# 前端架构说明

## 当前架构

本项目采用 **Feature-Sliced + 页面分层**，当前只承载主播团队工作台 PC Web：

```text
src/
├── app/                    # 应用入口、全局样式、全局装配
├── pages/                  # 页面层：home / login / host-console
├── features/               # 业务能力模块
│   ├── auth/               # 工作台登录、登出、token 存储
│   └── auction/            # 后台拍品控制 API
├── shared/                 # 通用契约、配置、类型、工具、后台 UI
└── main.tsx                # Vite 入口
```

## Home 冻结状态

2026-05-21 用户确认 Home 页视觉调整完成并冻结。当前 Home 页只保留“进入后台”单一 CTA，拖拽调试工具已移除。后续除非用户明确要求解冻，不再调整 Home 的视觉风格、随机装饰元素分布、主卡比例和按钮样式。

## 当前前端项目边界

当前 `live-auction-bid-frontend` 只作为 **主播团队工作台 PC Web 项目**，不在同一个前端项目里承载用户 H5 或小程序入口。

- **本项目负责：主播团队工作台 / 主播团队运营后台**
  - 当前入口：`/home`、`/login`、`/host`、`/admin`；
  - 面向主播主账号及其团队子账号，PC Web 优先；
  - 不再拆“平台管理端 / 商家端 / 主播端”三套前端概念；
  - 一个主播主账号对应一个后台空间，并固定绑定一个直播间；
  - 一个主播空间可创建多个团队子账号给运营、商品、控场、订单等团队成员协作；
  - 添加拍品页不提供选择多个直播间，也不提供“主播选择”：直播间固定展示当前主播绑定直播间，本场执行人只可选择当前账号或已授权团队子账号；
  - 权限管理是主播团队工作台内部能力，用于控制子账号能否拍品准备、竞拍玩法配置、开拍控场、成交处理、数据查看和风控处置；
  - 后台负责拍品准备、竞拍玩法配置、开拍/控场/异常取消、成交处理、数据看板、实时链路诊断和团队子账号权限。

- **独立项目负责：用户 H5 竞拍端 / 小程序端**
  - 面向买家/用户，移动端优先；
  - 负责看直播拍品、实时出价、排名/倒计时、提醒、订单与模拟支付；
  - 后续应作为独立前端项目或独立端开发，复用同一套后端 API/WebSocket；
  - 不在当前后台 Home、Login、路由或页面中提供入口。

因此，本项目 Home 页只需要引导进入主播团队工作台；用户 H5/小程序端只在系统架构和答辩材料中说明为独立端。

## 后端更新后的前端主链路

- 后台读接口：`ListLots`；
- 主播/运营操作：创建、开拍、揭示、Duel、落锤需要 anchor/operator/admin；本地开发默认登录 `admin / admin_dev_password`；
- 所有 HTTP reply 都走 `reply.result` 统一业务语义；
- 当前后台不实现用户端出价页面；出价、排名、倒计时、用户结果与模拟支付属于后续独立 H5/小程序端。

## 工程思考与设计模式

### Result Envelope Adapter

后端要求业务错误统一包装进 reply，而不是 `reply + error` 两套语义。前端在 `shared/api/result.ts` 做统一检查：

- `result.code === 0`：继续返回业务 payload；
- `result.code !== 0`：抛 `ApiResultError`，页面只展示 `result.message`。

这样页面不用关心 Kratos/HTTP 状态码细节，也不会把业务拒绝当网络异常。

### Auth Token Adapter

`features/auth/api` 持有 localStorage、access token、登录/登出。其他 feature 只通过 API client 自动带 token，不直接读取存储。

这样鉴权基础设施留在 auth feature，不污染 auction 业务模块。

### Page Composition

页面只组合 feature：

- `HomePage` 是当前后台 Web 的产品入口，承接 PDF 课题背景和主播团队工作台入口；
- `LoginPage` 是独立工作台登录页，登录成功后进入主播团队工作台；
- `HostConsolePage` 组合后台 AuthPanel 和拍品控制台。

页面不直接写 token、fetch、WebSocket 解析细节。

## 重要规则

- 前端 UI 原型阶段允许使用本地 mock 数据快速搭建 PC 后台信息架构和视觉交互，但必须明确标记为过渡态，不能冒充已接真实接口；
- 接入真实后端 API/WebSocket 时，必须删除所有 mock 数据、mock 类型、mock 动画/fallback 和相关注释痕迹，不允许 mock 与真实数据并存；
- 当前后台最终状态必须以服务端接口和 WebSocket 返回为准；
- 金额统一使用最小单位，显示时格式化；
- 后端契约变化后，先同步契约层，再改页面交互；
- stale generated schema 不能和真实契约并存误导开发；
- 未接真实契约的模块在交付前必须补齐 API/WebSocket 数据源，或明确降级为不可点击/待接入状态。

## 后台范式迁移说明（2026-05-20）

本轮参考了本机浅克隆参考仓库：

- `/tmp/liveauction-refs/cli-proxy`：提炼 `MainLayout` / `DashboardPage` 的「管理中心」范式：固定侧边栏、顶部操作区、快速指标卡、状态自检、刷新动作和模块化页面组合；
- `/tmp/liveauction-refs/sub2api`：提炼 `AppLayout` / `AppSidebar` / `AppHeader` 与 `DataTable`、`StatCard`、`StatusBadge`、`EmptyState` 这类后台通用组件范式，以及 payment / order / user 管理页的「统计卡 + 表格 + 筛选/动作 + 空态」结构。

迁移原则：

- 不复制 Vue/Tailwind/Zustand/router/i18n 等技术栈代码；当前项目仍保持 React + TypeScript + Vite 与轻量路径分发；
- 现阶段为 PC 后台 UI/信息架构原型，允许在页面文件中短期保留本地 mock 数据用于展示竞拍、商品、订单、WebSocket、风控和权限角色页面；
- mock 仅用于前端展示验证，不能作为正式数据层，也不能沉淀为服务或长期 fixture；
- 后续接入真实 API/WebSocket 时，必须一次性删除对应 mock 数组、mock 动画和 fallback 展示逻辑，页面数据全部改由 API service / WebSocket adapter 提供；
- 后台通用 UI 抽象应逐步沉淀为可复用组件，真实数据接入后再拆分 feature service 和页面容器。

当前前端入口：

```text
/home   产品 Home 页，默认首页
/login  独立工作台登录页，可用 ?next=/host 指定登录后跳转
/host   主播团队工作台，未登录时进入 LoginPage
/admin  同一个主播团队工作台入口别名，便于对齐管理中心语义
```

当前后台页面组件结构：

```text
src/shared/ui/admin/AdminPrimitives.tsx      # AdminLayout / StatCard / StatusBadge / DataTable / EmptyState
src/pages/host-console/HostConsolePage.tsx   # 竞拍运营后台组合页
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

## 本场拍品队列页数据边界（2026-05-21）

本场拍品队列页已经从原有占位表格替换为 `AuctionManagementPage`。该页面以当前主播固定直播间为唯一上下文，调用真实 lots/snapshot 接口，并订阅当前 roomId 的 WebSocket 事件更新列表和当前竞拍卡。页面保留订单、日志、实时出价扩展入口，但未接入的接口统一显示“待接入”，避免长期 mock 与真实数据并存。

## 实时链路诊断边界（2026-05-21）

实时链路诊断不再按跨直播间运维大屏设计，而是固定使用当前主播空间绑定的 `currentRoomId`。前端第一版只展示客户端可计算指标：WebSocket 连接状态、重连次数、最近心跳、服务器时间偏移、最近事件类型/序号、snapshot/currentLot/ranking 恢复情况。断线进入 reconnecting，重连成功后重新调用 `getRoomSnapshot(currentRoomId)` 恢复当前价、倒计时和排行榜。

## LiveAuction Studio 信息架构（2026-05-21）

当前后台不再按平台工作台或通用 SaaS 控制台组织，而按主播团队工作流组织：直播前（添加拍品、竞拍玩法、讲解卡、今日队列）、直播中（直播间中控台、实时出价、排行榜、控场操作）、直播后（成交处理、履约、复盘、复制重拍）。所有页面默认使用当前主播空间固定 `currentRoomId`，不得引入直播间切换。
