# Feature: Whole-Window Opacity Toggle

## Overview

`Ctrl+T` currently switches only the window's background color between two
configured values, so command output text, section labels and the frame
border stay fully opaque when the window is "dimmed". This feature replaces
that mechanism with a whole-window opacity toggle: two opacity levels are
configured in `config.yaml`, and `Ctrl+T` switches the GTK widget opacity of
the entire window — background, border, labels and output text together.

This is a retake of the `window-opacity-toggle` feature. Its background-only
mechanism (`window.style.background_alt`, `color.surface-alt`,
`ApplyBackgroundPattern`) is removed as part of this work.

## Objectives

- Dim the entire window, text included, rather than only the background.
- Keep both levels configurable from `config.yaml`, as the current
  background pair already is.
- Keep every existing config file working, with an unchanged startup
  appearance.

## User Stories

### US1: Get the widget out of the way without losing it

As a desktop user who keeps the widget above other windows, I want `Ctrl+T`
to fade the whole widget, so that I can read the window behind it through
the widget instead of having opaque text sitting on top of it.

**Acceptance Criteria:**
- [ ] Pressing `Ctrl+T` with the window focused fades the entire window —
      background, border, section labels and command output — to the
      alternate opacity.
- [ ] Pressing `Ctrl+T` again restores the primary opacity.
- [ ] The window's position, width and height do not change as a result, and
      section commands keep running across the toggle.

### US2: Configure both levels myself

As a user who already styles the widget through `config.yaml`, I want both
opacity levels to be normal style keys, so that I set them the same way I
set every other window style value.

**Acceptance Criteria:**
- [ ] `window.style.opacity` and `window.style.opacity_alt` are accepted and
      take effect.
- [ ] A config file with neither key loads and runs, using the built-in
      defaults.
- [ ] An unparsable or out-of-range value falls back to that key's default
      without turning the config into an error state.

## Technical Requirements

### Functional Requirements

- **FR1:** The window style schema gains the keys `opacity` and
  `opacity_alt`, user-overridable via `window.style.opacity` and
  `window.style.opacity_alt` in the YAML config, and merged through the same
  window-style merge path (`ui.MergeStyle` against `WindowStyleDefaults`) as
  every other window style key. Unknown keys stay ignored, so no change to
  the `config` package schema is needed — `window.style` is already a
  free-form `map[string]string`.
- **FR2:** `opacity` has the built-in default `"1.0"` and `opacity_alt` has
  the built-in default `"0.45"`. These defaults apply whenever the config
  omits the key.
- **FR3:** `design-system/tokens.yaml` gains an `opacity` group with the
  role-based tokens `opacity.window` (`"1.0"`) and `opacity.window-alt`
  (`"0.45"`), each with a description stating which runtime pattern it backs.
  The existing token/runtime drift-guard test is extended to cover both keys,
  so the defaults table and the tokens file cannot drift apart.
- **FR4:** Interpreting a configured opacity value is a pure, GTK-free
  function: the string is parsed as a decimal number and accepted when it
  lies within the inclusive range `0.0`–`1.0`. An unparsable value, or a
  value outside that range, falls back to that key's built-in default. No
  error banner is raised and the rest of the config is unaffected.
- **FR5:** The frame tracks which of the two patterns is active as
  process-local state. The active pattern selects which opacity value is
  applied to the window widget (GTK widget opacity, which composites the
  window's entire rendering). No other style value — background color,
  border, radius, error colors, section styles — is affected by the toggle.
- **FR6:** At process start the primary pattern (`opacity`) is always active.
  The active pattern is never read from or written to the config file.
- **FR7:** Pressing `Ctrl+T` while the window has keyboard focus flips the
  active pattern and applies the new opacity immediately, without recreating
  the window or its widgets, and without re-measuring, re-fitting or
  re-packing the layout.
- **FR8:** The key handler claims only `Ctrl+T`. Any other key event — and
  `T` with a different modifier combination, including `Alt`/`Super` held
  together with `Ctrl` — is left unclaimed so existing input behavior
  (drag-to-move, the right-click context menu, blocked scroll gestures) is
  unchanged. This is the existing `isToggleBackgroundKeyEvent` match rule,
  carried over unchanged in behavior.
- **FR9:** When the config file is reloaded at runtime, the active pattern
  survives the reload, and the reloaded opacity values take effect
  immediately for whichever pattern is active.
- **FR10:** The background-only toggle mechanism introduced by the previous
  feature is removed: `WindowStyleDefaults["background_alt"]`, the
  `ApplyBackgroundPattern` function and its call sites, the
  `color.surface-alt` design token, the `background_alt` documentation in
  `config.yaml.example`, and the tests that cover them. The primary
  `background` value (`#1e1e1eee`) is unchanged. A config file that still
  sets `background_alt` continues to load — the key simply becomes one of
  the ignored unknown keys.

### Non-Functional Requirements

- **NFR1 - Compatibility:** Existing config files that mention neither
  `opacity` nor `opacity_alt` keep working, and the appearance immediately
  after startup is identical to the current release (primary opacity `1.0`
  leaves the window's composited result unchanged).
- **NFR2 - Testability:** Opacity parsing, the fallback rule and the
  selection of the effective opacity value are pure, GTK-free functions
  covered by `go test ./...`. The state flip is testable independently of
  the GTK-dependent apply call. The `config` package continues not to import
  gotk4 or any GTK-related package.
- **NFR3 - Usability:** The switch takes effect on the next frame with no
  window resize, reflow or content reload; only the composited opacity
  changes.

## Implementation Approach

### Affected Components

```
config.yaml.example           # document opacity / opacity_alt, drop background_alt
design-system/tokens.yaml     # add opacity.window / opacity.window-alt, drop color.surface-alt (FR3, FR10)
internal/ui/style.go          # defaults table, opacity parsing + selection, drop ApplyBackgroundPattern (FR1, FR2, FR4, FR10)
internal/ui/style_test.go     # drift guard extension, parsing/selection tests, drop background_alt tests
internal/ui/window.go         # frame state, apply-opacity path, Ctrl+T controller retarget (FR5-FR9)
internal/ui/window_test.go    # toggle state and apply-path tests
README.md                     # feature list wording, if it mentions the background toggle
```

### Data Flow

```
config.yaml → config.Load → config.Window.Style
            → app layer → ui.WindowSettings.Style
            → frame.applySettings → MergeStyle(WindowStyleDefaults, overrides)
            → frame.windowStyle (holds BOTH opacity values as strings)
            → frame.applyOpacity() → win.SetOpacity(WindowOpacity(style, active))

Ctrl+T → key controller → flip active pattern → frame.applyOpacity()
```

The key difference from the previous implementation: the toggle no longer
goes through the CSS stylesheet path. `restyle()` rebuilds the stylesheet
from `f.windowStyle` directly again (the `backgroundPatternStyle()`
indirection is removed), and the opacity is applied as a widget property on
the window, which GTK composites over the window's entire rendering
including text.

### Key Functions (shape, not code)

- A pure parse-with-fallback function in `internal/ui/style.go` that turns a
  configured string into a `float64` opacity, applying the range check and
  the per-key default (FR4).
- A pure selection function that, given the merged window style map and the
  active-pattern boolean, returns the effective opacity (FR5). It reads
  `opacity` when the primary pattern is active and `opacity_alt` when the
  alternate one is, and falls back per FR4 for each key independently.
- A frame method that applies the currently effective opacity to the window
  widget, called both from `applySettings` (config reload, FR9) and from the
  `Ctrl+T` handler (FR7).
- The state flip stays a separate pure boolean negation so the sequence is
  testable without GTK (NFR2).

### Naming

Identifiers carried over from the previous feature that name the *background*
pattern (`altPatternActive`, `flipBackgroundPattern`,
`toggleBackgroundPattern`, `backgroundPatternActive`,
`isToggleBackgroundKeyEvent`, `attachBackgroundToggle`) should be renamed to
name the *opacity* pattern instead, so the code does not describe a mechanism
that no longer exists. Their behavior — in particular the key match rule —
is unchanged.

### Dependencies

**Internal Dependencies:**
- `internal/ui` window style merge path (`MergeStyle`, `WindowStyleDefaults`)
- `internal/ui` frame construction and key controller wiring

**External Dependencies:**
- gotk4 (GTK4), already a project dependency. Whole-window opacity uses the
  widget opacity property; no new dependency is introduced.

## Test Scenarios

### Unit Tests
- [ ] Merging a config with no opacity keys yields the defaults `"1.0"` and
      `"0.45"`.
- [ ] Overriding `opacity` and `opacity_alt` individually is reflected in the
      merged map and does not affect the other key.
- [ ] Parsing accepts well-formed decimals in range and returns them as
      `float64`.
- [ ] Parsing rejects unparsable values (e.g. `"abc"`, `""`) and returns the
      supplied default.
- [ ] Parsing rejects out-of-range values (e.g. `"-0.1"`, `"1.5"`) and
      returns the supplied default.
- [ ] Effective-opacity selection returns the `opacity` value when the
      primary pattern is active and the `opacity_alt` value when the
      alternate pattern is active.
- [ ] The active-pattern flip alternates false → true → false and never
      mutates the merged style map.
- [ ] The token/runtime drift guard fails if `opacity.window` /
      `opacity.window-alt` in `design-system/tokens.yaml` disagree with the
      defaults table.
- [ ] `config.yaml.example` documents `opacity` and `opacity_alt`, and the
      documented values round-trip through the merge path.
- [ ] The `Ctrl+T` match rule still accepts `t`/`T` with Control only and
      still rejects Control+Alt, Control+Super and unrelated keys.
- [ ] `background_alt` is no longer present in the defaults table, and a
      style map containing it merges without error (ignored unknown key).

### Integration Tests
- [ ] A config reload while the alternate pattern is active keeps the
      alternate pattern active and picks up the reloaded `opacity_alt` value.
- [ ] The stylesheet built from the merged style is independent of the active
      pattern (the toggle no longer touches CSS).

### E2E Tests
**Existing E2E tests**: None detected (no `e2e-tests/`, `tests/e2e/`,
`playwright.config.*` or `cypress.config.*` in the repository).
**Run command**: Not detected — E2E verification is manual.
- [ ] Manual: with the widget running, `Ctrl+T` visibly fades the whole
      window including command output text, and a second press restores it.

### Edge Cases
- [ ] `opacity_alt: "0.0"` — accepted (boundary); the window becomes fully
      transparent and a further `Ctrl+T` restores it. Documented as the
      user's choice, not guarded against.
- [ ] `opacity: "1.0"` — accepted (boundary); identical to the default.
- [ ] Both keys set to the same value — the toggle becomes a no-op visually;
      no error.
- [ ] `Ctrl+T` pressed before the window is realized/presented — mutates
      state and applies the property without panicking.

## Security Considerations

Not applicable. This feature reads two additional values from an existing
local config file and sets a GTK widget property. No network, no
authentication, no persistence, no user-supplied data reaches a shell or a
query.

## Error Handling

There are no error codes. The only failure mode is an unusable configured
value, handled by the FR4 fallback: the built-in default is used and the
application continues normally.

## Assumptions

Recorded per batch mode — decided without user confirmation, from the task
description and the existing code. Codex CLI is not installed in this
environment, so the batch consultation loop was skipped and these are the
author's decisions.

1. **Mechanism**: whole-window dimming is implemented with the GTK widget
   opacity property on the window, not by lowering the alpha of every color
   in the stylesheet. Widget opacity composites the entire rendered window,
   which is what "text included" requires; per-color alpha would need every
   style key to gain a second value.
2. **Config keys**: `window.style.opacity` and `window.style.opacity_alt`,
   following the existing `background` / `background_alt` naming shape, since
   the current mechanism is config-file driven (the task says to keep the
   existing configuration style).
3. **Default values**: primary `1.0`, alternate `0.45`. Primary is `1.0` so
   the startup appearance is byte-identical to the current release; `0.45`
   is chosen as clearly readable-through without making the widget
   unreadable.
4. **Removal of `background_alt`**: keeping both mechanisms would dim the
   background twice (color alpha plus widget opacity) while dimming text
   once, which is incoherent. The repository has no release tags, so the key
   has not shipped in a tagged release; existing config files that set it
   still load because unknown style keys are ignored.
5. **Invalid values**: fall back silently to the default rather than raising
   the config error banner, matching how the rest of the style layer treats
   unusable values.
6. **Toggle state is not persisted** — carried over from the previous
   feature's decision.
7. **Key binding stays `Ctrl+T`** and is not made configurable — the task's
   acceptance criteria name it explicitly.

## Success Criteria

- [ ] All functional requirements are implemented and tested
- [ ] All test scenarios pass (`make test`)
- [ ] `background_alt` and its token, docs and tests are fully removed
- [ ] A config file from before this change starts with an unchanged
      appearance
- [ ] Code review is completed

## References

- Previous (superseded) specification: `feature-docs/window-opacity-toggle/SPEC.md`
- Requirements document: `feature-docs/whole-window-opacity-toggle/REQUIREMENTS.md`
- Design tokens: `design-system/tokens.yaml`
- Config sample: `config.yaml.example`
