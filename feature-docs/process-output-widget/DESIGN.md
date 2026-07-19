# Design: process-output-widget

## Decisions

- Window: frame-only look — `surface` background, `frame` 1px (`border.frame-width`) outline, `radius.window` 8px rounded corners, inner padding 2 × `spacing.unit`. No titlebar, no shadow of our own (compositor default). Mockup: [design/mockups/screen-widget.html](design/mockups/screen-widget.html) `data-state="default"`.
- Sections: separated by divider rules (`color.divider`, `border.divider-width`), not by spacing. Between rows: a horizontal rule with zero extra vertical gap. Between sections sharing a row: a vertical rule with 1 × `spacing.unit` on each side. Every rule's ends are inset 1 × `spacing.unit`, so rules never touch each other or the window frame — no corners, T- or cross-junctions. `frame` is not used inside the window. Each section = optional name label (`typography.section-label`, `on-surface-dim`) + output text (`typography.output`, `on-surface`), 1 × `spacing.unit` between label and output. A section without `name` omits the label row entirely.
- Output text: `white-space: pre-wrap` equivalent (preserve command formatting, wrap long lines). All trailing newlines are stripped before rendering; interior blank lines are preserved (mockup state `trailing-newline`). Pango markup colors in the output override `on-surface`.
- Empty state (zero sections): the framed window renders with nothing inside, `min-height` keeps it grabbable. Mockup state `empty`.
- Config-error banner (FR9): full-width bar pinned above all sections, `error-surface` background, `error` text in `typography.output`, single line with the parser message. Last-good sections keep rendering below it. Mockup state `config-error`.
- Section command error (FR9): rendered inside the section as its output, `error` color, format `error: {message}`. Other sections unaffected. Mockup state `section-error`.
- Context menu (FR8): stock GTK4 PopoverMenu with a single 「終了」 item — no custom styling. Mockup state `context-menu` is illustrative only.
- Tokens: created `design-system/tokens.yaml` (new project, no design system existed) + generated `design-system/tokens.html`. Token values are the built-in defaults; user config `window.style` / `sections[].style` overrides them at runtime.

## Rationale

- Semi-transparent dark surface: blends into a desktop over wallpaper, the widget's purpose (NFR1); output text stays the visual subject.
- Monospace default for output: command output is column-aligned more often than not (FR4/FR5); user can override per-section via config.
- Divider rules between sections: on the device, spacing-only separation left residual vertical space and unclear horizontal boundaries. Floating 1px rules in `divider` (dimmer than `frame`) separate clearly without forming boxes, keeping the minimal intent.
- Trailing-newline strip: command stdout normally ends with a newline; rendering it draws a blank line between the output text and the divider below, contradicting "rules with zero extra gap" separation. Spacing is owned by the design, not by command output.
- Banner instead of dialog for config errors: a widget must not steal focus or require dismissal; the banner disappears on the next successful reload (FR3/FR9).
- Error color on dark surface (#e57373): readable at 4.5:1+ on `surface` while clearly distinct from normal output.
- Stock GTK context menu: one menu item does not justify custom chrome; GTK default guarantees GNOME consistency (FR8).

## Open items

- None. Runtime-configurable styles (colors/fonts per user config) are a spec concern (FR2/FR3), not open design; tokens define only the defaults.
