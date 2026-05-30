# Douyin H5 Migration QA Report

## Current Baseline

- Target app: `live-auction-user-h5`
- Stack requirement: React + TypeScript + Vite
- Reference app: `tmp/douyin-reference/douyin-master`
- Backend/admin scope: unchanged unless a real contract bug is proven.

## Verification Commands

Run after each vertical slice:

```bash
npm run lint
npm run build
```

Browser checks should cover at least:

- `/` home mixed feed
- `/m/room/:roomId` real LiveAuction room
- `/m/result/:lotId` auction result
- `/m/history` orders/bids
- `/m/profile` and `/me` profile alias once added
- `/shop`, `/message`, `/login` once added

## Known Current Gaps

1. Router coverage is far below reference coverage.
2. Bottom navigation includes shop and message tabs, but route-specific animation/state polish is still incomplete.
3. Shop, message, many profile utility pages, report, video detail, music detail, and route-level login parity are missing.
4. Current home/live/profile work is partial and must be checked against reference screenshots and gestures, not only build success.
5. CSS debt is expected during migration; do not treat raw visual parity CSS as complete until screenshots pass.

## Slice 1 Verification - Douyin Shell Routes

- Added `/home`, `/shop`, `/message`, `/publish`, `/login`, `/me` route coverage through the React router.
- Added five-tab Douyin-style bottom navigation for home/shop/publish/message/me surfaces.
- Updated `/m/result/:lotId` to fetch `GET /api/lots/{lotId}/result` first; query params are now only a visual fallback when server sync fails.
- `npm run lint`: passed.
- `npm run build`: passed with existing large-player chunk warning only.
- Browser route smoke at 390px: `/home`, `/shop`, `/message`, `/publish`, `/login`, `/me`, `/m/result/demo-lot?...`; no horizontal overflow found.
- Screenshots saved under `tmp/shot/h5-douyin-shell-*-v2.png`.

## Remaining High-Risk Gaps

- Home feed now has mixed short-video/live cards, but true video detail/music/follow state parity is still deferred.
- Comments/share sheets are still duplicated in home and live room; reference-style reusable draggable bottom sheet is not extracted yet.
- Live room now includes chat/join/gift/barrage effects, but persistence, real social/gift backend, exact timing polish, and deeper short-height compression remain deferred.
- Shop/message/publish pages are facade-level P0 route shells, not full reference parity.
- Mixed feed currently reuses `public/demo-live.mp4` and backend/live source fallbacks; owned replacement videos are still needed for richer media variety.
- Deep reference routes such as `/message/chat`, `/login/password`, and `/shop/detail` still collapse to broad facades; do not count them as one-to-one complete pages yet.
- `npm run check:ui-debt` is known to fail on raw colors, shadows, z-indexes, and durations during this migration slice; this needs a later token cleanup pass.
- Short-height live-room right rail still needs a mobile compression pass.

## QA Follow-Up Fixes

- Fixed `/login?next=//host` style open redirect by normalizing `next` to same-origin relative paths only.
- Fixed result page fallback trust issue: when server result sync fails, URL query data no longer presents as confirmed success/payment state.

## Slice 2 Verification - Mixed Home Feed

- Converted `HomePage.tsx` to a discriminated `HomeFeedItem` model with local short-video cards interleaved with real LiveAuction live cards.
- Live cards still use `listPublicRooms`, `getRoomSnapshot`, and `listRoomLots` for room, lot, price, and status metadata.
- Video cards use local/demo social data only; they do not mock auction room, lot, bidding, result, order, or payment state.
- Feed gestures now distinguish vertical swipes from horizontal movement and advance on fast flicks or meaningful vertical drags.
- Video cards tap to pause/play and show a centered play glyph plus progress bar; live cards tap through to `/m/room/:roomId`.
- Search overlay now includes both local video cards and real live room/lot metadata.
- `npm run lint`: passed.
- `npm run build`: passed with existing large-player chunk warning only.
- Browser smoke at 390x844 on `/home`: 6 rendered feed items, 3 video cards, 3 live cards, no horizontal/body overflow, bottom tab and action rail present, no console/page errors.
- Screenshot saved to `tmp/shot/h5-home-mixed-feed.png`.

## Slice 3 Verification - Live Room Effects

- Added live-room chat stream, join ticker, gift/heat burst, barrage lane, and viewer stack over `/m/room/:roomId`.
- Effects consume existing `room.recentBids`, `room.ranking`, `notices`, and `currentLot`; they do not replace snapshot/WebSocket/lots/bidding/order/result/payment flows.
- Comment/share sheets, auction drawer, quick bid/auth, result modal, deposit confirmation, and mock payment remain owned by the existing LiveAuction room stack.
- `npm run lint`: passed.
- `npm run build`: passed with existing large-player chunk warning only.
- Browser smoke at 390x844 and 390x667 on `/m/room/317989750961078272`: effects layer, chat, barrage, join ticker, gift burst, viewer stack, auction drawer button, quick bid, and composer present; no horizontal/body overflow or console/page errors.
- Screenshot saved to `tmp/shot/h5-live-room-effects.png`.

## Last Verified Baseline

- `npm run lint`: passed after live-room effects slice.
- `npm run build`: passed after live-room effects slice, with existing large-player chunk warning only.

## Regression Gates

- Production auth must not silently create demo users.
- `/m/room/:roomId` must keep room snapshot, WebSocket updates, bidding, auction drawer, result modal, order creation, and mock payment behavior.
- New Douyin facade pages must not import Vue, Pinia, or Vue Router.
- No copied GPL reference source/assets should be introduced into the target app without a deliberate license decision.
