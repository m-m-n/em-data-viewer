# Implementation Plan: Output Trailing-Newline Strip

## Overview

Strip all trailing newlines from section display text at render time, uniformly for the replace, append, and error display paths. Single-task feature.

## Technology Stack

- **Language**: Go (existing stack; no new dependencies — NFR1)

## Layer Structure

Unchanged. The change lives entirely in the GTK-free event-translation layer (`internal/app`), which already sits between the runner layer (`internal/runner`) and the UI layer (`internal/ui`). Runner, ring buffer, markup validation, and UI layers are not modified.

## Shared Components

None — single task; no cross-task contracts.

## Conventions

- Stripping is a display concern: it happens where runner events are translated into UI content, never in the runner or buffer layers (SPEC FR2: buffer contents stay raw).
- "Trailing newline" means newline characters at the very end of the display text, including a carriage return immediately preceding a stripped newline; interior line breaks and interior blank lines are untouched (SPEC FR4, Edge case 1).

## Cross-task Design Decisions

None beyond the conventions above (single task).

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Existing tests assert the old trailing-newline-preserving behavior | Low | Low | Task updates such assertions as part of its acceptance criteria |

## Open Questions

- None.
