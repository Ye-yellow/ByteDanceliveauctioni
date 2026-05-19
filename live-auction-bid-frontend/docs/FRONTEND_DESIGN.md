# Frontend Design

React 直播竞拍控制台/观众端初版。

## 页面能力

- 直播竞拍房间状态展示
- WebSocket 连接状态
- 当前拍品价格、最低下一口、倒计时
- 实时排行榜
- 一键加价/自定义出价
- AI 气氛官文案展示

## 后端连接

默认连接：

- `VITE_API_BASE=http://localhost:8080`
- `VITE_WS_BASE=ws://localhost:8080`

可在部署环境中覆盖。
