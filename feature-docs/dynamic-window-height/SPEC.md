# Feature: Dynamic Window Height

## Overview

The window height is currently fixed by the `window.height` config field
(default 600). This feature removes that field and instead sizes the window
dynamically from its content: the height follows the natural height of the
banner plus all sections, capped at the monitor height.

## Objectives

- Window height follows content, growing and shrinking as section output changes
- Remove `window.height` from the config schema; a `height` key in existing
  YAML is silently ignored
- Keep `window.width` behavior unchanged

## User Stories

### US1: Content-fitted window
As a user, I want the window height to match its content, so that there is no
wasted empty space and no clipped output.

**Acceptance Criteria:**
- [ ] The window height equals the content's natural height (banner + sections)
- [ ] When section output grows or shrinks, the window height follows
- [ ] When a config reload adds/removes sections, the window height follows

### US2: Height stays on screen
As a user, I want the window to never grow taller than my monitor, so that it
stays usable.

**Acceptance Criteria:**
- [ ] When the content's natural height exceeds the monitor height, the window
      height is capped at the monitor height and the overflow is clipped

### US3: Config without height
As a user, I want my existing config with a `height` entry to keep working,
so that the upgrade is painless.

**Acceptance Criteria:**
- [ ] A `window.height` key in config.yaml is silently ignored — no error,
      no warning
- [ ] `config.yaml.example` no longer contains a `height` line

## Technical Requirements

### Functional Requirements
- **FR1:** The window height is determined dynamically from the content's
  natural height (banner + sections view). `window.height` no longer controls
  the window size.
- **FR2:** The window height tracks content changes at runtime: section output
  updates, section add/remove via config reload, and style changes all cause
  the height to re-fit.
- **FR3:** The window height is capped at the monitor's height. Content beyond
  the cap is clipped.
- **FR4:** `window.height` is removed from the config schema (`config.Window`
  struct, `yamlWindow` struct, `DefaultWindowHeight` constant). Because config
  parsing uses non-strict `yaml.Unmarshal`, a `height` key present in YAML is
  silently ignored. `config.yaml.example` drops its `height` line.

### Non-Functional Requirements
- **NFR1 - Testability:** The config package remains GTK-free and testable
  with `go test ./...`.
- **NFR2 - Existing behavior:** The zero-sections minimum frame height
  (currently 48px via `minFrameHeight`) and all `window.width` behavior are
  preserved.

## Implementation Approach

### Affected Components

- `internal/config/config.go` — remove `Window.Height`, `DefaultWindowHeight`
- `internal/config/load.go` — remove `yamlWindow.Height` and its default/copy
  logic
- `internal/config/load_test.go` — update height-related cases (a `height`
  key in input YAML must load without error)
- `internal/ui/window.go` — `applySettings` no longer sets a fixed default
  height; the window sizes to content and caps at monitor height
- `internal/ui/ui.go` / `internal/ui/style.go` — update any `Height` plumbing
  in `WindowSettings`
- `config.yaml.example` — drop the `height` line

### Sizing Behavior (GTK4)

- The window's default height must not be forced; height derives from the
  natural size of the root box (banner + sections view).
- Shrinking must work too: when content gets shorter, the window gets shorter.
  (Note: a GTK4 resizable window does not shrink below its current size on its
  own; the implementation must actively re-fit, e.g. by making the window
  non-user-resizable in height or by resetting the default height when content
  height changes.)
- The monitor-height cap applies to the window's realized monitor. Overflow
  beyond the cap is clipped (no scrollbar requirement).
- Width behavior is unchanged: `SetDefaultSize(width, …)` semantics for width
  remain.

## Test Scenarios

### Unit Tests
- [ ] Config load: YAML with `window.height` present loads without error and
      the field has no effect
- [ ] Config load: YAML without `window.height` loads as before
- [ ] Config struct no longer exposes a Height field (compile-time)

### Integration Tests
None (GUI behavior is verified manually per test/README.md).

### E2E Tests
**Existing E2E tests**: None
**Run command**: Not detected

### Edge Cases
- [ ] Zero sections: window shows at the minimum frame height (48px)
- [ ] Content taller than the monitor: height caps at monitor height,
      overflow clipped
- [ ] Config reload removing sections: window shrinks accordingly

## Error Handling

No new error paths. A `height` key in YAML is not an error.

## Success Criteria

- [ ] All functional requirements are implemented and tested
- [ ] `go build ./...`, `go test ./...`, `gofmt -l -w .` pass
- [ ] Manual verification: window height follows content, caps at monitor
      height, and `height` in config is ignored silently

## References

- REQUIREMENTS.md: feature-docs/dynamic-window-height/REQUIREMENTS.md
