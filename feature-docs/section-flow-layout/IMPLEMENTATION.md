# Implementation Plan: Section Flow Layout

## Overview

Replace the vertical one-column sections container with a horizontally
reflowing container (GTK4 FlowBox): sections flow left→right in config
order and wrap when the row is full. Single-task feature.

## Technology Stack

- **Language**: Go (GTK4 via the already-vendored gotk4 bindings — no new
  dependencies; project.license is `none`, no license constraint applies)

## Layer Structure

Unchanged. All changes stay inside the `internal/ui` presentation layer
plus the design-token documents (`design-system/`). No other package's
contract changes: the sections container is consumed by the window layer
only as a generic widget.

## Shared Components

None (single task).

## Conventions

- Layout constants (spacing multiples of the 6px unit) live as named
  constants in the ui package's style file, mirroring
  `design-system/tokens.yaml` — token documents and constants must be
  updated together.
- Comment density/style follows the existing ui package files.

## Cross-task Design Decisions

None (single task). Task-level design lives in tasks/task0001.md.

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Flow container's auto-inserted child wrapper adds theme padding/selection visuals | Medium | Low | Selection disabled + stylesheet reset for the wrapper node; manual check M-3 |
| Removal during config reload leaks wrapper widgets | Low | Medium | Removal passes the stored section root; GTK resolves the wrapper. Manual check M-2 |
| Row height stretches when a tall (multi-line) section shares a row | Medium | Low | Accepted behavior per SPEC (natural-width cells); visible in M-1 |

## Open Questions

- [ ] None
