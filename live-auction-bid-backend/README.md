# 直播互动竞拍系统后端

当前阶段：V1 核心业务闭环 + 基础设施主路径落地。

## 项目最强纲领

最高需求基线是用户提供的《抖音电商AI全栈课题-直播竞拍全栈系统（宣讲版）》PDF。后续产品、架构、后端、前端、测试和答辩都必须按该课题评分点推进，不再自由发挥。

必须闭合主流程：商品上架 → 规则配置 → 主播开拍 → 用户实时出价 → WebSocket 同步价格/排名/倒计时 → 自动延时/封顶价自动成交/主播异常取消 → 竞拍结束 → 自动生成订单 → 用户查看结果/模拟支付。

P0 硬点：0 元起拍、固定加价、封顶价自动成交、10-30 秒自动延时、异常取消、订单管理、移动 H5 观众端、WebSocket 心跳重连/快照恢复、被超越/领先/延时/结束提醒、100+ 并发出价一致性。

工程铁律和完整纲领见根文档：[`PROJECT_CHARTER.md`](./PROJECT_CHARTER.md)。

## 已落地的主链路

- 核心业务：创建拍品、开拍、出价、排行榜、信任卡揭示、Duel、落锤、快照、WebSocket 事件广播。
- 数据主路径：GORM + MySQL 保存拍品、出价和持久幂等键；Redis 缓存出价幂等查询，缓存失败后以 MySQL 记录为准。
- 本地基础设施：Docker Compose 启动 backend / MySQL / Redis / Consul。
- 服务注册：启动时向 Consul 注册 `auction-backend`，使用 `/readyz` 作为 Consul HTTP health check。
- 事件流：伴随拍品/出价状态变更的领域事件与业务状态写入同一个 MySQL 事务；事务提交后写 Redis Stream，Stream 写入确认/错误回写 MySQL，后台 outbox worker 会重推未确认事件。
- 用户系统：自建 username/password 账号，用户 ID 使用雪花字符串；JWT access token + refresh session 支持注册、登录、刷新、登出、me 和 admin 角色管理。
- 鉴权权限：公开读接口可匿名访问；出价必须是 buyer；创建/开拍/揭示/Duel/落锤必须是 anchor/operator/admin；admin 接口仅 admin 可用。
- 健康检查：`/healthz` 存活检查，`/readyz` 检查 MySQL + Redis、事件 outbox worker 最近修复状态、Consul 注册可观测状态。
- 统一响应：service 对外只用 reply.result 表达业务成功/失败；Go `error` 不再承载可预期业务错误，避免前端同时解析 body 和 transport error。

被替代的 `MemoryStore`、`database/sql` repo、手写 `schema.go` 主路径已删除，不保留 fallback 或双实现开关。

## 本地运行

```bash
cd deploy
docker compose up --build
```

服务默认监听：`http://127.0.0.1:18080`。

Docker Compose 默认创建本地 admin：

```text
username: admin
password: admin_dev_password
```

常用检查：

```bash
curl http://127.0.0.1:18080/healthz
curl http://127.0.0.1:18080/readyz
```

用户与出价示例：

```bash
curl -X POST http://127.0.0.1:18080/api/users/register \
  -H 'Content-Type: application/json' \
  -d '{"username":"buyer1","password":"password123","nickname":"买家一号"}'

curl -X POST http://127.0.0.1:18080/api/lots/{lot_id}/bid \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer <access_token>' \
  -d '{"amount":{"amount":11000,"currency":"CNY"},"idempotency_key":"buyer1-11000"}'
```

如果宿主机安装了 Go：

```bash
go test ./...
go build ./app/auction/service/cmd/server
```

## 文档

主要文档位于 `docs/openclaw/v1/`。


### 统一响应与乐观锁冲突语义

当多实例或并发请求触发 lot expected-version 冲突时，data 层统一返回稳定哨兵错误，service 层包装进 reply.result：`code=409001`，`message=lot state changed, please refresh and retry`。前端应刷新拍品快照后提示用户重试，不应按普通 500 处理。

### 工程设计原则

- **统一返回模式（Result Envelope）**：service 层把可预期业务错误收敛到 `ReplyResult`，对前端形成单一解析入口；transport error 只留给不可包装的系统/链路故障。
- **Repository + Unit of Work**：data 层持有 GORM/Redis/事务边界，lot/bid/event 在必要场景进入同一个 MySQL transaction，biz 层只依赖 repo 接口。
- **Transactional Outbox**：业务状态与事件先在 MySQL 同事务落库，再由 outbox worker 推 Redis Stream，接受 at-least-once，消费侧按 event id 幂等。
- **Registry + Health Check**：server 层负责 Consul 注册与 `/readyz` 聚合观测，不让治理基础设施进入 biz。
- **测试隔离**：项目实现目录不放 `*_test.go`，测试集中在 `app/auction/service/test`，避免实现包被测试文件污染。
