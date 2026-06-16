SET NAMES utf8mb4;

DROP PROCEDURE IF EXISTS migrate_20260615_avatar_live_sync;

DELIMITER $$
CREATE PROCEDURE migrate_20260615_avatar_live_sync()
BEGIN
  SET @now_ms = CAST(UNIX_TIMESTAMP(CURRENT_TIMESTAMP(3)) * 1000 AS UNSIGNED);

  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = DATABASE()
      AND table_name = 'auction_users'
      AND column_name = 'avatar_url'
  ) THEN
    ALTER TABLE auction_users ADD COLUMN avatar_url VARCHAR(512) NOT NULL DEFAULT '' AFTER nickname;
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = DATABASE()
      AND table_name = 'auction_rooms'
      AND column_name = 'live_source_url'
  ) THEN
    ALTER TABLE auction_rooms ADD COLUMN live_source_url VARCHAR(512) NOT NULL DEFAULT '' AFTER platform_room_id;
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = DATABASE()
      AND table_name = 'auction_rooms'
      AND column_name = 'live_started_at_unix_ms'
  ) THEN
    ALTER TABLE auction_rooms ADD COLUMN live_started_at_unix_ms BIGINT NOT NULL DEFAULT 0 AFTER live_source_url;
  END IF;

  UPDATE auction_users
  SET avatar_url = CASE MOD(CRC32(id), 6)
    WHEN 0 THEN 'https://liveauction.tos-cn-beijing.volces.com/douyin-h5/images/avatar-71158770-d8597.jpeg'
    WHEN 1 THEN 'https://liveauction.tos-cn-beijing.volces.com/douyin-h5/images/avatar-lsy0508-160edjy.jpeg'
    WHEN 2 THEN 'https://liveauction.tos-cn-beijing.volces.com/douyin-h5/images/avatar-ll991221-1bmdvg4.jpeg'
    WHEN 3 THEN 'https://liveauction.tos-cn-beijing.volces.com/douyin-h5/images/avatar-sunmeng333-qheb8m.jpeg'
    WHEN 4 THEN 'https://liveauction.tos-cn-beijing.volces.com/douyin-h5/images/avatar-jingyiziran-176539n.jpeg'
    ELSE 'https://liveauction.tos-cn-beijing.volces.com/douyin-h5/images/avatar-8357999-1bd1vnm.jpeg'
  END
  WHERE avatar_url = '';

  UPDATE auction_rooms
  SET live_source_url = CASE MOD(CRC32(id), 8)
    WHEN 0 THEN 'https://liveauction.tos-cn-beijing.volces.com/douyin-h5/videos/6931271799195831566.mp4'
    WHEN 1 THEN 'https://liveauction.tos-cn-beijing.volces.com/douyin-h5/videos/7326744032997166387.mp4'
    WHEN 2 THEN 'https://liveauction.tos-cn-beijing.volces.com/douyin-h5/videos/7161000281575148800.mp4'
    WHEN 3 THEN 'https://liveauction.tos-cn-beijing.volces.com/douyin-h5/videos/6882368275695586568.mp4'
    WHEN 4 THEN 'https://liveauction.tos-cn-beijing.volces.com/douyin-h5/videos/6993228049399549198.mp4'
    WHEN 5 THEN 'https://liveauction.tos-cn-beijing.volces.com/douyin-h5/videos/7260749400622894336.mp4'
    WHEN 6 THEN 'https://liveauction.tos-cn-beijing.volces.com/douyin-h5/videos/7280132304427666722.mp4'
    ELSE 'https://liveauction.tos-cn-beijing.volces.com/douyin-h5/videos/7110263965858549003.mp4'
  END
  WHERE live_source_url = '';

  UPDATE auction_rooms
  SET live_started_at_unix_ms = CASE
    WHEN created_at_unix_ms > 0 THEN created_at_unix_ms
    ELSE @now_ms
  END
  WHERE live_started_at_unix_ms = 0;
END$$
DELIMITER ;

CALL migrate_20260615_avatar_live_sync();
DROP PROCEDURE IF EXISTS migrate_20260615_avatar_live_sync;
