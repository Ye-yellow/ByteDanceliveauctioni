package data

import (
	"context"
	"fmt"
	"strings"
	"time"

	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/biz/auction"
	userbiz "live-auction-bid/backend/app/auction/service/internal/biz/user"
	"live-auction-bid/backend/app/auction/service/internal/searchindex"
)

type lotSearchDocumentRow struct {
	AuctionLotModel  `gorm:"embedded"`
	RoomName         string
	RoomStatus       string
	MainUserStatus   int32
	MainHasOwnerRole bool
}

func (s *Store) ListLotSearchDocuments(ctx context.Context, updatedAfterMs int64, limit int) ([]searchindex.LotDocument, error) {
	if limit <= 0 {
		limit = 500
	}
	if limit > 1000 {
		limit = 1000
	}
	db := s.db.WithContext(ctx).
		Model(&AuctionLotModel{}).
		Select(`
			auction_lots.*,
			COALESCE(NULLIF(auction_users.nickname, ''), NULLIF(auction_users.username, ''), NULLIF(auction_rooms.name, ''), ?) AS room_name,
			COALESCE(auction_rooms.status, '') AS room_status,
			COALESCE(auction_users.status, 0) AS main_user_status,
			EXISTS (
				SELECT 1 FROM auction_user_roles
				WHERE auction_user_roles.user_id = auction_lots.main_account_id
				  AND auction_user_roles.role_code = ?
			) AS main_has_owner_role`, fallbackRoomName, userbiz.RoleMerchantOwner).
		Joins("LEFT JOIN auction_rooms ON auction_rooms.id = auction_lots.room_id AND auction_rooms.main_account_id = auction_lots.main_account_id").
		Joins("LEFT JOIN auction_users ON auction_users.id = auction_lots.main_account_id")
	if updatedAfterMs > 0 {
		db = db.Where("auction_lots.updated_at > ?", time.UnixMilli(updatedAfterMs))
	}
	var rows []lotSearchDocumentRow
	if err := db.Order("auction_lots.updated_at ASC").Order("auction_lots.id ASC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]searchindex.LotDocument, 0, len(rows))
	for i := range rows {
		doc, err := lotSearchDocumentFromRow(&rows[i])
		if err != nil {
			return nil, err
		}
		out = append(out, doc)
	}
	return out, nil
}

func lotSearchDocumentFromRow(row *lotSearchDocumentRow) (searchindex.LotDocument, error) {
	lot, err := modelToLot(&row.AuctionLotModel)
	if err != nil {
		return searchindex.LotDocument{}, err
	}
	price := searchPriceForLot(lot)
	publicVisible := row.RoomStatus == string(auction.RoomStatusActive) &&
		row.MainUserStatus == int32(v1.UserStatus_USER_STATUS_ACTIVE) &&
		row.MainHasOwnerRole &&
		auction.IsPublicVisibleLotStatus(lot.GetStatus())
	return searchindex.LotDocument{
		LotID:              lot.GetId(),
		RoomID:             lot.GetRoomId(),
		MainAccountID:      lot.GetMainAccountId(),
		Title:              lot.GetTitle(),
		SearchText:         buildLotSearchText(lot, row.RoomName, price),
		Status:             lot.GetStatus().String(),
		CurrentPrice:       price,
		Href:               "/m/room/" + lot.GetRoomId(),
		PublicVisible:      publicVisible,
		LotUpdatedAtUnixMs: modelTimeUnixMsOr(row.UpdatedAt, lot.GetUpdatedAtUnixMs()),
	}, nil
}

func searchPriceForLot(lot *v1.Lot) *v1.Money {
	if lot == nil {
		return nil
	}
	if price := lot.GetCurrentPrice(); price != nil && (price.GetAmount() > 0 || price.GetCurrency() != "") {
		return searchindex.CloneMoney(price)
	}
	if price := lot.GetRule().GetStartPrice(); price != nil {
		return searchindex.CloneMoney(price)
	}
	return nil
}

func buildLotSearchText(lot *v1.Lot, roomName string, price *v1.Money) string {
	if lot == nil {
		return ""
	}
	parts := []string{
		"直播间 " + strings.TrimSpace(roomName),
		"拍品 " + lot.GetTitle(),
		lot.GetDescription(),
		lot.GetCategory(),
	}
	if tags := strings.Join(lot.GetTags(), " "); tags != "" {
		parts = append(parts, "标签 "+tags)
	}
	if price != nil && price.GetAmount() > 0 {
		parts = append(parts, "当前价 "+formatSearchAmount(price.GetAmount(), price.GetCurrency()))
	}
	if estimate := lot.GetEstimatePrice(); estimate != nil && estimate.GetAmount() > 0 {
		parts = append(parts, "参考价 "+formatSearchAmount(estimate.GetAmount(), estimate.GetCurrency()))
	}
	return strings.Join(nonEmptyText(parts), "\n")
}

func nonEmptyText(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if text := strings.TrimSpace(value); text != "" {
			out = append(out, text)
		}
	}
	return out
}

func formatSearchAmount(amount int64, currency string) string {
	value := float64(amount) / 100
	if strings.TrimSpace(currency) == "" {
		currency = "CNY"
	}
	return strings.TrimRight(strings.TrimRight(strings.TrimSpace(fmt.Sprintf("%.2f", value)), "0"), ".") + " " + currency
}
