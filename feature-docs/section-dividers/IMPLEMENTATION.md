# Implementation Plan: Section Divider Rules

## Overview

Implement the updated design's divider rules: floating 1px horizontal rules
between rows and vertical rules between in-row sections, replacing
spacing-only separation, plus a 1-unit gap under the config-error banner.

## Technology Stack

- **Language**: Go (existing project)
- **UI**: GTK4 via gotk4 (already in go.mod) — no new dependencies (NFR1)

## Layer Structure

Unchanged. All changes stay inside `internal/ui` (styling + section layout).
No changes to config, runner, markup, or app packages.

## Shared Components

None — single-task feature; no cross-task contracts.

## Conventions

- Divider color/width are fixed constants mirroring the design tokens
  (`color.divider` #666666, `border.divider-width` 1px in
  design-system/tokens.yaml); they are NOT user-overridable style keys.
- Divider widgets carry dedicated CSS classes (`dv-hdivider`,
  `dv-vdivider`) styled from the window-level stylesheet, consistent with
  the existing `dv-*` class naming.

## Cross-task Design Decisions

None beyond the single task.

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| GTK CSS margin behavior differs from the intended inset/breathing-room layout | Low | Medium | Dividers are plain box widgets with explicit size + margins; manual scenario M-1 verifies against the mockup |
| Gap arithmetic drift between layout math and rendered divider footprint | Medium | Medium | One constant defines the divider footprint used by both the packing gap and the divider CSS; unit test pins it (TS-3) |

## Open Questions

- [ ] None
