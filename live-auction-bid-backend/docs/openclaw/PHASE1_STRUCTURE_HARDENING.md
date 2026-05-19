# Phase 1 Structure Hardening

Date: 2026-05-19
Owner note: OpenClaw-generated implementation note. Keep this under `docs/openclaw/`.

## What Changed

This batch implements the first architecture-hardening step from the office-hours review.

### 1. Realtime is now first-class

Moved WebSocket room/fanout concepts out of `internal/server/ws` into:

```text
app/auction/service/internal/realtime/
├── broadcaster.go
├── client.go
├── hub.go
├── message.go
├── presence.go
└── room.go
```

`server` now owns HTTP/WebSocket transport concerns. `realtime` owns room membership, message envelope, online count, and event fanout.

### 2. Server layer split

Added clearer server files:

```text
internal/server/http.go
internal/server/websocket.go
internal/server/grpc.go
internal/server/middleware.go
internal/server/server.go
```

`grpc.go` and `middleware.go` are placeholders for Kratos-generated gRPC registration and middleware setup.

### 3. Biz layer split by domain concept

Replaced generic `entity.go` / `repository.go` with domain files:

```text
internal/biz/auction_session.go
internal/biz/lot.go
internal/biz/bid.go
internal/biz/bid_rule.go
internal/biz/ranking.go
internal/biz/settlement.go
internal/biz/event.go
internal/biz/ai_pricing.go
internal/biz/ai_atmosphere.go
internal/biz/repo.go
internal/biz/money.go
```

This makes the competition story easier to explain: session, lot, bid, ranking, settlement, rule, and AI are visible domain concepts.

### 4. Data layer production seams

Added placeholders for production infrastructure:

```text
internal/data/data.go
internal/data/lot_repo_memory.go
internal/data/lot_repo_mysql.go
internal/data/bid_repo_redis.go
internal/data/bid_lua.go
internal/data/ranking_redis.go
internal/data/event_log.go
internal/data/transaction.go
internal/data/ai_client_ollama.go
internal/data/ai_client_stub.go
```

The important competition hook is `bid_lua.go`: it documents the intended Redis Lua atomic bid flow.

### 5. Proto contracts expanded

Expanded `api/auction/service/v1/auction.proto` with:

- AuctionSession
- BidRule
- PlaceBidReply
- RankingItem
- LiveRoomState
- Settlement
- AuctionEvent
- AIPriceSuggestion
- AI atmosphere messages

Also added a Phase 2 AI proto placeholder:

```text
api/ai/service/v1/ai.proto
```

### 6. Proto deps placeholder

Added:

```text
third_party/
```

for future `google/api`, `validate`, and Kratos proto dependencies.

## Validation

This machine currently has no Go toolchain, so full `go test ./...` was not run here.

Completed checks:

- Required structure files exist.
- Old `internal/server/ws` references removed from code.
- No stale `live-auction-bid/backend/internal/...` imports in app/api.
- `git diff --check` passed.

## Next Recommended Batch

1. Install Go/Kratos toolchain and run compile.
2. Add real `make api` generation with `third_party` proto deps.
3. Implement Redis Lua bid path behind a feature flag.
4. Add frontend Admin Console / Live Room split.
