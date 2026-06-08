-- P3-0 demo readiness migration.
-- The current schema requires auction_bids.idempotency_key for every bid.

DROP PROCEDURE IF EXISTS migrate_20260523_p3_0_bid_idem_required;

DELIMITER $$
CREATE PROCEDURE migrate_20260523_p3_0_bid_idem_required()
BEGIN
  IF EXISTS (
    SELECT 1
    FROM information_schema.tables
    WHERE table_schema = DATABASE()
      AND table_name = 'auction_bids'
  ) THEN
    ALTER TABLE auction_bids MODIFY COLUMN idempotency_key VARCHAR(128) NOT NULL;

    IF NOT EXISTS (
      SELECT 1
      FROM information_schema.statistics
      WHERE table_schema = DATABASE()
        AND table_name = 'auction_bids'
        AND index_name = 'idx_lot_user_idem'
    ) THEN
      ALTER TABLE auction_bids ADD UNIQUE INDEX idx_lot_user_idem (lot_id, user_id, idempotency_key);
    END IF;
  END IF;
END$$
DELIMITER ;

CALL migrate_20260523_p3_0_bid_idem_required();
DROP PROCEDURE IF EXISTS migrate_20260523_p3_0_bid_idem_required;
