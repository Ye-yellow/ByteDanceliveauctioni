# LiveAuction Admin Design System

This document is the source of truth for new UI work in
`live-auction-bid-frontend`. It records the target direction for future changes;
legacy CSS can remain until the related page is touched and verified.

## Product Direction

LiveAuction Admin is a real-time operations command center for a fixed live
auction room. The interface should be dense, calm, and scannable. Operators need
to see state, risk, queue progress, orders, and realtime health quickly without
decorative noise competing with decisions.

## Current Architecture

- React/Vite app with handwritten route selection in `src/app/App.tsx`.
- Admin pages route through `src/pages/host-console/HostConsolePage.tsx`.
- Admin shell and Studio primitives currently live under
  `src/pages/host-console/components/`.
- Styles are global CSS, not Tailwind:
  - `src/app/studio-tokens.css`
  - `src/app/styles.css`
  - `src/pages/host-console/styles/console-round06.css`
  - feature-level CSS such as `src/features/auction-manage/admin-dashboard.css`

## Preferred Token Source

New admin UI must prefer `src/app/studio-tokens.css`.

Use the `--studio-*` family for new code. Treat `--la-*`, `--merchant-*`, and
page-private variables as legacy or local compatibility aliases. Do not create a
new token family for one page.

Preferred token groups:

- Color: `--studio-color-*`
- Radius: `--studio-radius-*`
- Shadow: `--studio-shadow-*`
- Layering: `--studio-z-*`
- Motion: `--studio-duration-*`, `--studio-ease-*`

## New Code Rules

- Do not add raw hex, RGB, HSL, or named colors inside components.
- Do not add arbitrary z-index values. Use `--studio-z-*`.
- Do not add `!important` unless the selector is explicitly overriding a
  third-party widget or a verified legacy cascade conflict.
- Do not introduce another button, card, badge, toast, modal, table, or field
  system without first checking the Studio primitives.
- Keep navigation, shell, modal, drawer, toast, and table behavior stable before
  changing decorative styling.

## Preferred Primitives

Use the existing primitives in
`src/pages/host-console/components/studio-ui.tsx` for new admin UI:

| Primitive | Use |
|---|---|
| `StudioButton` | primary, secondary, ghost, danger, soft actions |
| `StudioCard` | page panels and grouped operational content |
| `StudioBadge` | compact status and metadata labels |
| `StudioMetricCard` | dashboard and realtime numeric summaries |
| `StudioTable` | tabular admin data |
| `StudioEmptyState`, `StudioLoadingState`, `StudioErrorState` | empty/loading/error surfaces |
| `StudioToast` / `StudioToastViewport` | transient feedback |

Only create a new primitive when one of these cannot express the interaction
without duplicating CSS or breaking accessibility.

## Layering Scale

Use these names for new or touched layering work:

| Token | Intent |
|---|---|
| `--studio-z-base` | normal content |
| `--studio-z-raised` | local raised/overlapping content |
| `--studio-z-sticky` | sticky section controls |
| `--studio-z-header` | topbar/sidebar shell |
| `--studio-z-dropdown` | menus, popovers, select menus |
| `--studio-z-overlay` | page scrims |
| `--studio-z-drawer` | side drawers |
| `--studio-z-modal` | modal content |
| `--studio-z-toast` | toasts and notifications |
| `--studio-z-tooltip` | tooltips |

## Status Tone Map

Use `StudioTone` for new admin feedback and primitives:

| Tone | Use |
|---|---|
| `success` | completed operations, paid orders, healthy realtime checks |
| `warning` | non-blocking risk, pending payment, countdown preparation |
| `danger` | failed operations, destructive actions, cancelled/abnormal states |
| `info` | neutral guidance, sync status, active informational states |
| `purple` | secondary emphasis for analytics and queue transitions |
| `neutral` | inactive, empty, draft, or unknown states |

New or touched CSS must use `--studio-color-*-bg`, `--studio-color-*-border`,
and `--studio-color-*-text` status tokens instead of raw green, red, yellow, or
blue literals.

## Motion Scale

Use these names for new or touched motion work:

| Token | Intent |
|---|---|
| `--studio-duration-fast` | hover, press, minor color changes |
| `--studio-duration-normal` | standard component state changes |
| `--studio-duration-slow` | drawers, sheets, complex transitions |
| `--studio-ease-standard` | default UI easing |
| `--studio-ease-emphasized` | entering surfaces and larger motion |

All nonessential animation must respect `prefers-reduced-motion`.

## Migration Policy

Do not clean old CSS globally. When a page is touched:

1. Identify the active CSS selectors for that page.
2. Move only the touched colors, layers, and motion values to preferred tokens.
3. Remove `!important` only when the replacement has been verified in the page.
4. Build and inspect the affected page at the agreed viewports.

## Acceptance Checks

For UI changes in this project:

```bash
npm run check:ui-debt
npm run build
```

From the suite root, use this when validating both frontends together:

```bash
./scripts/check-frontend-quality.sh
```

`check:ui-debt` is a focused guard for already-governed surfaces. It checks
that the Admin dashboard does not reintroduce hardcoded `#hex` / `rgba()`
colors. It is not a full CSS linter and should not be expanded to fail on
untouched legacy files without a migration phase.

Manual checks:

- Sidebar/topbar do not cover content.
- Drawer, modal, toast, and dropdown layers do not conflict.
- Focus states are visible on keyboard navigation.
- Disabled and loading states are visually distinct.
- Status color is paired with text or icon, not color alone.
