# Implementation Plan: process-output-widget

## Overview

data-viewer: a GTK4 (gotk4) desktop widget for Ubuntu/GNOME/Wayland that runs
YAML-configured commands and displays their stdout in stacked sections, with
live config reload. See SPEC.md for requirements, DESIGN.md-derived visual
intent is folded into the task plans.

## Technology Stack

- **Language**: Go (latest stable toolchain)
- **GUI**: github.com/diamondburned/gotk4/pkg (GTK4 bindings)
- **Config**: gopkg.in/yaml.v3
- **File watching**: github.com/fsnotify/fsnotify

## Layer Structure

| Layer | Package | Responsibility | May import |
|-------|---------|----------------|------------|
| entry | `cmd/data-viewer` | CLI parsing, app startup | app |
| app | `internal/app` | wiring: config watcher → runner → UI, main-loop dispatch, shutdown | config, runner, markup, ui |
| ui | `internal/ui` | GTK window, frame, sections, drag, menu, banner | markup (for escaping only) |
| logic | `internal/config`, `internal/runner`, `internal/markup` | parsing/validation/watching; process lifecycle; markup validation | stdlib + yaml/fsnotify only |

**Hard rule (NFR3)**: `internal/config`, `internal/runner`, `internal/markup`
must NOT import gotk4 or anything GTK-related. Only `internal/ui`,
`internal/app`, `cmd/` may.

## Bootstrap contract (empty repository)

Tasks run in parallel worktrees on an empty repo. Any task needing to build
creates `go.mod` if absent, EXACTLY as: module path `data-viewer`, current Go
version, dependencies limited to the three libraries above (plus their
transitive sums). Identical content makes merge conflicts trivial; on
conflict the parent side is adopted per the standard protocol.

## Shared Components

| Component | Responsibility | Contract (pre/postcondition) | Used by tasks |
|-----------|----------------|------------------------------|---------------|
| `config.Config` | parsed+validated config | Fields: `Window` (Width, Height int; Style map[string]string) and `Sections []SectionConfig`. Zero sections is valid. | task0001, task0002, task0004, task0005 |
| `config.SectionConfig` | one section definition | Fields: `Name string` (optional), `Command string` (required, run via shell), `Mode` (`interval` \| `stream`), `Interval` duration (required iff interval mode), `MaxLines int` (stream only, default 100), `Style map[string]string`. `Key() string` returns a stable identity derived from Command+Mode+Interval+MaxLines (style excluded) — style-only edits keep the same key. | task0001, task0002, task0005 |
| `config.Load` | load+validate one file | Pre: file path. Post: (`Config` with defaults applied, nil) or (zero, descriptive error). Validation failures listed in SPEC.md are errors. | task0001, task0005 |
| `config.Watcher` | watch config path | Post: emits a notification (channel) after the file changes, debounced, surviving editor rename/replace saves. Closeable. | task0001, task0005 |
| `runner.Event` | output/error event | Struct: `SectionKey string`, `Kind` (`replace` \| `append` \| `error`), `Text string`. `replace`: full new content (interval run). `append`: one line (stream). `error`: human-readable message for in-section display. | task0002, task0005 |
| `runner.Runner` | process lifecycle | `Apply([]SectionConfig)` reconciles: starts added keys, stops removed keys, restarts changed keys (same key = untouched). Events delivered on a single channel owned by the Runner. `Stop()` terminates all children (graceful signal, then kill after timeout) and closes the channel. Never blocks on a slow consumer beyond bounded buffering. | task0002, task0005 |
| `markup.Render` | markup safety | Pre: raw text. Post: (input unchanged, true) when it is valid Pango markup; (escaped plain text, false) otherwise. Never returns text that fails GTK markup parsing. Validation implemented without GTK (well-formed XML with a whitelist of Pango tags). | task0003, task0004 |
| `ui.UI` | GTK widget window | Constructed with a quit callback. Methods (all called on the GTK main loop only): `ApplyWindow(window settings)`, `SetSections(ordered section view models: key, label, style)`, `UpdateSection(key, markup string, isError bool)`, `SetConfigError(message string)` (empty string clears the banner), `Run()` blocking main loop. | task0004, task0005 |

## Conventions

- Errors: logic packages return errors (wrapped with context); only app/ui
  decide presentation. No logging framework — standard library logging to
  stderr for diagnostics.
- All GTK object access happens on the GTK main loop; app layer marshals
  goroutine events onto it via the main-loop scheduling mechanism.
- Test files sit next to sources; table-driven where natural (see
  test/README.md).
- Naming: package names are single lowercase words; exported API minimal.

## Cross-task Design Decisions

### Section identity = content-derived key
Reconciliation and event routing use `SectionConfig.Key()` (style excluded)
so style-only reloads never restart processes (FR3). UI display order is the
config list order, carried in the view models — key is identity, not order.
Affected: task0001, task0002, task0004, task0005.

### Events over callbacks
Runner communicates exclusively through the Event channel; the app layer is
the only consumer and the only place that touches the UI. Keeps logic
packages GTK-free (NFR3). Affected: task0002, task0004, task0005.

### Visual defaults come from design tokens
Default appearance (surface #1e1e1eee, frame #888888 1px, radius 8px,
spacing unit 6px, output monospace 12pt, error #e57373, error banner surface
#3a2222ee, dim labels #9e9e9e) mirrors `design-system/tokens.yaml`. User
config style values override these at runtime. Affected: task0004 (single
place: a defaults table in the ui package).

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| gotk4 first build is very slow (CGo) | High | Medium | Build once early in each worktree; commands already approved |
| Wayland interactive move API quirks | Medium | Medium | Manual verification scenario M-1; fallback documented in task0004 |
| Editor save patterns missed by watcher | Medium | Medium | TS-3 covers write and rename/replace; watcher re-arms on rename |
| Parallel-task go.mod conflicts | Medium | Low | Bootstrap contract above; parent-side adoption |

## Open Questions

- [ ] None.
