-- Clean auction terminal state after queue/payment lifecycle tightening.
-- Lot status values: SETTLED=3, CANCELLED=4, QUEUED=6, FAILED=8.
-- Queue status values: NONE=1, QUEUED=2.

SET @now_ms = UNIX_TIMESTAMP(CURRENT_TIMESTAMP(3)) * 1000;

UPDATE auction_lots
SET
  queue_status = 1,
  queue_position = 0,
  payload = JSON_SET(
    payload,
    '$.queueStatus', 'LOT_QUEUE_STATUS_NONE',
    '$.queuePosition', 0
  )
WHERE status IN (3, 4, 8)
  AND queue_status <> 1;

UPDATE auction_lots l
JOIN auction_orders o ON o.lot_id = l.id
SET
  l.status = 8,
  l.queue_status = 1,
  l.queue_position = 0,
  l.cancel_reason = 'payment expired',
  l.cancelled_at_unix_ms = o.expires_at_unix_ms,
  l.playbook_stage = 6,
  l.version = l.version + 1,
  l.payload = JSON_SET(
    l.payload,
    '$.status', 'LOT_STATUS_FAILED',
    '$.queueStatus', 'LOT_QUEUE_STATUS_NONE',
    '$.queuePosition', 0,
    '$.cancelReason', 'payment expired',
    '$.cancelledAtUnixMs', CAST(o.expires_at_unix_ms AS CHAR),
    '$.playbookStage', 'PLAYBOOK_STAGE_SETTLE_READY'
  )
WHERE o.status = 'PENDING_PAYMENT'
  AND o.payment_status = 'INIT'
  AND o.expires_at_unix_ms > 0
  AND o.expires_at_unix_ms <= @now_ms;

UPDATE auction_orders
SET
  status = 'EXPIRED',
  payment_status = 'CLOSED',
  updated_at_unix_ms = @now_ms,
  version = version + 1,
  payload = JSON_SET(
    payload,
    '$.status', 'EXPIRED',
    '$.paymentStatus', 'CLOSED',
    '$.updatedAtUnixMs', CAST(@now_ms AS UNSIGNED)
  )
WHERE status = 'PENDING_PAYMENT'
  AND payment_status = 'INIT'
  AND expires_at_unix_ms > 0
  AND expires_at_unix_ms <= @now_ms;
