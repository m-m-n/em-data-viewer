# Design: app-icon

## Decisions

- Motif: widget self-portrait — the icon depicts the app's own framed dark
  window (rounded rect, `color.surface` family, `color.frame` outline) with
  two sections of abstracted output rows separated by a floating divider
  rule, mirroring the process-output-widget look. Mockup:
  [design/mockups/screen-app-icon.html](design/mockups/screen-app-icon.html).
- Colors: taken from `design-system/tokens.yaml` with two icon-only
  deviations recorded here (no tokens added — the icon is a build asset,
  not runtime UI):
  - body fill `#1e1e1e` = `color.surface` made opaque (alpha dropped);
    a translucent icon degrades on arbitrary launcher backgrounds.
  - one accent row `#40ff40` — eM-family green (shared with eMterm's icon)
    for launcher-grid identification and brand consistency. Used exactly
    once; all other rows are `color.on-surface` / `color.on-surface-dim`.
- Geometry (canonical SVG source — implementation authors `assets/icon.svg`
  with exactly this content):

  ```svg
  <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 128 128">
    <rect x="8" y="8" width="112" height="112" rx="20" fill="#1e1e1e" stroke="#888888" stroke-width="5"/>
    <rect x="28" y="30" width="32" height="7" rx="3.5" fill="#9e9e9e"/>
    <rect x="28" y="44" width="64" height="9" rx="4.5" fill="#40ff40"/>
    <rect x="28" y="59" width="48" height="9" rx="4.5" fill="#e0e0e0"/>
    <line x1="26" y1="76" x2="102" y2="76" stroke="#666666" stroke-width="3" stroke-linecap="round"/>
    <rect x="28" y="86" width="56" height="9" rx="4.5" fill="#e0e0e0"/>
    <rect x="28" y="101" width="40" height="9" rx="4.5" fill="#e0e0e0"/>
  </svg>
  ```

- Scale behavior: single SVG rendered at 48/64/128/256 (FR2); row bars are
  ≥ 9/128 units thick so the smallest raster (48px) keeps ~3.4px rows and a
  ~1.1px divider — verified visually in the mockup's size strip.
- Corner radius 20/128: launcher-grid scale, deliberately NOT
  `radius.window` (8px is a window-chrome value; at icon scale it reads as
  square).

## Rationale

- Self-portrait motif: the app is itself a visual object on the desktop;
  the icon that looks like the running widget is the most honest
  identifier, and reuses the established design language (frame + floating
  dividers + row rhythm) from process-output-widget's DESIGN.md.
- Single green accent: the all-grey token palette is deliberately
  low-contrast (tone: 「主張しない」), which would vanish in a launcher
  grid; one eM-green row adds identification without breaking the minimal
  tone, and ties the app to the eM family (eMterm uses the same green).
- No new tokens: tokens.yaml documents runtime UI defaults that user config
  can override; icon colors are frozen build-time values, so they belong in
  this DESIGN.md, not the token SSOT.
- Divider ends inset, not touching the frame: same no-junction rule as the
  widget's own divider design.

## Open items

- None.
