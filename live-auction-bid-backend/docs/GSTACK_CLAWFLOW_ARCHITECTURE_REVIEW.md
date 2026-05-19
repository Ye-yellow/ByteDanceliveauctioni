# GStack × ClawFlow Architecture Review

Date: 2026-05-19
Scope: `auction-backend` + frontend collaboration boundary
Mode: planning/review only; no feature implementation in this pass.

## 1. Review Premise

The backend should follow the user's familiar Odin/Kratos mental model:

```text
api proto
 ↓ generated http/grpc/errors
app/<service>/service/internal/server
 ↓
internal/service
 ↓
internal/biz
 ↓
internal/data
```

The current repository has already moved from a small single-service DDD layout to a Kratos multi-service skeleton:

```text
auction-backend/
├── api/auction/service/v1/auction.proto
├── app/auction/service/
│   ├── cmd/server/
│   ├── configs/
│   └── internal/{conf,server,service,biz,data}
├── deploy/
├── docs/
├── Makefile
└── app_makefile
```

This is the right direction. The remaining work is to make it more generated-code-friendly, more production-realistic, and more competition-presentable.

## 2. Current Strengths

### 2.1 Directory direction is now correct

The repository now has the same high-level shape as an Odin/Kratos codebase:

- `api/` for protocol contracts;
- `app/` for service implementations;
- `internal/server/service/biz/data` for request flow;
- `deploy/` and `Makefile` placeholders for delivery.

This makes the project easier to explain to collaborators who already know the Odin structure.

### 2.2 Domain boundary is visible

The竞拍 domain is already recognizable:

- `Lot`
- `Bid`
- `RankingItem`
- `LotStatus`
- `PlaceBid`
- `Settle`
- `RequiredNextBid`

This is a good base for competition explanation: the core engine is not just CRUD.

### 2.3 Real-time path already exists

The WebSocket `Hub` can already broadcast:

- `lot.updated`
- `bid.accepted`
- `lot.settled`

This gives the project a live demo path early, which is important for a competition.

## 3. Main Gaps

### 3.1 `api/` is not yet the true source of generated interfaces

Current `auction.proto` exists, but the running HTTP/WS adapters are still hand-written. This is acceptable for prototype, but for Kratos style we should move toward:

```text
api/auction/service/v1/auction.proto
 ↓ make api
*.pb.go
*_http.pb.go
*_grpc.pb.go
```

Then `internal/server/http.go` should register generated handlers instead of manually wiring every route.

Priority: High.

### 3.2 `server` mixes REST and WebSocket responsibilities

Current server layer is usable, but it should be split:

```text
internal/server/
├── http.go
├── grpc.go
├── websocket.go
├── middleware.go
└── server.go
```

`server/ws/hub.go` is currently under server. For a real-time竞拍 engine, realtime is a first-class capability. Better shape:

```text
internal/realtime/
├── hub.go
├── room.go
├── message.go
├── broadcaster.go
└── presence.go
```

Then server owns connection upgrade; realtime owns room/session/broadcast semantics.

Priority: High.

### 3.3 `biz` files are still too generic

Current:

```text
internal/biz/entity.go
internal/biz/service.go
internal/biz/repository.go
```

Better for readability and domain storytelling:

```text
internal/biz/
├── lot.go
├── bid.go
├── ranking.go
├── settlement.go
├── rule.go
├── event.go
└── repo.go
```

This makes the competition narrative stronger: bidding, ranking, settlement, and rules are separated.

Priority: Medium-high.

### 3.4 `data` needs production seams

Current data layer is memory-only. Good for demo, but the high-concurrency topic needs Redis/MySQL seams early:

```text
internal/data/
├── data.go
├── lot_repo_memory.go
├── lot_repo_mysql.go
├── bid_redis.go
├── bid_lua.go
├── transaction.go
└── ai_client.go
```

For the competition, Redis Lua atomic bidding is a key technical highlight:

- validate minimum bid;
- update current price;
- append bid event;
- update ranking ZSET;
- publish room event.

Priority: High.

### 3.5 AI is currently only a stub

The topic explicitly asks for open-source AI model exploration. We should not bury it as only a `StubAI` in data.

Recommended staged design:

Short term inside auction service:

```text
internal/biz/ai_pricing.go
internal/biz/ai_atmosphere.go
internal/data/ai_client_ollama.go
```

Later split service:

```text
api/ai/service/v1/
app/ai/service/
```

AI capabilities to present:

- dynamic start price suggestion;
- min increment recommendation;
- live atmosphere copywriting;
- suspicious bidding/risk explanation.

Priority: Medium-high, because it is part of problem statement.

### 3.6 Missing `third_party/` and proto toolchain

For Kratos-style codegen, add:

```text
third_party/google/api/
third_party/validate/
```

And scripts/Makefile targets:

```bash
make api
make proto
make errors
make wire
```

Priority: Medium.

## 4. Recommended Next Architecture

```text
auction-backend/
├── api/
│   ├── auction/service/v1/
│   ├── room/service/v1/
│   └── ai/service/v1/
├── app/
│   ├── auction/service/
│   │   ├── cmd/server/
│   │   ├── configs/
│   │   └── internal/
│   │       ├── conf/
│   │       ├── server/
│   │       ├── service/
│   │       ├── biz/
│   │       ├── data/
│   │       └── realtime/
│   └── ai/service/              # optional after MVP
├── pkg/
│   ├── snowflake/
│   ├── money/
│   ├── pagination/
│   └── wsmsg/
├── third_party/
├── deploy/
└── docs/
```

Do not split too many services immediately. First make `auction/service` strong, then split `ai/service` only when the AI boundary becomes real.

## 5. Implementation Priority

### Phase 1 — Structure hardening

1. Split `internal/server/http.go` into HTTP / WebSocket / gRPC placeholders.
2. Move `server/ws` to `internal/realtime`.
3. Split `biz` into lot/bid/ranking/settlement/rule/repo.
4. Add `third_party/` placeholders and `make api` documentation.

### Phase 2 — High-concurrency story

1. Add Redis data layer interface.
2. Add Lua script placeholder for atomic bid.
3. Add ranking ZSET design doc.
4. Add concurrency test plan.

### Phase 3 — AI story

1. Add AI pricing/atmosphere interface in biz.
2. Add Ollama-compatible data client skeleton.
3. Add `api/ai/service/v1` or keep internal until demo is stable.

### Phase 4 — Competition delivery

1. Add architecture diagram.
2. Add sequence diagrams for bidding and settlement.
3. Add demo script.
4. Add docker-compose for frontend + backend + redis + mysql.

## 6. Decision

Accepted direction:

- Keep Odin/Kratos multi-service repo style.
- Keep `auction/service` as the first real service.
- Do not prematurely split `room/order/ai` services until the core bidding path is stable.
- Prioritize realtime consistency and Redis atomic bidding over cosmetic folders.

Recommended next step:

Implement Phase 1 only, then commit and push. That will make the architecture cleaner without overbuilding.
