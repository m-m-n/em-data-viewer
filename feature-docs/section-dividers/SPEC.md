# Feature: Section Divider Rules

## Overview

Replace spacing-only section separation with floating 1px divider rules,
implementing the updated design (design-system/tokens.yaml `color.divider` /
`border.divider-width`; process-output-widget DESIGN.md and mockup
`screen-widget.html`, both already updated). Rows are separated by a
horizontal rule with zero extra vertical gap; sections sharing a row are
separated by a vertical rule with 1 unit breathing room on each side. Every
rule's ends are inset 1 unit so rules never touch each other or the window
frame.

## Objectives

- Clear section boundaries without residual vertical space, keeping the
  minimal frame-only look.
- Divider layout matches the updated mockup exactly.
- Keep every existing section behavior unchanged.

## User Stories

### US1: Clear section separation
As a user, I want sections separated by subtle floating rules, so that
boundaries are clear without boxes, grids, or wasted vertical space.

**Acceptance Criteria:**
- [ ] Adjacent rows are separated by a 1px horizontal rule with no extra
      vertical gap; the rule's ends are inset 1 unit (6px) from the row
      edges.
- [ ] Adjacent sections in one row are separated by a 1px vertical rule
      with 1 unit of space on each side; the rule stretches with the row
      and its ends are inset 1 unit top/bottom.
- [ ] No rule touches the window frame or another rule (no corners, T- or
      cross-junctions).

## Technical Requirements

### Functional Requirements

- **FR1:** Horizontal divider between adjacent rows: a 1px-high widget in
  `color.divider` (#666666), horizontal margins 1 unit (6px), zero vertical
  margins. Rows N rows → N−1 horizontal dividers; a single row (or zero
  rows) renders none.
- **FR2:** Vertical divider between adjacent sections in one row: a
  1px-wide widget in `color.divider`, margins 1 unit on all sides (1 unit
  left/right breathing room; 1 unit top/bottom inset), stretching to the
  row height. A row with M sections holds M−1 vertical dividers.
- **FR3:** Divider color (#666666) and width (1px) are fixed constants
  mirroring the design tokens; no config override keys are added.
- **FR4:** The row-packing gap passed to `packRows` equals the vertical
  divider's real horizontal footprint: 1 unit + divider width + 1 unit
  (6 + 1 + 6 = 13px), replacing the previous 2-unit (12px) column gap. Row
  boxes use spacing 0 (the divider's margins provide the gaps).
- **FR5:** The config-error banner gets 1 unit (6px) of space between its
  bottom edge and the sections below it (mockup: banner margin-bottom
  1 unit).
- **FR6:** Existing behaviors are preserved: `setSections` rebuild
  semantics, `updateSection` (markup update, error style), reflow triggers
  and the unchanged-assignment skip, per-section style classes and
  `styleEntries`, empty-sections case.

### Non-Functional Requirements

- **NFR1 - Dependencies:** No new external dependencies.
- **NFR2 - Quality gates:** `go build ./...`, `go test ./...` pass and
  gofmt reports no diffs.

## Implementation Approach

- `internal/ui/sections.go` — `rebuildRows` inserts divider widgets:
  a horizontal divider box (CSS class `dv-hdivider`) between row boxes,
  and a vertical divider box (CSS class `dv-vdivider`) between section
  roots inside each row box. Row boxes get spacing 0. Dividers are plain
  widgets rebuilt with the rows (they carry no state).
- `internal/ui/style.go` — add divider constants (color `#666666`, width
  1px, matching design tokens) and the layout constants for the new gap
  arithmetic (`sectionColumnGap` becomes the divider footprint 13px, or an
  equivalent constant set); emit `.dv-hdivider` / `.dv-vdivider` CSS rules
  (background color, size, margins) from `windowCSS`; add the banner's
  bottom margin (FR5).
- `internal/ui/banner.go` — only if the banner spacing cannot be expressed
  in `windowCSS`.
- Tests: extend `style_test.go` (divider CSS present, constants), extend
  `sections_test.go` if divider counting is factored as a pure helper.

### Dependencies

**Internal Dependencies:**
- `internal/ui/sections.go`, `internal/ui/style.go`, tests. No changes to
  config, runner, markup, or app packages.

**External Dependencies:**
- gotk4 GTK4 bindings (already in go.mod). Dividers are plain `gtk.Box`
  widgets styled via CSS (no gtk.Separator needed; a Box guarantees exact
  1px sizing via CSS).

## Test Scenarios

### Unit Tests
- [ ] TS-1: `windowCSS` output contains a `.dv-hdivider` rule with the
      divider color, 1px height, and 6px horizontal margins.
- [ ] TS-2: `windowCSS` output contains a `.dv-vdivider` rule with the
      divider color, 1px width, and 6px margins.
- [ ] TS-3: The `packRows` gap constant equals 13px (unit + divider width
      + unit); row-box spacing is 0.
- [ ] TS-4: Banner CSS includes the 1-unit bottom spacing (FR5).
- [ ] TS-5: Existing `internal/ui` tests keep passing; full suite
      `go test ./...` passes.

### Integration Tests
None (GUI behavior is verified manually per test/README.md).

### E2E Tests
**Existing E2E tests**: None
**Run command**: Not detected

### Manual Verification
- [ ] M-1: With a multi-section config, rows are separated by floating
      horizontal rules and in-row sections by floating vertical rules,
      matching mockup state `default`; no rule touches the frame or
      another rule.
- [ ] M-2: Config reload, section updates, per-section style overrides and
      section errors render as before, with dividers intact.
- [ ] M-3: The config-error banner shows 1 unit of space above the
      sections (mockup state `config-error`).

### Edge Cases
- [ ] E-1: A single row renders no horizontal divider; a single-section
      row renders no vertical divider.
- [ ] E-2: Zero sections render the empty framed window with no dividers.

## Success Criteria

- [ ] All functional requirements are implemented and tested
- [ ] `go build ./...`, `go test ./...`, gofmt clean
- [ ] Manual scenarios M-1..M-3 confirmed by the user

## Open Questions

None.

## References

- REQUIREMENTS.md: feature-docs/section-dividers/REQUIREMENTS.md
- Design tokens: design-system/tokens.yaml (`color.divider`,
  `border.divider-width`, spacing description)
- Design decisions: feature-docs/process-output-widget/DESIGN.md (updated
  Sections bullet)
- Mockup: feature-docs/process-output-widget/design/mockups/screen-widget.html
- Prior layout feature: feature-docs/section-row-packing/
