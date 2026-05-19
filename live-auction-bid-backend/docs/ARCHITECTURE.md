# 架构设计

## DDD 分层

- Domain: `internal/domain/auction`，只表达竞拍领域规则：拍品、出价、排名、落锤、反狙击延时。
- Application: `internal/application/auction`，编排用例：创建拍品、出价、落锤。
- Infrastructure: `internal/infrastructure/*`，内存仓储、AI Stub、后续可替换 Redis/PostgreSQL/Ollama。
- Interfaces: `internal/interfaces/http|ws`，REST/WebSocket 适配器。

## 实时一致性策略初版

1. 后端为房间内拍品状态唯一写入点。
2. 每次出价在 Domain 内校验：状态、截止时间、最低加价。
3. 成功出价后更新版本号并广播 `bid.accepted` + `lot.updated`。
4. 前端以服务端广播状态覆盖本地，避免多端价格漂移。
5. 后续高并发升级：Redis Lua 原子出价、Kafka/Pulsar 事件流、房间分片、乐观版本号。

## AI 创新点预留

- 动态起拍价：商品描述 + 历史成交 + 直播热度估价。
- 直播气氛官：根据出价频率、剩余时间、价格跃迁生成主播话术。
- 风控：异常高频出价、机器人账号、恶意抬价检测。
