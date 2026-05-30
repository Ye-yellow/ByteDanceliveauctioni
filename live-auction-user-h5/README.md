# live-auction-user-h5

这是“实时竞拍大师”直播竞拍系统的**用户端移动 H5**，不是 PC 主播团队工作台。

PC 后台由主播团队使用，负责添加拍品、排品、开拍、控场和成交处理；本项目面向观众/买家，用于进入直播间、观看模拟直播、查看竞拍拍品、参与出价、实时查看排名、接收被超越/竞拍延时/竞拍结束提醒，并查看成交结果和模拟支付。

## 启动

```bash
npm install
npm run dev
```

访问：

```text
http://localhost:5173/
```

首页会从后端 `GET /api/rooms` 获取可进入的直播间。进入指定直播间时使用：

```text
http://localhost:5173/m/room/{roomId}
```

## 环境变量

```env
VITE_API_BASE_URL=http://localhost:18080
VITE_WS_BASE=ws://localhost:18080
VITE_AUTH_MODE=demo
VITE_DEMO_LIVE_URL=https://your-demo-live-stream.m3u8
VITE_DEMO_BUYER_USERNAME=
VITE_DEMO_BUYER_PASSWORD=
VITE_DEMO_BUYER_NICKNAME=H5 买家
```

- 本地开发不设置 `VITE_API_BASE_URL` / `VITE_WS_BASE` 时，会走 Vite 同源代理转发到 `http://127.0.0.1:18080`，避免浏览器跨端口 CORS 预检失败。
- 对比旧后端或其它本地后端时，可用 `VITE_DEV_PROXY_TARGET=http://127.0.0.1:18081 VITE_DEV_WS_PROXY_TARGET=ws://127.0.0.1:18081 npm run dev` 改代理目标。
- `VITE_AUTH_MODE=demo`：本地开发默认模式，H5 会自动 login/register demo buyer，方便联调出价。
- `VITE_AUTH_MODE=real`：生产模式，H5 不会静默创建 demo 用户；买家必须在直播页登录或注册后才能出价。
- 生产构建必须显式设置 `VITE_AUTH_MODE=real`，避免线上自动创建 demo 买家账号。

## 后端接口要求

- `GET /api/rooms/{roomId}/snapshot`
- `GET /api/rooms`
- `POST /api/lots/{lotId}/bid`
- `GET /api/lots/{lotId}/result`
- `POST /api/orders/{orderId}/mock-pay`
- `WebSocket /ws/rooms/{roomId}`

## 直播模拟

直播区域使用原生 `<video>` 播放 TOS 上的模拟直播视频。

- 默认使用 `VITE_DEMO_LIVE_URL`，没有配置时按内置 TOS MP4 播放列表循环。
- `.m3u8` 仅在浏览器原生支持 HLS 时播放；当前 demo 不再打包 HLS 播放器库。
- 不使用商品图假装直播；商品图只作为 poster / fallback。

## MVP 路由

- `/m/room/:roomId` 用户直播间主页面
- `/m/result/:lotId` 竞拍结果页
- `/m/history` 我的订单和竞拍记录页

## H5 页面布局约定

- 新增页面最外层统一使用 `<main className="mobileShell">`。
- `.mobileShell` 已经固定为当前视口内的手机容器，并在容器内部滚动；不要在普通页面再写 `min-height: 100vh`、`height: 100vh` 或额外顶层 margin，否则 Windows/桌面浏览器会出现页面外层被自动延长。
- 首页、直播间、我的页这类全屏沉浸页可以在 `mobileShell` 后追加专用 class，例如 `homeShell`、`douyinShell`、`profileShell`，但滚动也必须留在专用 shell 内部。
- 页面内容需要滚动时，让内容区或 `mobileShell` 滚动，不让 `body/html/#root` 滚动。
