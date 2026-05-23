# CancelLot 端到端验收清单

> 审查时间：2026-05-20 20:15+08  
> 当前优先级：QA 先围绕一个端到端功能——主播异常取消竞拍 `CancelLot`。  
> 范围：后端契约、biz 状态机、service `reply.result`、data/outbox/WebSocket、前端主播端按钮、观众端提醒、测试目录规则。

## 1. 当前结论

`CancelLot` 已经进入 proto 契约和生成文件：

- `api/auction/service/v1/auction.proto` 已有 `rpc CancelLot(CancelLotRequest) returns (CancelLotReply)`；
- HTTP 路由应为 `POST /api/lots/{lot_id}/cancel`；
- `AuctionEventType` 已有 `AUCTION_EVENT_TYPE_LOT_CANCELLED`；
- `LotStatus` 已有 `LOT_STATUS_CANCELLED`。

但当前仍不能算端到端完成：

- `app/auction/service/internal/service/auction.go` 尚未实现 `CancelLot` 方法；
- `app/auction/service/internal/biz/auction` 尚未发现 `CancelLot` 状态机函数/usecase；
- `data` 层 lot model/schema 尚未显式持久化 `cancel_reason`；
- 前端 `auctionApi.ts` 尚未封装 `cancelLot`；
- 主播端 `HostConsolePage.tsx` 尚无“异常取消/取消竞拍”按钮；
- 观众端已有 `LOT_STATUS_CANCELLED` 提醒逻辑，但需要真实 WebSocket 取消事件验收；
- 未发现专门覆盖取消场景的测试。

因此 QA 验收口径：**先把 CancelLot 从 LIVE 状态一路打通到前端提醒，再回到封顶价、订单等其它 P0。**

## 2. 端到端验收点

### C1. LIVE 可取消

验收标准：

- 主播/运营/admin 登录后，对 `LOT_STATUS_LIVE` 拍品调用 `CancelLot` 成功；
- 返回 `CancelLotReply.result.code == 0`；
- 返回 `lot.status == LOT_STATUS_CANCELLED`；
- 返回 `event.type == AUCTION_EVENT_TYPE_LOT_CANCELLED`；
- lot 版本号递增；
- 取消原因进入事件 `reason`，并进入 lot payload/显式字段（若新增 `cancel_reason` 字段，则数据库 schema 和 GORM model 同步）。

建议测试名：

- `TestCancelLotAllowsLiveLot`
- `TestCancelLotBroadcastsCancelledEvent`

### C2. DRAFT / SETTLED 不可取消

验收标准：

- `LOT_STATUS_DRAFT` 调 `CancelLot` 不改变状态；
- `LOT_STATUS_SETTLED` 调 `CancelLot` 不改变状态；
- service 不返回 Go transport error，必须返回 `CancelLotReply{result: ErrorResult(...)}, nil`；
- `result.code != 0`，message 能明确说明当前状态不可取消，例如 `only live lot can be cancelled`；
- 不生成 `LOT_CANCELLED` WebSocket 事件。

建议测试名：

- `TestCancelLotRejectsDraftLot`
- `TestCancelLotRejectsSettledLot`
- `TestCancelLotUsesResultEnvelopeForInvalidState`

### C3. 取消后不能出价 / 落锤 / Duel

验收标准：

取消成功后：

- 买家调用 `PlaceBid` 返回 `accepted=false`；
- `PlaceBidReply.result.code != 0` 或至少 `accepted=false + reject_reason` 表达业务拒绝；
- 拍品状态保持 `LOT_STATUS_CANCELLED`；
- 主播调用 `SettleLot` 必须失败，状态保持取消；
- 主播调用 `StartDuel` 必须失败，状态保持取消；
- 不创建 bid、不改变 ranking、不创建成交订单。

建议测试名：

- `TestCancelledLotRejectsBid`
- `TestCancelledLotRejectsSettle`
- `TestCancelledLotRejectsDuel`

### C4. WebSocket 广播

验收标准：

- `CancelLot` 成功时，后端通过现有 realtime publisher 广播 `AUCTION_EVENT_TYPE_LOT_CANCELLED`；
- event 至少包含：`room_id`、`lot_id`、`lot.status=CANCELLED`、`reason`、`occurred_at_unix_ms`；
- 事件先随业务状态同事务落 `auction_events`，再走 Redis Stream/outbox；
- outbox worker 失败时仍可补推；
- 前端重连后 `GetRoomSnapshot` 能恢复已取消状态。

建议测试/检查：

- usecase 测事件类型进入测试 publisher；
- data 测事件落库；
- 手工联调打开观众端，主播点取消，观众端无需刷新出现取消提醒。

### C5. 前端主播端按钮

验收标准：

- `src/features/auction/api/auctionApi.ts` 增加 `cancelLot(lotId, reason?)`；
- `src/pages/host-console/HostConsolePage.tsx` 在当前拍品区增加“异常取消”按钮；
- 按钮仅在 `selected.status === 'LOT_STATUS_LIVE'` 且非 busy 时可点；
- 点击前至少提供默认原因，例如 `主播异常取消`；如加 confirm，文案清晰说明这是外部可见状态变更；
- 成功后页面选中 lot 变为 CANCELLED，notice 显示取消成功；
- 失败时沿用 `resultMessage(e)` 展示 `reply.result.message`。

### C6. 前端观众端取消提醒

当前 `useAuctionRoom.ts` 已有基础逻辑：

```ts
if (prevLot.status === 'LOT_STATUS_LIVE' && nextLot.status === 'LOT_STATUS_CANCELLED') {
  pushFeed('ended', '竞拍已取消，请等待主播重新安排。');
}
```

验收标准：

- 收到 `AUCTION_EVENT_TYPE_LOT_CANCELLED` 后，观众端当前 lot 状态变为 CANCELLED；
- 提醒 feed 出现“竞拍已取消...”；
- 出价按钮在取消状态不可继续提交，或提交后后端稳定拒绝并提示；
- 重连后通过 snapshot 仍显示取消状态，不回退到 LIVE。

### C7. `reply.result` 错误语义

验收标准：

- `CancelLot` 的权限错误、lot 不存在、非法状态、缺少 lot_id/原因等可预期错误，都包装在 `CancelLotReply.result`；
- service 方法返回 `(*CancelLotReply, nil)`，不要 `reply + error` 双语义；
- 前端只需要 `assertOkResult`/`resultMessage` 一套解析路径；
- 未知系统错误可给统一文案，避免把 MySQL/Redis/JWT 内部错误直出。

### C8. 测试目录规则

验收标准：

- 新增测试全部放在 `app/auction/service/test`；
- 不在 `app/auction/service/internal/**` 新增 `*_test.go`；
- 测试覆盖状态机、usecase、service result envelope，必要时用 fake publisher/fake repo；
- 若涉及 data 事务/outbox，可继续在集中测试目录内构造测试 Store。

检查命令：

```bash
find app/auction/service/internal -name '*_test.go' -print
```

期望无输出。

## 3. 推荐实现顺序

1. `biz/auction/lot.go` 增加 `CancelLot(lot, reason, nowMs)` 状态机：只允许 LIVE -> CANCELLED；
2. `biz/auction/usecase.go` 增加 `CancelLot(ctx, lotID, operatorID, reason)`：查 lot、状态机、构造 `LOT_CANCELLED` event、保存、广播；
3. `service/auction.go` 实现 `CancelLot`：权限同 `StartLot/SettleLot`，所有错误走 `CancelLotReply.result`；
4. data/schema 如需展示取消原因，补 `cancel_reason` 显式列；否则至少确保 payload 和 event reason 持久化；
5. frontend API 增加 `cancelLot`；
6. HostConsole 增加 LIVE-only “异常取消”按钮；
7. 联调观众端 WebSocket 提醒和 snapshot 恢复；
8. 补 `app/auction/service/test` 下 CancelLot 专项测试。

## 4. 最小手工验收脚本

前提：后端、MySQL、Redis、Consul、前端均已启动，主播端已登录 admin，观众端已登录 buyer。

1. 主播端创建草稿拍品；
2. 验证 DRAFT 状态“异常取消”按钮不可点，或直接调用接口返回 `result.code != 0`；
3. 主播点击“开拍”；
4. 观众端确认看到 LIVE 拍品；
5. 主播点击“异常取消”；
6. 主播端显示状态 CANCELLED；
7. 观众端无需刷新出现取消提醒；
8. 观众尝试出价，被拒绝且文案来自后端 result/reject_reason；
9. 主播尝试落锤/Duel，被拒绝且文案来自 result；
10. 刷新观众端，snapshot 仍恢复为 CANCELLED。

## 5. 当前阻断清单

- [ ] `AuctionService.CancelLot` 未实现；
- [ ] `AuctionUsecase.CancelLot` 未实现；
- [ ] biz 状态机 `CancelLot` 未实现；
- [ ] 前端 `cancelLot` API 未实现；
- [ ] 主播端取消按钮未实现；
- [ ] CancelLot 专项测试未实现；
- [ ] 取消原因持久化策略未明确。
