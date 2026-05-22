-- Reference schema for the GORM AutoMigrate production path.
-- Runtime migrations are owned by app/auction/service/internal/data/models.go.

CREATE TABLE IF NOT EXISTS auction_lots (
  id VARCHAR(64) PRIMARY KEY,
  room_id VARCHAR(64) NOT NULL,
  title VARCHAR(255) NOT NULL,
  description TEXT NOT NULL,
  image_url VARCHAR(1024) NOT NULL,
  status INT NOT NULL,
  start_price_amount BIGINT NOT NULL,
  start_price_currency VARCHAR(16) NOT NULL,
  min_increment_amount BIGINT NOT NULL,
  min_increment_currency VARCHAR(16) NOT NULL,
  cap_price_amount BIGINT NULL,
  cap_price_currency VARCHAR(16) NULL,
  duration_seconds INT NOT NULL,
  anti_snipe_window_seconds INT NOT NULL,
  anti_snipe_extend_seconds INT NOT NULL,
  max_extend_count INT NOT NULL,
  current_price_amount BIGINT NOT NULL,
  current_price_currency VARCHAR(16) NOT NULL,
  leading_user_id VARCHAR(64) NOT NULL DEFAULT '',
  leading_nickname VARCHAR(128) NOT NULL DEFAULT '',
  started_at_unix_ms BIGINT NOT NULL DEFAULT 0,
  ends_at_unix_ms BIGINT NOT NULL DEFAULT 0,
  settled_at_unix_ms BIGINT NOT NULL DEFAULT 0,
  cancel_reason VARCHAR(512) NOT NULL DEFAULT '',
  cancelled_at_unix_ms BIGINT NOT NULL DEFAULT 0,
  winner_user_id VARCHAR(64) NOT NULL DEFAULT '',
  winner_nickname VARCHAR(128) NOT NULL DEFAULT '',
  final_price_amount BIGINT NOT NULL DEFAULT 0,
  final_price_currency VARCHAR(16) NOT NULL DEFAULT '',
  version BIGINT NOT NULL,
  playbook_stage INT NOT NULL,
  payload JSON NOT NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  INDEX idx_room_status (room_id, status),
  INDEX idx_room_updated (room_id, updated_at),
  INDEX idx_status_ends_at (status, ends_at_unix_ms)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS auction_bids (
  id VARCHAR(64) PRIMARY KEY,
  lot_id VARCHAR(64) NOT NULL,
  user_id VARCHAR(64) NOT NULL,
  nickname VARCHAR(128) NOT NULL,
  amount BIGINT NOT NULL,
  currency VARCHAR(16) NOT NULL,
  idempotency_key VARCHAR(128) NULL,
  created_at_unix_ms BIGINT NOT NULL,
  payload JSON NOT NULL,
  created_at DATETIME(3) NULL,
  INDEX idx_lot_created (lot_id, created_at_unix_ms),
  INDEX idx_lot_amount (lot_id, amount),
  INDEX idx_lot_user (lot_id, user_id),
  UNIQUE INDEX idx_lot_idem (lot_id, idempotency_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS auction_events (
  id VARCHAR(64) PRIMARY KEY,
  room_id VARCHAR(64) NOT NULL,
  lot_id VARCHAR(64) NOT NULL DEFAULT '',
  type INT NOT NULL,
  occurred_at_unix_ms BIGINT NOT NULL,
  reason VARCHAR(512) NOT NULL DEFAULT '',
  payload JSON NOT NULL,
  stream_id VARCHAR(64) NOT NULL DEFAULT '',
  streamed_at_unix_ms BIGINT NOT NULL DEFAULT 0,
  last_stream_error VARCHAR(512) NOT NULL DEFAULT '',
  created_at DATETIME(3) NULL,
  INDEX idx_room_occurred (room_id, occurred_at_unix_ms),
  INDEX idx_lot_occurred (lot_id, occurred_at_unix_ms),
  INDEX idx_type_occurred (type, occurred_at_unix_ms),
  INDEX idx_streamed_at (streamed_at_unix_ms)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS auction_users (
  id VARCHAR(64) PRIMARY KEY,
  username VARCHAR(64) NOT NULL,
  nickname VARCHAR(128) NOT NULL,
  password_hash VARCHAR(255) NOT NULL,
  role INT NOT NULL,
  created_at_unix_ms BIGINT NOT NULL,
  updated_at_unix_ms BIGINT NOT NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  UNIQUE INDEX idx_username (username),
  INDEX idx_role (role)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS auction_user_sessions (
  id VARCHAR(64) PRIMARY KEY,
  user_id VARCHAR(64) NOT NULL,
  refresh_token_hash VARCHAR(64) NOT NULL,
  refresh_expires_at_unix_ms BIGINT NOT NULL,
  revoked_at_unix_ms BIGINT NOT NULL DEFAULT 0,
  created_at_unix_ms BIGINT NOT NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  INDEX idx_user_sessions (user_id),
  UNIQUE INDEX idx_refresh_token_hash (refresh_token_hash),
  INDEX idx_session_expiry (refresh_expires_at_unix_ms)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;


CREATE TABLE IF NOT EXISTS asset_files (
  id VARCHAR(64) PRIMARY KEY,
  owner_user_id VARCHAR(64) NOT NULL,
  room_id VARCHAR(64) NOT NULL DEFAULT '',
  biz_type VARCHAR(64) NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'temporary',
  attached_lot_id VARCHAR(64) NOT NULL DEFAULT '',
  storage_provider VARCHAR(32) NOT NULL,
  bucket VARCHAR(128) NOT NULL,
  object_key VARCHAR(512) NOT NULL,
  public_url VARCHAR(1024) NOT NULL,
  original_name VARCHAR(255) NOT NULL DEFAULT '',
  mime_type VARCHAR(64) NOT NULL,
  size_bytes BIGINT NOT NULL,
  sha256 CHAR(64) NOT NULL,
  attached_at_unix_ms BIGINT NOT NULL DEFAULT 0,
  deleted_at_unix_ms BIGINT NOT NULL DEFAULT 0,
  expires_at_unix_ms BIGINT NOT NULL DEFAULT 0,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  INDEX idx_asset_owner (owner_user_id),
  INDEX idx_asset_room (room_id),
  INDEX idx_asset_biz_type (biz_type),
  INDEX idx_asset_status (status),
  INDEX idx_asset_attached_lot (attached_lot_id),
  INDEX idx_asset_sha256 (sha256),
  INDEX idx_asset_expiry (expires_at_unix_ms),
  INDEX idx_asset_deleted_at (deleted_at_unix_ms),
  UNIQUE INDEX idx_asset_object_key (object_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
