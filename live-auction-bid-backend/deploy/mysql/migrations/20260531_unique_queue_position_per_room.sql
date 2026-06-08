-- Enforce one queued/NEXT queue position per room.

DROP PROCEDURE IF EXISTS migrate_20260531_unique_queue_position_per_room;

DELIMITER $$
CREATE PROCEDURE migrate_20260531_unique_queue_position_per_room()
BEGIN
  SET @now_ms = CAST(UNIX_TIMESTAMP(CURRENT_TIMESTAMP(3)) * 1000 AS UNSIGNED);

  DROP TEMPORARY TABLE IF EXISTS tmp_queue_repair;
  DROP TEMPORARY TABLE IF EXISTS tmp_duplicate_queue_rooms;

  CREATE TEMPORARY TABLE tmp_duplicate_queue_rooms AS
  SELECT DISTINCT room_id
  FROM (
    SELECT room_id, queue_position
    FROM auction_lots
    WHERE queue_status IN (2, 3)
      AND queue_position > 0
    GROUP BY room_id, queue_position
    HAVING COUNT(*) > 1
  ) duplicated;

  CREATE TEMPORARY TABLE tmp_queue_repair AS
  SELECT
    id,
    version + 1 AS repaired_version,
    ROW_NUMBER() OVER (
      PARTITION BY room_id
      ORDER BY queue_position ASC, updated_at ASC, id ASC
    ) AS repaired_position
  FROM auction_lots
  WHERE room_id IN (SELECT room_id FROM tmp_duplicate_queue_rooms)
    AND queue_status IN (2, 3)
    AND queue_position > 0;

  UPDATE auction_lots l
  JOIN tmp_queue_repair r ON r.id = l.id
  SET
    l.queue_position = r.repaired_position,
    l.version = r.repaired_version,
    l.payload = JSON_SET(
      l.payload,
      '$.queuePosition', r.repaired_position,
      '$.version', r.repaired_version
    );

  UPDATE auction_room_states s
  JOIN (
    SELECT room_id, COALESCE(MAX(queue_position), 0) + 1 AS next_queue_position
    FROM auction_lots
    WHERE queue_status IN (2, 3)
      AND queue_position > 0
    GROUP BY room_id
  ) q ON q.room_id = s.room_id
  SET
    s.next_queue_position = GREATEST(s.next_queue_position, q.next_queue_position),
    s.updated_at_unix_ms = @now_ms;

  IF NOT EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema = DATABASE()
      AND table_name = 'auction_lots'
      AND column_name = 'queued_room_position_key'
  ) THEN
    ALTER TABLE auction_lots
      ADD COLUMN queued_room_position_key VARCHAR(96)
      GENERATED ALWAYS AS (
        CASE
          WHEN queue_status IN (2, 3) AND queue_position > 0 THEN CONCAT(room_id, '#', queue_position)
          ELSE NULL
        END
      ) STORED;
  END IF;

  IF NOT EXISTS (
    SELECT 1
    FROM information_schema.statistics
    WHERE table_schema = DATABASE()
      AND table_name = 'auction_lots'
      AND index_name = 'uidx_one_queued_position_per_room'
  ) THEN
    ALTER TABLE auction_lots ADD UNIQUE INDEX uidx_one_queued_position_per_room (queued_room_position_key);
  END IF;

  DROP TEMPORARY TABLE IF EXISTS tmp_queue_repair;
  DROP TEMPORARY TABLE IF EXISTS tmp_duplicate_queue_rooms;
END$$
DELIMITER ;

CALL migrate_20260531_unique_queue_position_per_room();
DROP PROCEDURE IF EXISTS migrate_20260531_unique_queue_position_per_room;
