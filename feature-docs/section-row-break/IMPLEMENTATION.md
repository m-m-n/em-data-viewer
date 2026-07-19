# Implementation Plan: Section Row Break

## Overview

Add a `- type: break` entry to the sections config that forces the row
packing to end the current row at that position. Single-task feature: the
change threads one concept (a break marker) through config → app → ui.

## Technology Stack

- **Language**: Go (existing project; no new dependencies — NFR1)
- **UI**: gotk4 (existing); all new logic stays in GUI-free code (NFR3)

## Layer Structure

Unchanged. Existing rules continue to apply:

- `internal/config` must not import gotk4 or anything GTK-related.
- `internal/runner` does not import `internal/config`; the app layer owns
  conversions.
- `internal/ui` receives display data only via view models.

## Shared Components

None — single task; no cross-task contracts.

## Conventions

- The break marker is represented as a variant of the existing section
  entry types (config entry with `type: break`), not a parallel list:
  order information lives solely in the one `sections` slice at every
  layer, so display order remains "slice order" throughout.
- Breaks are display-only at every layer: excluded from duplicate-key
  validation, runner reconciliation, rehydration, style entries, and
  widget creation.

## Cross-task Design Decisions

None beyond the conventions above (single task).

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Break entries leaking into runner/rehydration and spawning empty processes | Low | Medium | Explicit exclusion ACs + unit tests (TS-5) |
| Duplicate-key validation rejecting two breaks | Medium | High | Exclusion from the duplicate check is its own AC + test (TS-4) |
| Reflow regression (assignment caching, divider counts) | Low | Medium | Segment packing kept pure and covered by TS-6..TS-8; rendering code path untouched (FR5) |

## Open Questions

- [ ] None
