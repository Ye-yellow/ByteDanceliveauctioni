package data

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/biz/auction"
	userbiz "live-auction-bid/backend/app/auction/service/internal/biz/user"
	"live-auction-bid/backend/app/auction/service/internal/pkg/idgen"
)

const fallbackRoomName = "直播间"

func (s *Store) EnsureDefaultRoom(ctx context.Context, mainAccountID, createdByUserID string, nowMs int64) (*auction.Room, error) {
	mainAccountID = strings.TrimSpace(mainAccountID)
	if mainAccountID == "" {
		return nil, errors.New("main account id is required")
	}
	if nowMs <= 0 {
		return nil, errors.New("now ms is required")
	}
	var model AuctionRoomModel
	err := s.db.WithContext(ctx).
		Where("main_account_id = ?", mainAccountID).
		Order("created_at_unix_ms ASC").
		Order("id ASC").
		First(&model).Error
	if err == nil {
		if err := ensureRoomStateRecord(ctx, s.db, model.ID, model.MainAccountID, nowMs); err != nil {
			return nil, err
		}
		room, _, err := s.roomFromModelWithProfile(ctx, &model, false)
		return room, err
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	room := auction.Room{
		ID:              idgen.New("room"),
		MainAccountID:   mainAccountID,
		Name:            fallbackRoomName,
		Platform:        "douyin",
		Status:          auction.RoomStatusActive,
		CreatedByUserID: strings.TrimSpace(createdByUserID),
		CreatedAtUnixMs: nowMs,
		UpdatedAtUnixMs: nowMs,
	}
	model = *roomToModel(room)
	if err := s.db.WithContext(ctx).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "main_account_id"}}, DoNothing: true}).
		Create(&model).Error; err != nil {
		return nil, err
	}
	err = s.db.WithContext(ctx).
		Where("main_account_id = ?", mainAccountID).
		Order("created_at_unix_ms ASC").
		Order("id ASC").
		First(&model).Error
	if err != nil {
		return nil, err
	}
	if err := ensureRoomStateRecord(ctx, s.db, model.ID, model.MainAccountID, nowMs); err != nil {
		return nil, err
	}
	roomOut, _, err := s.roomFromModelWithProfile(ctx, &model, false)
	return roomOut, err
}

func (s *Store) ListRooms(ctx context.Context, query auction.RoomQuery) ([]auction.Room, error) {
	db := s.db.WithContext(ctx).Model(&AuctionRoomModel{})
	if mainAccountID := strings.TrimSpace(query.MainAccountID); mainAccountID != "" {
		db = db.Where("main_account_id = ?", mainAccountID)
	}
	if query.PublicOnly {
		db = db.Where("status = ?", string(auction.RoomStatusActive))
	}
	if query.PublicVisibleOnly {
		db = db.Where(
			"EXISTS (SELECT 1 FROM auction_lots WHERE auction_lots.room_id = auction_rooms.id AND auction_lots.main_account_id = auction_rooms.main_account_id AND auction_lots.status IN ?)",
			publicVisibleLotStatusValues(),
		)
	}
	var models []AuctionRoomModel
	if err := db.Order("created_at_unix_ms ASC").Order("id ASC").Find(&models).Error; err != nil {
		return nil, err
	}
	rooms := make([]auction.Room, 0, len(models))
	for i := range models {
		room, visible, err := s.roomFromModelWithProfile(ctx, &models[i], query.PublicOnly)
		if err != nil {
			return nil, err
		}
		if visible {
			rooms = append(rooms, *room)
		}
	}
	return rooms, nil
}

func publicVisibleLotStatusValues() []int32 {
	statuses := auction.PublicVisibleLotStatuses()
	values := make([]int32, 0, len(statuses))
	for _, status := range statuses {
		values = append(values, int32(status))
	}
	return values
}

func (s *Store) FindRoomByID(ctx context.Context, roomID string) (*auction.Room, bool, error) {
	roomID = strings.TrimSpace(roomID)
	if roomID == "" {
		return nil, false, errors.New("room id is required")
	}
	var model AuctionRoomModel
	if err := s.db.WithContext(ctx).Where("id = ?", roomID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	room, visible, err := s.roomFromModelWithProfile(ctx, &model, false)
	return room, visible, err
}

func roomToModel(room auction.Room) *AuctionRoomModel {
	return &AuctionRoomModel{
		ID:              strings.TrimSpace(room.ID),
		MainAccountID:   strings.TrimSpace(room.MainAccountID),
		Name:            strings.TrimSpace(room.Name),
		Platform:        strings.TrimSpace(room.Platform),
		PlatformRoomID:  strings.TrimSpace(room.PlatformRoomID),
		Status:          string(room.Status),
		CreatedByUserID: strings.TrimSpace(room.CreatedByUserID),
		CreatedAtUnixMs: room.CreatedAtUnixMs,
		UpdatedAtUnixMs: room.UpdatedAtUnixMs,
	}
}

func roomFromModel(model *AuctionRoomModel) *auction.Room {
	if model == nil {
		return nil
	}
	return &auction.Room{
		ID:              model.ID,
		MainAccountID:   model.MainAccountID,
		Name:            model.Name,
		Platform:        model.Platform,
		PlatformRoomID:  model.PlatformRoomID,
		Status:          auction.RoomStatus(model.Status),
		CreatedByUserID: model.CreatedByUserID,
		CreatedAtUnixMs: model.CreatedAtUnixMs,
		UpdatedAtUnixMs: model.UpdatedAtUnixMs,
	}
}

func (s *Store) roomFromModelWithProfile(ctx context.Context, model *AuctionRoomModel, publicOnly bool) (*auction.Room, bool, error) {
	room := roomFromModel(model)
	if room == nil {
		return nil, false, nil
	}
	var main AuctionUserModel
	err := s.db.WithContext(ctx).Where("id = ?", room.MainAccountID).First(&main).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return room, !publicOnly, nil
		}
		return nil, false, err
	}
	if main.Status != int32(v1.UserStatus_USER_STATUS_ACTIVE) {
		return room, !publicOnly, nil
	}
	var roleCount int64
	if err := s.db.WithContext(ctx).Model(&AuctionUserRoleModel{}).
		Where("user_id = ? AND role_code = ?", main.ID, userbiz.RoleMerchantOwner).
		Count(&roleCount).Error; err != nil {
		return nil, false, err
	}
	if roleCount == 0 {
		return room, !publicOnly, nil
	}
	room.Name = mainAccountRoomName(&main)
	return room, true, nil
}

func mainAccountRoomName(main *AuctionUserModel) string {
	if main == nil {
		return fallbackRoomName
	}
	if nickname := strings.TrimSpace(main.Nickname); nickname != "" {
		return nickname
	}
	if username := strings.TrimSpace(main.Username); username != "" {
		return username
	}
	return fallbackRoomName
}
