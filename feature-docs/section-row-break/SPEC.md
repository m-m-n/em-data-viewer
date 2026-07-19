# Feature: Section Row Break

## Overview

Add a forced row-break object to the sections config: an entry
`- type: break` in the `sections` array ends the current row at that
point, so the following section always starts a new row. Within the
segments delimited by breaks, the existing width-based greedy row
packing (`packRows`) is unchanged.

```
a │ b            ← row ended by break
─────────
c │ d │ e │ f    ← normal width-based wrapping resumes
─────────
g │ hij
```

## Objectives

- Let the user cut a row at an exact position regardless of available
  width.
- Keep every existing rendering rule (vdividers within rows, hdividers
  between rows, spacing) unchanged — a break-created row boundary looks
  identical to a natural wrap.
- Keep breaks display-only: no process, no runner entry, no rehydration.

## User Stories

### US1: Forced row break at a chosen position
As a user, I want to insert `- type: break` between two sections in
config.yaml, so that the next section always starts a new row even when
the current row still has room.

**Acceptance Criteria:**
- [ ] AC-1: With sections `a, b, break, c, d, e, f, g, hij` and a window
      wide enough for all of them on one row, the display shows `a │ b`
      on the first row and `c │ d │ e │ f …` starting on the second row.
- [ ] AC-2: Sections after a break wrap by width exactly as before
      within their segment.
- [ ] AC-3: The row boundary created by a break renders the same
      hdivider as a natural width-driven wrap.

## Technical Requirements

### Functional Requirements

- **FR1 — Config schema:** `yamlSection` gains `Type string`
  (`yaml:"type"`). `buildSection` semantics:
  - `type` omitted / empty → a normal section; existing validation
    unchanged.
  - `type: break` → a break entry: all other fields (`name`, `command`,
    `mode`, `interval`, `max_lines`, `style`) are ignored, and the
    section-level validations (command required, mode check, interval
    check) are skipped.
  - any other non-empty `type` value → validation error
    (`unknown type %q`), surfaced through the existing config-error /
    last-good path.
- **FR2 — Config model:** `config.SectionConfig` gains a way to
  represent a break (e.g. a `Type` field with `"break"`). Break entries:
  - are excluded from `checkDuplicateKeys` (multiple breaks are always
    allowed),
  - are excluded from `runnerSections` (no process is ever started),
  - are excluded from `rehydrationsFor`,
  - carry no style entry and render no widgets.
- **FR3 — View model:** the app layer conveys break positions to the ui
  layer (e.g. `ui.SectionViewModel.Break bool` or an equivalent
  mechanism chosen by the planner). Display order still equals config
  order.
- **FR4 — Packing:** row assignment treats breaks as hard segment
  boundaries: sections between breaks form segments; each segment is
  packed with the existing width-based greedy algorithm; the resulting
  rows are concatenated in order. An empty segment (leading break,
  trailing break, consecutive breaks, or a config of breaks only)
  produces no row — no empty rows, no extra dividers.
- **FR5 — Rendering unchanged:** `rebuildRows` / divider rules are
  untouched: N total rows produce N-1 hdividers between them regardless
  of whether a boundary came from width or from a break; vdividers
  within rows are unchanged.

### Non-Functional Requirements

- **NFR1 - Dependencies:** No new external dependencies.
- **NFR2 - Quality gates:** `go build ./...`, `go test ./...` pass and
  gofmt reports no diffs.
- **NFR3 - Testability:** Packing/segmentation logic stays in GUI-free
  code (no gotk4 import) so it is testable with `go test`.

## Implementation Approach

- `internal/config/load.go` / `config.go`: `type` field parsing,
  break-aware validation (FR1), duplicate-key exclusion (FR2).
- `internal/app/reload.go`: skip breaks in `runnerSections` /
  `rehydrationsFor`; include break markers in `sectionViewModels`
  (FR2, FR3).
- `internal/ui/sections.go`: `setSections` skips widget creation for
  break entries but records forced-break positions; `reflow` computes
  the assignment per segment (segment-split + existing `packRows`), then
  concatenates (FR4). `rebuildRows` unchanged (FR5).

## Test Scenarios

### Unit Tests
- [ ] TS-1: `Load` accepts `- type: break` between sections; the break
      appears at the right position in `Config.Sections` (FR1).
- [ ] TS-2: `Load` rejects `type: unknown-value` with an error naming
      the value (FR1).
- [ ] TS-3: A break entry with extra fields (command etc.) loads
      without error and stays a break (FR1).
- [ ] TS-4: Two break entries do not trigger the duplicate-key error
      (FR2).
- [ ] TS-5: `runnerSections` / `rehydrationsFor` outputs contain no
      break entries (FR2).
- [ ] TS-6: Segment packing — given widths and a break index, the
      assignment ends a row at the break even when width remains; the
      segment after the break packs by width as before (FR4, AC-1,
      AC-2).
- [ ] TS-7: Leading / trailing / consecutive breaks and a breaks-only
      config produce no empty rows (FR4).
- [ ] TS-8: Divider counts for a break-created boundary equal those of
      a natural wrap boundary (`rowDividerCounts` over the assignment;
      FR5, AC-3).

### E2E Tests
**Existing E2E tests**: None (GUI verified manually per test/README.md).
- [ ] Manual M-1: With a wide window and a config containing breaks,
      rows split at the break positions and the boundary looks
      identical to a natural wrap.

### Edge Cases
- [ ] Breaks only (no sections): valid config, zero rows (same as zero
      sections).
- [ ] Break as first / last element: no visible effect.

## Success Criteria

- [ ] All functional requirements are implemented and tested
- [ ] All unit test scenarios pass
- [ ] `go build ./...`, `go test ./...`, gofmt clean

## References

- REQUIREMENTS.md: feature-docs/section-row-break/REQUIREMENTS.md
- Related: feature-docs/section-row-packing/ (greedy packing),
  feature-docs/section-dividers/ (divider rendering)
