# Verification Document: Startup Window Fit

## Overview
**Feature**: startup-window-fit / **SPEC.md**: `feature-docs/startup-window-fit/SPEC.md` / **IMPLEMENTATION.md**: `feature-docs/startup-window-fit/IMPLEMENTATION.md`

## Build Verification
- Command: `go build ./...`
- Expected: exit code 0, no errors

## Test Verification
- Command: `go test ./...`
- Coverage target: new GUI-free logic (gate, width derivation, fit
  agreement) fully covered; GTK glue excluded per test/README.md

### Test Scenarios from SPEC.md
| ID | Scenario | Expected Result | Test Type |
|----|----------|-----------------|-----------|
| TS-1 | All armed sections receive first data | Ready fires exactly once; present dispatched after the completing update | Unit |
| TS-2 | A section's first event is an error | Counts as received; does not delay presentation | Unit |
| TS-3 | Stream section's first append event | Counts as received | Unit |
| TS-4 | Timeout elapses with sections still owed | Ready fires at the injected timeout; pending sections remain label-only/empty | Unit |
| TS-5 | Zero sections or first-load failure | Immediate fire without waiting | Unit |
| TS-6 | Data arrives after the gate fired | Routed through the normal update path; no re-fire | Unit |
| TS-7 | Pre-map fit vs allocation-driven fit | Same height for the same configured width and inputs | Unit |
| TS-8 | Width-0 degraded packing | Never the source of a presented height (pre-map path packs at derived width) | Unit |
| TS-9 | Map-time re-fit before allocation | Retries instead of no-oping permanently | Unit |

## Code Quality Verification
- Format: `gofmt -l -w .` (expect no diff on committed files)
- Static analysis: none configured

## SPEC.md Compliance

### Success Criteria
| ID | Criterion | How to Verify |
|----|-----------|---------------|
| SC-1 | No oversized-window phase at startup | Manual M-1 (X11 geometry log + visual on Wayland) |
| SC-2 | All-1m-interval config appears within ~1s | Manual M-2 |
| SC-3 | Slow first command: present at ~1s, section empty, fills in later | Manual M-3 |
| SC-4 | Config error / zero sections present immediately | Manual M-4 |
| SC-5 | No steady-state regressions | Manual M-5 |

### Functional Requirements Coverage
| Requirement | Tasks | Verification |
|-------------|-------|--------------|
| FR1 | task0001, task0002 | TS-1, TS-4, M-1, M-2 |
| FR2 | task0001 | TS-2, TS-3 |
| FR3 | task0001 | TS-4, TS-6, M-3 |
| FR4 | task0001 | TS-5, M-4 |
| FR5 | task0002 | TS-7, TS-8, TS-9, M-1 |
| NFR1 | task0001 | M-6 (code inspection: no polling / fixed-FPS loops) |
| NFR2 | task0001, task0002 | TS-1..TS-9 being GUI-free unit tests; M-6 |

## E2E Testing
None (no E2E infrastructure; GUI verified manually per test/README.md).

## Manual Testing (E2E Not Possible)
- [ ] M-1: Normal config (fast first commands) → window appears once, at
      fitted size; no visible resize after presentation. Verify under
      GDK_BACKEND=x11 with a geometry poll (no oversized sample ever
      observed) and visually on Wayland.
- [ ] M-2: Config with all intervals ≥ 1m → window appears roughly 1s
      after startup (timeout path), sections populated (first runs are
      immediate) or empty per timing.
- [ ] M-3: Config containing a `sleep 10`-style section → window at ~1s
      with that section label-only/empty; section fills in at ~10s and the
      height follows.
- [ ] M-4: Broken config file → window appears immediately with the error
      banner. Empty sections list → appears immediately.
- [ ] M-5: Steady-state regression pass: interval updates resize correctly,
      live config reload works, keep-above toggle, drag-to-move, context
      menu, monitor-change re-cap.
- [ ] M-6: Code inspection: no polling loops or fixed-FPS ticking introduced
      (NFR1); gate logic lives in GUI-free files (NFR2).

## Verification Summary
| Category | Items | Automated | E2E | Manual |
|----------|-------|-----------|-----|--------|
| Build/format | 2 | 2 | 0 | 0 |
| Unit scenarios | 9 | 9 | 0 | 0 |
| Manual scenarios | 6 | 0 | 0 | 6 |
