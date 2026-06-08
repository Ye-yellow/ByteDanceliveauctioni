# Douyin H5 Migration License Notes

## Reference License

The local Douyin reference at `tmp/douyin-reference/douyin-master` contains a `LICENSE` file with GPL-3.0 text. Treat the reference as a product/UX reference, not as a source-code or asset library for direct copying into `live-auction-user-h5`.

The reference README also describes the project as learning/research oriented. Until legal review says otherwise, do not copy source, media, datasets, icons, lyrics, screenshots, or exact product copy from the reference into this product.

## Allowed Use

- Study route structure, interaction patterns, layout hierarchy, sheet behavior, gesture behavior, and product journeys.
- Reimplement behavior in original React + TypeScript code.
- Use self-authored SVG/CSS shapes and existing target assets such as `public/demo-live.mp4`, `public/favicon.svg`, and `public/icons.svg`.
- Use local mock data for non-business social/video/shop/message facade content where LiveAuction has no backend contract.

## Avoid Without Explicit Review

- Copying Vue components, Less files, TypeScript utilities, Pinia stores, router code, or mock implementation code into the target app.
- Copying reference images, icons, posters, avatars, music covers, videos, JSON datasets, or lyrics into target `public`/`src`.
- Adding Vue, Pinia, Vue Router, or reference-only dependencies to the React H5 app.

## Asset Strategy

1. For P0 migration, use existing target demo media and self-authored CSS/SVG controls.
2. For shop/message/profile demo content, use generated/simple placeholder data and CSS gradients, not copied reference images.
3. If a specific reference asset becomes required, record the file path, reason, license implication, and replacement plan before adding it.

## Current Target Asset Inventory

- `public/demo-live.mp4`
- `public/favicon.svg`
- `public/icons.svg`

No reference asset has been approved for copying into the target app.

Current note from audit: `HomePage.tsx` uses `resolveLiveSource()` and `public/demo-live.mp4` for preview media. Additional feed media should be owned/generated replacement videos; do not use reference media URLs.
