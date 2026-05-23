# 07-本地 MySQL / Redis / Docker 配置

> 状态：V1 本地基础设施主路径说明
> 目标：GORM + MySQL + Redis + Consul/Docker Compose 作为默认系统路径，不保留内存或 `database/sql` fallback。

## 1. 当前边界

后端默认使用：

```text
internal/data/Store
```

其中：

- MySQL 通过 GORM 存储拍品聚合和出价流水；
- Redis 缓存出价幂等查询并承载事件 Stream；
- Consul 作为服务注册中心，后端启动时注册 `auction-backend` 并使用 `/readyz` 做健康检查；
- `/readyz` 同时观测 MySQL/Redis、事件 outbox worker 最近修复状态和 Consul 注册是否仍存在；
- 旧 `MemoryStore`、手写 `database/sql` repo、`schema.go` 主路径已删除。

本次配置 MySQL / Redis / Consul 的目的：

- 让 V1 默认具备真实持久化环境；
- 为出价幂等、后续排行榜缓存、事件流准备 Redis；
- 为 Docker 一键启动准备统一入口；
- 保持最终系统主路径清晰，不做 demo/fallback 双轨。

## 2. 后续职责划分

### MySQL

适合承载：

- 拍品主表；
- 出价流水归档；
- 成交记录；
- 信任卡片配置；
- 运营侧查询。

### Redis

适合承载：

- 出价幂等键；
- 房间当前状态缓存；
- 排行榜 ZSET；
- WebSocket 广播辅助；
- 后续 Redis Stream 事件流。

### 当前 Go 代码

当前 Go 服务启动链路默认创建 `data.NewStore(...)`，并要求 MySQL / Redis 可连接。

分层原则：

```text
biz 只依赖 Repository 接口
        ↓
data 提供 MySQL / Redis 实现
        ↓
cmd/server 负责读取环境变量并组装 data 实现
```

不要把 MySQL / Redis 细节泄漏进 biz。

## 3. 本地启动

启动基础设施和后端：

```bash
make docker-up
```

停止并清理容器：

```bash
make docker-down
```

查看日志：

```bash
make docker-logs
```

只启动基础设施：

```bash
cd deploy
docker compose up -d mysql redis consul
```

## 4. 默认端口

| 服务 | 容器端口 | 本机端口 |
| --- | --- | --- |
| auction-backend | 18080 | 18080 |
| MySQL | 3306 | 13306 |
| Redis | 6379 | 16379 |
| Consul | 8500 | 18500 |

选择 `13306` / `16379` / `18500` 是为了避免和本机已有服务冲突。

## 5. 默认账号

仅用于本地开发：

```text
MYSQL_DATABASE=live_auction
MYSQL_USER=auction
MYSQL_PASSWORD=auction_dev
MYSQL_ROOT_PASSWORD=auction_root
REDIS_PASSWORD=auction_redis
```

不要把这些配置用于线上环境。

## 6. 配置文件

当前配置文件位置：

```text
app/auction/service/configs/config.yaml
```

其中包含：

```yaml
data:
  mysql:
    dsn: "auction:auction_dev@tcp(mysql:3306)/live_auction?parseTime=true&charset=utf8mb4&loc=Local"
  redis:
    addr: "redis:6379"
    password: "auction_redis"
  consul:
    addr: "consul:8500"
```

Docker Compose 会通过环境变量传入同等配置：`AUCTION_CONSUL_ADDR`、`AUCTION_SERVICE_NAME`、`AUCTION_SERVICE_ADDR`。

## 7. 当前实现

当前实现已经完成第一版真实 data store：

- `auction_lots`：存储拍品聚合 JSON；
- `auction_bids`：存储出价流水 JSON；
- Redis `auction:idem:{lot_id}:{key}`：存储幂等键对应的出价。

数据库表由 GORM AutoMigrate 在服务启动时创建；`deploy/mysql/schema.sql` 只作为参考结构，不是运行时主路径。

后续仍可继续演进：

- 将拍品 JSON 拆成更细的结构化字段；
- 将排行榜改为 Redis ZSET；
- 将事件流改为 Redis Stream / Kafka；
- 增加正式 migration 工具。

每一步都必须保持：

- biz 不感知 MySQL / Redis；
- service 不做存储逻辑；
- data 负责具体实现；
- 测试继续覆盖业务状态机。
