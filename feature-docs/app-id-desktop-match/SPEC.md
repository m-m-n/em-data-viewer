# Feature: Application ID / .desktop Filename Match

## Overview

On Wayland (GNOME Shell), dock and Alt-Tab icons resolve by matching the
window's app_id against .desktop file names. A GTK4 + GtkApplication app's
app_id is its application ID, so the current `org.dataviewer.Widget` never
matches `em-data-viewer.desktop` and the dock shows a generic gear.
Change the application ID to the app-specific `io.github.m_m_n.EmDataViewer`
and rename the shipped .desktop file to match. The icon name
`em-data-viewer` stays untouched everywhere.

## Objectives

- Dock icon resolves on Wayland via app_id ↔ .desktop filename match
- X11 behavior preserved (StartupWMClass stays)
- No icon-name churn (hicolor / gresource / SetIconName / Icon= unchanged)

## User Stories

### US1: Icon in the Wayland dock
As an app user on a Wayland session, I want the dock to show the app icon,
so that the running app is identifiable.

**Acceptance Criteria:**
- [ ] app_id equals the .desktop basename (`io.github.m_m_n.EmDataViewer`)
- [ ] Dock shows the icon (manual, Wayland)

## Technical Requirements

### Functional Requirements

- **FR1:** The GTK application ID constant in `internal/ui` becomes exactly
  `io.github.m_m_n.EmDataViewer` (valid GApplication ID, 4 elements).
- **FR2:** The dpkg target installs the desktop entry as
  `usr/share/applications/io.github.m_m_n.EmDataViewer.desktop` (mode 644).
  The file CONTENT is byte-identical to the current one — including
  `Icon=em-data-viewer` and `StartupWMClass=em-data-viewer` (X11 matching
  path). `em-data-viewer.desktop` is no longer shipped.

### Non-Functional Requirements

- **NFR1 - Icon-name stability:** the string `em-data-viewer` used for
  hicolor file names, the gresource aliases/base path, `windowIconName` /
  `SetIconName`, and the .desktop `Icon=` value is not changed anywhere.
- **NFR2 - Packaging continuity:** apart from the .desktop filename, the
  deb payload structure is unchanged; upgrading replaces the old
  `em-data-viewer.desktop` automatically (dpkg file-list replacement).

## Implementation Approach

Two one-line-scale changes:

1. `internal/ui/ui.go`: `applicationID` constant value (and its comment,
   which currently claims the ID has no meaning — now it does: it must
   equal the .desktop basename for Wayland icon matching).
2. `Makefile` (dpkg target): the .desktop output filename.

Any test asserting the old constant value is updated alongside.

### File Structure

```
internal/ui/ui.go   # applicationID constant
Makefile            # .desktop filename in dpkg target
```

## Test Scenarios

### Unit Tests

- [ ] `make test` passes (any constant-assertion test updated).

### Integration Tests

- [ ] `make dpkg`: deb contains
  `usr/share/applications/io.github.m_m_n.EmDataViewer.desktop` and does
  NOT contain `em-data-viewer.desktop`.
- [ ] The installed .desktop content equals the previous content
  byte-for-byte.
- [ ] Remaining payload structure (hicolor icons, postinst/postrm)
  unchanged.

### Manual Tests

- [ ] On the Wayland session, after reinstalling the deb and relaunching,
  the dock shows the app icon (gear gone).

## Error Handling

None — static renames only.

## Success Criteria

- [ ] All functional requirements are implemented and tested
- [ ] `make build` / `make test` / `make dpkg` pass
- [ ] Dock icon appears on Wayland (manual)

## Open Questions

None.

## References

- Requirements: feature-docs/app-id-desktop-match/REQUIREMENTS.md
- Diagnosis context: feature-docs/app-icon/ (theme install),
  feature-docs/app-icon-embed/ (embedded resource) — both unchanged by
  this feature.
