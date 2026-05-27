-- Lot stats read model for high-concurrency auction runtime projection.

CREATE TABLE IF NOT EXISTS auction_lot_stats (
  lot_id VARCHAR(64) NOT NULL,
  room_id VARCHAR(64) NOT NULL,
  bid_count BIGINT NOT NULL DEFAULT 0,
  participant_count BIGINT NOT NULL DEFAULT 0,
  last_bid_id VARCHAR(64) NOT NULL DEFAULT '',
  last_bid_at_unix_ms BIGINT NOT NULL DEFAULT 0,
  projected_version BIGINT NOT NULL DEFAULT 0,
  updated_at_unix_ms BIGINT NOT NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  PRIMARY KEY (lot_id),
  KEY idx_lot_stats_room (room_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS auction_lot_participants (
  lot_id VARCHAR(64) NOT NULL,
  user_id VARCHAR(64) NOT NULL,
  room_id VARCHAR(64) NOT NULL,
  first_bid_id VARCHAR(64) NOT NULL,
  first_bid_at_unix_ms BIGINT NOT NULL,
  created_at DATETIME(3) NULL,
  PRIMARY KEY (lot_id, user_id),
  KEY idx_lot_participants_room (room_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

INSERT IGNORE INTO auction_lot_participants (
  lot_id,
  user_id,
  room_id,
  first_bid_id,
  first_bid_at_unix_ms,
  created_at
)
SELECT
  b.lot_id,
  b.user_id,
  COALESCE(l.room_id, ''),
  MIN(b.id),
  MIN(b.created_at_unix_ms),
  NOW(3)
FROM auction_bids b
LEFT JOIN auction_lots l ON l.id = b.lot_id
WHERE b.lot_id <> '' AND b.user_id <> ''
GROUP BY b.lot_id, b.user_id, l.room_id;

INSERT INTO auction_lot_stats (
  lot_id,
  room_id,
  bid_count,
  participant_count,
  last_bid_id,
  last_bid_at_unix_ms,
  projected_version,
  updated_at_unix_ms,
  created_at,
  updated_at
)
SELECT
  b.lot_id,
  COALESCE(l.room_id, ''),
  COUNT(*) AS bid_count,
  COUNT(DISTINCT b.user_id) AS participant_count,
  SUBSTRING_INDEX(GROUP_CONCAT(b.id ORDER BY b.created_at_unix_ms DESC, b.id DESC), ',', 1) AS last_bid_id,
  MAX(b.created_at_unix_ms) AS last_bid_at_unix_ms,
  COALESCE(MAX(l.version), 0) AS projected_version,
  MAX(b.created_at_unix_ms) AS updated_at_unix_ms,
  NOW(3),
  NOW(3)
FROM auction_bids b
LEFT JOIN auction_lots l ON l.id = b.lot_id
WHERE b.lot_id <> ''
GROUP BY b.lot_id, l.room_id
ON DUPLICATE KEY UPDATE
  room_id = VALUES(room_id),
  bid_count = VALUES(bid_count),
  participant_count = VALUES(participant_count),
  last_bid_id = VALUES(last_bid_id),
  last_bid_at_unix_ms = VALUES(last_bid_at_unix_ms),
  projected_version = GREATEST(projected_version, VALUES(projected_version)),
  updated_at_unix_ms = GREATEST(updated_at_unix_ms, VALUES(updated_at_unix_ms)),
  updated_at = NOW(3);
