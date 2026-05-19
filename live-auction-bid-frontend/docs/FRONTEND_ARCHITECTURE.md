# 前端架构说明

## 1. 当前主流前端架构怎么选

现在 React 项目比较流行的不是把所有代码堆在 `components/`，而是按“业务能力”组织，常见有三类：

1. **页面分层架构**
   - `pages` 放页面；
   - `components` 放组件；
   - `api` 放请求。
   - 优点是简单，适合小项目。

2. **Feature-Sliced / 按功能切片架构**
   - `features/auction` 管竞拍能力；
   - `features/realtime` 管实时连接；
   - `features/ranking` 管排行榜；
   - `shared` 放通用能力。
   - 优点是业务边界清楚，适合中大型 React 项目。

3. **领域驱动前端架构**
   - 前端也按业务领域拆：竞拍、直播间、玩法、订单、用户。
   - 优点是和后端 DDD 更容易对齐。

本项目建议采用 **Feature-Sliced + 页面分层** 的折中方案。

原因：

- 比赛项目需要快速演示，不宜过度复杂；
- 但直播竞拍有明确业务模块，不能继续单文件；
- 后续要做观众端、主播端、玩法引擎，按 feature 拆更稳。

## 2. 当前前端目录

```text
src/
├── app/                    # 应用入口、全局样式、全局装配
├── pages/                  # 页面层
│   ├── live-room/          # 观众直播间页面
│   └── host-console/       # 主播/运营控制台页面
├── features/               # 业务功能模块
│   ├── auction/            # 拍品、出价、竞拍 API
│   ├── realtime/           # WebSocket 连接与房间状态
│   ├── ranking/            # 实时排行榜
│   └── playbook/           # 玩法引擎 UI：信任揭示、群体共振、Duel
├── shared/                 # 通用能力
│   ├── config/             # 环境变量
│   ├── lib/                # 通用工具函数
│   ├── types/              # 通用类型
│   └── ui/                 # 通用 UI 组件
└── main.tsx                # Vite 入口，仅引入 app/main
```

## 3. 分层职责

### app

只放应用装配：

- React root；
- 全局样式；
- 后续 router/provider/store。

不写具体竞拍业务。

### pages

页面负责组合 feature，不直接写太多业务细节。

当前：

- `LiveRoomPage`：观众端直播竞拍页；
- `HostConsolePage`：主播/运营后台预留页。

### features

每个 feature 是一个业务能力。

#### auction

负责：

- 拍品接口；
- 出价组件；
- 拍品展示；
- 当前价格与下一口价。

#### realtime

负责：

- WebSocket 连接；
- 接收服务端事件；
- 同步服务端权威状态；
- 发送出价消息。

#### ranking

负责：

- 展示实时排名；
- 后续可加排名动画、Top2 PK 高亮。

#### playbook

负责玩法引擎 UI：

- 信任揭示卡片；
- 群体共振进度条；
- Duel Mode 双人巅峰竞拍；
- 主播控场提示。

### shared

跨功能共享：

- 环境配置；
- 金额格式化；
- 类型定义；
- 通用 UI。

## 4. 和后端契约的关系

前端不写 mock，不内置模拟服务。

前端只连接真实后端契约：

```text
VITE_API_BASE=http://localhost:8080
VITE_WS_BASE=ws://localhost:8080
```

后端当前契约：

```text
GET  /api/lots
POST /api/lots
POST /api/lots/{id}/bid
POST /api/lots/{id}/settle
WS   /ws/rooms/{roomId}
```

后续应该逐步由 `api/auction/service/v1/auction.proto` 生成类型或接口客户端，减少手写类型漂移。

## 5. 后续演进

下一步建议：

1. 引入 React Router，拆 `/live/:roomId` 和 `/host`；
2. 拆 HostConsole 的商品上架、规则配置、落锤操作；
3. 为 playbook 增加三个组件：
   - `TrustRevealPanel`
   - `CrowdPowerBar`
   - `DuelModeBanner`
4. 增加 API client 统一错误处理；
5. 从 proto/openapi 生成 TypeScript 类型。

## 6. 重要规则

- 不在前端仓库写 mock 服务；
- 不在前端内置假数据冒充真实接口；
- 前端状态以服务端 WebSocket 广播为准；
- 金额统一使用分/厘等最小单位，显示时再格式化；
- 玩法展示要服务于比赛主题：短视频直播互动竞拍新玩法。
