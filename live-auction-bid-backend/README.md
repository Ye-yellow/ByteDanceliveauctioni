# Live Auction Bid Backend

直播电商竞拍系统后端：Go + Kratos 风格工程 + DDD 分层 + WebSocket 实时出价。

## 能力

- 竞拍商品上架/查询
- 自定义起拍价、加价幅度、竞拍时长
- WebSocket 房间加入、实时出价、排行榜广播
- 最高价更新、最低加价校验、落锤成交
- AI 气氛官/动态定价扩展接口

## 架构

```text
cmd/auction-server              启动入口
configs/                        配置
api/auction/v1                  Proto 契约
internal/domain/auction         领域模型与领域服务
internal/application/auction    用例编排
internal/infrastructure/memory  内存仓储与 AI Stub
internal/interfaces/http        REST 接口
internal/interfaces/ws          WebSocket Hub
```

## 本地运行

```bash
go mod tidy
go run ./cmd/auction-server
```

默认地址：

- HTTP: `http://localhost:8080`
- WebSocket: `ws://localhost:8080/ws/rooms/demo`

## 前端仓库

前端已拆分到兄弟项目：`live-auction-bid-frontend`。

前端通过环境变量连接后端：

```bash
VITE_API_BASE=http://localhost:8080
VITE_WS_BASE=ws://localhost:8080
```
