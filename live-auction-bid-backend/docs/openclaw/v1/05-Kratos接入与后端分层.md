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
