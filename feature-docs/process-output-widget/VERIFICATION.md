# Verification Document: process-output-widget

## Overview
**Feature**: process-output-widget / **SPEC.md**: `feature-docs/process-output-widget/SPEC.md` / **IMPLEMENTATION.md**: `feature-docs/process-output-widget/IMPLEMENTATION.md`

## Build Verification
- Command: `go build ./...`
- Expected: exit code 0, no errors

## Test Verification
- Command: `go test ./...`
- Coverage target: logic packages (`internal/config`, `internal/runner`, `internal/markup`) covered by unit tests; GUI excluded per test/README.md

### Test Scenarios from SPEC.md
| ID | Scenario | Expected Result | Test Type |
|----|----------|-----------------|-----------|
| TS-1 | Valid YAML loads with defaults applied (window size, max_lines) | Config populated, defaults filled | Unit |
| TS-2 | Invalid YAML / validation errors | Error reported; holder retains last good config | Unit |
| TS-3 | Watcher detects write and rename/replace saves | Notification emitted within debounce window | Unit |
| TS-4 | Reconcile computes start/stop/restart/unchanged sets | Correct process lifecycle per key diff | Unit |
| TS-5 | Interval captures full stdout per run; stream appends lines honoring max_lines | Replace/append events; ring buffer bounded | Unit |
| TS-6 | Command failure / abnormal exit | Error event for that section only; others continue | Unit |
| TS-7 | Invalid Pango markup falls back to escaped plain text; valid passes through | Render contract honored | Unit |

## Code Quality Verification
- Format: `gofmt -l -w .` (expect no files needing changes at verify time)
- Static analysis: none configured

## SPEC.md Compliance

### Success Criteria
| ID | Criterion | How to Verify |
|----|-----------|---------------|
| SC-1 | All FRs implemented | Requirements coverage below + manual scenarios |
| SC-2 | TS-1..TS-7 pass | `go test ./...` |
| SC-3 | Manual scenarios verified on GNOME/Wayland | M-1..M-3 |
| SC-4 | gofmt clean | format command output empty |

### Functional Requirements Coverage
| Requirement | Tasks | Verification |
|-------------|-------|--------------|
| FR1 | task0004 | M-1 (drag, frame look) |
| FR2 | task0001, task0005 | TS-1, unit-tested path resolution, M-1 |
| FR3 | task0001, task0002, task0005 | TS-3, TS-4, M-2 |
| FR4 | task0002 | TS-5 |
| FR5 | task0002 | TS-5, TS-6 |
| FR6 | task0003, task0004 | TS-7, M-1 |
| FR7 | task0004 | M-1 |
| FR8 | task0004, task0005 | M-3 |
| FR9 | task0001, task0002, task0004, task0005 | TS-2, TS-6, M-2 |
| NFR1 | task0004 | M-1 (runs on GNOME/Wayland) |
| NFR2 | task0002, task0005 | TS-5 (bounded buffer), M-2 (~1s reload) |
| NFR3 | task0001, task0002, task0003 | TS-1..TS-7 run headlessly; import check |

## E2E Testing
None (no E2E framework; GUI verified manually).

## Manual Testing (E2E Not Possible)
- [ ] M-1: Launch on GNOME/Wayland with a multi-section config (interval + stream). Frame-only window appears, sections stack in order with default styling, drag moves the window. モックとの目視照合: compare against `design/mockups/screen-widget.html` states `default` and `empty`.
- [ ] M-2: While running, edit the config: (a) style-only change applies without process restart; (b) add/remove/change a command reconciles processes; (c) save an invalid YAML → banner appears, last content stays (compare mockup state `config-error`); (d) fix it → banner clears. A failing command shows an in-section error (compare mockup state `section-error`). Changes visible within ~1s.
- [ ] M-3: Right-click opens the menu; 終了 exits the app and leaves no child processes behind (verify with a process listing).

## Verification Summary
| Category | Items | Automated | E2E | Manual |
|----------|-------|-----------|-----|--------|
| Build | 1 | 1 | 0 | 0 |
| Unit tests | 7 (TS-1..7) | 7 | 0 | 0 |
| Format | 1 | 1 | 0 | 0 |
| Manual | 3 (M-1..3) | 0 | 0 | 3 |
