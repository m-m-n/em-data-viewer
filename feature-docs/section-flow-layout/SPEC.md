# Feature: Section Flow Layout

## Overview

Replace the vertical one-column stack of sections with a horizontally
reflowing layout: sections are laid out left to right in config order and
wrap to the next row when the remaining width is insufficient. Implemented
with GtkFlowBox (horizontal orientation) in `internal/ui/sections.go`.

## Objectives

- Use horizontal space when the window is wide (e.g. width 800): narrow
  sections share a row instead of each occupying a full-width line.
- Keep every existing section behavior (reload, per-section update,
  per-section styling, error rendering) unchanged.

## User Stories

### US1: Compact display on a wide window
As a user, I want narrow sections to pack side by side, so that a wide
window shows the same information in less vertical space.

**Acceptance Criteria:**
- [ ] At window width 800, multiple narrow sections render on one row.
- [ ] A section that does not fit in the remaining row width starts the
      next row.
- [ ] Visual order equals config order (left→right, then top→bottom).

## Technical Requirements

### Functional Requirements

- **FR1:** The sections container is a `gtk.FlowBox` with horizontal
  orientation. Section root widgets (label + output vertical `gtk.Box`,
  unchanged) are appended in config order; the flow box wraps them into
  rows based on available width.
- **FR2:** Cells use their natural width: `SetHomogeneous(false)`. A
  section wider than the available row width occupies its own row; output
  label wrapping behavior is unchanged. The flow box must not impose an
  artificial cap on sections per row (`SetMaxChildrenPerLine` set high
  enough to be a non-constraint, e.g. 100).
- **FR3:** Spacing: `SetColumnSpacing(12)` (2 spacing units) and
  `SetRowSpacing(0)`. The `sectionGap` constant's role is replaced by
  these two values; `design-system/tokens.yaml` and `tokens.html` spacing
  descriptions are updated to "セクション間 横 2 unit / 縦 0".
- **FR4:** No container-added visuals: `SetSelectionMode(gtk.SelectionNone)`,
  `SetActivateOnSingleClick(false)` is not required, and the app
  stylesheet resets the auto-inserted wrapper:
  `flowboxchild { padding: 0; margin: 0; }`.
- **FR5:** Existing behaviors are preserved with the container swap:
  - `setSections` rebuilds the displayed set (removal must account for
    the `GtkFlowBoxChild` wrapper: passing the original child widget to
    `FlowBox.Remove` is sufficient — GTK resolves the wrapper).
  - `updateSection` (markup update, error style) is untouched.
  - Per-section style classes and `styleEntries` are untouched.

### Non-Functional Requirements

- **NFR1 - Dependencies:** No new external dependencies; FlowBox comes
  from the already-vendored gotk4 GTK4 bindings.
- **NFR2 - Quality gates:** `go build ./...`, `go test ./...` pass and
  gofmt reports no diffs.

## Implementation Approach

`internal/ui/sections.go`:
- `sectionsView.widget` changes type from `*gtk.Box` to `*gtk.FlowBox`;
  `newSectionsView` constructs it with orientation horizontal, selection
  none, homogeneous false, column spacing 12, row spacing 0,
  min children per line 1, max children per line 100.
- `setSections`: `sv.widget.Append(root)` → `FlowBox.Append` (wrapper
  inserted automatically); removal passes the stored root widget to
  `FlowBox.Remove`.
- `window.go` appends `sv.widget` as a `gtk.Widgetter` — no change
  besides type compatibility.

`internal/ui/style.go`:
- Replace `sectionGap` with `sectionColumnGap = 2 * spacingUnit` and
  `sectionRowGap = 0` (layout constants consumed by `newSectionsView`).
- Base stylesheet gains `flowboxchild { padding: 0; margin: 0; }`.

`design-system/tokens.yaml`, `design-system/tokens.html`:
- Spacing description updated: "セクション間 横 2 unit / 縦 0".

### Dependencies

**Internal Dependencies:**
- `internal/ui/sections.go`: container swap (primary change)
- `internal/ui/style.go`: layout constants + stylesheet addition
- `internal/ui/window.go`: consumes `sectionsView.widget` (type only)

**External Dependencies:**
- `github.com/diamondburned/gotk4/pkg/gtk/v4`: already in go.mod;
  provides `FlowBox`.

## Test Scenarios

### Unit Tests
- [ ] TS-1: Base stylesheet contains the `flowboxchild` reset rule
      (padding 0, margin 0).
- [ ] TS-2: Existing `internal/ui` stylesheet/style tests keep passing
      unchanged.
- [ ] TS-3: All other package tests (`go test ./...`) pass.

### Integration Tests
None (GUI behavior is verified manually per test/README.md).

### E2E Tests
**Existing E2E tests**: None
**Run command**: Not detected

### Manual Verification
- [ ] M-1: At width 800, narrow sections (e.g. co2 / pressure /
      disk-usage / uptime) share rows; overflow wraps to the next row in
      config order.
- [ ] M-2: Config reload (edit config.yaml while running) rebuilds the
      flow layout without leaked widgets or crashes.
- [ ] M-3: Section error rendering and per-section `style` overrides
      look unchanged aside from the new layout.

### Edge Cases
- [ ] Edge-1: A section wider than the window (e.g. wide-status at a
      narrow width) occupies its own row and wraps its output text as
      before.
- [ ] Edge-2: Empty sections list (config with no sections) renders an
      empty container without error.

## Success Criteria

- [ ] All functional requirements are implemented and tested
- [ ] `go build ./...`, `go test ./...`, gofmt clean
- [ ] Manual scenarios M-1..M-3 confirmed by the user

## Open Questions

None.

## References

- REQUIREMENTS.md: feature-docs/section-flow-layout/REQUIREMENTS.md
- GTK4 GtkFlowBox: reflowing grid container (gotk4 binding confirmed)
