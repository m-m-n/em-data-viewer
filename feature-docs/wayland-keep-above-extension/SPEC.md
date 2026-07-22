# Feature: Wayland Keep-Above GNOME Shell Extension

## Overview

A minimal GNOME Shell extension, bundled in this repository, that keeps
em-data-viewer windows always-on-top on native Wayland sessions. Wayland has
no client-side always-on-top protocol; the extension calls the Mutter
compositor API `Meta.Window.make_above()` (the same operation as the
Alt+Space "Always on Top" menu item) for windows whose app id matches
`io.github.m_m_n.EmDataViewer`. Makefile targets install/enable and
disable/remove the extension for the current user.

## Objectives

- Keep em-data-viewer above other windows from login onward on Wayland
- Ship the extension inside this repository (`contrib/gnome-shell-extension/`)
- Provide `make install-extension` / `make uninstall-extension`

## User Stories

### US1: Always-on-top from login
As a user who autostarts em-data-viewer, I want its window kept above other
windows from the moment I log in, so that the viewer stays visible without
manual Alt+Space interaction.

**Acceptance Criteria:**
- [ ] With the extension enabled, a newly created em-data-viewer window is
      set above automatically
- [ ] Windows already open when the extension is enabled are also set above

### US2: Simple install/uninstall
As a user, I want single make targets to install and remove the extension,
so that setup does not require manual file copying.

**Acceptance Criteria:**
- [ ] `make install-extension` installs and enables the extension
- [ ] `make uninstall-extension` disables and removes it

## Technical Requirements

### Functional Requirements

- **FR1:** Extension source lives at
  `contrib/gnome-shell-extension/keep-above@em-data-viewer.m-m-n.github.io/`
  containing `extension.js` and `metadata.json`. `metadata.json` declares
  `uuid: keep-above@em-data-viewer.m-m-n.github.io`,
  `shell-version: ["50"]`, and a name/description identifying it as the
  em-data-viewer keep-above helper. The extension is written in ESM form
  (GNOME Shell 45+ `export default class ... extends Extension`).
- **FR2:** The extension applies `make_above()` to every window whose app
  id is `io.github.m_m_n.EmDataViewer` (matched via
  `Meta.Window.get_gtk_application_id()` or `get_wm_class()`). A window
  matches when EITHER identifier equals the target id; a non-matching value
  in one identifier does not finalize a non-match while the other
  identifier is still unset. It connects to the display's `window-created`
  signal for new windows. Because the app id may not be set yet at
  `window-created` time on Wayland, the extension must handle late-arriving
  ids (e.g. re-check on the window's `notify::wm-class` /
  `notify::gtk-application-id` and disconnect the per-window handlers once
  decided). Before setting a matched window above, the extension verifies
  the window's process identity: the compositor-reported PID must resolve
  to an executable named `em-data-viewer` (via the proc filesystem; a
  kernel-appended ` (deleted)` marker on the resolved path is ignored);
  windows failing this verification (no PID, unreadable process info,
  different executable) are not set above. It does not read the
  application's config file; the extension's enable/disable state is the
  on/off switch.
- **FR3:** On `enable()`, the extension scans all existing windows
  (`global.display.list_all_windows()` or equivalent) and applies FR2
  matching/`make_above()` to them, then starts watching `window-created`.
- **FR4:** On `disable()`, the extension disconnects every signal handler it
  connected (display-level and per-window) and calls `unmake_above()` on the
  windows it set above (tracked at apply time and still present), so
  disabling reverts exactly the extension's effect — no revert misses for
  windows matched via either identifier, and no reverting of windows the
  extension never touched.
- **FR5:** Makefile target `install-extension`: copies the extension
  directory to `~/.local/share/gnome-shell/extensions/{uuid}/` (replacing
  any previous contents), then runs `gnome-extensions enable {uuid}`. When
  the running Shell does not yet know the extension (fresh install), the
  target prints a message that re-login is required on Wayland. The target
  fails fast with a clear message when `gnome-extensions` is not on PATH.
- **FR6:** Makefile target `uninstall-extension`: runs
  `gnome-extensions disable {uuid}` (failure tolerated), then removes
  `~/.local/share/gnome-shell/extensions/{uuid}/`.

### Non-Functional Requirements

- **NFR1 - Compatibility:** Targets GNOME Shell 50 on Wayland. Uses only
  APIs present in GNOME 50's Mutter (`Meta.Window.make_above` /
  `unmake_above`). No external dependencies; the extension is
  self-contained GJS.
- **NFR2 - Isolation:** No changes to the Go application's code or
  behavior. Existing `make build` / `go test ./...` results are unaffected.
  The extension (GJS) is excluded from automated tests per the project's
  testing policy; its behavior is verified manually.

## Implementation Approach

### File Structure

```
contrib/
└── gnome-shell-extension/
    └── keep-above@em-data-viewer.m-m-n.github.io/
        ├── extension.js      # ~30-line ESM extension
        └── metadata.json     # uuid, shell-version ["50"], name, description
Makefile                      # + install-extension / uninstall-extension targets
```

### Extension Behavior

```
enable()
  → for each existing window: maybeApply(window)
  → connect display "window-created" → maybeApply(window)

maybeApply(window)
  → either id matches io.github.m_m_n.EmDataViewer?
      yes: process identity verified (PID → executable name) → make_above()
      no, but an id may still arrive: watch notify::wm-class /
           notify::gtk-application-id, re-check on change, disconnect the
           per-window watch once decided (match, both-set non-match, or
           window unmanaged)

disable()
  → disconnect display + per-window handlers
  → unmake_above() on the tracked windows this extension set above
```

### Dependencies

**Internal Dependencies:**
- Makefile: existing `APP_ID` variable (`io.github.m_m_n.EmDataViewer`)
  documents the id the extension hardcodes; the extension source itself
  hardcodes both the app id and its uuid.

**External Dependencies:**
- GNOME Shell 50 / Mutter (Meta, Shell ESM modules) — runtime only
- `gnome-extensions` CLI — used by the make targets

## Test Scenarios

### Unit Tests

None. The extension is GJS (excluded from automated tests, NFR2); the make
targets are shell operations verified manually. Existing Go tests must keep
passing.

### Manual Verification

- [ ] M-1: `make install-extension` copies the extension and enables it
      (fresh install prints the re-login guidance)
- [ ] M-2: After re-login, launching em-data-viewer results in an
      always-on-top window (Alt+Space shows "Always on Top" checked)
- [ ] M-3: With em-data-viewer already running, enabling the extension sets
      the existing window above
- [ ] M-4: Disabling the extension reverts the window (no longer above)
- [ ] M-5: `make uninstall-extension` disables and removes the extension
      directory
- [ ] M-6: Non-matching windows (other apps) are never set above
- [ ] M-7: `go test ./...` passes; `make build` unaffected
- [ ] M-8: A window presenting the target app id from a different owning
      executable (spoofed id) is not set above

### Edge Cases

- [ ] Window whose app id arrives after `window-created` is still matched
      (FR2 late-id handling)
- [ ] Window whose first-arriving identifier is a non-matching value while
      the other identifier later arrives with the target id is still
      matched (FR2 either-id semantics)
- [ ] A running instance whose on-disk binary was replaced or deleted
      (proc path carries the ` (deleted)` marker) still passes process
      verification
- [ ] `install-extension` treats "install directory absent but Shell still
      knows the uuid" as a known install: an enable failure there fails
      the target (fresh detection is based on Shell recognition only)
- [ ] When the Shell recognition query itself fails (no running Shell /
      D-Bus session, e.g. headless or SSH install), `install-extension`
      treats the install as fresh: files are copied, the enable failure is
      tolerated, and the re-login guidance is printed (enabling is
      impossible in that state regardless, so failing the target would
      only break headless installs)
- [ ] `make install-extension` with the extension already installed
      re-copies files without error (idempotent)
- [ ] `make uninstall-extension` when the extension is not installed
      completes without error
- [ ] User manually unsets "Always on Top" via Alt+Space: the extension
      does not re-apply it for that existing window (re-apply happens only
      on window creation / enable scan)

## Success Criteria

- [ ] FR1–FR6 implemented
- [ ] M-1..M-7 verified manually on GNOME Shell 50 / Wayland
- [ ] `go test ./...` and `make build` unaffected

## Open Questions

None.

## References

- Research report: tmp/wayland-keep-above-research-2026-07-21.md
- Mutter API: https://mutter.gnome.org/meta/method.Window.make_above.html
- Prior spec (X11 keep-above, Wayland inert):
  feature-docs/window-drag-keep-above/SPEC.md (FR4)
