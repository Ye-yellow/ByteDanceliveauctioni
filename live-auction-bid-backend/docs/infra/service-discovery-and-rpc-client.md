# 服务发现与 RPC Client

## 通用目标

服务之间通过稳定的服务名调用，而不是硬编码实例 IP。线上走注册中心，本地开发允许显式直连 override。RPC client 默认带 timeout、trace、recovery，必要时加熔断。

## 适用场景

适用于多服务部署、服务动态扩缩容、本地开发和线上服务发现策略不同的系统。

## 通用抽象

- `Registrar`：服务启动后把自身 HTTP/gRPC 地址注册到注册中心。
- `Discovery`：RPC client 通过服务名发现目标实例。
- `ServiceEndpoint`：形如 `discovery:///service.name` 的逻辑地址。
- `DirectEndpointOverride`：本地开发或故障排查用的直连环境变量。
- `RPCClientFactory`：创建带 middleware、timeout、discovery 的 client。
- `ClientMiddleware`：tracing、recovery、circuit breaker、retry、auth metadata。

## 核心流程

1. 服务启动读取 registry 配置，创建 registrar 和 discovery。
2. `newApp` 把 registrar 注入应用框架，服务运行时自动注册。
3. 需要调用下游服务时，先设置默认 `discovery:///service.name`。
4. 如果发现直连环境变量存在，则使用直连 endpoint 并跳过 discovery。
5. 构造 RPC client options：timeout、middleware、endpoint、discovery。
6. Dial 失败直接启动失败，避免服务半可用。

## 可变点

- 注册中心可替换为 Consul、Nacos、Etcd、Kubernetes DNS。
- 本地直连可用环境变量、配置项、命令行 flag。
- 熔断只给不稳定或非关键链路加，不要所有内部 RPC 一刀切。
- 内部调用是否重试必须看接口幂等性。

## 落地模板

```go
func NewGameClient(discovery registry.Discovery, tracer Tracer) GameClient {
    endpoint := "discovery:///game.service"
    opts := []grpc.ClientOption{
        grpc.WithTimeout(10 * time.Second),
        grpc.WithMiddleware(
            tracing.Client(tracing.WithTracerProvider(tracer)),
            recovery.Recovery(),
        ),
    }

    if direct := os.Getenv("GAME_GRPC_ENDPOINT"); direct != "" {
        endpoint = direct
    } else {
        opts = append(opts, grpc.WithDiscovery(discovery))
    }

    opts = append(opts, grpc.WithEndpoint(endpoint))
    conn, err := grpc.DialInsecure(context.Background(), opts...)
    if err != nil {
        panic(err)
    }
    return NewGameClient(conn)
}
```