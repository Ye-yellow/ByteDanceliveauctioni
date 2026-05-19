# Live Auction Bid Backend

直播电商竞拍系统后端。当前按你现有 Odin 项目的阅读方式重构为 **Go + Kratos 多服务仓库骨架**：`api/` 管协议，`app/` 管服务实现，服务内部走 `server → service → biz → data`。

## 顶层结构

```text
auction-backend/
├── api/                    # 对外协议层：proto + 后续生成 pb/http/grpc/errors
├── app/                    # 后端服务实现区，每个子服务一套 Kratos 分层
├── pkg/                    # 跨服务共享工具包预留
├── deploy/                 # Docker / compose / k8s 部署配置预留
├── docs/                   # 架构与路线文档
├── Makefile                # 顶层构建入口
└── app_makefile            # 各服务共用构建模板
```

## 当前服务

```text
app/auction/service/
├── cmd/server/             # 进程入口：main.go，后续补 wire.go
├── configs/                # 本地/环境配置 yaml
├── internal/conf/          # 配置结构
├── internal/server/        # HTTP/WebSocket server，注册路由和中间件
├── internal/service/       # 协议适配层：接请求，调 usecase
├── internal/biz/           # 业务核心层：竞拍规则、出价校验、排名、落锤
├── internal/data/          # 数据访问层：当前内存仓储，后续 Redis/MySQL
└── Makefile
```

请求流：

```text
Web / Admin / Mobile
 ↓ HTTP / WebSocket
api/auction/service/v1/*.proto
 ↓ 生成代码/手写适配
app/auction/service/internal/server
 ↓ 路由注册与连接管理
app/auction/service/internal/service
 ↓ 用例编排
app/auction/service/internal/biz
 ↓ 领域规则
app/auction/service/internal/data
 ↓
Memory / Redis / MySQL / AI 服务
```

## 本地运行

> 当前机器如果没有 Go，需要先安装 Go 1.22+。

```bash
make -C app/auction/service run
```

默认地址：

- HTTP: `http://localhost:8080`
- WebSocket: `ws://localhost:8080/ws/rooms/demo`

## 常用命令

```bash
make auction     # 构建 auction 服务
make test        # 全仓测试
```

## 前端仓库

前端独立仓库：`git@github.com:Ye-yellow/auction-frontend.git`

前端环境变量：

```bash
VITE_API_BASE=http://localhost:8080
VITE_WS_BASE=ws://localhost:8080
```
