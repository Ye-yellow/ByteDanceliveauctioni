# 07-本地 MySQL / Redis / Docker 配置

> 状态：V1 本地基础设施配置说明  
> 目标：先把 MySQL、Redis、Docker 运行环境准备好，业务代码仍保持 V1 内存存储，后续再按 data 层接口替换。

## 1. 当前边界

当前后端使用：

```text
internal/data/MemoryStore
```

它服务于 V1 本地 demo，优点是闭环快、依赖少。

本次配置 MySQL / Redis 的目的不是立刻把业务强行迁移过去，而是：

- 为后续数据持久化准备环境；
- 为后续出价幂等、排行榜、事件流准备 Redis；
- 为 Docker 一键启动准备统一入口；
- 让架构表达从“内存 demo”自然演进到“可落地工程”。

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

当前 Go 代码仍然默认使用 `MemoryStore`。

后续替换原则：

```text
biz 只依赖 Repository 接口
        ↓
data 提供 Memory / MySQL / Redis 实现
        ↓
cmd/server 负责按配置选择 data 实现
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

只启动数据库和 Redis：

```bash
cd deploy
docker compose up -d mysql redis
```

## 4. 默认端口

| 服务 | 容器端口 | 本机端口 |
| --- | --- | --- |
| auction-backend | 18080 | 18080 |
| MySQL | 3306 | 13306 |
| Redis | 6379 | 16379 |

选择 `13306` / `16379` 是为了避免和本机已有 MySQL / Redis 冲突。

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
```

Docker Compose 会通过环境变量传入同等配置，后续接入真实 data 实现时再读取。

## 7. 后续迁移建议

建议分三步迁移，不要一次性改大：

1. 保留 MemoryStore，新增 MySQL schema 文档或 migration 草案；
2. 先把 LotRepository 替换为 MySQL 实现；
3. 再把 BidRepository 的幂等和排行榜能力拆到 Redis / MySQL 组合。

每一步都必须保持：

- biz 不感知 MySQL / Redis；
- service 不做存储逻辑；
- data 负责具体实现；
- 测试继续覆盖业务状态机。
