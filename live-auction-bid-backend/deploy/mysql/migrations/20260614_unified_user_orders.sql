SET NAMES utf8mb4;

CREATE TABLE IF NOT EXISTS user_orders (
  id VARCHAR(64) PRIMARY KEY,
  source VARCHAR(32) NOT NULL,
  source_order_id VARCHAR(64) NOT NULL,
  order_no VARCHAR(64) NOT NULL DEFAULT '',
  main_account_id VARCHAR(64) NOT NULL DEFAULT '',
  user_id VARCHAR(64) NOT NULL,
  nickname VARCHAR(128) NOT NULL DEFAULT '',
  status VARCHAR(32) NOT NULL,
  payment_status VARCHAR(32) NOT NULL,
  payment_id VARCHAR(64) NOT NULL DEFAULT '',
  title VARCHAR(255) NOT NULL DEFAULT '',
  shop_name VARCHAR(128) NOT NULL DEFAULT '',
  total_amount BIGINT NOT NULL,
  currency VARCHAR(16) NOT NULL DEFAULT 'CNY',
  shipping_address_id VARCHAR(64) NOT NULL DEFAULT '',
  shipping_address_snapshot JSON NULL,
  address_snapshot VARCHAR(512) NOT NULL DEFAULT '',
  created_at_unix_ms BIGINT NOT NULL,
  updated_at_unix_ms BIGINT NOT NULL,
  paid_at_unix_ms BIGINT NOT NULL DEFAULT 0,
  expires_at_unix_ms BIGINT NOT NULL DEFAULT 0,
  version BIGINT NOT NULL DEFAULT 1,
  payment_idempotency_key VARCHAR(128) NOT NULL DEFAULT '',
  source_payload JSON NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  UNIQUE INDEX uk_user_order_source_id (source, source_order_id),
  INDEX idx_user_order_user_status (user_id, status, source),
  INDEX idx_user_order_payment_status (payment_status),
  INDEX idx_user_order_no (order_no),
  INDEX idx_user_order_main_created (main_account_id, created_at_unix_ms),
  INDEX idx_user_order_source_created (source, created_at_unix_ms),
  INDEX idx_user_order_shipping_address (shipping_address_id),
  INDEX idx_user_order_created (created_at_unix_ms),
  INDEX idx_user_order_expiry (expires_at_unix_ms)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS user_order_items (
  id VARCHAR(64) PRIMARY KEY,
  order_id VARCHAR(64) NOT NULL,
  source VARCHAR(32) NOT NULL,
  source_item_id VARCHAR(64) NOT NULL DEFAULT '',
  product_id VARCHAR(64) NOT NULL DEFAULT '',
  sku_id VARCHAR(64) NOT NULL DEFAULT '',
  lot_id VARCHAR(64) NOT NULL DEFAULT '',
  room_id VARCHAR(64) NOT NULL DEFAULT '',
  title VARCHAR(255) NOT NULL DEFAULT '',
  image_url VARCHAR(1024) NOT NULL DEFAULT '',
  sku_name VARCHAR(128) NOT NULL DEFAULT '',
  quantity BIGINT NOT NULL DEFAULT 1,
  unit_amount BIGINT NOT NULL,
  total_amount BIGINT NOT NULL,
  currency VARCHAR(16) NOT NULL DEFAULT 'CNY',
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  INDEX idx_user_order_item_order (order_id),
  INDEX idx_user_order_item_source (source),
  INDEX idx_user_order_item_product (product_id),
  INDEX idx_user_order_item_lot (lot_id),
  INDEX idx_user_order_item_room (room_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS user_order_payments (
  id VARCHAR(64) PRIMARY KEY,
  order_id VARCHAR(64) NOT NULL,
  source VARCHAR(32) NOT NULL,
  provider VARCHAR(32) NOT NULL DEFAULT 'mock',
  main_account_id VARCHAR(64) NOT NULL DEFAULT '',
  lot_id VARCHAR(64) NOT NULL DEFAULT '',
  user_id VARCHAR(64) NOT NULL,
  status VARCHAR(32) NOT NULL,
  amount BIGINT NOT NULL,
  currency VARCHAR(16) NOT NULL DEFAULT 'CNY',
  idempotency_key VARCHAR(128) NOT NULL,
  created_at_unix_ms BIGINT NOT NULL,
  updated_at_unix_ms BIGINT NOT NULL DEFAULT 0,
  succeeded_at_unix_ms BIGINT NOT NULL DEFAULT 0,
  source_payload JSON NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  UNIQUE INDEX uk_user_order_payment_idem (order_id, idempotency_key),
  INDEX idx_user_order_payment_order (order_id),
  INDEX idx_user_order_payment_source (source),
  INDEX idx_user_order_payment_main_created (main_account_id, created_at_unix_ms),
  INDEX idx_user_order_payment_lot (lot_id),
  INDEX idx_user_order_payment_user (user_id),
  INDEX idx_user_order_payment_status (status),
  INDEX idx_user_order_payment_created (created_at_unix_ms)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

INSERT IGNORE INTO user_orders (
  id, source, source_order_id, order_no, main_account_id, user_id, nickname,
  status, payment_status, payment_id, title, shop_name, total_amount, currency,
  shipping_address_id, shipping_address_snapshot, created_at_unix_ms, updated_at_unix_ms,
  paid_at_unix_ms, expires_at_unix_ms, version, source_payload, created_at, updated_at
)
SELECT
  id,
  'auction',
  id,
  id,
  main_account_id,
  buyer_user_id,
  buyer_nickname,
  CASE status
    WHEN 'PENDING_PAYMENT' THEN 'pending_payment'
    WHEN 'PAID' THEN 'paid'
    WHEN 'CANCELLED' THEN 'cancelled'
    WHEN 'EXPIRED' THEN 'expired'
    WHEN 'REFUNDED' THEN 'refunded'
    ELSE LOWER(status)
  END,
  CASE payment_status
    WHEN 'INIT' THEN 'init'
    WHEN 'PROCESSING' THEN 'processing'
    WHEN 'SUCCESS' THEN 'success'
    WHEN 'FAILED' THEN 'failed'
    WHEN 'CLOSED' THEN 'closed'
    ELSE LOWER(payment_status)
  END,
  payment_id,
  lot_title,
  '直播竞拍',
  amount,
  currency,
  shipping_address_id,
  shipping_address_snapshot,
  created_at_unix_ms,
  updated_at_unix_ms,
  paid_at_unix_ms,
  expires_at_unix_ms,
  version,
  payload,
  created_at,
  updated_at
FROM auction_orders;

INSERT IGNORE INTO user_order_items (
  id, order_id, source, source_item_id, lot_id, room_id, title, image_url,
  sku_name, quantity, unit_amount, total_amount, currency, created_at, updated_at
)
SELECT
  CONCAT('auction_item_', id),
  id,
  'auction',
  lot_id,
  lot_id,
  room_id,
  lot_title,
  lot_image_url,
  '竞拍拍品',
  1,
  amount,
  amount,
  currency,
  created_at,
  updated_at
FROM auction_orders;

INSERT IGNORE INTO user_order_payments (
  id, order_id, source, provider, main_account_id, lot_id, user_id, status,
  amount, currency, idempotency_key, created_at_unix_ms, updated_at_unix_ms,
  succeeded_at_unix_ms, source_payload, created_at, updated_at
)
SELECT
  id,
  order_id,
  'auction',
  'mock',
  main_account_id,
  lot_id,
  buyer_user_id,
  CASE status
    WHEN 'INIT' THEN 'init'
    WHEN 'PROCESSING' THEN 'processing'
    WHEN 'SUCCESS' THEN 'success'
    WHEN 'FAILED' THEN 'failed'
    WHEN 'CLOSED' THEN 'closed'
    ELSE LOWER(status)
  END,
  amount,
  currency,
  IFNULL(NULLIF(idempotency_key, ''), CONCAT('legacy-', id)),
  created_at_unix_ms,
  updated_at_unix_ms,
  succeeded_at_unix_ms,
  payload,
  created_at,
  updated_at
FROM auction_payments;

INSERT IGNORE INTO user_orders (
  id, source, source_order_id, order_no, user_id, nickname, status, payment_status,
  payment_id, title, shop_name, total_amount, currency, shipping_address_id,
  shipping_address_snapshot, address_snapshot, created_at_unix_ms, updated_at_unix_ms,
  paid_at_unix_ms, payment_idempotency_key, created_at, updated_at
)
SELECT
  id,
  'shop',
  id,
  order_no,
  user_id,
  nickname,
  LOWER(status),
  LOWER(payment_status),
  payment_id,
  shop_name,
  shop_name,
  total_amount,
  currency,
  shipping_address_id,
  shipping_address_snapshot,
  address_snapshot,
  created_at_unix_ms,
  updated_at_unix_ms,
  paid_at_unix_ms,
  payment_idempotency_key,
  created_at,
  updated_at
FROM shop_orders;

INSERT IGNORE INTO user_order_items (
  id, order_id, source, source_item_id, product_id, sku_id, title, image_url,
  sku_name, quantity, unit_amount, total_amount, currency, created_at, updated_at
)
SELECT
  id,
  order_id,
  'shop',
  id,
  product_id,
  sku_id,
  title,
  image_url,
  sku_name,
  quantity,
  unit_amount,
  total_amount,
  currency,
  created_at,
  updated_at
FROM shop_order_items;

INSERT IGNORE INTO user_order_payments (
  id, order_id, source, provider, user_id, status, amount, currency,
  idempotency_key, created_at_unix_ms, succeeded_at_unix_ms, created_at, updated_at
)
SELECT
  id,
  order_id,
  'shop',
  'mock',
  user_id,
  LOWER(status),
  amount,
  currency,
  idempotency_key,
  created_at_unix_ms,
  succeeded_at_unix_ms,
  created_at,
  updated_at
FROM shop_payments;

DROP TABLE IF EXISTS auction_payments;
DROP TABLE IF EXISTS auction_orders;
DROP TABLE IF EXISTS shop_payments;
DROP TABLE IF EXISTS shop_order_items;
DROP TABLE IF EXISTS shop_orders;
