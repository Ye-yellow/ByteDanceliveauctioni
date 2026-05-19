# Live Auction Bid Frontend

直播电商竞拍系统前端：React + TypeScript + Vite。

## 能力

- 展示直播竞拍拍品
- WebSocket 实时同步价格和排名
- 一键加价/自定义出价
- AI 气氛官文案展示
- 响应式比赛演示页面

## 本地运行

```bash
npm install
npm run dev
```

默认连接真实后端：

```bash
VITE_API_BASE=http://localhost:8080
VITE_WS_BASE=ws://localhost:8080
```

如果后端地址不同：

```bash
VITE_API_BASE=http://your-backend:8080 VITE_WS_BASE=ws://your-backend:8080 npm run dev
```

## 后端仓库

后端已拆分到兄弟项目：`live-auction-bid-backend`。
## Development Rule

Do not add mock services or mock data into this frontend repository. The frontend should connect to the real backend API/WebSocket contract, or use externally managed test environments.

## 前端架构

当前采用 Feature-Sliced + 页面分层结构：

```text
src/app          应用入口和全局装配
src/pages        页面层：观众直播间、主播控制台
src/features     业务功能：auction/realtime/ranking/playbook
src/shared       通用配置、类型、工具、UI
```

详细说明见：`docs/FRONTEND_ARCHITECTURE.md`。
