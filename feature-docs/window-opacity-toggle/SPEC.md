# Feature: Window Opacity Toggle

## Overview

The widget window's transparency currently comes from the alpha channel of a
single configured color, `window.style.background` (default `#1e1e1eee`).
This feature adds a second, independently configured background pattern and a
`Ctrl+T` key binding that toggles the window between the two at runtime, so a
window pinned above everything else can be made unobtrusive without editing
the config file.

## Objectives

- Let the config file carry two background patterns instead of one.
- Toggle between them with `Ctrl+T` while the window has keyboard focus.
- Keep every existing config file working unchanged, with an unchanged
  startup appearance.

## User Stories

### US1: Dim the widget when it is in the way

As a desktop user who keeps the widget above other windows, I want to press
`Ctrl+T` to switch the window to a more transparent background, so that it
stops obscuring what is behind it.

**Acceptance Criteria:**
- [ ] Pressing `Ctrl+T` with the window focused switches the background to
      the alternate pattern immediately.
- [ ] Pressing `Ctrl+T` again switches it back to the primary pattern.
- [ ] The window's position, width and height do not change as a result.

### US2: Configure both levels myself

As a user who already styles the widget through `config.yaml`, I want the
second pattern to be a normal style key, so that I set it the same way I set
the first one.

**Acceptance Criteria:**
- [ ] `window.style.background_alt` is accepted with the same value syntax as
      `window.style.background`.
- [ ] A config file without `background_alt` still loads and runs, using the
      built-in default for the alternate pattern.

## Technical Requirements

### Functional Requirements

- **FR1:** The window style schema gains the key `background_alt`, a CSS color
  string with the same syntax and validation as the existing `background`
  key. It is user-overridable via `window.style.background_alt` in the YAML
  config, and merged through the same window-style merge path as every other
  window style key (unknown keys stay ignored).
- **FR2:** `background_alt` has the built-in default `#1e1e1e99` — the same
  hue as the default `background` (`#1e1e1eee`) with a lower alpha. This
  default applies whenever the config omits the key.
- **FR3:** `design-system/tokens.yaml` gains the role-based token
  `color.surface-alt` with value `#1e1e1e99` and a description stating it is
  the alternate (more transparent) widget window background selected by the
  runtime toggle. The runtime window-style defaults table and this token must
  agree, so the existing token/runtime drift-guard test covers the new key.
- **FR4:** The frame tracks which of the two patterns is active as
  process-local state. The active pattern selects which style value is
  emitted as the window's CSS `background-color`; every other style value
  (border, radius, error colors, section styles) is unaffected by the toggle.
- **FR5:** At process start the primary pattern (`background`) is always
  active. The active pattern is never read from or written to the config file.
- **FR6:** Pressing `Ctrl+T` while the window has keyboard focus flips the
  active pattern and re-applies the stylesheet live, without recreating the
  window or its widgets.
- **FR7:** The key handler claims only `Ctrl+T`. Any other key event — and
  `T` with a different modifier combination, including `Alt`/`Super` held
  together with `Ctrl` — is left unclaimed so existing input behavior
  (drag-to-move, the right-click context menu, blocked scroll gestures) is
  unchanged.
- **FR8:** When the config file is reloaded at runtime, the active pattern
  index survives the reload, and the reloaded values take effect immediately
  for whichever pattern is active.

### Non-Functional Requirements

- **NFR1 - Compatibility:** Existing config files that do not mention
  `background_alt` keep working, and the appearance immediately after startup
  is identical to the current release.
- **NFR2 - Testability:** The toggle's state transition and the selection of
  the effective background value are pure, GTK-free functions covered by
  `go test ./...`. The `config` package continues not to import gotk4 or any
  GTK-related package.
- **NFR3 - Usability:** The switch takes effect on the next frame with no
  window resize, reflow, or content reload; only the background color changes.

## Implementation Approach

### Affected Components

```
config.yaml.example           # document window.style.background_alt
design-system/tokens.yaml     # color.surface-alt token (FR3)
internal/ui/style.go          # WindowStyleDefaults + effective-background selection
internal/ui/window.go         # frame state, Ctrl+T controller, restyle path
```

The config layer needs no schema change: `window.style` is already a free-form
`map[string]string` (`config.Window.Style`), so `background_alt` reaches the ui
layer through the existing path. Filtering of unknown keys happens in
`ui.MergeStyle` against `WindowStyleDefaults`, so adding the default entry is
what makes the key effective.

### Data Flow

```
config.yaml → config.Load → config.Window.Style
            → app layer → ui.WindowSettings.Style
            → frame.applySettings → MergeStyle(WindowStyleDefaults, overrides)
            → frame.windowStyle (holds BOTH patterns)

Ctrl+T → key controller → flip active pattern → frame.restyle()
       → buildStylesheet uses the active pattern's value as background-color
```

Because both patterns live in the merged style map, the toggle only changes
which key is read; no re-merge and no config access is involved.

### Dependencies

**Internal:** `internal/ui` (style + frame), `design-system/tokens.yaml`
(drift guard in `internal/ui/style_test.go`).

**External:** none beyond the existing gotk4 GTK4/GDK bindings already in
`go.mod` (`gtk.EventControllerKey`, `gdk.ControlMask`).

## Test Scenarios

### Unit Tests
- [ ] `MergeStyle` accepts a `background_alt` override and falls back to the
      default when the key is absent.
- [ ] The window stylesheet emits the primary background as
      `background-color` when the primary pattern is active, and the alternate
      value when the alternate pattern is active.
- [ ] The pure toggle transition flips primary → alternate → primary.
- [ ] The pure key-match predicate accepts `Ctrl+T` and rejects bare `T`,
      `Ctrl+Shift+T` variants that carry `Alt`/`Super`, and unrelated keyvals.
- [ ] The token/runtime drift guard passes with `color.surface-alt` present.

### Manual Verification
- [ ] Start the app with the example config: the window looks exactly as
      before.
- [ ] Focus the window, press `Ctrl+T`: the background becomes visibly more
      transparent; geometry does not change.
- [ ] Press `Ctrl+T` again: the original background returns.
- [ ] Set a distinctive `window.style.background_alt` in the config, restart,
      and confirm `Ctrl+T` selects that value.
- [ ] With the alternate pattern active, edit the config file and save:
      the reload keeps the alternate pattern active.

### Edge Cases
- [ ] `background_alt` present but with an invalid/unparsable color value: it
      is passed through to CSS exactly like an invalid `background` value is
      today — no new validation, no crash.
- [ ] `background_alt` set to the same value as `background`: the toggle is a
      visual no-op and must not error.
- [ ] `Ctrl+T` pressed before the window has been presented or realized: no
      crash; the state flips and applies when the stylesheet is next applied.
- [ ] Repeated rapid `Ctrl+T` presses: each press flips the state exactly
      once, with no accumulated drift.

## Assumptions

Recorded because this feature was specified in batch mode with no user
dialogue available (Codex CLI was also unavailable for consultation):

- **A1:** The second pattern is expressed as a full color string
  (`background_alt`), not as a separate numeric opacity field. Rationale:
  transparency in this app is already carried by the alpha channel of
  `window.style.background`, and the task requires the second pattern to live
  where the first one lives.
- **A2:** The toggle state is process-local and never written back to the
  config file, unlike `window.keep_above`.
- **A3:** Startup always uses the primary pattern.
- **A4:** Only the window background toggles. The error banner background
  (`error-background`), the frame border and section styles do not.
- **A5:** The default alternate value is `#1e1e1e99` (default hue, alpha
  `0x99` ≈ 60%).
- **A6:** No context-menu entry is added for the toggle; `Ctrl+T` is the only
  affordance.
- **A7:** The binding is a window-level key controller and therefore requires
  the widget window to hold keyboard focus; no global/compositor-level hotkey
  is registered.

## Success Criteria

- [ ] All functional requirements are implemented and covered by tests.
- [ ] `go test ./...` passes.
- [ ] Existing configs without `background_alt` run unchanged.
- [ ] Manual verification steps above pass on the developer's desktop.

## Open Questions

- [ ] None.

## References

- Task: https://www.notion.so/3a83509ec8ee80608676fbf7fe0b519b
- REQUIREMENTS.md: `feature-docs/window-opacity-toggle/REQUIREMENTS.md`
- Existing style implementation: `internal/ui/style.go`
- Design tokens: `design-system/tokens.yaml`
