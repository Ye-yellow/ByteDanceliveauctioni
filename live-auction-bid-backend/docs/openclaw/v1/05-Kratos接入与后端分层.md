# V1 Kratos 接入与后端分层说明

## 1. 当前问题

上一版后端虽然能跑，但有两个问题：

1. `internal/service` 没有真正接 Kratos proto 生成的 service；
2. `internal/biz` 文件都堆在一个目录，后续玩法、排名、出价、信任揭示继续增加会变乱。

这不符合最开始定的 Odin/Kratos 风格。

## 2. 目标分层

后端目标结构：

```text
api/auction/service/v1/auction.proto

app/auction/service/
├── cmd/server/                 # 启动入口，负责依赖组装
└── internal/
    ├── server/                 # HTTP/WebSocket server 注册，不写业务规则
    ├── service/                # Kratos service 适配层，实现 proto 生成接口
    ├── biz/                    # 业务领域层
    │   └── auction/            # 竞拍领域：聚合、规则、用例、仓储接口
    ├── data/                   # 仓储实现，内存/MySQL/Redis 都在这里
    └── realtime/               # WebSocket 房间与广播
```

## 3. biz 为什么用文件夹

`biz` 不是单纯放几个 Go 文件的目录，而是领域层。

V1 先只有 `auction` 一个领域：

```text
internal/biz/auction/
├── command.go       # 应用命令
├── event.go         # 领域事件构造
├── id.go            # ID/时间工具
├── lot.go           # Lot 聚合行为：开拍、出价、揭示、Duel、落锤
├── model.go         # 领域模型
├── ranking.go       # 排名规则
├── repo.go          # 仓储接口
├── usecase.go       # 用例编排
└── lot_test.go      # 领域规则测试
```

后续如果玩法复杂，可以继续拆：

```text
internal/biz/playbook/
internal/biz/risk/
internal/biz/settlement/
```

但 V1 不提前过度拆。

## 4. service 层应该怎么接 Kratos

最终应该是：

```go
type AuctionService struct {
    v1.UnimplementedAuctionServiceServer
    auction *auction.AuctionUsecase
}

func (s *AuctionService) CreateLot(ctx context.Context, req *v1.CreateLotRequest) (*v1.CreateLotReply, error) {
    cmd := convertCreateLotRequest(req)
    lot, err := s.auction.CreateLot(ctx, cmd)
    if err != nil {
        return nil, err
    }
    return &v1.CreateLotReply{Lot: convertLot(lot)}, nil
}
```

也就是说：

```text
proto request
 ↓
service 转换
 ↓
biz usecase
 ↓
data repository
```

service 只做协议适配和 DTO 转换，不写竞拍规则。

## 5. 为什么现在还没完全生成 Kratos service

当前仓库还没有提交 Kratos/protoc 生成产物，也没有完整生成链路。

为了不继续手写一堆假的 generated code，当前先做了两件事：

1. 把 `internal/service/AuctionService` 命名和职责固定为 Kratos service adapter；
2. 把 biz 领域层拆干净，避免后续生成 pb 后还要大搬家。

下一步应该补：

```text
api proto 生成 pb.go / http.pb.go / grpc.pb.go
internal/service 实现生成接口
internal/server 注册生成的 HTTP/gRPC server
```

## 6. 当前已经完成的修正

- 删除旧的 `internal/biz/service.go` 大杂烩；
- 新增 `internal/biz/auction/` 领域包；
- `data` 只实现仓储接口；
- `realtime` 只做 WebSocket 房间和事件广播；
- `server` 只处理 HTTP 参数和响应；
- `service` 改为 `AuctionService`，作为 Kratos service adapter 预备层；
- 补充领域单测。

## 7. 下一步建议

优先级：

1. 补 Kratos/protoc 生成工具链；
2. 给 proto 增加 HTTP annotation；
3. 生成 v1 包；
4. `internal/service` 实现生成出来的接口；
5. `internal/server` 注册 Kratos HTTP/gRPC server；
6. 删除临时手写 HTTP route 或将其降级为 debug route。
## 8. 本次 Kratos 接入结果

已完成：

```text
api/auction/service/v1/auction.pb.go
api/auction/service/v1/auction_grpc.pb.go
api/auction/service/v1/auction_http.pb.go
```

`auction.proto` 已补充 `google.api.http` 注解，HTTP 路由由 Kratos `protoc-gen-go-http` 生成。

当前启动链路：

```text
cmd/server/main.go
 ↓
server.NewHTTPServer
 ↓
v1.RegisterAuctionServiceHTTPServer
 ↓
internal/service.AuctionService
 ↓
internal/biz/auction.AuctionUsecase
 ↓
internal/data.Store (GORM + MySQL + Redis)
```

`internal/server` 现在只负责 Kratos HTTP server 组装、WebSocket 非 proto 路由注册和健康检查，不再手写业务 HTTP 路由。旧 `MemoryStore` / `database/sql` repo 主路径已删除。

生成命令：

```bash
make api
```

注意：Kratos/proto JSON 会把 `int64` 编码为字符串，这是 protobuf JSON 的正常行为。前端如果直接消费生成接口，需要按契约处理金额和时间字段。


### 统一响应与乐观锁冲突语义

service 对外采用统一返回包装（Result Envelope）：所有可预期业务错误都落到 reply.result，Go `error` 不再作为前端业务判断入口。这样前端只解析一套结构，避免同一个操作既可能拿 body 又可能拿 transport error。

当多实例或并发请求触发 lot expected-version 冲突时，data 层统一返回稳定哨兵错误，service 层包装为 `ReplyResult{code=409001, message="lot state changed, please refresh and retry"}`。前端应刷新拍品快照后提示用户重试，不应按普通 500 处理。

### 工程思考与设计模式

- service 层采用 **Adapter + Result Envelope**：只做 proto 入参/出参适配和错误包装，不写竞拍业务规则。
- biz 层采用 **Usecase + Domain Service**：状态机、出价规则、Duel 选择留在业务层，不持有 MySQL/Redis/Consul。
- data 层采用 **Repository + Unit of Work**：GORM、Redis、事务、缓存失败语义都属于基础设施层；出价成功时 lot/bid/event 使用同一 MySQL transaction。
- 事件采用 **Transactional Outbox**：MySQL 同事务落事件，事务后推 Redis Stream，worker 补偿未确认事件；下游按 event id 幂等消费。
- server 层采用 **Registry + Health Check**：Consul 注册和 `/readyz` 聚合观测属于 transport/governance，不进入 biz。
- 测试采用 **External Test Package**：`*_test.go` 集中在 `app/auction/service/test`，从外部视角验证接口和业务闭环，避免实现目录混入测试文件。
