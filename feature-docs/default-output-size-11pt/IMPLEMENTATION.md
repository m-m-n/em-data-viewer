# Implementation Plan: Default Output Font Size 11pt

## Overview

Change the default output-text font size from 12pt to 11pt in the section
style defaults, and keep the design token SSOT (design-system/tokens.yaml)
in sync with the code default.

## Technology Stack

- **Language**: Go (existing project; no new dependencies)

## Layer Structure

No change. The edit is confined to the UI layer's style defaults and the
project-level design token file.

## Shared Components

| Component | Responsibility | Contract (pre/postcondition) | Used by tasks |
|-----------|----------------|------------------------------|---------------|
| (none — single task) | | | |

## Conventions

- The section style default table and the design token file must state the
  same output font size (token SSOT stays truthful).

## Cross-task Design Decisions

None — the feature is a single task.

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Content-derived window height shifts slightly | High | Low | Height is computed dynamically; no action needed |

## Open Questions

- (none)
