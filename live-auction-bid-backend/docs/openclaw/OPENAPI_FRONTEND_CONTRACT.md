# 前后端契约生成方案

## 方案选择

采用：

```text
后端 proto → OpenAPI → 前端 TypeScript 类型/API client
```

原因：

- 后端仍以 `api/auction/service/v1/auction.proto` 作为源头；
- 前端不直接猜字段；
- React 侧适合消费 HTTP/OpenAPI 生成的 TS 类型；
- 后续接 Kratos OpenAPI 生成时，只需要替换 `openapi/auction.openapi.json` 的来源。

## 当前落地

当前先提交手工维护的契约文件：

```text
openapi/auction.openapi.json
```

它和当前 HTTP 接口保持一致：

```text
GET  /api/lots
POST /api/lots
POST /api/lots/{lotId}/bid
POST /api/lots/{lotId}/settle
```

前端通过 `openapi-typescript` 读取这个文件生成：

```text
src/shared/api/generated/auction.schema.ts
```

## 后续演进

等 Go/Kratos 工具链就绪后，将改成：

```text
api/auction/service/v1/auction.proto
 ↓ make api/openapi
openapi/auction.openapi.json
 ↓ 前端 npm run generate:api
auction.schema.ts
```

原则：

- proto/OpenAPI 是契约源；
- 前端不手写后端 DTO；
- 生成文件可以提交，便于协作者不用本地装生成器也能编译。
