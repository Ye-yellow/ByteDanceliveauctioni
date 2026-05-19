package data

import (
	"context"
	"database/sql"
	"errors"

	"google.golang.org/protobuf/encoding/protojson"
	v1 "live-auction-bid/backend/api/auction/service/v1"
)

func (s *Store) Create(ctx context.Context, lot *v1.Lot) error {
	if lot == nil {
		return errors.New("lot is required")
	}
	payload, err := protojson.Marshal(lot)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
INSERT INTO auction_lots (id, room_id, status, payload)
VALUES (?, ?, ?, ?)`, lot.Id, lot.RoomId, int32(lot.Status), string(payload))
	return err
}

func (s *Store) Save(ctx context.Context, lot *v1.Lot) error {
	if lot == nil {
		return errors.New("lot is required")
	}
	payload, err := protojson.Marshal(lot)
	if err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, `
UPDATE auction_lots
SET room_id = ?, status = ?, payload = ?
WHERE id = ?`, lot.RoomId, int32(lot.Status), string(payload), lot.Id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return errors.New("lot not found")
	}
	return nil
}

func (s *Store) FindByID(ctx context.Context, lotID string) (*v1.Lot, error) {
	if lotID == "" {
		return nil, errors.New("lot id is required")
	}
	var payload []byte
	if err := s.db.QueryRowContext(ctx, `
SELECT payload
FROM auction_lots
WHERE id = ?`, lotID).Scan(&payload); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("lot not found")
		}
		return nil, err
	}
	lot := &v1.Lot{}
	if err := protojson.Unmarshal(payload, lot); err != nil {
		return nil, err
	}
	return lot, nil
}

func (s *Store) List(ctx context.Context, roomID string, status v1.LotStatus) ([]*v1.Lot, error) {
	if roomID == "" {
		return nil, errors.New("room id is required")
	}
	query := `
SELECT payload
FROM auction_lots
WHERE room_id = ?`
	args := []any{roomID}
	if status != 0 {
		query += ` AND status = ?`
		args = append(args, int32(status))
	}
	query += ` ORDER BY updated_at DESC, id ASC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lots := make([]*v1.Lot, 0)
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		lot := &v1.Lot{}
		if err := protojson.Unmarshal(payload, lot); err != nil {
			return nil, err
		}
		lots = append(lots, lot)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return lots, nil
}
