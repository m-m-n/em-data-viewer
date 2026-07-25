# Implementation Plan: Window Opacity Toggle

## Overview

Add a second, independently configured window background pattern and a
`Ctrl+T` binding that switches the live window between the two. The work
splits into a style layer (the alternate value, its token, and the pure
selection rule) and a frame layer (the toggle state, the key controller, and
the live restyle).

## Technology Stack

- **Language**: Go (existing module, no new dependency).
- **UI toolkit**: GTK4 via the gotk4 bindings already in `go.mod`. The key
  controller and modifier constants come from the same bindings; nothing new
  is added, so `project.license` (MIT) is unaffected.
- **Design tokens**: `design-system/tokens.yaml`, already the SSOT the ui
  package's default tables are drift-guarded against.

## Layer Structure

| Layer | Responsibility | Depends on |
|-------|----------------|------------|
| config (`internal/config`) | Carries `window.style` as an opaque string map. **Unchanged by this feature** — the new key needs no schema work and this package must stay free of GTK imports. | — |
| style (`internal/ui/style.go`) | Owns the window style default table, the merge rule, the stylesheet builder, and the new pure background-pattern selection rule. GTK-free logic. | design tokens |
| frame (`internal/ui/window.go`) | Owns the live window: holds the merged style map, the active-pattern state, the key controller, and calls the stylesheet builder. | style |

Dependency direction is one-way: frame → style. The style layer never learns
about key events or window state.

## Shared Components

| Component | Responsibility | Contract (pre/postcondition) | Used by tasks |
|-----------|----------------|------------------------------|---------------|
| `background_alt` style key | Names the alternate window background in the merged window style map and in user config (`window.style.background_alt`). | Pre: the key participates in the window style default table, so the merge rule accepts a user override for it. Post: after merging, the map always holds both `background` and `background_alt` entries. | task0001, task0002 |
| `color.surface-alt` design token | Design SSOT for the alternate background default. | Pre: lives in `design-system/tokens.yaml` under `color`. Post: its value string equals the runtime default for `background_alt` exactly (`#1e1e1e99`), so the existing token/runtime drift guard passes. | task0001 |
| `ApplyBackgroundPattern` (style layer, exported within the ui package) | Pure selection rule: produces the window style map to render for a given active pattern. | Pre: input is a merged window style map (both keys present) and a boolean stating whether the alternate pattern is active. Post: returns a map identical to the input except that, when the alternate pattern is active, the `background` entry carries the `background_alt` value; the input map is never mutated; every other entry (frame, radius, error colors) is passed through untouched; a missing `background_alt` entry leaves `background` unchanged. | task0001 (implements), task0002 (calls) |
| Frame active-pattern state | Process-local record of which pattern is showing. | Pre: initialized to "primary active" when the frame is constructed. Post: survives style re-application (config reload) and is never read from or written to the config file. | task0002 |

Both tasks run in parallel: task0002 codes against the
`ApplyBackgroundPattern` contract above without reading task0001's plan.

## Conventions

- **Style key naming**: user-facing style keys stay snake_case
  (`background`, `border_color`, `border_width`), so the new key is
  `background_alt`. Like `background`, it needs no alias entry in the
  SPEC-key translation table — the public key name and the internal key name
  are the same string.
- **Token naming**: role-based, matching the file's stated convention
  (`surface`, `error-surface`), so the alternate surface is `surface-alt`.
- **No new validation**: style values are user input passed through to CSS.
  An unparsable `background_alt` behaves exactly like an unparsable
  `background` does today — no new error path, no logging, no fallback.
- **Purity**: everything that can be decided without GTK (pattern selection,
  key matching, state flip) is a pure function so it is covered by
  `go test ./...` headlessly.

## Cross-task Design Decisions

### D1: The alternate pattern is a full color value, not a numeric opacity

Transparency in this application is already carried by the alpha channel of
`window.style.background`. Introducing a numeric opacity field would create a
second, independent transparency mechanism that would compound with the
existing alpha. The alternate pattern is therefore another color string
handled by the same merge and stylesheet paths (SPEC A1).

Affected tasks: task0001, task0002.

### D2: Selection happens at stylesheet-build time, not at merge time

The merged style map holds BOTH patterns for the whole process lifetime; the
active pattern is applied as a late, pure transformation just before the
stylesheet is built. This is what makes FR8 (state survives a config reload)
fall out for free: re-merging fresh config values cannot clobber the active
pattern, because the pattern is not stored in the map.

Affected tasks: task0001 (provides the transformation), task0002 (applies it
on every restyle).

### D3: Only the window background participates

The transformation replaces exactly one entry. The error banner background,
the frame border color and width, the radius and all section styles are
untouched, so the toggle can never change layout or the error presentation
(SPEC A4, NFR3).

Affected tasks: task0001, task0002.

### D4: The binding is a window-level key controller

The toggle is bound on the application window, in the same capture-phase
style the existing scroll blocker and drag gesture use, and therefore
requires the widget window to hold keyboard focus. No global/compositor
hotkey is registered (SPEC A7). The controller claims an event only when the
match rule accepts it; every other key event is left unclaimed so existing
input behavior is untouched (FR7).

Affected tasks: task0002.

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| The token/runtime drift-guard test fails because the new token is added on only one side | Medium | Low (fails loudly in CI/test) | Both sides are inside task0001's file set and covered by its acceptance criteria |
| The key controller swallows events other components rely on (drag, context menu, scroll blocking) | Low | Medium | The match rule is a pure, unit-tested predicate; the controller claims only on a match (FR7) |
| A user's existing config already defines `background_alt` for some other purpose | Very low | Low | The key is currently ignored by the merge rule, so no existing config can depend on it having an effect |
| Modifier semantics differ between X11 and Wayland | Low | Low | The match rule reads the event's modifier state through the toolkit's backend-independent abstraction; manual verification covers the developer's session |

## Open Questions

- [ ] None.
