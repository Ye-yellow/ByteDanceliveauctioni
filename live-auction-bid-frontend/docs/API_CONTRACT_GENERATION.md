# 前端 API 契约同步说明

## 当前状态

后端最新契约源头是：

```text
../live-auction-bid-backend/api/auction/service/v1/auction.proto
../live-auction-bid-backend/api/auction/service/v1/user.proto
../live-auction-bid-backend/api/auction/service/v1/shop.proto
```

`../live-auction-bid-backend/openapi/auction.openapi.json` 已同步到当前 proto HTTP binding，前端可用已有命令生成类型：

```bash
npm run generate:api
```

生成产物：

```text
src/shared/api/generated/auction.schema.ts
```

当前前端手写 DTO 集中在：

```text
src/shared/api/types.ts
```

它按后端 proto 的 Result Envelope、JWT 用户系统、AuctionEvent、UserService、AuctionService 同步；生成 schema 是合同校验与补全来源，业务 feature 仍通过本地 adapter/normalizer 隔离后端字段细节。

## 工程思考与设计模式

- **Contract Anti-Corruption Layer**：`src/shared/api/types.ts` 是前端契约防腐层，业务 feature 不直接依赖后端生成细节。
- **Result Envelope Adapter**：`src/shared/api/result.ts` 统一检查 `reply.result.code`，让页面不再同时处理 HTTP error 与业务 error 两套语义。
- **Auth Token Adapter**：`features/auth/api` 只负责 token 存取和 Authorization header，竞拍 feature 不直接碰 localStorage。
- **Realtime Normalizer**：WebSocket 事件进入 UI 前先规范化事件枚举，兼容后端 gorilla JSON 可能输出数字 enum 的事实。

## 生成规则

修改后端 proto HTTP API 后必须确认 OpenAPI 覆盖：

- AuctionService 全部 reply envelope；
- UserService 登录/注册/刷新/me/admin；
- ShopService 商品、地址、保证金、商城订单、统一订单；
- Authorization header；
- proto int64 在前端类型中允许 `number | string`；
- WebSocket AuctionEvent 的 enum 表达。

规则：不能保留 stale generated schema 与手写契约并行误导；手写 DTO 只能作为薄映射层，不能成为新的事实源。

## CancelLot 前端预期契约（待后端落地对齐）

本轮前端已按课题 P0“主播异常取消”预留真实接口 adapter，不写 mock：

```text
POST /api/lots/{lot_id}/cancel
Authorization: Bearer <anchor/operator/admin token>
body: { "lotId": "...", "reason": "主播设备/商品状态异常，竞拍取消" }
reply: { "lot": { "status": "LOT_STATUS_CANCELLED", "cancelReason": "..." }, "event": { "type": "AUCTION_EVENT_TYPE_LOT_CANCELLED", "reason": "..." }, "result": { "code": 0, "message": "ok" } }
```

需要后端最终确认/补齐：

- RPC/HTTP path 是否固定为 `/api/lots/{lot_id}/cancel`；
- 请求字段名是否为 `reason`，是否还需要 `operator_id`（JWT 后建议不需要前端传）；
- `Lot` 是否增加 `cancel_reason` 并在 proto JSON 中映射为 `cancelReason`；
- WebSocket 是否广播 `AUCTION_EVENT_TYPE_LOT_CANCELLED`，或继续用 `LOT_UPDATED + lot.status=CANCELLED`；
- 业务错误继续只走 `reply.result`，例如非 LIVE 取消、无权限、版本冲突。
