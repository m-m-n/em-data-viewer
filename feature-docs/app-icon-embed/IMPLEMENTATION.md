# Implementation Plan: App Icon Binary Embedding

## Overview

Embed the app icon into the binary as a GResource bundle generated at build
time, registered on the icon theme's resource path at startup so the
existing theme-name lookup resolves without installed icons.

## Technology Stack

- **Language**: Go (existing), GNU Make (existing)
- **Build tools**: rsvg-convert (existing build-time dep),
  glib-compile-resources (NEW build-time dep; part of GLib tooling, not a
  runtime or Go dependency — license impact: none, MIT project unaffected;
  no new Go module is introduced)

## Layer Structure

Single task; layers touched:

- **Build/packaging (Makefile, assets/)**: resource pipeline.
- **UI (internal/ui)**: embed + registration at UI construction.
- **Repo config (.gitignore)**: one entry.

## Shared Components

| Component | Responsibility | Contract (pre/postcondition) | Used by tasks |
|-----------|----------------|------------------------------|---------------|
| Resource base path `/net/laser5/em-data-viewer/icons` | Single base path linking the gresource manifest layout and the runtime icon-theme registration | Precondition: the bundle places PNGs at `{base}/{S}x{S}/apps/em-data-viewer.png` (hicolor-like structure). Postcondition: runtime adds exactly `{base}` to the icon theme resource path; `em-data-viewer` then resolves from the bundle. | task0001 |
| Generated bundle `internal/ui/icon.gresource` | go:embed source of the resource bytes | Precondition: `make resources` has run (Makefile guarantees order for build/test). Postcondition: file exists and is git-ignored. | task0001 |

## Conventions

- Generated binaries stay out of git: intermediates under `build/`, the
  bundle git-ignored in place (go:embed cannot reach outside the package
  directory — recorded decision, see below).
- Makefile style follows the existing file (tab recipes, top variables,
  `.PHONY` maintained).

## Cross-task Design Decisions

### Bundle generated inside internal/ui, git-ignored

go:embed only reads files within the embedding package's directory tree,
so the bundle cannot live under `build/`. It is generated at
`internal/ui/icon.gresource` and git-ignored (user-approved). Consequence:
a fresh checkout cannot compile with bare `go build ./...` until
`make resources` has run once — hence the make-entry decision below.

### Build/test entry points move to make

`make build` (extended) and `make test` (new) both depend on the
`resources` target, guaranteeing generation order. workflow.yaml
`build_command` / `test_command` are `make build` / `make test`
(user-approved). `icons` / `dpkg` targets stay unchanged (NFR2).

### Fail-closed tooling, non-fatal runtime

Missing rsvg-convert or glib-compile-resources fails the build naming the
tool (consistent with app-icon's FR2). At runtime, a resource registration
failure logs and continues — the app must keep working with theme-installed
icons (SPEC FR3).

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| GTK icon cache misses resource icons added after theme lookup init | Low | Medium | Register before any window/widget creation (task plan pins the call site); manual verification M-1 |
| gotk4 resource-registration API misuse crashes at startup | Low | High | Registration wrapped with error handling (log-and-continue per SPEC FR3); `make test` exercises package init |

## Open Questions

- [ ] None.
