# GStack Office Hours：直播竞拍系统架构优化建议

Date: 2026-05-19
Mode: Hackathon / demo / competition
Scope: architecture planning only, no implementation in this pass.

## 0. Goal

比赛题目不是普通电商 CRUD，而是要证明我们能做一个短视频直播场景下的“实时竞拍交易引擎”。

核心评审点应该被我们主动放大：

1. 高并发实时出价；
2. WebSocket 毫秒级广播；
3. 竞拍状态一致性；
4. 动态排名；
5. 落锤成交闭环；
6. AI 动态定价与直播气氛营造。

所以后端架构不能只像一个简单商品服务，而要像一个“交易状态机 + 实时通信引擎 + AI 辅助决策”的组合。

## 1. Current Understanding

当前后端已经从初版 DDD 单服务目录，调整成更接近 Odin/Kratos 的多服务仓库：

```text
auction-backend/
├── api/auction/service/v1/auction.proto
├── app/auction/service/
│   ├── cmd/server/
│   ├── configs/
│   └── internal/
│       ├── conf/
│       ├── server/
│       ├── service/
│       ├── biz/
│       └── data/
├── deploy/
├── docs/
├── Makefile
└── app_makefile
```

这个方向是对的。它已经符合：

```text
api → server → service → biz → data
```

但它现在更像“能跑 demo 的 Kratos 骨架”，还不够像“比赛级实时竞拍平台”。

## 2. Office Hours Diagnosis

### 2.1 The real product is not 商品上架

普通电商系统的中心是商品和订单。

这个题目的中心应该是：

```text
Auction Session / 竞拍会话
```

也就是说，核心不是 `Product`，而是：

- 某个直播间正在拍哪件货；
- 当前竞拍规则是什么；
- 谁在第几毫秒出了多少钱；
- 这口价是否有效；
- 排名如何变化；
- 是否触发反狙击延时；
- 最终谁落锤成交。

因此后端领域模型应该从 `Lot/Bid` 继续升级到：

```text
AuctionSession
AuctionLot
BidEvent
BidRule
RankingBoard
Settlement
```

### 2.2 Current weakest point: 状态一致性故事还不够硬

题目强调高并发与实时交互。评委会关心：

> 如果 1000 人同时出价，谁赢？

现在内存仓储可以演示，但不能讲“硬核架构能力”。我们需要在架构图和代码结构上提前表达：

- 单拍品出价必须原子；
- 最低加价校验和当前价更新必须同一事务；
- 排行榜必须和成交价一致；
- WebSocket 广播不能先于状态落库；
- 客户端显示以服务端版本为准。

这要求我们把 Redis Lua / 单线程竞价队列 / 事件日志作为下一阶段核心。

### 2.3 AI 不能只是“文案生成”

题目鼓励 AI 在动态定价与气氛营造中的创新。只做一句直播话术会显得薄。

更好的 AI 叙事是三层：

1. **拍前：动态估价**
   - 输入：品类、成色、图片描述、参考价、历史成交；
   - 输出：建议起拍价、保留价、加价档位。

2. **拍中：气氛官**
   - 输入：出价频率、剩余时间、价格跃迁、在线人数；
   - 输出：主播提示语、催拍话术、稀缺性描述。

3. **拍后：复盘与风控**
   - 输入：出价曲线、异常账号、成交价偏离；
   - 输出：异常提示、运营复盘、下一场建议。

这样 AI 不只是装饰，而是参与交易闭环。

## 3. Architecture Alternatives

### Option A — Keep single `auction/service`, make it excellent

```text
app/auction/service
├── internal/server
├── internal/service
├── internal/biz
├── internal/data
└── internal/realtime
```

Pros:

- 最快；
- 比赛演示风险最低；
- 不会过度拆服务；
- 当前代码改动最小。

Cons:

- 服务边界不够宏大；
- AI/订单/直播间概念都在一个服务内。

Verdict: **Recommended for next 1-2 iterations.**

### Option B — Split `auction/service` + `ai/service`

```text
app/auction/service
app/ai/service
api/auction/service/v1
api/ai/service/v1
```

Pros:

- AI 创新点更突出；
- 评委容易看到“开源 AI 模型服务化”；
- 未来可独立部署 Ollama/Qwen/Llama client。

Cons:

- 当前会增加工程量；
- 需要多服务调用、错误处理、超时降级；
- 若 demo 时间短，可能不如单服务稳定。

Verdict: **Good Phase 2, not immediate Phase 1.**

### Option C — Full commerce microservices

```text
app/room/service
app/auction/service
app/order/service
app/payment/service
app/ai/service
app/gateway/service
```

Pros:

- 最像完整电商平台；
- 架构图漂亮。

Cons:

- 对比赛初版过重；
- 大量服务只有空壳；
- 容易被问“你们真正做深的是哪块？”

Verdict: **Do not do now. Mention as future evolution only.**

## 4. Recommended Architecture Direction

采用 Option A 为主，Option B 预留。

也就是：

```text
auction-backend/
├── api/
│   ├── auction/service/v1/
│   └── ai/service/v1/              # 先放 proto 草案，服务后拆
├── app/
│   └── auction/service/
│       ├── cmd/server/
│       ├── configs/
│       └── internal/
│           ├── conf/
│           ├── server/
│           ├── service/
│           ├── biz/
│           ├── data/
│           └── realtime/
├── pkg/
│   ├── money/
│   ├── idgen/
│   ├── wsmsg/
│   └── errors/
├── third_party/
├── deploy/
└── docs/
```

## 5. Concrete Optimization Plan

### 5.1 Add `internal/realtime`

Current WebSocket hub should move out of `server/ws`.

Target:

```text
internal/realtime/
├── hub.go             # connection registry
├── room.go            # room lifecycle
├── client.go          # connected client abstraction
├── message.go         # WS message envelope
├── broadcaster.go     # broadcast strategy
└── presence.go        # online users / heartbeat
```

Why:

- `server` should only upgrade HTTP to WebSocket;
- `realtime` owns room state, presence, fanout;
- competition narrative becomes clearer: “实时通信引擎” is a named module.

### 5.2 Split `biz` by domain concept

Target:

```text
internal/biz/
├── auction_session.go
├── lot.go
├── bid.go
├── bid_rule.go
├── ranking.go
├── settlement.go
├── event.go
├── ai_pricing.go
├── ai_atmosphere.go
└── repo.go
```

Why:

- 当前 `entity.go/service.go` 还能用，但不利于展示；
- 竞拍规则、排名、落锤应该被看见；
- AI 能力应该在 biz 层有接口，不只是 data stub。

### 5.3 Upgrade `data` to show production seams

Target:

```text
internal/data/
├── data.go
├── lot_repo_memory.go
├── lot_repo_mysql.go
├── bid_repo_redis.go
├── bid_lua.go
├── ranking_redis.go
├── event_log.go
├── transaction.go
└── ai_client_ollama.go
```

High-concurrency design:

```text
PlaceBid
 ↓
Redis Lua atomic script
 ↓
1. read current price
2. validate min increment
3. update current price/version
4. append bid event
5. update ranking zset
6. publish stream event
 ↓
WebSocket broadcaster consumes event
 ↓
clients receive authoritative state
```

This is the strongest backend story for the competition.

### 5.4 Make proto contracts more complete

Current proto only covers basic lot and bid.

Need add:

- `AuctionSession`;
- `BidRule`;
- `RankingItem`;
- `LiveRoomState`;
- `AuctionEvent`;
- `AIPriceSuggestion`;
- `AIAtmosphereRequest/Reply`.

REST/gRPC endpoints should include:

```proto
CreateAuctionSession
CreateLot
StartLot
PlaceBid
GetLiveRoomState
ListRanking
SettleLot
SuggestAuctionRule
GenerateAtmosphereLine
```

### 5.5 Add anti-failure design

Must explicitly handle:

- duplicate bid;
- stale client price;
- WebSocket reconnect;
- bid accepted but broadcast failed;
- Redis success but MySQL delayed;
- AI timeout;
- auction ends while bid arrives;
- network delay near final seconds;
- malicious rapid bids.

These should appear in docs and eventually in tests.

### 5.6 Frontend architecture improvement

Frontend is currently one-page demo. It should split by role:

```text
src/
├── app/
├── pages/
│   ├── LiveRoomPage.tsx       # 观众端
│   └── AdminConsolePage.tsx   # 主播/运营端
├── features/
│   ├── auction/
│   ├── realtime/
│   ├── ranking/
│   └── ai-assistant/
├── shared/
│   ├── api/
│   ├── ws/
│   ├── ui/
│   └── money/
└── types/
```

Competition demo should show two views:

1. 主播后台：上架、规则、开拍、落锤、AI 建议；
2. 观众端：实时出价、排名、气氛官。

## 6. Demo Narrative

比赛展示不应该说“我们做了一个竞拍页面”。

应该说：

> 我们构建了一个直播竞拍交易引擎，把非标品交易拆成竞拍会话、实时出价、原子排名、落锤成交和 AI 运营辅助五个核心模块。系统用 WebSocket 承载毫秒级互动，用 Redis Lua 保证高并发下的状态一致性，用 AI 模型辅助起拍价和直播气氛生成。

Suggested architecture diagram:

```text
React Live Room      React Admin Console
      |                       |
      | HTTP / WebSocket      |
      v                       v
Gateway / Auction HTTP Server / WS Server
      |
      v
Auction Service Layer
      |
      v
Biz: Session / BidRule / Ranking / Settlement / AI Policy
      |
      +--> Redis Lua: atomic bid + ranking zset + event stream
      |
      +--> MySQL: lot/order/bid history persistence
      |
      +--> AI Client: pricing + atmosphere + risk explanation
      |
      v
Realtime Broadcaster → WebSocket clients
```

## 7. Priority List

### Must do next

1. Move WS hub to `internal/realtime`.
2. Split `biz` into session/bid/ranking/settlement/rule.
3. Add Redis Lua design and file placeholders.
4. Expand proto to include session/ranking/event/AI contracts.
5. Add frontend admin/participant page split.

### Should do after

1. Add `third_party/` proto deps.
2. Add `make api` and generated-code workflow.
3. Add docker-compose with Redis/MySQL/Ollama placeholders.
4. Add concurrency test plan.
5. Add architecture diagrams under official docs.

### Do not do yet

1. Do not split five microservices immediately.
2. Do not implement payment/guarantee deposit deeply before bidding engine is solid.
3. Do not spend too much time on UI polish before realtime correctness is demonstrable.

## 8. Final Recommendation

The next best move is not more UI and not more microservices. It is:

> Make `auction/service` a convincing realtime transaction engine.

Concretely, the next implementation batch should be:

```text
server split + realtime module + biz split + Redis/Lua seams + richer proto contracts
```

That gives us the best competition leverage: architecture looks professional, demo remains stable, and the hardest technical claim—high-concurrency real-time bidding consistency—has a clear implementation path.
