-- P2 audit fix migration.
-- Re-runnable for databases created before or after the audit fix.

CREATE TABLE IF NOT EXISTS auction_orders (
  id VARCHAR(64) PRIMARY KEY,
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
  INDEX idx_order_room_created (room_id, created_at_unix_ms),
  INDEX idx_order_buyer_status (buyer_user_id, status),
  INDEX idx_order_payment_status (payment_status),
  INDEX idx_order_expiry (expires_at_unix_ms)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS auction_payments (
  id VARCHAR(64) PRIMARY KEY,
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
  INDEX idx_payment_order (order_id),
  INDEX idx_payment_lot (lot_id),
  INDEX idx_payment_buyer (buyer_user_id),
  INDEX idx_payment_status (status),
  INDEX idx_payment_created (created_at_unix_ms),
  UNIQUE INDEX uk_order_payment_idem (order_id, idempotency_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

DROP PROCEDURE IF EXISTS migrate_20260523_audit_fix_bid_idem;

DELIMITER $$
CREATE PROCEDURE migrate_20260523_audit_fix_bid_idem()
BEGIN
  IF EXISTS (
    SELECT 1
    FROM information_schema.statistics
    WHERE table_schema = DATABASE()
      AND table_name = 'auction_bids'
      AND index_name = 'idx_lot_idem'
  ) THEN
    ALTER TABLE auction_bids DROP INDEX idx_lot_idem;
  END IF;

  IF NOT EXISTS (
    SELECT 1
    FROM information_schema.statistics
    WHERE table_schema = DATABASE()
      AND table_name = 'auction_bids'
      AND index_name = 'idx_lot_user_idem'
  ) THEN
    ALTER TABLE auction_bids ADD UNIQUE INDEX idx_lot_user_idem (lot_id, user_id, idempotency_key);
  END IF;
END$$
DELIMITER ;

CALL migrate_20260523_audit_fix_bid_idem();
DROP PROCEDURE IF EXISTS migrate_20260523_audit_fix_bid_idem;
