-- Enforce one active LIVE/EXTENDED lot per room.

DROP PROCEDURE IF EXISTS migrate_20260531_one_active_lot_per_room;

DELIMITER $$
CREATE PROCEDURE migrate_20260531_one_active_lot_per_room()
BEGIN
  SET @now_ms = CAST(UNIX_TIMESTAMP(CURRENT_TIMESTAMP(3)) * 1000 AS UNSIGNED);

  CREATE TABLE IF NOT EXISTS auction_room_states (
    room_id VARCHAR(64) PRIMARY KEY,
    main_account_id VARCHAR(64) NOT NULL,
    active_lot_id VARCHAR(64) NOT NULL DEFAULT '',
    active_lot_version BIGINT NOT NULL DEFAULT 0,
    next_queue_position INT NOT NULL DEFAULT 1,
    updated_at_unix_ms BIGINT NOT NULL,
    created_at DATETIME(3) NULL,
    updated_at DATETIME(3) NULL,
    INDEX idx_room_state_main (main_account_id)
  ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

  INSERT INTO auction_room_states (
    room_id,
    main_account_id,
    active_lot_id,
    active_lot_version,
    next_queue_position,
    updated_at_unix_ms,
    created_at,
    updated_at
  )
  SELECT
    r.id,
    r.main_account_id,
    COALESCE((
      SELECT l.id
      FROM auction_lots l
      WHERE l.room_id = r.id
        AND l.status IN (2, 7)
      ORDER BY l.updated_at DESC, l.id ASC
      LIMIT 1
    ), ''),
    COALESCE((
      SELECT l.version
      FROM auction_lots l
      WHERE l.room_id = r.id
        AND l.status IN (2, 7)
      ORDER BY l.updated_at DESC, l.id ASC
      LIMIT 1
    ), 0),
    COALESCE((
      SELECT MAX(NULLIF(l.queue_position, 0)) + 1
      FROM auction_lots l
      WHERE l.room_id = r.id
    ), 1),
    @now_ms,
    NOW(3),
    NOW(3)
  FROM auction_rooms r
  ON DUPLICATE KEY UPDATE
    main_account_id = VALUES(main_account_id),
    updated_at_unix_ms = VALUES(updated_at_unix_ms),
    updated_at = NOW(3);

  UPDATE auction_lots l
  JOIN auction_room_states s ON s.room_id = l.room_id
  SET
    l.status = 4,
    l.cancel_reason = CASE
      WHEN l.cancel_reason IS NULL OR l.cancel_reason = '' THEN '系统修复：同一直播间存在多个竞拍中拍品，保留最新一件'
      ELSE l.cancel_reason
    END,
    l.cancelled_at_unix_ms = CASE
      WHEN l.cancelled_at_unix_ms IS NULL OR l.cancelled_at_unix_ms = 0 THEN @now_ms
      ELSE l.cancelled_at_unix_ms
    END,
    l.version = l.version + 1,
    l.payload = JSON_SET(
      l.payload,
      '$.status', 'LOT_STATUS_CANCELLED',
      '$.cancelReason', CASE
        WHEN JSON_UNQUOTE(JSON_EXTRACT(l.payload, '$.cancelReason')) IS NULL
          OR JSON_UNQUOTE(JSON_EXTRACT(l.payload, '$.cancelReason')) = ''
          THEN '系统修复：同一直播间存在多个竞拍中拍品，保留最新一件'
        ELSE JSON_UNQUOTE(JSON_EXTRACT(l.payload, '$.cancelReason'))
      END,
      '$.cancelledAtUnixMs', @now_ms,
      '$.version', l.version + 1
    )
  WHERE l.status IN (2, 7)
    AND s.active_lot_id <> ''
    AND l.id <> s.active_lot_id;

  IF NOT EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema = DATABASE()
      AND table_name = 'auction_lots'
      AND column_name = 'active_room_key'
  ) THEN
    ALTER TABLE auction_lots
      ADD COLUMN active_room_key VARCHAR(64)
      GENERATED ALWAYS AS (
        CASE
          WHEN status IN (2, 7) THEN room_id
          ELSE NULL
        END
      ) STORED;
  END IF;

  IF NOT EXISTS (
    SELECT 1
    FROM information_schema.statistics
    WHERE table_schema = DATABASE()
      AND table_name = 'auction_lots'
      AND index_name = 'uidx_one_active_lot_per_room'
  ) THEN
    ALTER TABLE auction_lots ADD UNIQUE INDEX uidx_one_active_lot_per_room (active_room_key);
  END IF;
END$$
DELIMITER ;

CALL migrate_20260531_one_active_lot_per_room();
DROP PROCEDURE IF EXISTS migrate_20260531_one_active_lot_per_room;
