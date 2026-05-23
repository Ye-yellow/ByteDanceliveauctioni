# 06-夜间多 Agent 开发方案

> 状态：夜间开发执行草案
> 适用场景：用户明确说“我睡了 / 开始夜间开发 / 今晚跑起来”之后才启动。
> 当前约束：夜间开发只保留工作区变更，不 commit，不 push。

## 1. 目标

夜间开发不是让一个 agent 闷头重构，而是基于 `docs/openclaw/v1/agents/` 下已经拆好的角色工作间，按产品、架构、后端、前端、测试答辩分工推进。

今晚目标控制为：

```text
把 liveauction 从“能跑的原型”推进到“V1 demo 闭环更清晰、代码边界更干净、测试能兜底、答辩有亮点”。
```

不追求大而全，不做无授权大重构。

## 2. 硬规则

- 不 commit。
- 不 push。
- 不长期启动 dev server。
- 不在前端写 mock 服务或模拟数据。
- 不新增无价值的 `model`、`mapper`、`command`、小 helper。
- 后端关键入参采用 fail-fast，不静默补默认值。
- 返回给调用方的错误/返回信息用英文。
- Go 测试放在服务项目的 `test/` 目录，例如 `app/auction/service/test/`。
- 每个 agent 只在自己职责边界内判断，跨角色影响写入 `HANDOFF.md`。

## 3. 总控角色

### night commander

职责：

- 读取本方案和各角色工作间；
- 分波次启动角色 agent；
- 约束每个 agent 的修改范围；
- 收集各角色产出；
- 检查工作区 diff 和测试结果；
- 生成早晨交接。

不负责：

- 不直接做大代码改动；
- 不替产品/架构做最终判断；
- 不 commit / push。

## 4. 第一波：产品、架构、后端主链路

### 4.1 产品需求负责人：`product-owner`

工作目录：

```text
docs/openclaw/v1/agents/product-owner/
```

任务：

- 明确 V1 产品边界；
- 梳理主播、观众、系统三类角色；
- 明确 V1 demo 主流程；
- 输出今晚能做/不能做清单；
- 补充验收标准。

建议产出：

```text
docs/openclaw/v1/agents/product-owner/outputs/v1-night-product-scope.md
```

### 4.2 玩法产品 / 交易玩法设计

暂挂在 `product-owner` 下，不先新建独立目录。

任务：

- 梳理信任揭示竞拍；
- 梳理群体共振竞拍；
- 梳理 Duel Auction；
- 判断 V1 demo 主玩法和辅助玩法；
- 明确前端展示点、主播操作点、后端字段/事件建议；
- 避免项目退化成普通“出价 + 排行榜”。

建议产出：

```text
docs/openclaw/v1/agents/product-owner/outputs/playbook-product-plan.md
```

### 4.3 系统架构负责人：`system-architect`

工作目录：

```text
docs/openclaw/v1/agents/system-architect/
```

任务：

- 检查 proto、service、biz、data、realtime 边界；
- 明确 V1 内存版和后续 MySQL/Redis 演进关系；
- 检查事件流、状态机、服务端权威、幂等、原子出价表达是否清楚；
- 输出架构风险和改动建议。

建议产出：

```text
docs/openclaw/v1/agents/system-architect/outputs/v1-night-architecture-review.md
```

### 4.4 后端交易核心负责人：`backend-auction-core`

工作目录：

```text
docs/openclaw/v1/agents/backend-auction-core/
```

任务：

- 检查 `internal/biz/auction` 的 fail-fast 规则；
- 检查 CreateLot / StartLot / PlaceBid / SettleLot 状态机；
- 去掉静默默认值；
- 不抽低价值 helper；
- 补充最小必要测试。

建议产出：

```text
docs/openclaw/v1/agents/backend-auction-core/outputs/v1-night-core-review.md
```

### 4.5 后端实时链路负责人：`backend-realtime`

工作目录：

```text
docs/openclaw/v1/agents/backend-realtime/
```

任务：

- 检查 WebSocket 事件广播；
- 检查 snapshot 恢复；
- 检查事件类型是否覆盖前端 demo；
- 只做低风险修补。

建议产出：

```text
docs/openclaw/v1/agents/backend-realtime/outputs/v1-night-realtime-review.md
```

## 5. 第二波：前端与 QA

### 5.1 前端观众端负责人：`frontend-live-room`

任务：

- 检查观众端是否连接真实后端；
- 检查当前拍品、当前价、出价、排行榜、信任卡片、Duel 展示；
- 不写 mock。

### 5.2 前端主播端负责人：`frontend-host-console`

任务：

- 检查创建拍品、开拍、揭示卡片、落锤流程；
- 检查与后端 proto / HTTP JSON 的契约适配；
- 不做无授权 UI 大改版。

### 5.3 测试与答辩负责人：`qa-demo-defense`

任务：

- 整理 demo 检查清单；
- 补充最小测试；
- 输出答辩亮点：直播互动玩法、服务端权威状态机、实时广播、可演进架构。

## 6. 每个 agent 必须写回的内容

每个角色完成后必须更新：

```text
docs/openclaw/v1/agents/<role>/outputs/
docs/openclaw/v1/agents/<role>/HANDOFF.md
docs/openclaw/v1/agents/<role>/memory/MEMORY.md
```

如果只做审查、不改代码，也要写清楚审查结论。

## 7. 早晨交接格式

早晨交接必须包含：

```text
1. 总体结论
2. git status
3. git diff --stat
4. 后端改动摘要
5. 前端改动摘要
6. 各角色 agent 产出列表
7. 测试/构建结果
8. 风险与 TODO
9. 是否有服务仍在运行
```

如果发现误 commit / push，只报告事实，不擅自 reset / revert。
