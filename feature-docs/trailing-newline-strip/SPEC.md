# Feature: Output Trailing-Newline Strip

## Overview

Strip all trailing newlines from section output before it is rendered, across every display path (interval replace, stream append, section error). Interior blank lines are preserved. This implements the process-output-widget DESIGN.md decision "All trailing newlines are stripped before rendering; interior blank lines are preserved" (mockup state `trailing-newline`).

## Objectives

- Remove the phantom blank line between output text and the divider rule below it
- Keep spacing owned by the design (dividers / padding), never by command output
- Apply one uniform display rule to all render paths

## User Stories

### US1: Output sits directly above the rule
As a widget user, I want section output to end at its last visible line, so that the gap between text and the divider matches the design (rules with zero extra gap).

**Acceptance Criteria:**
- [ ] A command whose stdout ends with `\n` shows no blank line below its output
- [ ] A command that intentionally prints interior blank lines still shows them

## Technical Requirements

### Functional Requirements

- **FR1:** Interval (replace) output: all trailing newlines are stripped from the text before rendering.
- **FR2:** Stream (append) display: after joining the buffered lines for display, trailing newlines (i.e. trailing blank lines) are stripped. The ring buffer's stored lines are NOT modified — stripping is a display concern only.
- **FR3:** Section error display (`error: {message}`): trailing newlines in the message are stripped before rendering.
- **FR4:** Interior blank lines are preserved verbatim in all paths.
- **FR5:** Output consisting only of newlines renders as an empty string (the section shows its label only; no output line box).

### Non-Functional Requirements

- **NFR1:** No new external dependencies.
- **NFR2:** Quality gates pass: `go build ./...`, `go test ./...`, `gofmt -l -w .` clean.

## Implementation Approach

Stripping happens at render time in the GTK-free translation layer (`internal/app/events.go` `translateEvent`), which is the single point all three display paths already flow through (KindReplace / KindAppend / KindError). Markup rendering (`markup.Render`) and the runner/ring-buffer layers stay unchanged.

### Dependencies

**Internal Dependencies:**
- `internal/app/events.go`: `translateEvent` — the render-time translation point
- `feature-docs/process-output-widget/DESIGN.md`: design decision SSOT (Output text)

**External Dependencies:**
- None added.

## Test Scenarios

### Unit Tests

- [ ] TS-1: Replace event with text ending in one `\n` — rendered text has no trailing newline
- [ ] TS-2: Replace event with text ending in multiple `\n` — all are stripped
- [ ] TS-3: Append event whose snapshot ends with empty lines — joined display text has no trailing newlines; buffer contents unchanged
- [ ] TS-4: Error event with message ending in `\n` — displayed error text has no trailing newline
- [ ] TS-5: Text with interior blank lines (`"a\n\nb\n"`) — interior blank line preserved, trailing newline stripped
- [ ] TS-6: Text of only newlines (`"\n\n"`) — renders as empty string

### Integration Tests

- [ ] Existing `internal/app` and `internal/ui` test suites pass unchanged except where they assert the old trailing-newline behavior

### E2E Tests

**Existing E2E tests**: None
**Run command**: Not detected
- [ ] Not applicable

### Manual Verification

- [ ] M-1: On the device, with interval commands whose stdout ends in `\n`, the gap between output text and the horizontal rule below matches the mockup state `trailing-newline` (no blank line)

### Edge Cases

- [ ] Edge case 1: `\r\n` line endings — only `\n` (and any `\r` immediately preceding it) at the end of the text is subject to stripping; interior `\r\n` untouched
- [ ] Edge case 2: Pango markup whose visible text ends before trailing newlines — stripping operates on the raw text before markup rendering, so markup validity is unaffected

## Success Criteria

- [ ] All functional requirements are implemented and tested
- [ ] All unit test scenarios pass
- [ ] Quality gates (build / test / gofmt) pass
- [ ] Manual verification M-1 confirms the on-device gap matches the mockup

## Open Questions

- None.

## References

- feature-docs/process-output-widget/DESIGN.md: Output text decision (SSOT)
- feature-docs/process-output-widget/design/mockups/screen-widget.html: mockup state `trailing-newline`
