# Implementation Plan: window-drag-keep-above

## Overview

Fix whole-window drag-to-move and add a keep-above option (config default
ON, menu toggle with config write-back, X11-EWMH / Wayland-inert backend
split) to the existing data-viewer app. See SPEC.md.

## Technology Stack

- Existing: Go, gotk4 (GTK4), gopkg.in/yaml.v3, fsnotify
- **New**: github.com/jezek/xgb (pure-Go X11 client, EWMH client messages)

## Layer Structure

Unchanged from feature-docs/process-output-widget/IMPLEMENTATION.md:
`config`/`runner`/`markup` GTK-free; `ui` GTK; `app` composition root.
X11-specific code lives ONLY in `internal/ui` (isolated file), never in
logic packages. xgb may be imported only by `internal/ui`.

## Shared Components (contracts — tasks compile against these)

| Component | Responsibility | Contract (pre/postcondition) | Used by tasks |
|-----------|----------------|------------------------------|---------------|
| `config.Config.Window.KeepAbove` | keep-above setting | `bool` field; `Load` applies default `true` when the key is absent; non-boolean YAML value is a validation error (existing parse-error path). | task0001, task0002, task0003 |
| `config.UpdateKeepAbove` | persist toggle | `UpdateKeepAbove(path string, value bool) error`. Post: the file at path has `window.keep_above: <value>`; ALL other content including comments preserved (yaml.v3 node round-trip); creates `window:`/`keep_above:` when absent; returns error without partial writes on failure (write temp + rename). | task0001, task0003 |
| `ui.UI` keep-above surface | GTK-side apply | New methods, main-loop-only like the rest: `SetKeepAbove(enabled bool)` (applies EWMH on X11 when the toplevel is realized — deferred to first realize otherwise — and updates the menu check state; no-op apply on Wayland but check state still tracks), `KeepAboveSupported() bool` (true iff X11 backend and XID reachable). Constructor gains a toggle callback `onKeepAboveToggle func(enabled bool)` invoked from the menu item with the NEW desired state. On Wayland the menu item is insensitive with a WM-usage hint, so the callback never fires there. | task0002, task0003 |
| menu item label | 「最前面に固定」 check item | Stock GTK check/toggle menu entry; no custom styling. | task0002 |

## Conventions

Same as the prior feature (errors wrapped, GTK only on main loop via
dispatch, tests beside sources). The app layer routes ALL keep-above flows:
config→ui at startup/reload, menu callback→immediate `SetKeepAbove` +
`UpdateKeepAbove` write-back executed OFF the GTK main loop (on the
serialized reload worker or a dedicated goroutine feeding it), tolerating
the self-triggered watcher reload (idempotent same-value apply).

## Cross-task Design Decisions

### Backend detection lives in ui, policy in app
`KeepAboveSupported()` answers capability (GDK display type + XID). The app
layer decides what to do with it (apply or skip); config stores intent
regardless of backend so switching backends later honors the setting.
Affected: task0002, task0003.

### Write-back is single-key and atomic
Only the `keep_above` scalar changes; file rewritten via temp+rename so a
crash cannot truncate the user's config. Comment preservation via yaml.v3
Node API is required (the config file is user-authored). Affected:
task0001, task0003.

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| yaml.v3 node round-trip loses some formatting (indent/flow styles) | Medium | Low | Acceptance limited to preserving keys+comments; TS-2 locks it |
| EWMH apply before window mapped is ignored | Medium | Medium | Defer to realize/map signal; re-apply on realize |
| xgb connection failure on X11 | Low | Low | Degrade to unsupported (item insensitive), log |
| Capture-phase drag steals menu right-click | Low | Medium | Gesture restricted to button 1 only; M-1 verifies |

## Open Questions

- [ ] None.
