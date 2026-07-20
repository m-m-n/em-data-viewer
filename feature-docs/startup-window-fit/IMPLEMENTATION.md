# Implementation Plan: Startup Window Fit

## Overview

Defer the window's first presentation until every section's first data has
arrived (or a fixed 1-second timeout), and compute the presented height from
the real row-packed layout at the configured width — never from the width-0
degraded layout. Two tasks: a GUI-free readiness gate in the app layer, and
deferred presentation + width-derived fitting in the ui layer.

## Technology Stack

- **Language**: Go (existing)
- **UI**: gotk4 / GTK4 (existing) — no new dependencies, so no new license
  obligations (project license: MIT).

## Layer Structure

- `internal/app` — startup readiness accounting (which sections still owe
  their first data), the 1-second timeout decision, and the immediate-present
  decision for config-error / zero-section startups. GUI-free and
  `go test`-able (NFR2). Talks to the ui layer only through the narrow
  interfaces defined in app.go.
- `internal/ui` — deferred presentation and window size fitting. Owns all
  GTK interaction. Must not know why presentation was requested, only when.
- `internal/runner` — unchanged.

## Shared Components

| Component | Responsibility | Contract (pre/postcondition) | Used by tasks |
|-----------|----------------|------------------------------|---------------|
| `PresenterUI` (app-layer narrow interface, mirrored by a ui.UI method `Present`) | Request the window's first presentation | Pre: called on the GTK main loop (via the app layer's dispatch seam). Post: the ui layer presents the window at a fitted size; the first call wins, later calls are no-ops; a call arriving before the GTK frame exists is remembered and honored when the frame is built. The app layer invokes it via a type assertion on its `UI` interface value (same pattern as the existing `KeepAboveUI`), so either task compiles and passes tests independently of the other's merge state. | task0001 (caller), task0002 (implementer) |

Integration wiring owner: task0001 owns calling `Present` through the type
assertion at gate-ready time; task0002 owns implementing the method on the
concrete ui type. The type-assertion pattern means no placeholder needs
replacing — when both tasks are merged the assertion starts succeeding, which
is the same mechanism `KeepAboveUI` already uses in this codebase.

## Conventions

- Follow the existing app-layer seam style: narrow per-capability interfaces,
  function-type seams for side effects, dispatch via the existing
  main-loop-dispatch function value so tests can substitute a synchronous
  dispatcher.
- Timeout constant lives in the app layer with the other startup constants;
  the duration must be injectable for tests (default 1 second).

## Cross-task Design Decisions

### 1. Presentation readiness is decided in the app layer, applied in the ui layer

The gate (which sections owe first data, timeout, immediate-present cases) is
pure app-layer state fed by runner events — it must stay GUI-free (NFR2). The
ui layer only receives a single "present now" request. Affects both tasks.

### 2. Ordering guarantee via FIFO dispatch

The app layer's event consumer schedules per-section UI updates onto the main
loop in receive order (existing behavior). The gate's present request is
dispatched through the same FIFO dispatch seam, strictly after the section
event that completed the gate was scheduled. Consequently, by the time the
present request runs, every first-data UI update scheduled before it has
already been applied — the presented window shows the data that satisfied the
gate. Affects task0001 (must dispatch in this order) and task0002 (may assume
content is already applied when `Present` runs).

### 3. Pre-map fitting uses a width derived from the configured width, not the allocation

GTK allocates widgets only after the window is mapped, so a hidden window can
never satisfy "wait for a settled allocation". Therefore the presented size
is computed from the configured window width: derive the content-area width
(configured width minus the window's own CSS chrome), row-pack the sections
against that width, and measure the resulting height — all of which GTK
supports on unmapped widgets. The width-0 degraded packing must never be what
the presented height was computed from (FR5). After presentation, the
existing allocation-driven triggers keep the window in sync; the map-time
re-fit remains as a safety net and must re-arm instead of silently no-oping
when it runs before the allocation exists. Affects task0002 (implements) and
task0001 (relies on: present request needs no allocation to exist).

### 4. The gate applies to the first presentation only

Live reloads after presentation never re-hide the window and never re-arm the
gate. The gate is one-shot; once fired (data-complete, timeout, or
immediate), it stays fired for the process lifetime. Affects task0001.

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Width-derived pre-map height differs from the post-map allocation-driven height (visible snap after present) | Medium | Medium | Derive both from the same inputs (configured width, chrome computation, row packing); unit-test that the two computations agree on identical inputs; keep post-map triggers as correction |
| Present request races frame construction (activate not yet run) | Medium | High | `Present` contract: pre-frame calls are remembered and honored at frame construction (task0002 AC) |
| Gate never fires if runner events are dropped before the consumer starts | Low | High | Gate is armed in the same reload transaction that applies sections; timeout guarantees presentation regardless |
| Regression in steady-state refit behavior | Low | Medium | Keep existing triggers untouched except the map-time re-arm; manual regression checklist in VERIFICATION.md |

## Open Questions

- [ ] None.
