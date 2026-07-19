# Implementation Plan: App Icon

## Overview

Add an application icon: a git-tracked SVG source rendered to PNGs at build
time, packaged into the deb (hicolor theme + .desktop + maintainer scripts),
and referenced at runtime as the window icon name.

## Technology Stack

- **Language**: Go (existing), GNU Make (existing build system)
- **Build tool**: rsvg-convert (librsvg) — build-time only, NOT a Go or deb
  runtime dependency. No new Go dependency is introduced (license impact:
  none; project.license MIT unaffected).

## Layer Structure

Two independent surfaces; no shared code path:

- **Build/packaging layer** (Makefile, assets/): renders and packages the
  icon. Owned by task0001.
- **UI layer** (internal/ui): sets the window icon name at construction.
  Owned by task0002.

## Shared Components

| Component | Responsibility | Contract (pre/postcondition) | Used by tasks |
|-----------|----------------|------------------------------|---------------|
| Icon name string `em-data-viewer` | Single identifier linking theme files, .desktop and runtime lookup | Precondition: hicolor files are named `em-data-viewer.png` / `em-data-viewer.svg` and the .desktop sets `Icon=em-data-viewer` (task0001). Postcondition: the runtime window icon name is set to exactly this string (task0002); icon resolution then succeeds on an installed system. | task0001, task0002 |

## Conventions

- Generated artifacts go under `build/` only (git-ignored); nothing binary
  is committed (NFR1).
- Makefile style follows the existing file: tab-indented recipes, variables
  at top, `.PHONY` declarations maintained.

## Cross-task Design Decisions

### Naming: plain name, unchanged application ID

The GTK application ID (`org.dataviewer.Widget`) stays as is. Desktop
integration uses the plain name `em-data-viewer` for the .desktop file
name, `Icon=` value, hicolor file names, and the runtime icon name;
taskbar association relies on `StartupWMClass=em-data-viewer` matching the
process WM_CLASS derived from the binary name. Affects both tasks (the
shared contract above).

### Fail-closed icon rendering

Missing rsvg-convert fails the build with a clear message — no fallback
tool, no silent skip (FR2). Rationale: never ship a deb with missing icons.
Affects task0001 only, recorded here because it is a deliberate deviation
from the emterm precedent (which warns and skips).

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| WM_CLASS of the running process differs from `em-data-viewer`, breaking taskbar icon matching | Low | Medium | Manual verification item in VERIFICATION.md; StartupWMClass value can be corrected in the Makefile-generated .desktop without code changes |
| rsvg-convert output differs across librsvg versions | Low | Low | Icon uses simple shapes only (rects, line); no filters or text |

## Open Questions

- [ ] None.
