# Feature: Startup Window Fit

## Overview

At startup the window currently appears at an oversized height and only snaps
to the correct fitted size when the smallest-interval section's second run
fires (about 5 seconds with the reference config). This feature reworks the
startup sequence so the window is presented once, already at its fitted size:
Present is deferred until every section's first data has arrived (or a fixed
1-second timeout), and the height is computed from the real row-packed layout
instead of the width-0 degraded one.

Root cause (established by investigation): every refit during startup runs
before the content container has a settled allocation and silently no-ops
(including the map-signal deferred one); the row packing degrades to
one-section-per-row at width 0, and the height gets finalized from that
degraded layout. The next effective refit is the first periodic update event.

## Objectives

- Present the window exactly once, already at its final fitted size, in the
  normal case.
- Bound the wait: a slow or hung first command must not delay presentation
  beyond 1 second.
- Preserve the existing event-driven architecture and all steady-state
  behavior (interval updates, live reload, keep-above, styling).

## User Stories

### US1: Clean startup
As a user, I want the widget to appear already at its correct size, so that
startup does not flash an oversized window for several seconds.

**Acceptance Criteria:**
- [ ] With a config whose first commands all complete quickly, the window
      appears once, at the fitted size, with all sections populated.
- [ ] No visible resize occurs after presentation in this case.

### US2: Slow first command
As a user with a slow first command (e.g. `sleep 10 && ...`), I want the
widget to appear within about 1 second, so that one slow section does not
block the whole widget.

**Acceptance Criteria:**
- [ ] The window is presented about 1 second after startup.
- [ ] Sections whose first data has not arrived show their label only, with
      empty output text.
- [ ] Late data fills in through the normal update path and the height
      follows.

## Technical Requirements

### Functional Requirements

- **FR1:** Defer the window's first `Present()` until either (a) every
  section's first data has been received, or (b) the fixed 1-second timeout
  has elapsed, measured from when the first successful config application
  arms the readiness gate — whichever comes first. While deferred, the
  window must not be shown in any form (no map, no taskbar/dock presence).
- **FR2:** First-data semantics: an interval section counts as received on
  its first run's result event (replace or error); a stream section counts as
  received on its first event (append or error). Error events count as
  received — an immediately-failing command must not delay presentation.
- **FR3:** The presentation timeout is fixed at 1 second and is not
  configurable. On timeout, sections without first data are shown with their
  label only and empty output text; data arriving later flows through the
  normal `UpdateSection` → refit path (a post-present height change is
  acceptable only in this timeout case).
- **FR4:** When the initial config load fails (error-banner-only state) or
  the config has zero sections, present immediately without waiting.
- **FR5:** The height applied at presentation must be computed from the real
  row-packed layout at the content width derived from the configured window
  width (GTK provides no allocation before the window is mapped, so the
  pre-presentation fit must not depend on one; the derived content width
  must always be positive). The width-0 degraded layout (one section per
  row) must never be the layout from which a presented window's height was
  finalized. After presentation, the allocation-driven refit path remains as
  a safety net: its map-time trigger re-arms (bounded, without busy-spinning
  the main loop) until an allocation exists instead of no-oping permanently.

### Non-Functional Requirements

- **NFR1 - Architecture:** No polling loops and no fixed-FPS ticking. The
  wait for first data and the timeout must use the existing event-driven /
  blocking-wait style (idle CPU ~0).
- **NFR2 - Testability:** The startup-gate logic (readiness tracking,
  timeout decision, first-data accounting) must live in GUI-free code paths
  testable with `go test`, per test/README.md. GTK-touching glue stays thin.

## Implementation Approach

### Current startup flow (for reference)

```
App.Run ─▶ reloadLoop ─▶ reload ─▶ runner.Apply (first runs start)
        └▶ ui.Run ─▶ activate ─▶ newFrame ─▶ win.Present()   ← immediate today
events ─▶ eventLoop ─▶ IdleAdd ─▶ UpdateSection ─▶ refit     ← all no-op pre-allocation
```

### Target startup flow

```
App.Run ─▶ reload ─▶ runner.Apply (first runs start)
        └▶ ui.Run ─▶ activate ─▶ newFrame (window built, NOT presented)
events ─▶ UpdateSection (widgets fill in, window still hidden)
gate: all sections received first data (FR2)  OR  1s timeout (FR3)
  └▶ pre-map fit: row-pack at the width derived from the configured width (FR5)
      └▶ win.Present()   ← single presentation at final size
```

The gate needs both conditions resolved in order: data-complete (or timeout)
first, then the width-derived pre-map fit, then Present. The app layer owns
the readiness/timeout accounting (GUI-free); the ui layer owns the pre-map
fit and the deferred Present. After presentation, the existing
allocation-driven refit triggers keep the window in sync (safety net).

### Dependencies

**Internal:** internal/app (startup wiring, event routing), internal/ui
(frame construction, Present timing, refit triggers), internal/runner
(event kinds — no changes expected).

**External:** gotk4 (existing dependency); no new libraries.

### File Structure (expected touch points)

```
internal/
├── app/
│   ├── app.go            # startup wiring: readiness gate hookup
│   ├── events.go         # first-data accounting per section key
│   └── reload.go         # initial-load projection: gate arming / immediate present
└── ui/
    ├── ui.go             # deferred Present entry point
    └── window.go         # allocation-settled refit guarantee
```

## Test Scenarios

### Unit Tests
- [ ] Readiness gate: all sections received → ready fires once.
- [ ] Readiness gate: error event counts as received (FR2).
- [ ] Readiness gate: stream section's first append counts as received (FR2).
- [ ] Timeout path: the gate fires at the injected timeout while sections
      are still owed (FR3).
- [ ] Zero sections / config error → immediate-present decision (FR4).
- [ ] Late-arriving data after timeout routes through the normal update path.

### Integration Tests
None (per test/README.md — GUI behavior is verified manually).

### E2E Tests
**Existing E2E tests**: None
**Run command**: Not detected
- [ ] Manual: normal config → single presentation at fitted size (US1).
- [ ] Manual: config with `sleep 10` section → presented at ~1s with empty
      section, fills in later (US2).
- [ ] Manual: broken config → immediate presentation with error banner (FR4).

### Edge Cases
- [ ] All first commands fail instantly → all sections count received;
      present as soon as allocation settles, well under 1s.
- [ ] Section with empty label and no data at timeout → row renders at
      near-zero height (existing label-hiding behavior; acceptable).
- [ ] Live reload after startup → unchanged behavior (gate applies to the
      first presentation only).

## Success Criteria

- [ ] All functional requirements are implemented and tested.
- [ ] Startup shows no oversized-window phase under X11 and Wayland.
- [ ] With all-1m intervals, the window appears within ~1s of startup.
- [ ] `go test ./...`, `go build ./...`, `gofmt -l .` all clean.
- [ ] No regression in steady-state behavior (interval updates, live
      reload, keep-above, drag-to-move, context menu).

## Open Questions

None — all requirements resolved during the discussion phase.
