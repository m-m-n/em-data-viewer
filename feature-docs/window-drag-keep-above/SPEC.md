# Feature: window-drag-keep-above

## Overview

Fixes whole-window drag-to-move (dragging over section text currently does
not move the window) and adds a keep-above (always on top) option to
data-viewer: config-driven with default ON, toggleable from the right-click
menu with persistence back to the config file, honoring the X11/Wayland
backend capability split.

## Objectives

- Left-drag anywhere on the window (including over rendered text) moves it
- Keep the widget above other windows by default, controllable per config and menu
- Behave correctly on both GDK backends (X11/XWayland and native Wayland)

## User Stories

### US1: Move the widget from anywhere
As a desktop user, I want to drag the widget from any point of its surface,
so that moving it never depends on hitting empty background.

**Acceptance Criteria:**
- [ ] A left-button drag started over a section label, output text, the error banner, or padding initiates a compositor interactive move
- [ ] Right-click anywhere still opens the context menu
- [ ] A plain click (no movement) does nothing

### US2: Keep the widget on top
As a desktop user, I want the widget to stay above other windows by default
and to toggle that from the menu, so that it works as a persistent overlay.

**Acceptance Criteria:**
- [ ] With defaults, on X11/XWayland the window has `_NET_WM_STATE_ABOVE` set from startup
- [ ] The menu shows a check item「最前面に固定」reflecting the current state; toggling applies immediately and updates the config file
- [ ] On native Wayland the item is disabled (insensitive) and the config value is not applied; no crash on either backend

## Technical Requirements

### Functional Requirements
- **FR1:** Whole-window drag: a left-button drag gesture attached at the window level in the CAPTURE phase (so child labels cannot swallow it) starts `begin_move` on drag threshold; right-button context menu behavior is unchanged; plain clicks are not consumed.
- **FR2:** Config key `window.keep_above` (bool, default `true`), applied at startup and on every successful reload.
- **FR3:** Menu check item「最前面に固定」: reflects effective state; toggling applies immediately AND persists the new value to the active config file by updating only the `keep_above` key (round-trip the YAML preserving other content and comments via the yaml.v3 node API; create the key under `window:` when absent). The self-triggered watcher reload (same values) must be harmless.
- **FR4:** Backend split: on X11 (including XWayland) apply/remove keep-above via EWMH `_NET_WM_STATE_ABOVE` on the toplevel's XID; on native Wayland the feature is inert — menu item insensitive with a hint that the WM's "Always on Top" (Alt+Space) can be used — and no X11 calls are made. Backend detection at runtime from the GDK display type.

### Non-Functional Requirements
- **NFR1 - Platform:** Works on Ubuntu/GNOME whether launched natively (Wayland) or with `GDK_BACKEND=x11`; never crashes due to backend mismatch.
- **NFR2 - Testability:** Config write-back logic, keep-above state decisions, and menu-state derivation are GTK-free and unit-tested; X11-specific calls are isolated in one component behind an interface.

## Implementation Approach

### Architecture

Existing layering unchanged (`config` / `runner` / `markup` GTK-free; `ui`
GTK; `app` composition root). Additions:

- `internal/config`: `Window.KeepAbove *bool`-equivalent with default true; `UpdateKeepAbove(path, value)` — comment-preserving single-key YAML update (yaml.v3 node round-trip).
- `internal/ui`: window-level capture-phase drag gesture; menu check item wired to a callback; `SetKeepAbove(enabled, supported)` applying EWMH on X11 or updating menu sensitivity on Wayland. X11 specifics isolated (e.g. `topmost_x11.go`) behind a small interface so the rest of ui stays backend-agnostic.
- `internal/app`: routes config value → ui on startup/reload; routes menu toggle → immediate apply + `config.UpdateKeepAbove` (write-back happens off the GTK main loop, on the reload/serialized worker or a dedicated goroutine feeding it).

### EWMH mechanism (X11/XWayland)

Obtain the toplevel XID via gotk4's gdkx11 surface accessor; send the
`_NET_WM_STATE` client message (add/remove `_NET_WM_STATE_ABOVE`) to the
root window over a direct X connection (pure-Go X client, e.g.
github.com/jezek/xgb) — GTK4 exposes no keep-above API. When the window is
not yet mapped, apply on first map/realize.

### Config Schema addition

```yaml
window:
  keep_above: true   # optional, default true; X11/XWayland only (Wayland: inert)
```

Validation: any non-boolean value is a validation error (treated like the
existing ones per FR9 of process-output-widget).

### Dependencies

**New External Dependency:**
- github.com/jezek/xgb (pure-Go X11 client) — EWMH client messages

### File Structure (expected changes)

```
internal/config/config.go        # Window.KeepAbove field + default
internal/config/load.go          # validation
internal/config/writeback.go     # UpdateKeepAbove (yaml node round-trip)
internal/ui/window.go            # capture-phase whole-window drag
internal/ui/menu.go              # check item + callback wiring
internal/ui/topmost.go           # backend detection + interface
internal/ui/topmost_x11.go       # EWMH implementation (XID + xgb)
internal/app/app.go / reload.go  # wiring: config→ui, toggle→apply+write-back
```

## Test Scenarios

### Unit Tests
- [ ] TS-1: config: `keep_above` absent → true; explicit false → false; non-bool → validation error
- [ ] TS-2: config: `UpdateKeepAbove` flips only the key, preserves other keys AND comments; creates the key when absent; errors cleanly on unwritable path
- [ ] TS-3: app: startup/reload routes keep_above to the ui seam; menu toggle triggers immediate apply and a write-back call with the new value (fakes)
- [ ] TS-4: ui logic: menu check state and sensitivity derivation from (enabled, backend-supported) — pure function
- [ ] TS-5: drag: gesture wiring is in capture phase (constructive/unit where testable; behavior verified manually)

### E2E Tests
None automated (GUI).

### Manual Testing
- [ ] M-1: Drag from over the clock text, the stream text, and padding — window moves in all cases; right-click still opens the menu (Wayland and X11)
- [ ] M-2: `GDK_BACKEND=x11` launch: widget stays above a maximized window by default; menu toggle drops/restores it and the config file's `keep_above` updates (comments intact)
- [ ] M-3: Native Wayland launch: menu item visible but insensitive; no crash; config keep_above untouched by startup

## Error Handling

| Case | Behavior |
|------|----------|
| Config write-back fails (permissions, missing file) | Keep runtime state applied; surface the error in the config-error banner; do not crash |
| X11 connection/XID unavailable on X11 backend | Log and degrade to inert (menu stays enabled but no-op is avoided by marking unsupported) |
| Non-bool keep_above in config | Validation error → last-good config + banner (existing FR9 path) |

## Security Considerations

- The write-back only ever writes to the config path the app was started with (already user-owned); no new input surface. The X connection is local-display only.

## Success Criteria

- [ ] FR1–FR4 implemented; TS-1..TS-5 pass via `go test ./...`
- [ ] M-1..M-3 verified manually on GNOME
- [ ] `gofmt` clean; existing tests unaffected

## Open Questions

- None.

## References

- REQUIREMENTS.md: feature-docs/window-drag-keep-above/REQUIREMENTS.md
- Prior feature: feature-docs/process-output-widget/ (SPEC/DESIGN)
