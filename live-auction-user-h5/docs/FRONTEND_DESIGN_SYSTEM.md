# LiveAuction H5 Design System

This document is the source of truth for new UI work in
`live-auction-user-h5`. It records the target direction for future changes;
legacy CSS can remain until the related screen is touched and verified.

## Product Direction

LiveAuction H5 is an immersive mobile live-auction experience. The UI should
feel video-first, touch-first, and fast. The main user jobs are entering a live
room, following the current lot, bidding, seeing the result, and completing the
order/payment path.

## Current Architecture

- React/Vite app.
- App root is `src/app/App.tsx`.
- Handwritten path routing lives in `src/app/router.tsx`.
- Feature components live under `src/features/*/components`.
- Styles are global CSS, not Tailwind:
  - `src/index.css`
  - `src/app/styles.css`

## Preferred Token Source

New H5 UI must prefer `src/app/styles.css`.

Use the `--live-*` family for product-level tokens. Existing aliases such as
`--panel-bg`, `--card-bg`, `--text-main`, and `--primary-red` remain for
compatibility, but new work should use semantic `--live-color-*`,
`--live-z-*`, and `--live-duration-*` tokens.

## New Code Rules

- Do not add raw hex, RGB, HSL, or named colors inside components.
- Do not add arbitrary z-index values. Use `--live-z-*`.
- Do not add `!important` unless overriding the video player or a verified
  legacy cascade conflict.
- Keep all important touch targets at least 44px high or wide.
- Fixed bottom actions must account for `env(safe-area-inset-bottom)`.
- Video/player overlays, bid panels, result sheets, and toasts must use the
  documented layer scale.

## Preferred Feedback Patterns

H5 does not yet have a formal component primitive package. Until one exists,
new or touched feedback UI should reuse the existing CSS patterns in
`src/app/styles.css`:

| Pattern | Use |
|---|---|
| `noticeLayer` | transient notices |
| `modalMask` | modal/sheet blocking overlays |
| `statusPill` / `orderStatePill` | compact status labels |
| `emptyState` | empty and error states |
| `connectionWarn` / `cancelBanner` | blocking or payment-adjacent warnings |

Do not create another toast, status pill, or modal layer without first checking
these patterns.

## Layering Scale

Use these names for new or touched layering work:

| Token | Intent |
|---|---|
| `--live-z-base` | normal content |
| `--live-z-feed` | feed item content |
| `--live-z-raised` | local raised content inside one screen |
| `--live-z-local-overlay` | local overlays inside one screen, below global video/overlay layers |
| `--live-z-video` | video/player media layer |
| `--live-z-overlay` | live overlay controls and readable scrims |
| `--live-z-sticky` | sticky room/header/bottom controls |
| `--live-z-sheet` | bid panels and bottom sheets |
| `--live-z-modal` | modal content |
| `--live-z-toast` | toasts and transient notices |
| `--live-z-tooltip` | tips and helper bubbles |

## Status Tone Map

Use the shared `--live-color-*` status tokens for new or touched H5 feedback:

| Tone | Use |
|---|---|
| `success` | paid orders, successful bids, healthy queue state |
| `warning` | pending payment, almost-ending auction, recoverable attention |
| `error` | failed payment, rejected bid, destructive or blocked state |
| `info` | neutral guidance, processing, current room/status hints |
| `neutral` | inactive, empty, unknown, or disabled states |

Fixed overlays, bid panels, order cards, notices, and status pills must use
`--live-color-*-bg`, `--live-color-*-border`, and `--live-color-*-text` when a
status color is touched.

## Motion Scale

Use these names for new or touched motion work:

| Token | Intent |
|---|---|
| `--live-duration-fast` | tap/press feedback |
| `--live-duration-normal` | panel and state changes |
| `--live-duration-slow` | route-like or sheet transitions |
| `--live-ease-standard` | default UI easing |
| `--live-ease-swipe` | swipe/pager motion |

All nonessential animation must respect `prefers-reduced-motion`.

## Mobile UX Rules

- Primary bid/payment actions must remain reachable without colliding with the
  gesture bar.
- Do not rely on hover-only behavior.
- Do not block vertical scrolling with nested horizontal gestures unless the
  interaction has visible affordance and a drag threshold.
- Loading longer than 300ms needs spinner, skeleton, or clear status text.
- Destructive or payment-adjacent actions need clear confirmation or recovery.

## Migration Policy

Do not clean old CSS globally. When a screen is touched:

1. Identify its active selectors in `src/app/styles.css`.
2. Move only touched colors, layers, and motion values to preferred tokens.
3. Preserve video/player compatibility selectors until verified on device-sized
   viewports.
4. Build, lint, and inspect the affected screen.

## Acceptance Checks

For UI changes in this project:

```bash
npm run check:ui-debt
npm run build
npm run lint
```

From the suite root, use this when validating both frontends together:

```bash
./scripts/check-frontend-quality.sh
```

`check:ui-debt` is a focused guard for already-governed H5 surfaces. It checks
that `src/app/styles.css` does not reintroduce raw numeric `z-index` values or
direct `env(safe-area-inset-bottom)` usage outside `--live-safe-bottom`. It also
checks that component-level animation and transition durations use
`--live-duration-*` aliases instead of raw `ms` / `s` literals. It is not a full
CSS linter and should not be expanded to fail on untouched legacy visual
identity styles without a migration phase.

Manual checks:

- 375x812, 390x844, 430x932, and landscape viewports.
- Video layer, overlay, bid panel, toast, and modal do not conflict.
- Bottom actions respect safe area.
- Touch targets are large enough and visibly respond to tap.
- Reduced-motion users are not forced through decorative animation.
