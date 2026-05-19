package data

import (
	"context"
	"errors"

	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/encoding/protojson"
	v1 "live-auction-bid/backend/api/auction/service/v1"
)

func (s *Store) Append(ctx context.Context, bid v1.Bid) error {
	if bid.Id == "" {
		return errors.New("bid id is required")
	}
	if bid.LotId == "" {
		return errors.New("lot id is required")
	}
	if bid.UserId == "" {
		return errors.New("user id is required")
	}
	if bid.GetAmount() == nil || bid.GetAmount().GetCurrency() == "" {
		return errors.New("bid amount and currency are required")
	}
	payload, err := protojson.Marshal(&bid)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
INSERT INTO auction_bids (id, lot_id, user_id, amount, currency, created_at_unix_ms, payload)
VALUES (?, ?, ?, ?, ?, ?, ?)`, bid.Id, bid.LotId, bid.UserId, bid.GetAmount().GetAmount(), bid.GetAmount().GetCurrency(), bid.CreatedAtUnixMs, string(payload))
	return err
}

func (s *Store) ListByLot(ctx context.Context, lotID string) ([]v1.Bid, error) {
	if lotID == "" {
		return nil, errors.New("lot id is required")
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT payload
FROM auction_bids
WHERE lot_id = ?
ORDER BY created_at_unix_ms ASC, id ASC`, lotID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	bids := make([]v1.Bid, 0)
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		bid := v1.Bid{}
		if err := protojson.Unmarshal(payload, &bid); err != nil {
			return nil, err
		}
		bids = append(bids, bid)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return bids, nil
}

func (s *Store) FindByIdempotencyKey(ctx context.Context, lotID, key string) (v1.Bid, bool, error) {
	if lotID == "" {
		return v1.Bid{}, false, errors.New("lot id is required")
	}
	if key == "" {
		return v1.Bid{}, false, errors.New("idempotency key is required")
	}
	payload, err := s.redis.Get(ctx, idempotencyKey(lotID, key)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return v1.Bid{}, false, nil
		}
		return v1.Bid{}, false, err
	}
	bid := v1.Bid{}
	if err := protojson.Unmarshal(payload, &bid); err != nil {
		return v1.Bid{}, false, err
	}
	return bid, true, nil
}

func (s *Store) SaveIdempotencyKey(ctx context.Context, lotID, key string, bid v1.Bid) error {
	if lotID == "" {
		return errors.New("lot id is required")
	}
	if key == "" {
		return errors.New("idempotency key is required")
	}
	payload, err := protojson.Marshal(&bid)
	if err != nil {
		return err
	}
	return s.redis.Set(ctx, idempotencyKey(lotID, key), payload, 0).Err()
}

func idempotencyKey(lotID, key string) string {
	return "auction:idem:" + lotID + ":" + key
}
