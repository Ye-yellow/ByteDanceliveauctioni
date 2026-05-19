# 直播互动竞拍玩法引擎蓝图

Date: 2026-05-19
Owner note: OpenClaw-generated blueprint. This is a concept/design document, not final product documentation yet.

## 1. 项目重新定位

这个项目不要做成普通“直播间竞拍页面”，而要做成：

> 面向短视频直播电商的互动竞拍玩法引擎。

核心价值不是简单让用户加价，而是把直播间里的围观、互动、信任建立、最后争夺这些短视频直播原生元素，设计成一套动态竞拍机制。

传统竞拍是：

```text
商品展示 → 用户出价 → 最高价成交
```

我们的竞拍是：

```text
商品上架
 ↓
直播间互动热度进入竞拍系统
 ↓
AI/规则引擎识别当前玩法阶段
 ↓
动态触发信任揭示、群体共振、双人巅峰竞拍
 ↓
高并发实时出价与排行榜变化
 ↓
反狙击落锤成交
 ↓
拍后玩法复盘
```

一句话：

> 让直播间氛围成为竞拍机制的一部分，而不只是旁边的弹幕背景。

## 2. 为什么这个方向更贴题

赛题关键词：

- 直播电商生态；
- 珠宝、二手奢侈品等高价值非标品；
- 动态竞拍；
- 强互动与竞技感；
- 高并发与实时交互；
- AI 动态定价与竞拍气氛营造。

普通竞拍系统只能覆盖“实时出价”。

我们这版玩法引擎覆盖：

1. 非标品信任问题；
2. 直播间围观互动问题；
3. 最后几秒竞技感问题；
4. 主播控场问题；
5. AI 如何真正参与玩法，而不是硬塞一句话术。

## 3. 三大核心玩法

### 3.1 Trust-Reveal Auction：信任揭示竞拍

#### 背景

珠宝、二奢、收藏品这类非标品，用户不出价往往不是因为不喜欢，而是因为不确定：

- 真不真？
- 成色怎么样？
- 瑕疵在哪里？
- 值不值？
- 有没有证书？
- 售后怎么保障？

所以这类商品的竞拍不应该只是“喊价”，还应该是一个“逐步建立信任”的过程。

#### 玩法

竞拍过程中，系统根据直播间状态逐步解锁商品信息：

```text
开拍前：基础信息
出价启动：高清细节图
热度上升：瑕疵说明
竞价激烈：鉴定证书/来源说明
临近落锤：保真/售后承诺
```

AI/规则引擎判断当前用户犹豫点：

```text
价格犹豫 → 展示历史参考价/同类成交区间
真伪犹豫 → 展示鉴定证书/检测报告
成色犹豫 → 展示瑕疵特写/成色等级
售后犹豫 → 展示退换/保真承诺
```

#### 示例

```text
当前围观人数高，但出价人数低。
系统判断：用户可能缺少信任信息。
触发：解锁鉴定证书卡片 + 主播提示“证书编号已展示，支持回放查看”。
```

#### 价值

- 解决非标品“不敢拍”的问题；
- 让商品信息披露成为竞拍节奏的一部分；
- 比单纯 AI 定价更安全、更合理。

---

### 3.2 Crowd-Powered Auction：群体共振竞拍

#### 背景

抖音直播间里，真正出价的人可能不多，但围观、点赞、评论的人很多。

普通竞拍系统忽略了这批人。

我们的设计是：

> 围观用户不能改变成交价格，但可以影响竞拍氛围、信息揭示和玩法节奏。

#### 玩法

围观用户通过互动触发“共振事件”：

```text
点赞数达到阈值 → 解锁商品细节卡
评论关键词达到阈值 → 解锁主播讲解提示
围观人数上涨 → 开启限时热场倒计时
投票结果达成 → 进入最后 PK 模式
```

重要边界：

- 围观用户不能直接改变最终价格；
- 不能凭点赞降低成交价；
- 只能影响信息展示、节奏、互动权益；
- 保证竞拍公平性。

#### 示例

```text
30 秒内新增 200 个点赞，且评论里“证书”关键词频繁出现。
系统触发：展示证书卡片，AI 提醒主播重点讲解来源和鉴定。
```

#### 价值

- 让非出价用户也参与直播间事件；
- 更贴近短视频直播互动；
- 强化“交易新玩法”的差异化。

---

### 3.3 Duel Auction：双人巅峰竞拍

#### 背景

直播竞拍最有戏剧性的时刻，往往是最后两个用户反复加价。

传统系统只是延时。

我们把它包装成直播原生的“巅峰 PK”。

#### 触发条件

系统检测：

- 最后 30 秒；
- 前两名用户连续交替出价；
- 出价差距小；
- 围观热度上升。

触发：

```text
Duel Mode / 双人巅峰竞拍
```

#### 玩法

进入 Duel 后：

- 当前前二进入主战区；
- 其他用户仍可出价，但前端重点展示前二 PK；
- 倒计时变成短轮次；
- 每次最后几秒出价会触发反狙击延时；
- 围观用户可以点赞/评论助威；
- 主播获得 AI 控场提示。

#### 示例

```text
用户 A：¥2,300
用户 B：¥2,350
系统：检测到双人连续争夺，进入 30 秒巅峰竞拍。
围观区：显示“当前 A/B 双人争夺中”。
```

#### 价值

- 强竞技感；
- 直播间戏剧效果强；
- 很适合比赛演示；
- 和普通拍卖拉开差异。

## 4. AI 在这里真正做什么

AI 不直接决定价格，也不替商家鉴定商品。

AI 作为：

```text
Playbook Engine Assistant / 玩法引擎助手
```

负责辅助判断：

1. 当前应该进入哪个玩法阶段；
2. 用户为什么不出价；
3. 应该释放什么商品信息；
4. 竞拍是否进入高热度争夺；
5. 是否存在异常出价行为；
6. 主播下一句应该怎么控场。

## 5. AI 能力模块

### 5.1 Playbook Detector：玩法阶段识别

输入：

- 在线人数；
- 评论速度；
- 点赞速度；
- 出价频率；
- 出价人数；
- 当前价格变化；
- 剩余时间；
- 前两名竞争强度。

输出：

```text
WARM_UP        热场
TRUST_BLOCKED  信任阻塞
BIDDING_ACTIVE 出价活跃
DUEL_READY     双人争夺即将形成
DUEL_MODE      双人巅峰竞拍
COOLING        冷场
SETTLE_READY   可落锤
```

### 5.2 Trust Signal Advisor：信任信息建议

输入：

- 商品类型；
- 用户评论关键词；
- 当前出价转化率；
- 已展示信息；
- 停留时间。

输出：

```text
建议展示：鉴定证书
原因：评论中“真假/证书”占比上升，围观多但出价少。
```

### 5.3 Crowd Event Planner：群体事件触发

输入：

- 点赞/评论/围观变化；
- 当前竞拍阶段；
- 商品风险等级。

输出：

```text
触发：群体共振事件
动作：点赞达到 500 解锁瑕疵细节图
```

### 5.4 Duel Trigger：PK 模式判断

输入：

- Top2 出价差；
- Top2 交替出价次数；
- 剩余时间；
- 直播间热度。

输出：

```text
建议进入 Duel Mode，持续 30 秒，最多延时 3 次。
```

### 5.5 Host Copilot：主播控场助手

输入玩法阶段，输出主播提示：

```text
现在用户主要在问证书，建议先展示鉴定卡，再提醒最后 30 秒落锤。
```

## 6. 系统模块蓝图

```text
auction-backend
├── api/
│   ├── auction/service/v1
│   └── playbook/service/v1          # 可选，后期拆
│
├── app/auction/service
│   └── internal/
│       ├── server
│       ├── service
│       ├── biz
│       │   ├── auction_session.go
│       │   ├── lot.go
│       │   ├── bid.go
│       │   ├── ranking.go
│       │   ├── settlement.go
│       │   ├── playbook.go
│       │   ├── trust_reveal.go
│       │   ├── crowd_power.go
│       │   ├── duel_mode.go
│       │   └── host_copilot.go
│       │
│       ├── realtime
│       │   ├── hub.go
│       │   ├── room.go
│       │   ├── message.go
│       │   └── broadcaster.go
│       │
│       └── data
│           ├── lot_repo_memory.go
│           ├── bid_repo_redis.go
│           ├── playbook_event_log.go
│           └── ai_playbook_client.go
```

## 7. 前端蓝图

前端不只做一个观众页，而是两个视角：

```text
auction-frontend
├── pages/
│   ├── LiveRoomPage        # 观众端
│   └── HostConsolePage     # 主播/运营端
│
├── features/
│   ├── auction             # 出价、拍品、倒计时
│   ├── ranking             # 排名
│   ├── realtime            # ws 连接
│   ├── trust-reveal        # 信任揭示卡片
│   ├── crowd-power         # 群体共振事件
│   ├── duel-mode           # 双人巅峰竞拍 UI
│   └── host-copilot        # AI 控场助手
```

### 观众端重点

- 当前拍品；
- 出价按钮；
- 实时排名；
- 信任揭示卡片；
- Duel Mode 视觉效果；
- 群体共振进度条。

### 主播端重点

- 商品上架；
- 规则设置；
- 当前玩法阶段；
- AI 建议展示哪张信息卡；
- 是否进入 Duel Mode；
- 一键落锤。

## 8. 核心事件流

### 8.1 Trust-Reveal Flow

```text
用户围观但不出价
 ↓
系统检测 trust_blocked
 ↓
AI/规则判断阻塞点：真伪/成色/售后/价格
 ↓
主播端提示展示对应信息卡
 ↓
观众端解锁信息卡
 ↓
出价转化率变化
```

### 8.2 Crowd-Powered Flow

```text
直播间点赞/评论/围观上升
 ↓
系统累计 crowd signal
 ↓
达到阈值
 ↓
触发共振事件
 ↓
解锁信息/进入热场/提示主播
```

### 8.3 Duel Flow

```text
最后 30 秒
 ↓
Top2 用户交替出价
 ↓
系统判断 DuelReady
 ↓
进入 DuelMode
 ↓
短倒计时 + 反狙击延时
 ↓
最终落锤成交
```

## 9. 数据模型草案

### PlaybookState

```go
type PlaybookStage string

const (
    StageWarmUp       PlaybookStage = "WARM_UP"
    StageTrustBlocked PlaybookStage = "TRUST_BLOCKED"
    StageActive       PlaybookStage = "BIDDING_ACTIVE"
    StageDuelReady    PlaybookStage = "DUEL_READY"
    StageDuelMode     PlaybookStage = "DUEL_MODE"
    StageCooling      PlaybookStage = "COOLING"
    StageSettleReady  PlaybookStage = "SETTLE_READY"
)
```

### TrustRevealCard

```go
type TrustRevealCard struct {
    ID       string
    LotID    string
    Type     string // CERTIFICATE / FLAW / DETAIL / PRICE_REF / SERVICE
    Title    string
    Content  string
    Revealed bool
}
```

### CrowdSignal

```go
type CrowdSignal struct {
    RoomID        string
    LikeDelta     int
    CommentDelta  int
    OnlineDelta   int
    KeywordHits   map[string]int
    WindowSeconds int
}
```

### DuelState

```go
type DuelState struct {
    LotID        string
    UserA        string
    UserB        string
    StartedAt    time.Time
    EndsAt       time.Time
    ExtendCount  int
    MaxExtends   int
}
```

## 10. 安全边界

这些玩法必须遵守边界：

1. AI 不决定最终成交价；
2. AI 不鉴定真伪，只辅助展示已有资料；
3. 围观互动不能直接降低成交价；
4. Duel Mode 不能排除其他合法用户出价，除非规则开拍前明确说明；
5. 所有玩法规则必须在开拍前可见；
6. 所有出价以服务端原子校验为准；
7. 竞拍结束后事件日志可追溯。

## 11. 比赛展示话术

可以这样介绍：

> 我们不是做一个普通竞拍页面，而是设计了一套直播互动竞拍玩法引擎。它把短视频直播间里的点赞、评论、围观、信任建立和最后争夺，转化成可配置、可追踪、可实时广播的竞拍机制。系统通过 WebSocket 实现毫秒级互动，通过服务端竞拍状态机保证出价一致性，通过 AI 玩法助手判断何时释放信任信息、何时触发群体共振、何时进入双人巅峰竞拍，从而提升非标品直播交易的参与度和成交效率。

## 12. 下一步实现建议

第一批只做三个玩法，不要贪多：

1. Trust-Reveal Auction
2. Crowd-Powered Auction
3. Duel Auction

落地顺序：

```text
1. biz 增加 playbook/trust/crowd/duel 模型
2. proto 增加 PlaybookState / TrustRevealCard / CrowdSignal / DuelState
3. realtime 增加 playbook.updated 广播事件
4. 前端增加信任卡片、群体进度条、Duel UI
5. 主播端增加 AI/规则推荐面板
```

这样既创新，又能演示，而且和赛题“短视频时代交易新玩法”强相关。
