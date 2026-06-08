# Anchor Studio UI and Data Guidelines

Status: `ACTIVE_PRE_LAUNCH`

This document defines how the current Admin frontend should present the LiveAuction Studio experience. It is a product and UI boundary document for future Home, Login, Workbench, Auction Queue, Realtime Console, Orders, and Diagnostics changes.

The current frontend is a **PC workbench for the streamer and the streamer's operations team**. It is not a company internal admin console, not a platform-wide BI dashboard, and not the buyer H5 client.

## Source Basis

- `docs/1.pdf` describes a full-stack live auction competition project: lot publishing, rule configuration, realtime bidding, dynamic ranking, settlement, and abnormal cancellation.
- The scoring direction emphasizes the complete technical chain, realtime consistency, WebSocket stability, high-concurrency handling, and a convincing auction atmosphere.
- The PC management side in the PDF is for merchants/streamers: publishing auction lots, managing lot status and results, and handling generated orders.
- `$ui-ux-pro-max` and `$frontend-design` guidance applies here as a utilitarian operations product: high clarity, dense but scannable information, semantic actions, strong accessibility, and no decorative admin noise.

## Product Positioning

LiveAuction Studio should answer one question quickly: **Can the streamer team control today's auction room safely and confidently?**

Primary users:

- Streamer owner: sees the room status, current auction, final business result, and abnormal risks.
- Live operator: controls start, reveal, duel, hammer, cancel, and sync during the live room.
- Product assistant: prepares lots, checks rules, images, trust cards, and queue order.
- Order support: follows settlement, payment state, and post-auction handling.
- Data reviewer: reviews room-level auction outcome and operational rhythm after the stream.

Account boundary:

- The streamer or merchant owner account is provisioned by the platform/company backend and binds one merchant workspace plus the fixed live room.
- The Studio can manage only team subaccounts under that workspace, such as live operator or operations roles.
- The Studio must not expose buyer registration, buyer account creation, or buyer role management. Buyer identity belongs to the H5 client.
- If an Admin team-account API returns buyer accounts, the frontend should fail fast and the backend contract should be corrected.

Default context:

- One streamer workspace is bound to one current live room.
- Pages should default to the current room and today's auction workflow.
- Do not introduce platform-wide room switching or company-wide views unless the product explicitly changes scope.

## Data Display Principle

Company backend data is not automatically UI data. A field should appear in the normal workbench only if it helps the streamer team make or verify an auction operation.

Before adding a new field to a page, classify it:

| Tier | Meaning | UI placement |
| --- | --- | --- |
| P0 | Required to run the live auction safely | Main card, table primary columns, sticky action area |
| P1 | Useful for review or troubleshooting | Detail drawer, secondary panel, expandable row |
| P2 | Technical diagnostics only | Realtime diagnostics page or explicit debug section |
| Hidden | Internal, sensitive, or irrelevant to streamer operations | Never render in routine UI |

Because the project is still pre-launch, follow `docs/COMPATIBILITY_POLICY.md`: missing required P0 data should fail fast with a clear error path. Do not invent fallback values, stale demo data, or legacy compatibility branches.

## What To Show

### P0: Main Workflow Data

Show these where the operator needs them, with strong visual hierarchy:

| Area | Required visible data |
| --- | --- |
| Current room | Room name or room code, connection state, last successful sync, reconnecting/error state |
| Current lot | Image, title, status, queue position, responsible operator, whether rules are complete |
| Auction rules | Start price, minimum increment, duration, cap price, extension window, max extension count |
| Live bidding | Current price, leading bidder display name, top ranking, latest accepted bids, server-driven countdown |
| Control actions | Start auction, reveal trust card, enter duel, hammer settlement, abnormal cancel |
| Risk and consistency | WebSocket state, event sequence freshness, snapshot recovery state, countdown source |
| Order result | Winning bidder display name, final price, generated order status, payment/handling state |
| Daily workbench | Today's lots, live/in-progress count, pending setup, pending settlement, abnormal items |

Use normalized business language. For example, show "最近同步 3 秒前" or "事件序号已恢复到 128" only where that helps the operator trust the room state.

### P1: Secondary Data

Show these in detail panels, drawers, or expandable rows:

| Area | Secondary data |
| --- | --- |
| Lot detail | Short lot ID, created time, updated time, version, creator/operator |
| Rules detail | Full extension configuration, trust card reveal state, validation errors |
| Bid history | Accepted bid list, rejected bid reason summarized for operations, rank changes |
| Orders | Short order ID, payment deadline, fulfillment note, support status |
| Troubleshooting | Request ID, event type, event timestamp, retry action, readable backend error message |

P1 data should support a human operator. Do not expose raw payloads as a substitute for designed UI.

### P2: Diagnostics-Only Data

Keep these out of normal Workbench, Auction Queue, and Realtime Console primary views:

| Data | Allowed location |
| --- | --- |
| Raw WebSocket event payload | Realtime diagnostics detail, explicit debug mode only |
| Raw server timestamp, offset calculation, sequence internals | Realtime diagnostics page |
| Redis, lock, cache, process, or DB implementation metrics | Diagnostics or backend observability, not streamer workbench |
| Full request/response traces | Developer diagnostics, not routine operations UI |
| Company-wide connection count, global broadcast success rate | Platform observability only, not this streamer workspace |

### Hidden Data

Never show these in routine frontend UI:

- Access tokens, refresh tokens, API keys, secrets, DSNs, hostnames, internal service names.
- Full buyer private identifiers, full phone numbers, private addresses, or payment credentials.
- Internal lock owners, Redis keys, database row internals, idempotency keys, and raw auth claims.
- Company-wide revenue, tenant-wide performance, or other streamer rooms unless the product scope changes.

## Page Guidance

### Home

Home is a public product entry. It can keep a softer visual atmosphere, but it should still point to one product: the streamer team workbench. Do not add buyer H5 or platform admin entrances to this project.

### Login

Login should feel connected to Home but more operational. The card can be prominent and comfortable, with clear account/password flow, error state, loading state, and no role picker unless the backend contract requires one. Workspace and permissions come from the authenticated account.

### Today's Workbench

The workbench should not be a generic stats dashboard. It should prioritize:

- What is happening now.
- What must be prepared before the next auction.
- What requires operator attention today.
- What has settled and still needs order handling.

Recommended first-screen rhythm:

- Top status strip: room state, realtime state, last sync, active operator.
- Primary panel: current lot or next actionable lot.
- Supporting panels: today's queue, pending setup, pending settlement, abnormal/risk items.
- Metrics: only room-level operational metrics that affect today's work.

Avoid large company-level KPIs, raw backend counters, and decorative cards that do not drive an action.

### Auction Queue

The queue is the product assistant and operator's source of truth for today's lots. It should show status, order, readiness, rule completeness, current/next relation, and allowed actions. It should not become a platform-wide inventory table.

### Realtime Console

The console is the live control surface. Its information hierarchy should be stricter than other pages:

- Current lot, current price, countdown, leading bidder, latest bids, and ranking get the strongest weight.
- Live business actions use semantic color and spacing.
- Utility actions such as return, refresh, and sync stay visually quiet.
- Dangerous actions are separated, labeled clearly, and require an explicit reason where relevant.

The current top-bar pattern should be treated as a utility header. A sync button should not use the same visual strength as hammer settlement, abnormal cancellation, or other live-control actions.

### Orders

Orders should support post-auction handling, not expose private buyer records. Show final price, winner display name, payment state, deadline, support state, and required next action. Keep private buyer data masked unless a dedicated support permission and backend contract justify revealing it.

### Realtime Diagnostics

Diagnostics exists to prove the technical challenge from `1.pdf`: realtime synchronization, reconnect, consistency, and observability. Keep it scoped to the current room and current frontend session unless a platform observability product is explicitly introduced.

## Visual And Interaction Rules

- Use dense but organized layouts. This is a workbench for repeated operation, not a marketing page.
- Preserve clear typography levels: page title, section title, metric number, table label, helper text.
- Use tabular numbers for prices, countdowns, bid counts, and order amounts.
- Keep utility buttons quiet. Reserve strong primary/danger/warning colors for business-critical actions.
- Map button color to action semantics, not only to a shared variant name.
- Use icons from the existing icon library for tools and actions. Do not use emoji as production UI icons.
- Keep hit targets large enough for fast live operation, especially control buttons.
- Every action needs visible loading, disabled, success, and error states.
- Do not rely on color alone for status. Use text labels and accessible focus states.
- Motion should clarify state changes, such as bid accepted, rank changed, countdown extended, or settlement completed. Avoid decorative motion in core operations.
- Respect reduced-motion settings.
- Maintain WCAG AA contrast for text and controls.

## Backend Contract Guidance

Frontend display requirements should drive explicit contracts:

- If a P0 field is required by the UI, the backend should return it explicitly.
- If the backend has internal fields the streamer team does not need, the frontend should ignore them.
- If a required contract is missing or malformed, fail fast and surface an integration error in development/testing.
- Do not add frontend fallback logic for missing required fields during pre-launch.
- Do not keep old payload compatibility until the project status changes in `docs/COMPATIBILITY_POLICY.md`.

When a frontend page cannot be completed because an API is missing, implement or request the backend contract instead of hiding the gap behind mock data.

## Change Checklist

Before merging a UI change to the streamer workbench, verify:

- Can the operator answer: what is being auctioned, current price, remaining time, leading bidder, next action, and sync state?
- Can the product assistant answer: which lots are ready, which rules are incomplete, and what can still be edited?
- Can order support answer: which settlements produced orders, payment state, and what needs handling?
- Does every displayed backend field serve a streamer-team workflow?
- Are secrets, raw internals, company-wide metrics, and private buyer data hidden?
- Are utility actions visually quieter than live-control actions?
- Do required missing fields fail fast instead of falling back to fake values?
- Does the page remain usable on the target large viewport and normal laptop widths?
- Do focus states, keyboard use, contrast, and loading/error states meet baseline accessibility expectations?

## Cross-Project Marker

The same product boundary should be mirrored in the backend and buyer H5 projects:

- Backend owns strict contracts, validation, and safe data shaping for the streamer workbench.
- Admin frontend owns streamer-team operational presentation and action hierarchy.
- H5 owns buyer-facing bidding, ranking, reminders, result, and payment simulation.

Until launch compatibility is explicitly required, all three projects should stay fail-fast and avoid legacy fallback behavior.
