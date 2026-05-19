# 前端 API 类型生成说明

## 方案

前端采用后端 OpenAPI 契约生成 TypeScript 类型。

```text
后端 proto
 ↓ 后端生成/维护 OpenAPI
openapi/auction.openapi.json
 ↓ 前端 npm run generate:api
src/shared/api/generated/auction.schema.ts
 ↓
src/shared/api/types.ts
 ↓
features/* 使用类型
```

## 命令

```bash
npm run generate:api
```

当前读取后端兄弟目录：

```text
../live-auction-bid-backend/openapi/auction.openapi.json
```

## 规则

- 前端不手写后端 DTO；
- `src/shared/api/generated/auction.schema.ts` 是生成文件，不手动改；
- 业务代码从 `src/shared/api/types.ts` 引用类型；
- 后端契约变化后，先更新 OpenAPI，再运行 `npm run generate:api`。
