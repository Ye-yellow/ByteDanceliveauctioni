# Live Auction Bid Frontend

直播竞拍系统商家/主播/运营后台前端：React + TypeScript + Vite。

## 当前项目边界

当前 `live-auction-bid-frontend` **只承载后台 Web**：主播主账号、场控、商品助理、订单客服、数据复盘使用的 PC 工作台。

- 本项目入口：`/home`、`/login`、`/host`、`/admin`
- 本项目负责：拍品准备、竞拍玩法配置、开拍/取消/结束、信任揭示、后台状态看板、后续成交订单/支付/会员工作台
- 不在本项目实现：用户 H5 竞拍端、小程序端、观众直播间入口或页面
- 用户 H5 / 小程序端后续应作为独立项目复用同一后端 API/WebSocket 契约

Home 页已按浅蓝/粉、梦幻、手绘、玻璃拟态方向冻结；后续除非用户明确解冻，不主动改 Home 视觉与布局。

## 能力

- 工作台登录：调用真实后端 `POST /api/users/login`，使用 JWT token adapter
- 拍品后台：创建草稿、列表、开拍、信任揭示、Duel、落锤、异常取消
- 状态总览：以真实 `GET /api/lots?room_id=demo` 返回为唯一数据源
- 后台预留：会员与角色、订单与支付、运营监控仅做禁用态/契约待扩展说明，不请求不存在接口，不构造 mock 数据

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

Do not add mock services or mock data into this frontend repository. The frontend should connect to the real backend API/WebSocket contract, or use externally managed test environments. Features without a backend contract must stay disabled and documented as contract extensions.

## 前端架构

当前采用 Feature-Sliced + 页面分层结构：

```text
src/app          应用入口和全局装配
src/pages        页面层：home / login / host-console
src/features     业务功能：auth / auction
src/shared       通用配置、类型、工具、后台 UI
```

详细说明见：`docs/FRONTEND_ARCHITECTURE.md`。

## API 类型生成

前端类型来自后端 OpenAPI 契约，不手写后端 DTO。

```bash
npm run generate:api
```

详细说明见：`docs/API_CONTRACT_GENERATION.md`。

## 本地联调启动

后端默认运行在 `18080`，前端默认运行在 `5173`。

```bash
# 终端 1：后端仓库
cd ../live-auction-bid-backend
export PATH=/home/ye/OpenClaw/state/tools/go/bin:$PATH
go run ./app/auction/service/cmd/server

# 终端 2：前端仓库
cd ../live-auction-bid-frontend
npm run dev
```

访问：

```text
Home：http://127.0.0.1:5173/home
登录：http://127.0.0.1:5173/login
后台：http://127.0.0.1:5173/host
后台别名：http://127.0.0.1:5173/admin
```

说明：前端通过 Vite proxy 访问后端，所以浏览器只需要访问 `5173`。如果 `5173` 拒绝连接，说明前端 dev server 停了，需要重新运行 `npm run dev`。

## 后台管理中心入口

参考 cli-proxy 与 sub2api 的后台范式后，主播端已升级为 PC 工作台：

```text
http://127.0.0.1:5173/host
http://127.0.0.1:5173/admin
```

已接入真实后端契约：创建拍品、列表、开拍、信任揭示、Duel、落锤、异常取消。会员/成交订单/支付模块仅保留禁用态结构预留，不构造 mock 数据。
