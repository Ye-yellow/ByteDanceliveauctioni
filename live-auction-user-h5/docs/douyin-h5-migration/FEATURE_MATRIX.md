# Douyin H5 Feature Matrix

This matrix tracks product behavior to reproduce from `tmp/douyin-reference/douyin-master` in the React H5 app while preserving LiveAuction business correctness.

## P0 Product Shell

| Feature | Reference files | Target status | Required target behavior |
|---|---|---|---|
| Mobile app frame | `src/App.vue`, `src/assets/less/layout.less`, `src/components/BaseFooter.vue` | partial via `.mobileShell`, `.homeShell`, `.douyinShell` | Use a Douyin-like full-screen mobile viewport with safe-area handling, no body scroll leaks, bottom tab bar, and route-level transitions where practical. |
| Home vertical feed | `src/pages/home/index.vue`, `src/components/slide/SlideVertical.vue`, `src/components/slide/SlideVerticalInfinite.vue`, `src/components/slide/SlideItem.vue` | complete for current P0 slice | Feed vertically pages through mixed short-video and live cards. Local video/social decoration is allowed, but room identity, current lot, title, price/status, and live-room navigation stay real. |
| Video playback card | `src/components/slide/BaseVideo.vue`, `src/components/slide/ItemDesc.vue`, `src/components/slide/ItemToolbar.vue` | partial in `HomePage.tsx` | Tap toggles play/pause, centered pause mark and progress bar exist, bottom description/action rail exist. Still missing scrubber, full video-detail route, music route, and persistent social state. |
| Live feed card | `src/pages/home/LivePage.vue`, `src/pages/home/slide/*` plus target auction APIs | complete for P0 data integration | Live cards visually sit in the same feed as short videos. Preview media may fall back to local demo video, but room metadata/current lot must come from real APIs and tap enters real `/m/room/:roomId`. |
| Live room | reference live page + target `LiveRoomView.tsx`, `AuctionDrawer.tsx` | P0 effects slice complete; broader parity partial | Full-screen live experience with Douyin chrome, anchor pill, viewer stack, rail, chat stream, join/gift effects, barrage, comment/share sheets, and auction drawer. Bidding, result, orders, payment must stay real. |
| Bottom navigation | `src/components/BaseFooter.vue` | shell exists | Tabs: 首页, 商城, 发布, 消息, 我 now exist via `DouyinTabBar`. Need interaction polish and route-specific active motion later. |
| Comments sheet | `src/components/Comment.vue`, `src/components/AutoInput.vue` | P0 facade complete; persistence deferred | Bottom sheet with count, avatar list, like/reply metadata, input toolbar. Used by video feed and live room; draggable close/reply persistence still deferred. |
| Share sheet | `src/components/Share.vue`, `src/pages/home/components/VideoShare.vue`, `src/pages/home/components/ShareToFriend.vue` | P0 facade complete; friend/report actions partial | Friend row, action grid, report/not interested/copy link entries. Copy link is real for live cards; friend/report flows remain facade-level. |
| Search | `src/pages/home/SearchPage.vue`, `src/components/Search.vue` | P0 overlay complete; full route partial | Full-screen overlay with hot tags and results across local video cards plus real room/lot metadata. Dedicated route/history/results parity is deferred. |
| Profile/me | `src/pages/me/Me.vue`, `src/pages/me/Me.less` | partial | Need closer top chrome, avatar/stat/action layout, right drawer, tabs, works/private/likes/collection blocks, and order/history entry. |
| Login/auth | `src/pages/login/*`, target `shared/auth/*` | route shell exists | `/login` wraps target buyer login/register/reset APIs. Production auth mode must never silently create demo users. |
| Orders/result/payment | target-specific `HistoryPage.tsx`, `ResultPage.tsx`, `MockPayModal.tsx` | exists | Result route now calls the real result API first. Orders/payment remain real LiveAuction flows and must be integrated into Douyin profile/message/shop surfaces. |

## P1 Reachable Flows

| Feature | Reference files | Target status | Required target behavior |
|---|---|---|---|
| Shop home | `src/pages/shop/Shop.vue`, `src/components/WaterfallList.vue`, `src/components/ScrollList.vue` | facade exists | Douyin shop tab shell with search, quick options, waterfall goods. Non-auction goods can be local/demo. |
| Goods detail | `src/pages/shop/GoodsDetail.vue`, `src/components/slide/SlideHorizontal.vue` | missing | Product detail visual parity; avoid fake auction checkout. |
| Message center | `src/pages/message/Message.vue`, notice/chat pages | facade exists | Message tab with notices, interactions, chat rows. Auction order/payment notices link to real history/result. |
| Publish | `src/pages/home/Publish.vue` | route facade and sheet exist | Full-screen publish facade exists. Needs closer camera/mode parity if creator posting becomes in scope. |
| Report flow | `src/pages/home/Report.vue`, `src/pages/home/SubmitReport.vue` | missing | Reachable from share sheet. |
| Video detail | `src/pages/other/VideoDetail.vue` | missing | Deep-linked feed item page. |
| Music detail | `src/pages/home/Music.vue` | missing | Reachable from rotating disc/music label. |
| Profile utilities | `src/pages/me/rightMenu/Setting.vue`, `LookHistory.vue`, `MyCard.vue`, collections | mostly missing | Support natural clicks from profile/drawer. |

## P2 Deep Facades

Profile school/location utilities, minor protection settings, scan/address/face-to-face, red packet detail, music rank list, and reference test pages can be added after the core shell and reachable flows are stable.

## Business Correctness Rules

1. Never replace `listPublicRooms`, `getRoomSnapshot`, `listRoomLots`, bidding, WebSocket, order, result, or mock payment calls with local mock data in production flows.
2. Local/demo data is acceptable for Douyin social/video/shop/message facade content that has no LiveAuction backend contract.
3. Home feed social/video mock data is allowed only for non-auction facade content: preview media, likes, comments, friend rows, tags, and local labels.
4. Live-room social effects may use local/demo engagement data, but room snapshot, lots, bids, prices, WebSocket events, orders, result, and payment must remain real.
5. If Douyin parity conflicts with auction correctness, auction correctness wins and the visual pattern adapts around the real state.
6. Every vertical slice must remain `npm run lint` and `npm run build` clean before the next slice starts.
