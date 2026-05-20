# Live Auction Bid Frontend

直播电商竞拍系统前端：React + TypeScript + Vite。

## 项目最强纲领

前端必须服务《抖音电商AI全栈课题-直播竞拍全栈系统（宣讲版）》PDF 的评分点，不再做普通 demo 页面。

前端 P0：移动 H5 观众端、WebSocket 心跳保活与自动重连、断线后快照恢复、实时排名一致展示、被超越/领先/延时/结束提醒、模拟支付与历史竞拍结果展示。

全项目最高纲领见后端根文档：`../live-auction-bid-backend/PROJECT_CHARTER.md`。

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
观众端：http://127.0.0.1:5173/
主播端：http://127.0.0.1:5173/host
```

说明：前端通过 Vite proxy 访问后端，所以浏览器只需要访问 `5173`。如果 `5173` 拒绝连接，说明前端 dev server 停了，需要重新运行 `npm run dev`。

## 后台管理中心入口

参考 cli-proxy 与 sub2api 的后台范式后，主播端已升级为 PC 管理后台：

```text
http://127.0.0.1:5173/host
http://127.0.0.1:5173/admin
```

已接入真实后端契约：创建拍品、列表、开拍、信任揭示、Duel、落锤、异常取消。用户/订单/支付模块仅保留结构预留，不构造 mock 数据。
