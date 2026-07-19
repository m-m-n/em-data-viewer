# Implementation Plan: Dynamic Window Height

## Overview

Remove the fixed `window.height` config setting and size the window
dynamically from its content, capped at the monitor height. Single-task
feature: the height plumbing (config → app → ui) is one tightly-coupled
contract change and is removed atomically in one task.

## Technology Stack

- **Language**: Go (existing)
- **UI toolkit**: GTK4 via gotk4 (existing)
- No new dependencies. License constraint (MIT) unaffected.

## Layer Structure

Unchanged. The existing dependency direction holds:
`cmd` → `internal/app` → `internal/ui` / `internal/config`.
The config package stays GTK-free (NFR1).

## Shared Components

Single task — no cross-task contracts.

## Conventions

- Existing project conventions apply (comment style referencing FR/AC IDs,
  table-driven tests, test files next to sources).
- The config schema change must be silent: an obsolete `height` key in YAML
  is neither an error nor a warning (non-strict YAML decoding already
  provides this once the field is removed from the schema structs).

## Cross-task Design Decisions

Single task — the sizing-behavior design lives in tasks/task0001.md.

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| GTK4 window does not shrink when content shrinks (toolkit keeps the larger size) | High | High | Task design requires an active re-fit on content change; acceptance criteria include the shrink case explicitly |
| Monitor geometry unavailable before the window is realized | Medium | Low | Apply the cap from the realized window's monitor; before realize, no cap is needed |

## Open Questions

None.
