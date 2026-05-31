-- Full RBAC replacement.
-- Run this once before deploying the RBAC-only application build.

DROP PROCEDURE IF EXISTS migrate_20260531_full_rbac_replace;

DELIMITER $$
CREATE PROCEDURE migrate_20260531_full_rbac_replace()
BEGIN
  SET @now_ms = CAST(UNIX_TIMESTAMP(CURRENT_TIMESTAMP(3)) * 1000 AS UNSIGNED);

  CREATE TABLE IF NOT EXISTS auction_roles (
    code VARCHAR(64) PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    description VARCHAR(512) NOT NULL DEFAULT '',
    `system` BOOLEAN NOT NULL DEFAULT TRUE,
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

  INSERT INTO auction_roles (code, name, description, `system`, created_at_unix_ms, updated_at_unix_ms, created_at, updated_at)
  VALUES
    ('merchant_owner', '商家主账号', '商家/主播空间主账号', TRUE, @now_ms, @now_ms, NOW(3), NOW(3)),
    ('anchor', '主播/场控', '直播控场子账号', TRUE, @now_ms, @now_ms, NOW(3), NOW(3)),
    ('operator', '运营子账号', '拍品与订单运营子账号', TRUE, @now_ms, @now_ms, NOW(3), NOW(3)),
    ('buyer', '买家', 'H5 买家账号', TRUE, @now_ms, @now_ms, NOW(3), NOW(3))
  ON DUPLICATE KEY UPDATE
    name = VALUES(name),
    description = VALUES(description),
    `system` = VALUES(`system`),
    updated_at_unix_ms = VALUES(updated_at_unix_ms),
    updated_at = NOW(3);

  INSERT INTO auction_permissions (code, name, module, description, created_at_unix_ms, updated_at_unix_ms, created_at, updated_at)
  VALUES
    ('team.user.create', '团队账号创建', 'team', '创建主播/运营子账号', @now_ms, @now_ms, NOW(3), NOW(3)),
    ('team.user.list', '团队账号查询', 'team', '查询当前主账号空间团队账号', @now_ms, @now_ms, NOW(3), NOW(3)),
    ('team.user.update_role', '团队角色调整', 'team', '调整主播/运营子账号角色', @now_ms, @now_ms, NOW(3), NOW(3)),
    ('team.user.update_status', '团队状态调整', 'team', '启用或禁用子账号', @now_ms, @now_ms, NOW(3), NOW(3)),
    ('lot.create', '拍品创建', 'lot', '创建拍品或草稿', @now_ms, @now_ms, NOW(3), NOW(3)),
    ('lot.update', '拍品编辑', 'lot', '编辑拍品资料', @now_ms, @now_ms, NOW(3), NOW(3)),
    ('lot.queue', '拍品入队', 'lot', '加入本场队列', @now_ms, @now_ms, NOW(3), NOW(3)),
    ('lot.view_admin', '后台拍品查看', 'lot', '查看后台拍品列表', @now_ms, @now_ms, NOW(3), NOW(3)),
    ('auction.control', '直播控场', 'auction', '开拍、落锤、取消和 Duel 控制', @now_ms, @now_ms, NOW(3), NOW(3)),
    ('order.manage', '订单管理', 'order', '后台查看和处理订单', @now_ms, @now_ms, NOW(3), NOW(3)),
    ('realtime.view', '实时状态查看', 'realtime', '查看房间实时状态和事件', @now_ms, @now_ms, NOW(3), NOW(3)),
    ('upload.image', '图片上传', 'asset', '上传拍品图片', @now_ms, @now_ms, NOW(3), NOW(3)),
    ('bid.place', '买家出价', 'buyer', 'H5 买家出价', @now_ms, @now_ms, NOW(3), NOW(3)),
    ('order.pay', '买家支付', 'buyer', 'H5 买家模拟支付', @now_ms, @now_ms, NOW(3), NOW(3)),
    ('order.view_own', '本人订单查看', 'buyer', '查看本人订单和出价', @now_ms, @now_ms, NOW(3), NOW(3))
  ON DUPLICATE KEY UPDATE
    name = VALUES(name),
    module = VALUES(module),
    description = VALUES(description),
    updated_at_unix_ms = VALUES(updated_at_unix_ms),
    updated_at = NOW(3);

  INSERT IGNORE INTO auction_role_permissions (id, role_code, permission_code, created_at_unix_ms, created_at, updated_at)
  VALUES
    (CONCAT('rp_', MD5('merchant_owner:team.user.create')), 'merchant_owner', 'team.user.create', @now_ms, NOW(3), NOW(3)),
    (CONCAT('rp_', MD5('merchant_owner:team.user.list')), 'merchant_owner', 'team.user.list', @now_ms, NOW(3), NOW(3)),
    (CONCAT('rp_', MD5('merchant_owner:team.user.update_role')), 'merchant_owner', 'team.user.update_role', @now_ms, NOW(3), NOW(3)),
    (CONCAT('rp_', MD5('merchant_owner:team.user.update_status')), 'merchant_owner', 'team.user.update_status', @now_ms, NOW(3), NOW(3)),
    (CONCAT('rp_', MD5('merchant_owner:lot.create')), 'merchant_owner', 'lot.create', @now_ms, NOW(3), NOW(3)),
    (CONCAT('rp_', MD5('merchant_owner:lot.update')), 'merchant_owner', 'lot.update', @now_ms, NOW(3), NOW(3)),
    (CONCAT('rp_', MD5('merchant_owner:lot.queue')), 'merchant_owner', 'lot.queue', @now_ms, NOW(3), NOW(3)),
    (CONCAT('rp_', MD5('merchant_owner:lot.view_admin')), 'merchant_owner', 'lot.view_admin', @now_ms, NOW(3), NOW(3)),
    (CONCAT('rp_', MD5('merchant_owner:auction.control')), 'merchant_owner', 'auction.control', @now_ms, NOW(3), NOW(3)),
    (CONCAT('rp_', MD5('merchant_owner:order.manage')), 'merchant_owner', 'order.manage', @now_ms, NOW(3), NOW(3)),
    (CONCAT('rp_', MD5('merchant_owner:realtime.view')), 'merchant_owner', 'realtime.view', @now_ms, NOW(3), NOW(3)),
    (CONCAT('rp_', MD5('merchant_owner:upload.image')), 'merchant_owner', 'upload.image', @now_ms, NOW(3), NOW(3)),
    (CONCAT('rp_', MD5('anchor:lot.view_admin')), 'anchor', 'lot.view_admin', @now_ms, NOW(3), NOW(3)),
    (CONCAT('rp_', MD5('anchor:auction.control')), 'anchor', 'auction.control', @now_ms, NOW(3), NOW(3)),
    (CONCAT('rp_', MD5('anchor:order.manage')), 'anchor', 'order.manage', @now_ms, NOW(3), NOW(3)),
    (CONCAT('rp_', MD5('anchor:realtime.view')), 'anchor', 'realtime.view', @now_ms, NOW(3), NOW(3)),
    (CONCAT('rp_', MD5('operator:lot.create')), 'operator', 'lot.create', @now_ms, NOW(3), NOW(3)),
    (CONCAT('rp_', MD5('operator:lot.update')), 'operator', 'lot.update', @now_ms, NOW(3), NOW(3)),
    (CONCAT('rp_', MD5('operator:lot.queue')), 'operator', 'lot.queue', @now_ms, NOW(3), NOW(3)),
    (CONCAT('rp_', MD5('operator:lot.view_admin')), 'operator', 'lot.view_admin', @now_ms, NOW(3), NOW(3)),
    (CONCAT('rp_', MD5('operator:order.manage')), 'operator', 'order.manage', @now_ms, NOW(3), NOW(3)),
    (CONCAT('rp_', MD5('operator:realtime.view')), 'operator', 'realtime.view', @now_ms, NOW(3), NOW(3)),
    (CONCAT('rp_', MD5('operator:upload.image')), 'operator', 'upload.image', @now_ms, NOW(3), NOW(3)),
    (CONCAT('rp_', MD5('buyer:bid.place')), 'buyer', 'bid.place', @now_ms, NOW(3), NOW(3)),
    (CONCAT('rp_', MD5('buyer:order.pay')), 'buyer', 'order.pay', @now_ms, NOW(3), NOW(3)),
    (CONCAT('rp_', MD5('buyer:order.view_own')), 'buyer', 'order.view_own', @now_ms, NOW(3), NOW(3));

  SELECT COUNT(*) INTO @has_users_table
  FROM information_schema.tables
  WHERE table_schema = DATABASE()
    AND table_name = 'auction_users';

  SELECT COUNT(*) INTO @has_user_role_column
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'auction_users'
    AND column_name = 'role';

  IF @has_users_table > 0 AND @has_user_role_column > 0 THEN
    SET @sql = 'UPDATE auction_users SET main_account_id = id WHERE role = 4 AND (main_account_id IS NULL OR main_account_id = '''')';
    PREPARE stmt FROM @sql;
    EXECUTE stmt;
    DEALLOCATE PREPARE stmt;

    SET @sql = 'INSERT IGNORE INTO auction_user_roles (id, user_id, role_code, main_account_id, granted_by_user_id, created_at_unix_ms, created_at, updated_at)
      SELECT
        CONCAT(''ur_'', MD5(CONCAT(id, '':'', role, '':'', COALESCE(main_account_id, '''')))),
        id,
        CASE role
          WHEN 1 THEN ''buyer''
          WHEN 2 THEN ''anchor''
          WHEN 3 THEN ''operator''
          WHEN 4 THEN ''merchant_owner''
        END,
        COALESCE(main_account_id, ''''),
        COALESCE(created_by_user_id, ''''),
        CASE WHEN created_at_unix_ms > 0 THEN created_at_unix_ms ELSE @now_ms END,
        NOW(3),
        NOW(3)
      FROM auction_users
      WHERE role IN (1, 2, 3, 4)';
    PREPARE stmt FROM @sql;
    EXECUTE stmt;
    DEALLOCATE PREPARE stmt;
  END IF;

  IF @has_users_table > 0 AND EXISTS (
    SELECT 1 FROM information_schema.statistics
    WHERE table_schema = DATABASE()
      AND table_name = 'auction_users'
      AND index_name = 'idx_user_main_role'
  ) THEN
    ALTER TABLE auction_users DROP INDEX idx_user_main_role;
  END IF;

  IF @has_users_table > 0 AND EXISTS (
    SELECT 1 FROM information_schema.statistics
    WHERE table_schema = DATABASE()
      AND table_name = 'auction_users'
      AND index_name = 'idx_role'
  ) THEN
    ALTER TABLE auction_users DROP INDEX idx_role;
  END IF;

  IF @has_users_table > 0 AND NOT EXISTS (
    SELECT 1 FROM information_schema.statistics
    WHERE table_schema = DATABASE()
      AND table_name = 'auction_users'
      AND index_name = 'idx_user_main_status'
  ) THEN
    ALTER TABLE auction_users ADD INDEX idx_user_main_status (main_account_id, status);
  END IF;

  IF @has_users_table > 0 AND @has_user_role_column > 0 THEN
    ALTER TABLE auction_users DROP COLUMN role;
  END IF;
END$$
DELIMITER ;

CALL migrate_20260531_full_rbac_replace();
DROP PROCEDURE IF EXISTS migrate_20260531_full_rbac_replace;
