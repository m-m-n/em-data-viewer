# Feature: Section Row Packing (greedy flow layout)

## Overview

Replace the GtkFlowBox sections container (section-flow-layout) with a
hand-rolled greedy row-packing layout. GtkFlowBox aligns children into
uniform columns (per-column max width — confirmed in GTK's
`fit_aligned_item_requests`), which stretches every column to the widest
member. The new layout packs each row greedily by natural width, like CSS
flex-wrap: sections flow left→right in config order and wrap when the next
section does not fit.

## Objectives

- True natural-width packing: a row holds as many sections as fit; no
  column alignment, no stretching to the widest section.
- Layout follows window width changes and content-driven width changes.
- Keep every existing section behavior unchanged.

## User Stories

### US1: Dense multi-column display
As a user, I want each row to hold as many sections as actually fit at
their natural widths, so that a wide window shows a dense, flex-wrap-like
layout instead of a sparse column grid.

**Acceptance Criteria:**
- [ ] Rows contain variable numbers of sections depending on their natural
      widths (e.g. width 6 with items 1,2,2,2,2,2,… packs as
      `[1 2 2] [2 2 2] …`).
- [ ] Visual order equals config order (left→right, top→bottom).
- [ ] Resizing the window re-packs the rows.

## Technical Requirements

### Functional Requirements

- **FR1:** The sections container is a vertical `gtk.Box` (spacing =
  row gap) holding one horizontal `gtk.Box` per row (spacing = column
  gap). Section root widgets (label + output vertical box, unchanged) are
  packed into row boxes greedily in config order.
- **FR2:** Row assignment is computed by a pure function
  `packRows(widths []int, gap, avail int) [][]int`: greedy accumulation —
  wrap when `used + gap + width > avail`; at least one section per row;
  `avail <= 0` yields one section per row; empty input yields no rows.
- **FR3:** Widths are the sections' natural horizontal sizes (GTK widget
  measurement, natural value, for_size -1). A section wider than the
  available width occupies its own row; its output label wraps as before.
- **FR4:** Reflow (re-measure → packRows → rebuild only when the
  assignment changed) is triggered by: (a) `setSections`, (b) every
  `updateSection`, (c) the window's `default-width` property notify,
  deferred one main-loop iteration, (d) the window's `map` signal,
  deferred one main-loop iteration. An unchanged assignment must not
  rebuild widgets.
- **FR5:** Spacing stays 12px (2 units) within a row and 0 between rows;
  design tokens are unchanged. The FlowBox and the `flowboxchild`
  stylesheet reset are removed (constant `maxSectionsPerLine` too).
- **FR6:** Existing behaviors are preserved: `setSections` rebuild
  semantics, `updateSection` (markup update, error style), per-section
  style classes and `styleEntries`, empty-sections case.

### Non-Functional Requirements

- **NFR1 - Dependencies:** No new external dependencies.
- **NFR2 - Quality gates:** `go build ./...`, `go test ./...` pass and
  gofmt reports no diffs.

## Implementation Approach

Per tmp/section-flow-layout-rework-plan.md (agreed plan):

- `internal/ui/sections.go` — `sectionsView.widget` becomes a vertical
  `*gtk.Box`; add `rows []*gtk.Box`, a cached last assignment, `packRows`,
  `reflow` (measure → pack → compare → rebuild), and row rebuild logic
  (unparent roots from old rows, drop old row boxes, build new ones).
  Remove FlowBox construction and `maxSectionsPerLine`.
- `internal/ui/style.go` — remove `sectionsContainerCSS` and its
  `buildStylesheet` call; keep `sectionColumnGap` / `sectionRowGap`.
- `internal/ui/style_test.go` — remove the flowboxchild and
  maxSectionsPerLine tests; keep the gap-constants test.
- `internal/ui/sections_test.go` — new: table-driven `packRows` tests.
- `internal/ui/window.go` — wire `default-width` notify and `map` to a
  deferred reflow (idle callback).

### Dependencies

**Internal Dependencies:**
- `internal/ui/sections.go`, `internal/ui/style.go`,
  `internal/ui/window.go` (wiring), tests.

**External Dependencies:**
- gotk4 GTK4 bindings (already in go.mod): `Widget.Measure`,
  `glib.IdleAdd`, `Object.NotifyProperty`, `Widget.FirstChild` —
  availability confirmed.

## Test Scenarios

### Unit Tests
- [ ] TS-1: `packRows` packs greedily by natural width (multiple table
      cases incl. the width-6 example shape).
- [ ] TS-2: `packRows` boundary cases — exact fit stays in row; +1
      wraps; gap counts toward used width.
- [ ] TS-3: `packRows` — oversize item gets its own row; `avail <= 0`
      yields one per row; empty input yields no rows.
- [ ] TS-4: Stylesheet no longer contains a `flowboxchild` rule.
- [ ] TS-5: Existing `internal/ui` tests keep passing (gap constants
      test retained).
- [ ] TS-6: Full suite `go test ./...` passes.

### Integration Tests
None (GUI behavior is verified manually per test/README.md).

### E2E Tests
**Existing E2E tests**: None
**Run command**: Not detected

### Manual Verification
- [ ] M-1: With the sensor config, rows hold multiple natural-width
      sections with no column alignment; order follows config order.
- [ ] M-2: Resizing the window re-packs rows accordingly.
- [ ] M-3: Config reload rebuilds the layout; section errors and style
      overrides render unchanged.

### Edge Cases
- [ ] E-1: A section wider than the window occupies its own row and its
      output wraps.
- [ ] E-2: Zero sections render an empty window without error.

## Success Criteria

- [ ] All functional requirements are implemented and tested
- [ ] `go build ./...`, `go test ./...`, gofmt clean
- [ ] Manual scenarios M-1..M-3 confirmed by the user

## Open Questions

None.

## References

- REQUIREMENTS.md: feature-docs/section-row-packing/REQUIREMENTS.md
- Agreed implementation plan: tmp/section-flow-layout-rework-plan.md
- Prior feature (superseded layout): feature-docs/section-flow-layout/
