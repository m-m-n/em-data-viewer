# Implementation Plan: Application ID / .desktop Filename Match

## Overview

Rename the GTK application ID to `io.github.m_m_n.EmDataViewer` and ship
the .desktop file under that name so Wayland app_id ↔ .desktop matching
resolves the dock icon. No other identifier changes.

## Technology Stack

- Go / GNU Make (existing). No new dependency (license impact: none).

## Layer Structure

Single task; two files (`internal/ui/ui.go`, `Makefile`).

## Shared Components

| Component | Responsibility | Contract (pre/postcondition) | Used by tasks |
|-----------|----------------|------------------------------|---------------|
| App ID string `io.github.m_m_n.EmDataViewer` | Single identifier linking the GTK application ID and the shipped .desktop basename | Precondition: `internal/ui` constructs its GtkApplication with exactly this ID. Postcondition: dpkg ships exactly `usr/share/applications/io.github.m_m_n.EmDataViewer.desktop`; Wayland matching then succeeds. | task0001 |

## Conventions

- The icon-name string `em-data-viewer` (hicolor, gresource, SetIconName,
  Icon=) is frozen — this feature must not touch it (SPEC NFR1).

## Cross-task Design Decisions

### ID chosen as io.github.m_m_n.EmDataViewer

App-specific ID (user decision; net.laser5 excluded for this app). Valid
GApplication ID (4 dot-separated elements). The constant's comment in
`internal/ui` must be updated: the ID is no longer meaningless — it must
equal the .desktop basename for Wayland icon resolution.

### StartupWMClass stays em-data-viewer

X11 sessions match via WM_CLASS (derived from the binary name), not the
application ID — renaming the ID must not drop the X11 path.

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| A test asserts the old ID value | Low | Low | Grep confirmed no test references applicationID today; implementer re-checks in-worktree |

## Open Questions

- [ ] None.
