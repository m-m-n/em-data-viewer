# Implementation Plan: Wayland Keep-Above GNOME Shell Extension

## Overview

Bundle a minimal GNOME Shell extension that keeps em-data-viewer windows
always-on-top on Wayland, plus Makefile targets to install/enable and
disable/remove it for the current user.

## Technology Stack

- **GJS (GNOME JavaScript), ESM extension form**: the extension itself.
  Runtime-only dependency on GNOME Shell 50 / Mutter — no new build- or
  vendored dependencies, so no license impact (project stays MIT).
- **GNU Make**: install/uninstall automation, extending the existing
  Makefile.

## Layer Structure

Two independent artifacts, no code-level dependency between them:

- `contrib/gnome-shell-extension/` — extension source (GJS). Never imported
  by the Go application; runs inside GNOME Shell.
- `Makefile` — user-facing automation that copies the extension source and
  drives the `gnome-extensions` CLI.

The Go application is untouched (NFR2).

## Shared Components

| Component | Responsibility | Contract (pre/postcondition) | Used by tasks |
|-----------|----------------|------------------------------|---------------|
| Extension UUID | Identity of the extension | Literal string `keep-above@em-data-viewer.m-m-n.github.io`. It is simultaneously: the `uuid` value in metadata.json, the source directory basename under `contrib/gnome-shell-extension/`, and the install directory basename under `~/.local/share/gnome-shell/extensions/`. Both tasks hardcode this exact string. | task0001, task0002 |
| Source directory path | Location of the extension source in the repo | `contrib/gnome-shell-extension/keep-above@em-data-viewer.m-m-n.github.io/` containing exactly `extension.js` and `metadata.json`. task0001 creates it; task0002 copies its entire contents (file-name agnostic — copy the directory contents, not a hardcoded file list). | task0001, task0002 |
| Target app id | Window-matching key | Literal string `io.github.m_m_n.EmDataViewer` (equals the existing Makefile `APP_ID` variable). task0001 hardcodes it in the extension source. | task0001 |

## Conventions

- Follow the existing Makefile style: tool presence is probed up front with
  a clear failure message (see the existing `resources` target pattern);
  `.PHONY` entries kept current.
- The extension follows GNOME Shell 45+ extension conventions: ESM module,
  a default-exported extension class with enable/disable lifecycle methods,
  and full cleanup on disable (every connected handler disconnected, every
  effect reverted).

## Cross-task Design Decisions

### UUID doubles as directory name

The source directory basename equals the UUID, so the install step is a
plain recursive copy of one directory to
`~/.local/share/gnome-shell/extensions/` with no renaming. Affects
task0001 (creates the directory), task0002 (copies it).

### Unconditional application

The extension applies always-on-top to every matching window, with no
config linkage; enable/disable of the extension is the user's on/off
switch (SPEC.md FR2, user-confirmed decision). Affects task0001 only, but
task0002's uninstall target is the documented "turn it off permanently"
path.

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| App id not yet set at window-created time on Wayland | Medium | Matching silently misses the window | FR2 late-id handling: per-window watch until the id arrives, then decide and disconnect |
| GNOME Shell does not recognize a freshly installed extension until re-login | High (fresh install) | `gnome-extensions enable` fails or has no effect | install target detects the fresh-install case and prints re-login guidance (FR5) |
| Future GNOME major upgrade breaks the extension | Medium | Extension stops loading | Out of scope; shell-version pinned to 50 by decision, updated per release |

## Open Questions

None.
