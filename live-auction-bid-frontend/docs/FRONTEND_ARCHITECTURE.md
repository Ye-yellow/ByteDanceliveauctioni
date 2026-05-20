# 前端架构说明

## 当前架构

本项目采用 **Feature-Sliced + 页面分层**：

```text
src/
├── app/                    # 应用入口、全局样式、全局装配
├── pages/                  # 页面层：live-room / host-console
├── features/               # 业务能力模块
│   ├── auth/               # 登录、注册、token 存储
│   ├── auction/            # 拍品、出价、竞拍 API 与 UI
│   ├── realtime/           # WebSocket 连接与房间状态同步
│   ├── ranking/            # 实时排行榜
│   └── playbook/           # 信任揭示、Duel 等玩法 UI
├── shared/                 # 通用契约、配置、工具、UI
└── main.tsx                # Vite 入口
```



## Home 冻结状态

2026-05-21 用户确认 Home 页视觉调整完成并冻结。当前 Home 页只保留“进入后台”单一 CTA，拖拽调试工具已移除。后续除非用户明确要求解冻，不再调整 Home 的视觉风格、随机装饰元素分布、主卡比例和按钮样式。

## 当前前端项目边界

当前 `live-auction-bid-frontend` 只作为 **商家主播/运营后台 Web 项目**，不在同一个前端项目里承载用户 H5 或小程序入口。

- **本项目负责：商家主播后台 / 运营后台**
  - 当前入口：`/home`、`/login`、`/host`、`/admin`；
  - 面向商家、主播、运营、管理员，PC Web 优先；
  - 负责商品上架、规则配置、开拍/取消/结束、订单管理、数据看板；
  - 商家端和主播端当前可在同一后台中按角色权限区分，不拆成多个前端入口。

- **独立项目负责：用户 H5 竞拍端 / 小程序端**
  - 面向买家/用户，移动端优先；
  - 负责看直播拍品、实时出价、排名/倒计时、提醒、订单与模拟支付；
  - 后续应作为独立前端项目或独立端开发，复用同一套后端 API/WebSocket；
  - 不在当前后台 Home 页中提供同级入口。

因此，本项目 Home 页只需要引导进入商家主播后台；用户 H5/小程序端在系统架构和答辩材料中说明为独立端。

## 后端更新后的前端主链路

- 读接口：`ListLots`、`GetRoomSnapshot` 可匿名访问；
- 主播操作：创建、开拍、揭示、Duel、落锤需要 anchor/operator/admin；本地开发默认登录 `admin / admin_dev_password`；
- 观众出价：必须 buyer 登录，前端不再传 `userId/nickname`，由后端从 JWT claims 取；
- 所有 HTTP reply 都走 `reply.result` 统一业务语义；
- WebSocket 仍用 `/ws/rooms/{roomId}`，前端先做事件 enum 规范化再进入 UI 状态。

## 工程思考与设计模式

### Result Envelope Adapter

后端要求业务错误统一包装进 reply，而不是 `reply + error` 两套语义。前端在 `shared/api/result.ts` 做统一检查：

- `result.code === 0`：继续返回业务 payload；
- `result.code !== 0`：抛 `ApiResultError`，页面只展示 `result.message`。

这样页面不用关心 Kratos/HTTP 状态码细节，也不会把业务拒绝当网络异常。

### Auth Token Adapter

`features/auth/api` 持有 localStorage、access token、登录/注册/登出。其他 feature 只通过 API client 自动带 token，不直接读取存储。

这样鉴权基础设施留在 auth feature，不污染 auction/realtime 业务模块。

### Realtime Normalizer

后端 WebSocket 当前可能由 gorilla `WriteJSON` 输出 proto enum 数字；HTTP/proto JSON 更可能输出字符串。前端在 `normalizeAuctionEvent` 统一转成字符串 enum，避免 UI 到处写数字/字符串兼容判断。

### Page Composition

页面只组合 feature：

- `HomePage` 是当前后台 Web 的产品入口，承接 PDF 课题背景和商家主播后台入口；
- `LoginPage` 是独立登录页，参考 Sub2API 的居中卡片式登录体验，登录成功后进入商家主播后台；
- `HostConsolePage` 组合 host AuthPanel 和拍品控制台。

页面不直接写 token、fetch、WebSocket 解析细节。

## 重要规则

- 不写 mock 服务，不内置假数据冒充真实接口；
- 前端状态以服务端快照和 WebSocket 广播为准；
- 金额统一使用最小单位，显示时格式化；
- 后端契约变化后，先同步契约层，再改页面交互；
- stale generated schema 不能和真实契约并存误导开发。

## 后台范式迁移说明（2026-05-20）

本轮参考了本机浅克隆参考仓库：

- `/tmp/liveauction-refs/cli-proxy`：提炼 `MainLayout` / `DashboardPage` 的「管理中心」范式：固定侧边栏、顶部操作区、快速指标卡、状态自检、刷新动作和模块化页面组合；
- `/tmp/liveauction-refs/sub2api`：提炼 `AppLayout` / `AppSidebar` / `AppHeader` 与 `DataTable`、`StatCard`、`StatusBadge`、`EmptyState` 这类后台通用组件范式，以及 payment / order / user 管理页的「统计卡 + 表格 + 筛选/动作 + 空态」结构。

迁移原则：

- 不复制 Vue/Tailwind/Zustand/router/i18n 等技术栈代码；当前项目仍保持 React + TypeScript + Vite 与轻量路径分发；
- 可执行按钮只接真实后端契约：`CreateLot`、`ListLots`、`StartLot`、`RevealTrustCard`、`StartDuel`、`SettleLot`、`CancelLot`；
- 用户管理、支付结算、订单分析仅作为后台信息架构预留卡片展示，明确标注「契约待扩展」，不发请求、不构造 mock 数据；
- 后台通用 UI 抽象集中在 `src/shared/ui/admin/AdminPrimitives.tsx`，用于后续扩展真实 user/order/payment 页面。

当前前端入口：

```text
/home   产品 Home 页，默认首页
/login  独立登录页，可用 ?next=/host 指定登录后跳转
/host   商家主播/运营后台，未登录时进入 LoginPage
/admin  同一个商家主播后台入口别名，便于对齐管理中心语义
```

这次参考 Sub2API 的 Home/Login 分层：公开首页先解释产品与课题背景，Login 只处理身份进入，管理后台承载复杂运营能力，避免一进站就暴露后台。

当前后台页面组件结构：

```text
src/shared/ui/admin/AdminPrimitives.tsx  # AdminLayout / StatCard / StatusBadge / DataTable / EmptyState
src/pages/host-console/HostConsolePage.tsx # 竞拍运营后台组合页
```

## 后台主链路 API 契约清单

| 页面区块 | 真实后端契约 | 说明 |
| --- | --- | --- |
| 快速创建拍品 | `POST /api/lots` | 创建草稿拍品，带竞拍规则与信任卡 |
| 拍品列表 | `GET /api/lots?room_id=demo` | 后台表格唯一数据源 |
| 开拍 | `POST /api/lots/{lot_id}/start` | DRAFT → LIVE |
| 信任揭示 | `POST /api/lots/{lot_id}/trust-cards/{card_id}/reveal` | 卡片揭示后刷新 lot |
| Duel | `POST /api/lots/{lot_id}/duel` | LIVE 阶段控场动作 |
| 落锤成交 | `POST /api/lots/{lot_id}/settle` | LIVE → SETTLED |
| 异常取消 | `POST /api/lots/{lot_id}/cancel` | LIVE → CANCELLED，提交原因 |

