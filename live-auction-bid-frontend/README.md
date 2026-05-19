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

默认连接后端：

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
