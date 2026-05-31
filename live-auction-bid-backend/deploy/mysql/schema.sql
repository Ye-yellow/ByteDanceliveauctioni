-- Reference schema for the GORM AutoMigrate production path.
-- Runtime migrations are owned by app/auction/service/internal/data/models.go.

CREATE TABLE IF NOT EXISTS auction_rooms (
  id VARCHAR(64) PRIMARY KEY,
  main_account_id VARCHAR(64) NOT NULL,
  name VARCHAR(128) NOT NULL,
  platform VARCHAR(32) NOT NULL DEFAULT 'douyin',
  platform_room_id VARCHAR(128) NOT NULL DEFAULT '',
  status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE',
  created_by_user_id VARCHAR(64) NOT NULL DEFAULT '',
  created_at_unix_ms BIGINT NOT NULL,
  updated_at_unix_ms BIGINT NOT NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  UNIQUE KEY uidx_room_main_account (main_account_id),
  INDEX idx_room_main_status (main_account_id, status),
  INDEX idx_room_created_by (created_by_user_id),
  INDEX idx_platform_room (platform_room_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS auction_lots (
  id VARCHAR(64) PRIMARY KEY,
  main_account_id VARCHAR(64) NOT NULL,
  room_id VARCHAR(64) NOT NULL,
  title VARCHAR(255) NOT NULL,
  description TEXT NOT NULL,
  image_url VARCHAR(1024) NOT NULL,
  status INT NOT NULL,
  queue_status INT NOT NULL DEFAULT 1,
  queue_position INT NOT NULL DEFAULT 0,
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
  INDEX idx_lot_main_room_status (main_account_id, room_id, status),
  INDEX idx_lot_main_room_queue (main_account_id, room_id, queue_status, queue_position),
  INDEX idx_lot_main_updated (main_account_id, updated_at),
  INDEX idx_room_status (room_id, status),
  INDEX idx_room_queue (room_id, queue_status, queue_position),
  INDEX idx_room_updated (room_id, updated_at),
  INDEX idx_status_ends_at (status, ends_at_unix_ms)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS auction_bids (
  id VARCHAR(64) PRIMARY KEY,
  main_account_id VARCHAR(64) NOT NULL,
  lot_id VARCHAR(64) NOT NULL,
  user_id VARCHAR(64) NOT NULL,
  nickname VARCHAR(128) NOT NULL,
  amount BIGINT NOT NULL,
  currency VARCHAR(16) NOT NULL,
  idempotency_key VARCHAR(128) NOT NULL,
  created_at_unix_ms BIGINT NOT NULL,
  payload JSON NOT NULL,
  created_at DATETIME(3) NULL,
  INDEX idx_bid_main_lot_created (main_account_id, lot_id, created_at_unix_ms),
  INDEX idx_lot_created (lot_id, created_at_unix_ms),
  INDEX idx_lot_amount (lot_id, amount),
  INDEX idx_lot_user (lot_id, user_id),
  UNIQUE INDEX idx_lot_user_idem (lot_id, user_id, idempotency_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS auction_lot_stats (
  lot_id VARCHAR(64) PRIMARY KEY,
  main_account_id VARCHAR(64) NOT NULL,
  room_id VARCHAR(64) NOT NULL,
  bid_count BIGINT NOT NULL DEFAULT 0,
  participant_count BIGINT NOT NULL DEFAULT 0,
  last_bid_id VARCHAR(64) NOT NULL DEFAULT '',
  last_bid_at_unix_ms BIGINT NOT NULL DEFAULT 0,
  projected_version BIGINT NOT NULL DEFAULT 0,
  updated_at_unix_ms BIGINT NOT NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  INDEX idx_lot_stats_room (room_id),
  INDEX idx_lot_stats_main_room (main_account_id, room_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS auction_lot_participants (
  lot_id VARCHAR(64) NOT NULL,
  user_id VARCHAR(64) NOT NULL,
  main_account_id VARCHAR(64) NOT NULL,
  room_id VARCHAR(64) NOT NULL,
  first_bid_id VARCHAR(64) NOT NULL,
  first_bid_at_unix_ms BIGINT NOT NULL,
  created_at DATETIME(3) NULL,
  PRIMARY KEY (lot_id, user_id),
  INDEX idx_lot_participants_room (room_id),
  INDEX idx_lot_participants_main_room (main_account_id, room_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS auction_events (
  id VARCHAR(64) PRIMARY KEY,
  main_account_id VARCHAR(64) NOT NULL,
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
  INDEX idx_event_main_room_occurred (main_account_id, room_id, occurred_at_unix_ms),
  INDEX idx_room_occurred (room_id, occurred_at_unix_ms),
  INDEX idx_lot_occurred (lot_id, occurred_at_unix_ms),
  INDEX idx_type_occurred (type, occurred_at_unix_ms),
  INDEX idx_streamed_at (streamed_at_unix_ms)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS auction_orders (
  id VARCHAR(64) PRIMARY KEY,
  main_account_id VARCHAR(64) NOT NULL,
  lot_id VARCHAR(64) NOT NULL,
  room_id VARCHAR(64) NOT NULL,
  lot_title VARCHAR(255) NOT NULL,
  lot_image_url VARCHAR(1024) NOT NULL,
  buyer_user_id VARCHAR(64) NOT NULL,
  buyer_nickname VARCHAR(128) NOT NULL,
  status VARCHAR(32) NOT NULL,
  payment_status VARCHAR(32) NOT NULL,
  payment_id VARCHAR(64) NOT NULL DEFAULT '',
  amount BIGINT NOT NULL,
  currency VARCHAR(16) NOT NULL,
  created_at_unix_ms BIGINT NOT NULL,
  updated_at_unix_ms BIGINT NOT NULL,
  expires_at_unix_ms BIGINT NOT NULL,
  paid_at_unix_ms BIGINT NOT NULL DEFAULT 0,
  version BIGINT NOT NULL,
  payload JSON NOT NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  UNIQUE INDEX uk_lot_order (lot_id),
  INDEX idx_order_main_room_created (main_account_id, room_id, created_at_unix_ms),
  INDEX idx_order_main_status (main_account_id, status),
  INDEX idx_order_room_created (room_id, created_at_unix_ms),
  INDEX idx_order_buyer_status (buyer_user_id, status),
  INDEX idx_order_payment_status (payment_status),
  INDEX idx_order_expiry (expires_at_unix_ms)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS auction_payments (
  id VARCHAR(64) PRIMARY KEY,
  main_account_id VARCHAR(64) NOT NULL,
  order_id VARCHAR(64) NOT NULL,
  lot_id VARCHAR(64) NOT NULL,
  buyer_user_id VARCHAR(64) NOT NULL,
  status VARCHAR(32) NOT NULL,
  amount BIGINT NOT NULL,
  currency VARCHAR(16) NOT NULL,
  idempotency_key VARCHAR(128) NULL,
  created_at_unix_ms BIGINT NOT NULL,
  updated_at_unix_ms BIGINT NOT NULL,
  succeeded_at_unix_ms BIGINT NOT NULL DEFAULT 0,
  payload JSON NOT NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  INDEX idx_payment_main_created (main_account_id, created_at_unix_ms),
  INDEX idx_payment_order (order_id),
  INDEX idx_payment_lot (lot_id),
  INDEX idx_payment_buyer (buyer_user_id),
  INDEX idx_payment_status (status),
  INDEX idx_payment_created (created_at_unix_ms),
  UNIQUE INDEX uk_order_payment_idem (order_id, idempotency_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS auction_users (
  id VARCHAR(64) PRIMARY KEY,
  username VARCHAR(64) NOT NULL,
  nickname VARCHAR(128) NOT NULL,
  password_hash VARCHAR(255) NOT NULL,
  main_account_id VARCHAR(64) NOT NULL DEFAULT '',
  created_by_user_id VARCHAR(64) NOT NULL DEFAULT '',
  status INT NOT NULL DEFAULT 1,
  created_at_unix_ms BIGINT NOT NULL,
  updated_at_unix_ms BIGINT NOT NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  UNIQUE INDEX idx_username (username),
  INDEX idx_user_main_status (main_account_id, status),
  INDEX idx_user_created_by (created_by_user_id),
  INDEX idx_user_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS auction_roles (
  code VARCHAR(64) PRIMARY KEY,
  name VARCHAR(128) NOT NULL,
  description VARCHAR(512) NOT NULL DEFAULT '',
  system BOOLEAN NOT NULL DEFAULT TRUE,
  created_at_unix_ms BIGINT NOT NULL,
  updated_at_unix_ms BIGINT NOT NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS auction_permissions (
  code VARCHAR(96) PRIMARY KEY,
  name VARCHAR(128) NOT NULL,
  module VARCHAR(64) NOT NULL,
  description VARCHAR(512) NOT NULL DEFAULT '',
  created_at_unix_ms BIGINT NOT NULL,
  updated_at_unix_ms BIGINT NOT NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  INDEX idx_permission_module (module)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS auction_user_roles (
  id VARCHAR(64) PRIMARY KEY,
  user_id VARCHAR(64) NOT NULL,
  role_code VARCHAR(64) NOT NULL,
  main_account_id VARCHAR(64) NOT NULL DEFAULT '',
  granted_by_user_id VARCHAR(64) NOT NULL DEFAULT '',
  created_at_unix_ms BIGINT NOT NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  UNIQUE INDEX uk_user_role_scope (user_id, role_code, main_account_id),
  INDEX idx_user_role_user (user_id),
  INDEX idx_user_role_role (role_code),
  INDEX idx_user_role_main (main_account_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS auction_role_permissions (
  id VARCHAR(64) PRIMARY KEY,
  role_code VARCHAR(64) NOT NULL,
  permission_code VARCHAR(96) NOT NULL,
  created_at_unix_ms BIGINT NOT NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  UNIQUE INDEX uk_role_permission (role_code, permission_code),
  INDEX idx_role_permission_role (role_code),
  INDEX idx_role_permission_permission (permission_code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS auction_user_permissions (
  id VARCHAR(64) PRIMARY KEY,
  user_id VARCHAR(64) NOT NULL,
  permission_code VARCHAR(96) NOT NULL,
  effect VARCHAR(16) NOT NULL DEFAULT 'allow',
  granted_by_user_id VARCHAR(64) NOT NULL DEFAULT '',
  created_at_unix_ms BIGINT NOT NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  UNIQUE INDEX uk_user_permission (user_id, permission_code),
  INDEX idx_user_permission_user (user_id),
  INDEX idx_user_permission_permission (permission_code)
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
  main_account_id VARCHAR(64) NOT NULL DEFAULT '',
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
  INDEX idx_asset_main_room (main_account_id, room_id),
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
