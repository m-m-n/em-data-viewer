# Feature: App Icon

## Overview

Provide an application icon for em-data-viewer. The icon source is a single
SVG under git control; build time renders PNGs from it (git-ignored build
artifacts), and the dpkg target ships hicolor theme icons, a .desktop entry,
and maintainer scripts so the icon appears in app launchers, the taskbar,
and the window.

## Objectives

- Single SVG source of truth (`assets/icon.svg`), no binaries in git
- Build-time PNG rendering via rsvg-convert
- Full desktop integration: hicolor icons, .desktop entry, runtime icon-name

## User Stories

### US1: Icon visible after install
As an app user, I want the app to show an icon in the app list, taskbar and
window after installing the deb, so that I can identify it visually.

**Acceptance Criteria:**
- [ ] The deb contains hicolor icons, a .desktop file and maintainer scripts
- [ ] The running app sets its window icon-name

### US2: Reproducible icon build
As a developer, I want PNGs rendered from the SVG at build time, so that no
binary icon files need to be committed.

**Acceptance Criteria:**
- [ ] PNGs are generated under `build/` (git-ignored)
- [ ] The build fails loudly when rsvg-convert is missing

## Technical Requirements

### Functional Requirements

- **FR1:** Icon source `assets/icon.svg` is git-tracked. Its visual design
  is decided in the design step (DESIGN.md); the file is authored during
  implementation from that decision.
- **FR2:** A Makefile `icons` target renders
  `build/icons/em-data-viewer-{48,64,128,256}.png` from `assets/icon.svg`
  using `rsvg-convert -w S -h S`. When `rsvg-convert` is not on PATH the
  target exits non-zero with a clear error message (no silent skip, no
  fallback tool). The `dpkg` target depends on `icons`.
- **FR3:** The `dpkg` target installs into the package root:
  - `usr/share/icons/hicolor/{48x48,64x64,128x128,256x256}/apps/em-data-viewer.png`
  - `usr/share/icons/hicolor/scalable/apps/em-data-viewer.svg` (the SVG
    copied verbatim)
- **FR4:** The `dpkg` target installs
  `usr/share/applications/em-data-viewer.desktop` (mode 644) with exactly:
  ```ini
  [Desktop Entry]
  Name=eM Data Viewer
  Comment=Data viewer application
  Exec=em-data-viewer
  Icon=em-data-viewer
  Terminal=false
  Type=Application
  Categories=Utility;
  StartupWMClass=em-data-viewer
  ```
  The GTK application ID (`org.dataviewer.Widget`) stays unchanged; taskbar
  matching relies on `StartupWMClass` = the process WM_CLASS (derived from
  the binary name on X11).
- **FR5:** At startup the UI sets the top-level window's icon name to
  `em-data-viewer` (gotk4: `gtk.Window.SetIconName("em-data-viewer")`), so
  the icon resolves from the installed hicolor theme.
- **FR6:** The deb gains `DEBIAN/postinst` and `DEBIAN/postrm` maintainer
  scripts (mode 755) that run, best-effort (`|| true`, guarded by
  `command -v`):
  - `gtk-update-icon-cache -f -t /usr/share/icons/hicolor`
  - `update-desktop-database /usr/share/applications`

### Non-Functional Requirements

- **NFR1 - No binaries in git:** All rendered PNGs live under `build/`
  (already git-ignored). No new binary file is committed.
- **NFR2 - Testability:** Logic that does not require GTK stays out of the
  gotk4-importing packages, per the project's testing conventions. The icon
  work is build-script- and UI-wiring-only; no new logic package is
  expected.

## Implementation Approach

### Architecture

Two independent surfaces change:

1. **Build system (Makefile):** new `icons` target; `dpkg` grows icon /
   desktop / maintainer-script packaging steps.
2. **UI wiring (`internal/ui`):** one call to set the window icon name at
   window construction.

The SVG itself (`assets/icon.svg`) is a new asset authored per DESIGN.md.

### File Structure

```
assets/
└── icon.svg                 # icon source (git-tracked)
build/                       # git-ignored
└── icons/
    └── em-data-viewer-{48,64,128,256}.png
Makefile                     # icons target + dpkg packaging additions
internal/ui/window.go        # SetIconName call (exact location per plan)
```

### Dependencies

**External Dependencies:**
- `rsvg-convert` (librsvg): build-time only, required for `icons`/`dpkg`
  targets. Not a runtime dependency of the deb.

## Test Scenarios

### Unit Tests

- [ ] Existing `go test ./...` passes unchanged (no logic packages change).

### Integration Tests

- [ ] `make dpkg` produces a deb; `dpkg-deb -c` lists the four PNGs, the
  scalable SVG, and `em-data-viewer.desktop`.
- [ ] `dpkg-deb -e` shows `postinst` and `postrm` present and executable.
- [ ] With `rsvg-convert` masked from PATH, `make icons` exits non-zero.

### E2E Tests
**Existing E2E tests**: None
**Run command**: Not detected
- [ ] Manual: after installing the deb, the app list shows "eM Data Viewer"
  with the icon; the running window/taskbar shows the icon (GUI is verified
  manually per project convention).

### Edge Cases

- [ ] `make icons` re-run is idempotent (overwrites existing PNGs).
- [ ] `git status` stays clean of generated PNGs after a full build.

## Security Considerations

- **Input Validation:** None required — build-time file generation from a
  repository-controlled SVG only.
- **Maintainer scripts:** postinst/postrm run only well-known system
  utilities guarded by `command -v`, with `|| true`.

## Error Handling

| Condition | Behavior |
|-----------|----------|
| `rsvg-convert` missing | `make icons` (and thus `make dpkg`) fails with a clear message |
| Icon cache tools missing on target system | postinst/postrm ignore and succeed |

## Success Criteria

- [ ] All functional requirements are implemented and tested
- [ ] `go test ./...` passes
- [ ] `make dpkg` output contains icons, .desktop and maintainer scripts
- [ ] No binary file added to git

## Open Questions

None.

## References

- Requirements: feature-docs/app-icon/REQUIREMENTS.md
- Icon handling precedent: emterm (`scripts/generate-icons.sh`,
  `scripts/build-dpkg.sh`) — pattern reference only, not copied.
