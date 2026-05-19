# Backend Architecture

本后端参考 Odin/Kratos 多服务仓库风格，而不是简单单体目录。

## 顶层

```text
auction-backend/
├── api/            # proto 协议定义，不写业务逻辑
├── app/            # 服务实现，每个服务独立 Kratos 分层
├── pkg/            # 多服务共享能力
├── deploy/         # 部署配置
└── docs/           # 文档
```

## api/

当前协议：

```text
api/auction/service/v1/auction.proto
```

后续建议通过 `make api` 生成：

- `*.pb.go`
- `*_http.pb.go`
- `*_grpc.pb.go`
- errors code

## app/auction/service/

```text
cmd/server/          进程入口
configs/             yaml 配置
internal/conf/       配置结构
internal/server/     HTTP/WebSocket/gRPC server，注册路由
internal/service/    协议适配层，转 usecase
internal/biz/        业务核心层，竞拍规则
internal/data/       数据层，DB/Redis/RPC/AI client
```

## 分层职责

### server

只做暴露层：

- REST API
- WebSocket 房间
- 后续 gRPC 注册
- 中间件、CORS、Recovery、Logging

### service

协议适配层：

- 接收请求 DTO
- 调用 biz/domain service
- 不塞复杂业务规则

### biz

竞拍核心规则：

- 拍品状态
- 最低加价
- 出价时间窗
- 反狙击延时
- 排名计算
- 落锤成交

### data

数据访问：

- 当前：内存仓储、AI Stub
- 下一步：Redis Lua 原子出价、MySQL/GORM 持久化、Ollama/Qwen AI client

## 比赛扩展路线

后续如果拆多服务：

```text
app/auction/service      竞拍核心
app/room/service         直播间/主播/观众
app/payment/service      保证金/支付/订单
app/ai/service           AI 定价/气氛官/风控
app/gateway/service      网关聚合
```
