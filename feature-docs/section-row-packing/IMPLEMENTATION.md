# Implementation Plan: Section Row Packing

## Overview

Replace the FlowBox sections container with a hand-rolled greedy
row-packing layout (vertical box of per-row horizontal boxes). Single-task
feature; the approach was agreed in tmp/section-flow-layout-rework-plan.md.

## Technology Stack

- **Language**: Go (GTK4 via the already-vendored gotk4 bindings — no new
  dependencies; project.license is `none`, no license constraint applies)

## Layer Structure

Unchanged. All changes stay inside the `internal/ui` presentation layer.
The window layer gains only signal wiring that calls back into the
sections view; no other package's contract changes.

## Shared Components

None (single task).

## Conventions

- The row-assignment computation must be a pure function, separated from
  widget code, with table-driven tests (test/README.md convention).
- Layout constants stay in the ui package's style file, mirroring
  design-system/tokens.yaml (tokens unchanged this feature).
- Comment density/style follows the existing ui package files.

## Cross-task Design Decisions

None (single task). Task-level design lives in tasks/task0001.md.

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Reflow churn: content updates change natural widths and re-pack rows visibly | Medium | Low | Rebuild only when the computed assignment differs from the current one |
| Resize notify fires before child allocation updates | Medium | Medium | Defer reflow one main-loop iteration (idle callback); periodic section updates self-heal within seconds |
| Reparenting section roots between row boxes leaks or double-parents widgets | Low | Medium | Explicitly unparent each root from its old row before appending to the new one; drop old row boxes from the container |
| Width unknown before first map (allocation 0) | High | Low | packRows with avail<=0 degrades to one section per row; map-triggered deferred reflow corrects it |

## Open Questions

- [ ] None
