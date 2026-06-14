CREATE TABLE IF NOT EXISTS user_delivery_addresses (
  id VARCHAR(64) PRIMARY KEY,
  user_id VARCHAR(64) NOT NULL,
  receiver_name VARCHAR(64) NOT NULL,
  phone VARCHAR(32) NOT NULL,
  province VARCHAR(64) NOT NULL DEFAULT '',
  city VARCHAR(64) NOT NULL DEFAULT '',
  district VARCHAR(64) NOT NULL DEFAULT '',
  street VARCHAR(128) NOT NULL DEFAULT '',
  detail VARCHAR(512) NOT NULL,
  postal_code VARCHAR(32) NOT NULL DEFAULT '',
  tag VARCHAR(32) NOT NULL DEFAULT '',
  is_default BOOLEAN NOT NULL DEFAULT FALSE,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  created_at_unix_ms BIGINT NOT NULL,
  updated_at_unix_ms BIGINT NOT NULL,
  deleted_at_unix_ms BIGINT NOT NULL DEFAULT 0,
  default_user_key VARCHAR(64)
    GENERATED ALWAYS AS (
      CASE WHEN status = 'active' AND is_default THEN user_id ELSE NULL END
    ) STORED,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  UNIQUE INDEX uk_one_default_address_per_user (default_user_key),
  INDEX idx_user_address_active (user_id, status, updated_at_unix_ms)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS auction_deposit_holds (
  id VARCHAR(64) PRIMARY KEY,
  main_account_id VARCHAR(64) NOT NULL,
  room_id VARCHAR(64) NOT NULL,
  lot_id VARCHAR(64) NOT NULL,
  buyer_user_id VARCHAR(64) NOT NULL,
  buyer_nickname VARCHAR(128) NOT NULL DEFAULT '',
  status VARCHAR(32) NOT NULL,
  amount BIGINT NOT NULL,
  currency VARCHAR(16) NOT NULL,
  payment_provider VARCHAR(32) NOT NULL DEFAULT 'mock',
  payment_id VARCHAR(64) NOT NULL DEFAULT '',
  idempotency_key VARCHAR(128) NOT NULL,
  address_id VARCHAR(64) NOT NULL DEFAULT '',
  address_snapshot JSON NULL,
  created_at_unix_ms BIGINT NOT NULL,
  updated_at_unix_ms BIGINT NOT NULL,
  held_at_unix_ms BIGINT NOT NULL DEFAULT 0,
  released_at_unix_ms BIGINT NOT NULL DEFAULT 0,
  payload JSON NOT NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  UNIQUE INDEX uk_deposit_lot_buyer (lot_id, buyer_user_id),
  UNIQUE INDEX uk_deposit_idem (lot_id, buyer_user_id, idempotency_key),
  INDEX idx_deposit_main_created (main_account_id, created_at_unix_ms),
  INDEX idx_deposit_room (room_id),
  INDEX idx_deposit_lot_status (lot_id, status),
  INDEX idx_deposit_buyer_status (buyer_user_id, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

ALTER TABLE shop_orders
  ADD COLUMN shipping_address_id VARCHAR(64) NOT NULL DEFAULT '',
  ADD COLUMN shipping_address_snapshot JSON NULL,
  ADD INDEX idx_shop_order_shipping_address (shipping_address_id);

ALTER TABLE auction_orders
  ADD COLUMN shipping_address_id VARCHAR(64) NOT NULL DEFAULT '',
  ADD COLUMN shipping_address_snapshot JSON NULL,
  ADD INDEX idx_order_shipping_address (shipping_address_id);
