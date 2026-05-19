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
INSERT INTO auction_lots (
  id, room_id, title, description, image_url, status,
  start_price_amount, start_price_currency, min_increment_amount, min_increment_currency,
  duration_seconds, anti_snipe_window_seconds, anti_snipe_extend_seconds, max_extend_count,
  current_price_amount, current_price_currency, leading_user_id, leading_nickname,
  started_at_unix_ms, ends_at_unix_ms, settled_at_unix_ms,
  winner_user_id, winner_nickname, final_price_amount, final_price_currency,
  version, playbook_stage, payload
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		lot.Id, lot.RoomId, lot.Title, lot.Description, lot.ImageUrl, int32(lot.Status),
		lot.GetRule().GetStartPrice().GetAmount(), lot.GetRule().GetStartPrice().GetCurrency(),
		lot.GetRule().GetMinIncrement().GetAmount(), lot.GetRule().GetMinIncrement().GetCurrency(),
		lot.GetRule().GetDurationSeconds(), lot.GetRule().GetAntiSnipeWindowSeconds(),
		lot.GetRule().GetAntiSnipeExtendSeconds(), lot.GetRule().GetMaxExtendCount(),
		lot.GetCurrentPrice().GetAmount(), lot.GetCurrentPrice().GetCurrency(), lot.LeadingUserId, lot.LeadingNickname,
		lot.StartedAtUnixMs, lot.EndsAtUnixMs, lot.SettledAtUnixMs,
		lot.WinnerUserId, lot.WinnerNickname, lot.GetFinalPrice().GetAmount(), lot.GetFinalPrice().GetCurrency(),
		lot.Version, int32(lot.PlaybookStage), string(payload))
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
SET room_id = ?, title = ?, description = ?, image_url = ?, status = ?,
  start_price_amount = ?, start_price_currency = ?, min_increment_amount = ?, min_increment_currency = ?,
  duration_seconds = ?, anti_snipe_window_seconds = ?, anti_snipe_extend_seconds = ?, max_extend_count = ?,
  current_price_amount = ?, current_price_currency = ?, leading_user_id = ?, leading_nickname = ?,
  started_at_unix_ms = ?, ends_at_unix_ms = ?, settled_at_unix_ms = ?,
  winner_user_id = ?, winner_nickname = ?, final_price_amount = ?, final_price_currency = ?,
  version = ?, playbook_stage = ?, payload = ?
WHERE id = ?`,
		lot.RoomId, lot.Title, lot.Description, lot.ImageUrl, int32(lot.Status),
		lot.GetRule().GetStartPrice().GetAmount(), lot.GetRule().GetStartPrice().GetCurrency(),
		lot.GetRule().GetMinIncrement().GetAmount(), lot.GetRule().GetMinIncrement().GetCurrency(),
		lot.GetRule().GetDurationSeconds(), lot.GetRule().GetAntiSnipeWindowSeconds(),
		lot.GetRule().GetAntiSnipeExtendSeconds(), lot.GetRule().GetMaxExtendCount(),
		lot.GetCurrentPrice().GetAmount(), lot.GetCurrentPrice().GetCurrency(), lot.LeadingUserId, lot.LeadingNickname,
		lot.StartedAtUnixMs, lot.EndsAtUnixMs, lot.SettledAtUnixMs,
		lot.WinnerUserId, lot.WinnerNickname, lot.GetFinalPrice().GetAmount(), lot.GetFinalPrice().GetCurrency(),
		lot.Version, int32(lot.PlaybookStage), string(payload), lot.Id)
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
