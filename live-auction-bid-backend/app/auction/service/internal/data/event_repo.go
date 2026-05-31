package data

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/encoding/protojson"
	"gorm.io/gorm"
	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/biz/auction"
)

func (s *Store) Publish(ctx context.Context, event v1.AuctionEvent) error {
	return s.PersistEvents(ctx, []v1.AuctionEvent{event})
}

func (s *Store) PersistEvents(ctx context.Context, events []v1.AuctionEvent) error {
	if len(events) == 0 {
		return nil
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return createEventModels(ctx, tx, events)
	}); err != nil {
		return err
	}
	return s.streamEvents(ctx, events)
}

func (s *Store) ListRoomEvents(ctx context.Context, query auction.RoomEventQuery) (auction.RoomEventList, error) {
	if strings.TrimSpace(query.RoomID) == "" {
		return auction.RoomEventList{}, errors.New("room id is required")
	}
	_, pageSize := auction.NormalizePagination(1, query.PageSize)
	offset := 0
	if token := strings.TrimSpace(query.PageToken); token != "" {
		nextOffset, err := strconv.Atoi(token)
		if err != nil || nextOffset < 0 {
			return auction.RoomEventList{}, errors.New("invalid page token")
		}
		offset = nextOffset
	}

	var models []AuctionEventModel
	db := s.db.WithContext(ctx).Where("room_id = ?", query.RoomID)
	if query.MainAccountID != "" {
		db = db.Where("main_account_id = ?", query.MainAccountID)
	}
	if err := db.
		Order("occurred_at_unix_ms DESC").
		Order("id DESC").
		Offset(offset).
		Limit(pageSize + 1).
		Find(&models).Error; err != nil {
		return auction.RoomEventList{}, err
	}

	hasNext := len(models) > pageSize
	if hasNext {
		models = models[:pageSize]
	}
	events := make([]*v1.AuctionEvent, 0, len(models))
	for i := range models {
		event := v1.AuctionEvent{}
		if err := protojson.Unmarshal([]byte(models[i].Payload), &event); err != nil {
			return auction.RoomEventList{}, err
		}
		events = append(events, &event)
	}

	nextPageToken := ""
	if hasNext {
		nextPageToken = strconv.Itoa(offset + pageSize)
	}
	return auction.RoomEventList{Events: events, NextPageToken: nextPageToken}, nil
}

func createEventModels(ctx context.Context, tx *gorm.DB, events []v1.AuctionEvent) error {
	for _, event := range events {
		model, err := eventToModel(event)
		if err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Create(model).Error; err != nil {
			return err
		}
	}
	return nil
}

func eventToModel(event v1.AuctionEvent) (*AuctionEventModel, error) {
	if event.Id == "" {
		return nil, errors.New("event id is required")
	}
	if event.Type == v1.AuctionEventType_AUCTION_EVENT_TYPE_UNSPECIFIED {
		return nil, errors.New("event type is required")
	}
	if event.RoomId == "" {
		return nil, errors.New("event room id is required")
	}
	if event.OccurredAtUnixMs == 0 {
		return nil, errors.New("event occurred time is required")
	}
	payload, err := protojson.Marshal(&event)
	if err != nil {
		return nil, err
	}
	return &AuctionEventModel{
		ID:               event.Id,
		MainAccountID:    event.GetMainAccountId(),
		RoomID:           event.RoomId,
		LotID:            event.LotId,
		Type:             int32(event.Type),
		OccurredAtUnixMs: event.OccurredAtUnixMs,
		Reason:           event.Reason,
		Payload:          string(payload),
	}, nil
}

func (s *Store) streamEvents(ctx context.Context, events []v1.AuctionEvent) error {
	for _, event := range events {
		payload, err := protojson.Marshal(&event)
		if err != nil {
			_ = s.markEventStreamError(ctx, event.Id, err)
			return err
		}
		streamID, err := s.writeEventToStream(ctx, event, string(payload))
		if err != nil {
			_ = s.markEventStreamError(ctx, event.Id, err)
			return err
		}
		if err := s.markEventStreamed(ctx, event.Id, streamID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) RepairEventStreamOutbox(ctx context.Context, limit int) error {
	if limit <= 0 {
		limit = 100
	}
	var models []AuctionEventModel
	if err := s.db.WithContext(ctx).
		Where("streamed_at_unix_ms = 0").
		Order("created_at ASC").
		Order("id ASC").
		Limit(limit).
		Find(&models).Error; err != nil {
		return err
	}
	for i := range models {
		event := v1.AuctionEvent{}
		if err := protojson.Unmarshal([]byte(models[i].Payload), &event); err != nil {
			_ = s.markEventStreamError(ctx, models[i].ID, err)
			return err
		}
		streamID, err := s.writeEventToStream(ctx, event, models[i].Payload)
		if err != nil {
			_ = s.markEventStreamError(ctx, models[i].ID, err)
			return err
		}
		if err := s.markEventStreamed(ctx, models[i].ID, streamID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) CountPendingEventOutbox(ctx context.Context) (int64, error) {
	var count int64
	if err := s.db.WithContext(ctx).
		Model(&AuctionEventModel{}).
		Where("streamed_at_unix_ms = 0").
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (s *Store) writeEventToStream(ctx context.Context, event v1.AuctionEvent, payload string) (string, error) {
	return s.redis.XAdd(ctx, &redis.XAddArgs{
		Stream: eventStreamKey(event.RoomId),
		ID:     "*",
		Values: map[string]any{
			"id":                  event.Id,
			"type":                strconv.Itoa(int(event.Type)),
			"lot_id":              event.LotId,
			"occurred_at_unix_ms": strconv.FormatInt(event.OccurredAtUnixMs, 10),
			"payload":             string(payload),
		},
	}).Result()
}

func (s *Store) markEventStreamed(ctx context.Context, eventID, streamID string) error {
	return s.db.WithContext(ctx).Model(&AuctionEventModel{}).
		Where("id = ?", eventID).
		Updates(map[string]any{
			"stream_id":           streamID,
			"streamed_at_unix_ms": time.Now().UnixMilli(),
			"last_stream_error":   "",
		}).Error
}

func (s *Store) markEventStreamError(ctx context.Context, eventID string, err error) error {
	message := err.Error()
	if len(message) > 512 {
		message = message[:512]
	}
	return s.db.WithContext(ctx).Model(&AuctionEventModel{}).
		Where("id = ?", eventID).
		Updates(map[string]any{"last_stream_error": message}).Error
}

func eventStreamKey(roomID string) string {
	return "auction:events:" + roomID
}
