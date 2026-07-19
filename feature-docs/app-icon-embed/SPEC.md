# Feature: App Icon Binary Embedding

## Overview

Embed the application icon into the binary as a GResource bundle so the
icon resolves in the dock, taskbar and window regardless of whether the
hicolor theme icons are installed. The existing theme-name lookup
(`SetIconName("em-data-viewer")` from the app-icon feature) stays; this
feature makes it resolve from an embedded resource as a fallback-free
source. Build and test entry points move to Makefile targets to guarantee
the resource is generated before compilation.

## Objectives

- Icon visible without installing the deb (development binaries included)
- GTK-native mechanism (GResource + icon-theme resource path), no manual
  EWMH property handling
- No binary files committed to git

## User Stories

### US1: Icon without installation
As a developer or user running the binary directly, I want the app icon to
appear in the dock/taskbar/window, so that the app is identifiable without
installing the deb.

**Acceptance Criteria:**
- [ ] Running a `make build` binary shows the icon with no hicolor icons
  installed
- [ ] Existing `SetIconName` call is unchanged

### US2: Reproducible build via make
As a developer, I want `make build` / `make test` to generate the resource
bundle before compiling, so that go:embed never fails on a fresh checkout.

**Acceptance Criteria:**
- [ ] `make build` and `make test` succeed on a clean tree
- [ ] Missing tools fail the build loudly

## Technical Requirements

### Functional Requirements

- **FR1:** A Makefile `resources` target generates
  `internal/ui/icon.gresource`:
  - renders `build/embed-icons/em-data-viewer-{16,32,48,64,128,256}.png`
    from `assets/icon.svg` via `rsvg-convert -w S -h S` (intermediate PNGs
    stay under git-ignored `build/`);
  - compiles them with `glib-compile-resources` using a git-tracked XML
    manifest `assets/icon.gresource.xml`;
  - resource paths inside the bundle follow the hicolor structure the GTK
    icon theme expects under a base path:
    `/net/laser5/em-data-viewer/icons/{S}x{S}/apps/em-data-viewer.png`;
  - exits non-zero with a clear error naming the missing tool when
    `rsvg-convert` or `glib-compile-resources` is not on PATH (no silent
    skip).
- **FR2:** `make build` depends on `resources` (before `go build`); a new
  `make test` target depends on `resources` and runs the Go test suite.
  These become the workflow's `build_command` / `test_command`. Existing
  `icons` / `dpkg` targets are unchanged.
- **FR3:** `internal/ui` embeds `icon.gresource` via go:embed and, during
  UI construction after GTK initialization and before the window is
  created, registers the resource globally and adds the base icon path
  (`/net/laser5/em-data-viewer/icons`) to the default display's icon theme
  resource path. The existing `SetIconName(windowIconName)` call is
  unchanged. Registration failure (malformed resource data) must not crash
  the app: log and continue (the theme lookup then falls back to installed
  icons, matching current behavior).
- **FR4:** `internal/ui/icon.gresource` is added to the root `.gitignore`.

### Non-Functional Requirements

- **NFR1 - No binaries in git:** intermediate PNGs live under `build/`;
  `icon.gresource` lives in `internal/ui/` but is git-ignored. No binary
  file is committed.
- **NFR2 - Packaging unchanged:** the deb contents (hicolor icons,
  .desktop, postinst/postrm) produced by `make dpkg` are byte-identical in
  structure to the app-icon feature's output (dpkg gains no new steps).

## Implementation Approach

### Architecture

Three surfaces:

1. **Build system (Makefile + assets/icon.gresource.xml):** new
   `resources` target; `build` gains a dependency; new `test` target.
2. **UI wiring (`internal/ui`):** one new file embedding and registering
   the resource; called from UI construction.
3. **Repository config (.gitignore):** one entry.

### Data Flow

```
assets/icon.svg
  → (rsvg-convert)      build/embed-icons/em-data-viewer-{S}.png
  → (glib-compile-resources + assets/icon.gresource.xml)
                        internal/ui/icon.gresource
  → (go:embed)          binary
  → (runtime register)  icon theme resource path
  → SetIconName("em-data-viewer") resolves everywhere
```

### File Structure

```
assets/
├── icon.svg                  # existing (unchanged)
└── icon.gresource.xml        # NEW: gresource manifest (git-tracked text)
build/embed-icons/            # git-ignored intermediate PNGs
internal/ui/
├── icon.gresource            # generated, git-ignored
└── resources.go              # NEW: embed + registration
Makefile                      # resources target, build dep, test target
.gitignore                    # + internal/ui/icon.gresource
```

### Dependencies

**External Dependencies:**
- `glib-compile-resources` (libglib2.0 tooling): build-time only.
- `rsvg-convert` (librsvg): build-time only (already required).
- No new Go module dependency (gotk4 already provides
  `gio.NewResourceFromData`, `gio.ResourcesRegister`,
  `gtk.IconThemeGetForDisplay`, `IconTheme.AddResourcePath`).

## Test Scenarios

### Unit Tests

- [ ] Existing `go test ./...` (via `make test`) passes.
- [ ] The embedded resource data is non-empty and registers without error
  (pure-data checks where feasible without GTK).

### Integration Tests

- [ ] Clean-tree `make build` produces `internal/ui/icon.gresource` and a
  binary; second run is idempotent.
- [ ] `make test` succeeds on a tree where `icon.gresource` was absent.
- [ ] With `glib-compile-resources` masked from PATH, `make resources`
  exits non-zero naming the tool; same for `rsvg-convert`.
- [ ] `git status` stays clean of generated artifacts after full build.
- [ ] `gresource list internal/ui/icon.gresource` lists the six PNG entries
  under `/net/laser5/em-data-viewer/icons/`.

### E2E Tests
**Existing E2E tests**: None
**Run command**: `make dpkg`
- [ ] `make dpkg` output unchanged in structure (NFR2).

### Edge Cases

- [ ] Registration failure path: corrupt resource data logs and continues
  (no crash) — code-review-verified (cannot be unit-tested without GTK).

### Manual Tests

- [ ] With hicolor icons NOT installed, launching the binary shows the app
  icon in dock/taskbar/window (X11).

## Security Considerations

- **Input Validation:** none required — build-time generation from
  repository-controlled sources only.

## Error Handling

| Condition | Behavior |
|-----------|----------|
| `rsvg-convert` missing | `make resources` (thus build/test) fails, names the tool |
| `glib-compile-resources` missing | same |
| Resource registration fails at runtime | log and continue; theme lookup falls back to installed icons |

## Success Criteria

- [ ] All functional requirements are implemented and tested
- [ ] `make build` / `make test` pass on a clean tree
- [ ] Icon appears without deb installation (manual)
- [ ] No binary file added to git
- [ ] deb packaging output structure unchanged

## Open Questions

None.

## References

- Requirements: feature-docs/app-icon-embed/REQUIREMENTS.md
- Prior feature: feature-docs/app-icon/ (theme-install path; unchanged)
