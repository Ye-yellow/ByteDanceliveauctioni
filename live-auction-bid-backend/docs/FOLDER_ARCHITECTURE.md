# 文件夹架构篇

本后端参考 Odin 的 Kratos 多服务仓库组织方式。先按顶层目录理解：

```text
auction-backend/
├── api/          # 对外协议层：proto + 后续生成 pb/http/grpc/errors
├── app/          # 服务实现区，每个子服务一套 Kratos 分层
├── pkg/          # 跨服务共享工具包预留
├── third_party/  # proto 编译依赖预留
├── deploy/       # Docker / Kubernetes / compose 部署配置
├── docs/         # 项目文档
├── Makefile      # 顶层构建入口
└── app_makefile  # 服务通用构建模板
```

## api/

`api/` 只定义协议，不写业务逻辑。

```text
api/auction/service/v1/auction.proto
```

后续竞拍接口如创建拍品、出价、落锤、查询房间状态，都应该先落到 proto，再生成 HTTP/gRPC 代码。

## app/

当前只有竞拍核心服务：

```text
app/auction/service/
├── cmd/server/       # 进程入口：main.go + wire.go
├── configs/          # 环境配置 yaml
├── internal/conf/    # 配置结构
├── internal/server/  # HTTP/WebSocket/gRPC server 注册
├── internal/service/ # 协议适配层：接 req，调 usecase
├── internal/biz/     # 业务核心：竞拍规则、出价、排名、落锤
├── internal/data/    # 数据层：Memory/Redis/MySQL/RPC/AI client
└── Makefile
```

按请求流记：

```text
React / Admin / App
 ↓ HTTP / WebSocket
api/auction/service/v1/*.proto
 ↓ 生成代码/适配
internal/server
 ↓ 注册路由、管理连接
internal/service
 ↓ 请求适配、用例编排
internal/biz
 ↓ 业务规则
internal/data
 ↓
Memory / Redis / MySQL / AI Model
```

## internal/server

只做暴露层，不塞业务：

- `/api/lots` REST 路由
- `/ws/rooms/{roomId}` WebSocket 房间
- 后续补 gRPC、日志、恢复、鉴权、限流中间件

## internal/service

协议适配层，像前台接待：

1. 接收请求 DTO；
2. 转为业务参数；
3. 调 `biz.DomainService`；
4. 返回响应。

不要在这里堆竞拍规则。

## internal/biz

业务核心层：

- 拍品状态机：DRAFT / LIVE / SETTLED
- 最低加价校验
- 最高价与 winner 更新
- 反狙击延时
- 排名计算
- 落锤成交

## internal/data

数据层：

- 当前：内存仓储 + AI Stub
- 下一步：Redis Lua 原子出价、MySQL 持久化、AI 模型 HTTP client

业务层通过接口调用 data，不直接到处飞 `gorm.DB` 或 Redis client。

## 后续可拆服务

比赛做大后可以演进成：

```text
app/auction/service  # 竞拍核心
app/room/service     # 直播间/主播/观众
app/order/service    # 成交订单/保证金/支付
app/ai/service       # AI 定价/气氛官/风控
app/gateway/service  # 网关聚合
```
