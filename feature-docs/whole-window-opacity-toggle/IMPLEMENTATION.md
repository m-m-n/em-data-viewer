# Implementation Plan: Whole-Window Opacity Toggle

## Overview

Replace the background-color-only `Ctrl+T` toggle with a whole-window opacity
toggle: the style layer gains two configurable opacity values plus a pure
selection rule, and the frame applies the selected value as a widget-level
opacity property so background, border, labels and output text fade together.

## Technology Stack

- **Language**: Go (existing project language; no new dependency)
- **UI toolkit**: gotk4 / GTK4 — already a project dependency. Whole-window
  dimming uses the widget opacity property that GTK already exposes; no new
  library is introduced, so the MIT project license is unaffected.
- **Config**: existing YAML config path (`window.style` free-form string map);
  no schema change in the `config` package.

## Layer Structure

Unchanged from the project's existing shape. Two layers matter here:

| Layer | File | Responsibility | May depend on |
|-------|------|----------------|---------------|
| style | `internal/ui/style.go` | Defaults table, style merging, pure value interpretation and selection. GTK-free. | nothing in this feature |
| frame | `internal/ui/window.go` | Owns the window widget, the active-pattern state, key controllers, and applying the selected value to GTK. | style |

Dependency direction is one-way: frame → style. The style layer must not gain
any GTK import, and the `config` package must keep importing neither.

## Shared Components

| Component | Responsibility | Contract (pre/postcondition) | Used by tasks |
|-----------|----------------|------------------------------|---------------|
| Effective-opacity selector | Given a merged window style map and the active-pattern flag, return the opacity to apply | **Pre**: the map is the result of merging user overrides over the window-style defaults (so both opacity keys are normally present); the flag is `false` for the primary pattern and `true` for the alternate one. **Post**: returns a float in the inclusive range 0.0–1.0. Reads the primary key when the flag is false and the alternate key when it is true. A missing, unparsable, or out-of-range value for the *selected* key yields that key's built-in default; the other key's value never influences the result. The input map is never mutated. | task0001 builds it, task0002 calls it |
| Opacity value parser | Turn one configured string into a float with a per-call fallback | **Pre**: an arbitrary string (possibly empty) and a fallback float already known to be valid. **Post**: the parsed value when the string is a decimal number within 0.0–1.0 inclusive; otherwise the fallback, unchanged. Never returns an error and never panics. | task0001 (internal to the selector; not called across tasks) |
| Window style defaults table | Source of the built-in values | **Post**: contains the primary opacity key with value `"1.0"` and the alternate opacity key with `"0.45"`, and no longer contains the alternate-background key. | task0001 owns it; task0002 reads through the selector only |

Both opacity keys are named exactly as the public config keys, matching how
`background` is handled today, so no key-alias entry is needed.

## Conventions

### Key names

The public config keys and the internal style-map keys are identical:
`opacity` (primary) and `opacity_alt` (alternate). Both live under
`window.style` in the YAML config.

### Naming: pattern identifiers

Identifiers that currently name the *background* pattern must be renamed to
name the *opacity* pattern, so no symbol describes a mechanism that no longer
exists. This spans both tasks, so the rule is fixed here: the words
"background pattern" in identifier and comment text become "opacity pattern",
and the state field, the flip helper, the toggle entry point, the
active-pattern accessor, the key match predicate and the controller-attach
helper are renamed accordingly. Behavior is unchanged by the rename.

### Error-handling policy

An unusable configured opacity value is NOT an error. It never raises the
config error banner, never fails the load, and never logs at error level: the
built-in default is substituted silently and the rest of the config applies
normally. This matches how the style layer already treats values it cannot
use.

### Removal policy

The previous feature's background-only mechanism is removed, not deprecated.
Nothing keeps reading the alternate-background key. Because unknown keys are
already discarded during the style merge, a user config that still sets it
keeps loading — no compatibility shim is written for it.

## Cross-task Design Decisions

### D1: Opacity is a widget property, not a stylesheet value

The toggle no longer participates in stylesheet construction. The stylesheet
is rebuilt from the merged style map directly, and the active pattern is
applied by setting the window widget's opacity. Consequences both tasks must
respect:

- The stylesheet output must be independent of the active pattern — the
  indirection that composed the pattern into the style map before building
  the stylesheet is deleted, and the restyle path reads the merged map
  directly.
- Applying the toggle must not rebuild the stylesheet, re-measure, re-fit or
  re-pack anything. It is a single property assignment on the window.

Rationale: GTK composites the widget opacity over the window's entire
rendering, which is what "text included" requires; per-color alpha would need
every style key to gain a second value and still would not cover text drawn
by Pango markup.

### D2: The active pattern lives outside the merged style map

The active-pattern flag stays a process-local field on the frame and is never
written into the merged style map, so re-merging fresh config values on
reload cannot clobber it. Both opacity values live in the merged map; only
the flag decides which is read. This is what makes FR9 hold without extra
bookkeeping.

### D3: Two application points, one path

The effective opacity is applied in exactly two situations — when settings are
applied (initial construction and every config reload) and when the toggle
fires. Both call the same frame-level apply step, so there is no second code
path that could drift.

### D4: Task split boundary

task0001 owns everything GTK-free (defaults table, parser, selector, design
tokens, config sample, their tests) and the deletion of the old selection
helper. task0002 owns everything that touches the window widget (state field,
apply step, key controller retarget, renames, their tests) and the deletion of
the old helper's call sites. The two tasks therefore both delete parts of the
old mechanism, each on its own side of the layer boundary; neither deletion
depends on the other having landed first, and after both merge no reference
remains.

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Widget opacity and the background color's existing alpha compound into an unexpectedly faint primary state | Medium | Medium | Primary default is 1.0, which leaves the composited result identical to today; verified by the unchanged-startup-appearance criterion |
| A compositor ignores widget opacity (notably on native Wayland) | Low | High for the feature's value | Flagged as a manual verification item; no code mitigation is planned |
| The two tasks both edit the `internal/ui` package and conflict on merge | Medium | Low | Each task's file set is disjoint (`style.go`/`style_test.go` vs `window.go`/`window_test.go`); the implementer's parent-side-adoption protocol covers any residue |
| The rename sweep misses a comment or symbol, leaving "background pattern" wording behind | Medium | Low | The rename rule is pinned here and is an acceptance criterion of task0002 |

## Open Questions

- [ ] Whether widget opacity behaves as expected under the native Wayland
      compositor used on the target desktop — resolvable only by running the
      built binary, so it is recorded as a manual verification item rather
      than blocking the plan.
