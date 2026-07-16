# Feature: process-output-widget

## Overview

data-viewer is a desktop widget application for Ubuntu / GNOME / Wayland,
built with Go and gotk4 (GTK4). It runs shell commands defined in a YAML
config file and displays their stdout in a single undecorated window with
multiple stacked sections. The config file is watched and all changes
(styles and command definitions) are applied live without restarting.

## Objectives

- Display the stdout of user-defined commands as an always-running desktop widget
- Apply config file changes in real time (styles and command lifecycle)
- Support text decoration (colors etc.) via Pango markup

## User Stories

### US1: View command output on the desktop
As a desktop user, I want the output of my commands displayed in a widget,
so that I can monitor information at a glance.

**Acceptance Criteria:**
- [ ] Starting `data-viewer` shows a window with one section per configured command
- [ ] Interval-mode sections replace their content on every run
- [ ] Stream-mode sections append output lines as they arrive

### US2: Edit the config and see it applied immediately
As a desktop user, I want config edits reflected without restart,
so that I can tune the widget quickly.

**Acceptance Criteria:**
- [ ] Style-only changes update the display without restarting processes
- [ ] Adding/removing/changing a command starts/stops/restarts only the affected processes
- [ ] An unparsable config keeps the previous settings and shows an error

### US3: Move and quit the widget
As a desktop user, I want to drag the frame-only window and quit from a
right-click menu, so that I can manage the widget without a titlebar.

**Acceptance Criteria:**
- [ ] Dragging the window background/frame moves the window
- [ ] Right-click opens a context menu containing Quit
- [ ] Quit terminates all child processes and exits

## Technical Requirements

### Functional Requirements
- **FR1:** Undecorated window (no server-side decorations; frame only, drawn by the app) that can be moved by dragging its background/frame. On GTK4/Wayland use `gtk.Window.SetDecorated(false)` and start an interactive move from a drag gesture (`gdk.Toplevel.BeginMove` equivalent in gotk4).
- **FR2:** Load YAML config from `$XDG_CONFIG_HOME/data-viewer/config.yaml` (fallback `~/.config/data-viewer/config.yaml`); a CLI positional/flag argument overrides the path.
- **FR3:** Watch the config file and hot-reload ALL settings: styles apply in place; command definition changes reconcile processes (start added, stop removed, restart changed). Reload applies within ~1 second of saving. Handle editor save patterns (rename/replace) correctly.
- **FR4:** Interval mode: run the command every configured interval, capture full stdout, replace the section content with it.
- **FR5:** Stream mode: start the command once, read stdout line-by-line, append lines to the section. Keep at most a configured number of lines (default 100), dropping the oldest. If the process exits, show an error in the section (no automatic restart within this feature).
- **FR6:** Render output as Pango markup in GTK labels. If the markup is invalid, fall back to rendering the same text as plain (escaped) text.
- **FR7:** Multiple sections stacked vertically in one window, in config order. A config with zero sections shows an empty window (frame only).
- **FR8:** Right-click anywhere on the window opens a context menu with at least "終了" (Quit). Quit stops all child processes (SIGTERM, then SIGKILL after a timeout) and exits.
- **FR9:** Error handling: config parse failure → keep last good config, show the error (in-window banner); command spawn/exit failure → show the error inside that section while other sections keep running.

### Non-Functional Requirements
- **NFR1 - Platform:** Runs on Ubuntu / GNOME / Wayland using GTK4 via gotk4. No layer-shell; the widget is a normal window.
- **NFR2 - Performance:** Config reload visible within ~1s. Stream sections are memory-bounded via the line limit. UI updates happen on the GTK main loop (`glib.IdleAdd`), never from goroutines directly.
- **NFR3 - Testability:** Config parsing, section reconciliation logic, markup validation/fallback, and stream line-buffer logic live in packages that do not import gotk4, covered by `go test ./...`.

## Implementation Approach

### Architecture

```
┌─────────────────────────────────────────────┐
│  ui (gotk4): window, frame, sections,       │
│  drag-move, context menu, error banner      │
├─────────────────────────────────────────────┤
│  app: wires config watcher → runner →  UI   │
│  updates via glib.IdleAdd                   │
├─────────────────────────────────────────────┤
│  config: YAML schema, load, validate,       │
│  file watch, last-good-config keeping       │
├─────────────────────────────────────────────┤
│  runner: process lifecycle (interval /      │
│  stream), stdout capture, line ring buffer, │
│  reconcile on config change                 │
├─────────────────────────────────────────────┤
│  markup: Pango markup validation +          │
│  plain-text fallback (escaping)             │
└─────────────────────────────────────────────┘
```

`config`, `runner`, `markup` must not import gotk4 (NFR3).

### Data Flow

```
config.yaml ──watch──▶ config ──reconcile──▶ runner ──output events──▶ app ──IdleAdd──▶ ui
                          │                                                        ▲
                          └──────────────style/window settings────────────────────┘
```

### Config Schema (YAML)

```yaml
window:
  width: 400          # optional, default 400
  height: 600         # optional, default 600
  style:               # optional; frame/background appearance (CSS-like values)
    background: "#1e1e1eee"
    border_color: "#888888"
    border_width: 1
sections:
  - name: clock        # optional display name
    command: "date '+%H:%M:%S'"   # required; run via sh -c
    mode: interval     # interval | stream (required)
    interval: 1s       # required when mode: interval (Go duration string)
    max_lines: 100     # stream only; optional, default 100
    style:             # optional
      font: "monospace 12"
      color: "#ffffff"
```

Validation errors (unknown mode, missing command, missing interval for
interval mode, unparsable duration) are treated like parse errors (FR9).

### Dependencies

**External Dependencies:**
- github.com/diamondburned/gotk4/pkg — GTK4 bindings (UI)
- gopkg.in/yaml.v3 — config parsing
- github.com/fsnotify/fsnotify — config file watching

### File Structure

```
cmd/data-viewer/main.go      # entry point, CLI arg parsing
internal/config/             # schema, load, validate, watcher (no gotk4)
internal/runner/             # process lifecycle, interval/stream, ring buffer (no gotk4)
internal/markup/             # Pango markup validation + escape fallback (no gotk4)
internal/app/                # wiring config/runner/ui
internal/ui/                 # gotk4 window, sections, menu, drag
```

## Test Scenarios

### Unit Tests
- [ ] TS-1: config: valid YAML loads with defaults applied (window size, max_lines)
- [ ] TS-2: config: invalid YAML / validation errors are reported and last-good config is retained by the holder type
- [ ] TS-3: config: watcher detects write and rename/replace save patterns (t.TempDir)
- [ ] TS-4: runner: reconcile computes correct start/stop/restart sets for added/removed/changed/unchanged sections
- [ ] TS-5: runner: interval mode captures full stdout per run; stream mode appends lines and honors max_lines ring buffer
- [ ] TS-6: runner: command failure / abnormal exit surfaces an error event for that section only
- [ ] TS-7: markup: invalid Pango markup falls back to escaped plain text; valid markup passes through

### Integration Tests
- None (GUI verified manually).

### E2E Tests
**Existing E2E tests**: None
**Run command**: Not detected
- [ ] Manual scenario 1: launch, drag-move, right-click quit on GNOME/Wayland
- [ ] Manual scenario 2: edit config while running → styles and command set update live

### Edge Cases
- [ ] Zero sections in config → empty framed window, no crash
- [ ] Stream command floods output → display bounded by max_lines
- [ ] Config file deleted while running → keep last good config, show error
- [ ] Stream process exits → error shown in that section, others unaffected

## Security Considerations

- **Command Execution:** Commands from the config run via `sh -c` with the user's privileges. The config file is user-owned by design; no privilege escalation surface.
- **Input Validation:** Config is validated on load; invalid configs never replace the last good one.
- **Markup Injection:** Process output is rendered as Pango markup by design (user opted in). Invalid markup is escaped and rendered plain, so malformed output cannot crash rendering.

## Error Handling

| Case | Behavior |
|------|----------|
| Config unparsable/invalid (startup) | Start with empty config + error banner |
| Config unparsable/invalid (reload) | Keep last good config + error banner |
| Command spawn failure / abnormal exit | Error text inside that section; others continue |
| Invalid Pango markup in output | Render as escaped plain text |

## Success Criteria

- [ ] All functional requirements (FR1–FR9) are implemented
- [ ] All unit test scenarios (TS-1..TS-7) pass via `go test ./...`
- [ ] Manual scenarios verified on Ubuntu / GNOME / Wayland
- [ ] `gofmt` clean

## Open Questions

- None.

## References

- REQUIREMENTS.md: feature-docs/process-output-widget/REQUIREMENTS.md
- gotk4: https://github.com/diamondburned/gotk4
