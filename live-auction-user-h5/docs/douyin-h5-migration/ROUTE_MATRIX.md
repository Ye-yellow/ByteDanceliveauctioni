# Douyin H5 Route Matrix

Reference source: `../tmp/douyin-reference/douyin-master/src/router/routes.ts`

Target app: React + TypeScript + Vite in `live-auction-user-h5`. Vue, Pinia, and Vue Router are not target dependencies.

## Priority Rules

- **P0**: Required for the main user journey and for the first one-to-one Douyin shell: mixed short-video/live home feed, live room, profile, search, shop shell, message shell, login, auction order/result/payment.
- **P1**: Secondary pages users naturally reach from P0 surfaces: detail pages, edit/profile utility pages, settings, collections, report/share flows.
- **P2**: Deep settings, low-frequency flows, reference demo/test pages, or pages that can remain a faithful placeholder until core parity is stable.

## Route Coverage

| Reference route | Reference file | Target route | Target file/status | Priority | Migration notes |
|---|---|---|---|---|---|
| `/` -> `/home` | `src/router/routes.ts` | `/` | `src/pages/HomePage.tsx` P0 mixed-feed entry complete | P0 | Defaults to a Douyin-style vertical home feed. Short-video/social facade data is local; LiveAuction live cards hydrate from real `listPublicRooms`, `getRoomSnapshot`, and `listRoomLots`. |
| `/home` | `src/pages/home/index.vue` | `/` | `src/pages/HomePage.tsx` P0 slice complete, parity partial | P0 | Complete for this slice: top tabs, vertical paging, short-video/live mixed cards, action rail, comments/share/search/menu overlays, bottom tabs, live-room CTA. Deferred: persistent social state, follow/recommend backend behavior, report route, video detail, music detail. |
| `/home/live` | `src/pages/home/LivePage.vue` | `/m/room/:roomId` and live feed cards | `src/pages/LiveRoomPage.tsx`, `src/features/auction-room/components/LiveRoomView.tsx` live-room effects slice complete, parity partial | P0 | Live cards sit in the Douyin feed. `/m/room/:roomId` remains canonical for real bidding, room snapshot, lots, WebSocket, orders, result, and payment. |
| `/publish` | `src/pages/home/Publish.vue` | `/publish` | `src/pages/PublishPage.tsx` facade plus `src/shared/ui/PublishSheet.tsx` | P0 | Route-level publish facade exists. Needs closer camera/mode parity later; business actions route back to real LiveAuction pages. |
| `/home/search` | `src/pages/home/SearchPage.vue` | `/home/search` or sheet | Home search overlay partial | P0 | Full-screen overlay searches visible local video cards plus real room/lot metadata. Dedicated route, search history, and full results page are deferred. |
| `/shop` | `src/pages/shop/Shop.vue` | `/shop` | `src/pages/ShopPage.tsx` facade | P0 | Douyin shop tab shell exists with local demo goods and real order link. Needs detail page parity later; no fake auction payment. |
| `/shop/detail` | `src/pages/shop/GoodsDetail.vue` | `/shop/detail` | routes to `src/pages/ShopPage.tsx` facade | P1 | Product detail visual still missing; no fake auction payment. |
| `/message` | `src/pages/message/Message.vue` | `/message` | `src/pages/MessagePage.tsx` facade | P0 | Message tab shell exists with transaction/live/shop/profile entries. Auction order notices link to real history/result pages. |
| `/message/all` | `src/pages/message/AllMessage.vue` | `/message/all` | missing | P1 | Secondary message aggregation. |
| `/message/more-search` | `src/pages/message/MoreSearch.vue` | `/message/more-search` | missing | P1 | Message search page. |
| `/message/joined-group-chat` | `src/pages/message/JoinedGroupChat.vue` | `/message/joined-group-chat` | missing | P2 | Low-risk placeholder after message shell lands. |
| `/message/fans` | `src/pages/message/Fans.vue` | `/message/fans` | missing | P1 | Follower notification list. |
| `/message/visitors` | `src/pages/message/Visitors.vue` | `/message/visitors` | missing | P1 | Visitor notification list. |
| `/message/douyin-helper` | `src/pages/message/notice/DouyinHelper.vue` | `/message/douyin-helper` | missing | P1 | System helper notice detail. |
| `/message/system-notice` | `src/pages/message/notice/SystemNotice.vue` | `/message/system-notice` | missing | P1 | System notices; can include auction settlement notices. |
| `/message/task-notice` | `src/pages/message/notice/TaskNotice.vue` | `/message/task-notice` | missing | P2 | Secondary notices. |
| `/message/live-notice` | `src/pages/message/notice/LiveNotice.vue` | `/message/live-notice` | missing | P1 | Should surface live/auction reminders. |
| `/message/money-notice` | `src/pages/message/notice/MoneyNotice.vue` | `/message/money-notice` | missing | P1 | Should link to real orders/history, not fake balances. |
| `/message/notice-setting` | `src/pages/message/notice/NoticeSetting.vue` | `/message/notice-setting` | missing | P2 | Settings facade. |
| `/message/chat` | `src/pages/message/chat/Chat.vue` | `/message/chat` | missing | P1 | Local/demo chat UI acceptable. |
| `/message/chat/detail` | `src/pages/message/chat/ChatDetail.vue` | `/message/chat/detail` | missing | P2 | Deep chat settings. |
| `/message/chat/red-packet-detail` | `src/pages/message/RedPacketDetail.vue` | `/message/chat/red-packet-detail` | missing | P2 | Not business-critical for LiveAuction. |
| `/message/share-to-friend` | `src/pages/message/Share2Friend.vue` | `/message/share-to-friend` | missing | P1 | Share-to-friend flow should back comments/share sheet behavior. |
| `/me` | `src/pages/me/Me.vue`, `src/pages/me/Me.less` | `/m/profile` and `/me` | `src/pages/ProfilePage.tsx` partial | P0 | `/me` alias exists. Still needs closer reference profile: top actions, stats, tabs, works/private/likes/collect, right drawer. Preserve buyer session entry points. |
| `/me/edit-userinfo` | `src/pages/me/userinfo/EditUserInfo.vue` | `/me/edit-userinfo` | missing | P1 | Can edit local profile facade; do not mutate backend unless matching API exists. |
| `/me/edit-userinfo-item` | `src/pages/me/userinfo/EditUserInfoItem.vue` | `/me/edit-userinfo-item` | missing | P2 | Detail edit page. |
| `/me/country-choose` | `src/pages/login/countryChoose.vue` | `/me/country-choose` | missing | P2 | Login/user utility. |
| `/me/my-card` | `src/pages/me/MyCard.vue` | `/me/my-card` | missing | P1 | QR/card page. |
| `/me/add-school` and choose/declare/display/location/city/province routes | `src/pages/me/userinfo/*` | same aliases | missing | P2 | Low-frequency profile utilities. |
| `/me/right-menu/look-history` | `src/pages/me/rightMenu/LookHistory.vue` | `/me/right-menu/look-history` | `/m/history` exists for auction orders | P1 | Need reference-style watch history facade plus links to real auction order history. |
| `/me/right-menu/setting` | `src/pages/me/rightMenu/Setting.vue` | `/me/right-menu/setting` | missing | P1 | Settings page should include auth/session controls from target app. |
| `/me/collect/music-collect` | `src/pages/me/collect/MusicCollect.vue` | same | missing | P2 | Collection facade. |
| `/me/collect/video-collect` | `src/pages/me/collect/VideoCollect.vue` | same | missing | P1 | Collection facade can reuse local feed cards. |
| `/me/my-music` | `src/pages/me/MyMusic.vue` | same | missing | P2 | Music page facade. |
| `/login` | `src/pages/login/Login.vue` | `/login` | `src/pages/LoginPage.tsx` | P0 | Buyer login/register/reset route exists and uses target auth APIs. Production auth must not silently create demo users. |
| `/login/other` | `src/pages/login/OtherLogin.vue` | same | missing | P1 | Visual parity; real auth options only if supported. |
| `/login/password` | `src/pages/login/PasswordLogin.vue` | same | missing | P0 | Buyer password login/register should use existing target auth APIs. |
| `/login/verification-code` | `src/pages/login/VerificationCode.vue` | same | missing | P1 | Facade unless SMS API exists. |
| `/login/retrieve-password` | `src/pages/login/RetrievePassword.vue` | same | missing | P2 | Facade unless backend supports it. |
| `/login/help` | `src/pages/login/Help.vue` | same | missing | P2 | Help page. |
| `/people/find-acquaintance` | `src/pages/people/FindAcquaintance.vue` | same | missing | P2 | Social discovery facade. |
| `/people/follow-and-fans` | `src/pages/people/FollowAndFans.vue` | same | missing | P1 | Needed from profile stats. |
| `/address-list` | `src/pages/people/AddressList.vue` | same | missing | P2 | Social address book facade. |
| `/scan` | `src/pages/people/Scan.vue` | same | missing | P2 | Scanner facade. |
| `/face-to-face` | `src/pages/people/FaceToFace.vue` | same | missing | P2 | Social utility facade. |
| `/set-remark` | `src/pages/message/SetRemark.vue` | same | missing | P2 | Chat utility. |
| `/home/music` | `src/pages/home/Music.vue` | same | missing | P1 | Music detail/cover behavior supports feed disc interactions. |
| `/home/music-rank-list` | `src/pages/home/MusicRankList.vue` | same | missing | P2 | Music ranking facade. |
| `/home/report` | `src/pages/home/Report.vue` | same | missing | P1 | Needed from share/report sheet. |
| `/home/submit-report` | `src/pages/home/SubmitReport.vue` | same | missing | P1 | Report submission facade. |
| `/video-detail` | `src/pages/other/VideoDetail.vue` | same | missing | P1 | Needed for shared/deep-linked feed items. |
| `/m/room/:roomId` | target-specific | `/m/room/:roomId` | `src/pages/LiveRoomPage.tsx` exists | P0 | LiveAuction-specific route with Douyin-like chat stream, join/gift toasts, barrage, anchor/viewer chrome, and real auction drawer. Keep real business flow. |
| `/m/result/:lotId` | target-specific | `/m/result/:lotId` | `src/pages/ResultPage.tsx` exists | P0 | LiveAuction-specific result route now fetches `GET /api/lots/{lotId}/result` first; query params are fallback only. |
| `/m/history` | target-specific | `/m/history` | `src/pages/HistoryPage.tsx` exists | P0 | LiveAuction-specific orders/bids route; should be surfaced through `/me/right-menu/look-history`, `/message/money-notice`, and profile/order entries. |
| `/test`, `/test4` | `src/pages/test/*` | none | intentionally omitted | P2 | Reference dev/test pages are not product parity requirements. |

## Immediate Route Gaps

1. Report, share-to-friend, video detail, music detail, shop detail, message detail, and profile utility routes are the next reachability layer after the shell lands.
2. The target now has first-class home/shop/message/publish/login/me route shells, but shop/message/publish are facade-level, not full one-to-one parity.
3. LiveAuction-specific routes must remain canonical for auction operations even when entered from Douyin-like tabs or sheets.
4. Home feed social/video mock data is allowed only for non-auction facade content: preview media, likes, comments, friend rows, tags, and local labels. Auction live cards must never mock room, lot, price, status, bidding, result, order, or payment data.
5. Live-room chat/join/gift/barrage effects are in-room facade behavior, not new route or backend contract coverage.
