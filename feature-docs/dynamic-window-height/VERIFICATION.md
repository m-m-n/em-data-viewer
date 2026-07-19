# Verification Document: Dynamic Window Height

## Overview
**Feature**: dynamic-window-height / **SPEC.md**:
`feature-docs/dynamic-window-height/SPEC.md` / **IMPLEMENTATION.md**:
`feature-docs/dynamic-window-height/IMPLEMENTATION.md`

## Build Verification
- Command: `go build ./...`
- Expected: exit code 0, no errors

## Test Verification
- Command: `go test ./...`
- Expected: all tests pass

### Test Scenarios from SPEC.md
| ID | Scenario | Expected Result | Test Type |
|----|----------|-----------------|-----------|
| TS-1 | Load YAML containing `window.height` | Loads without error or warning; height has no effect | Unit |
| TS-2 | Load YAML without `window.height` | Remaining fields behave as before (width default 400, style, keep_above) | Unit |
| TS-3 | Section output grows/shrinks at runtime | Window height follows the content | Manual |
| TS-4 | Content naturally taller than the monitor | Window height caps at monitor height; overflow clipped | Manual |
| TS-5 | Zero sections | Window shows at minimum frame height (48px) and stays draggable | Manual |
| TS-6 | Config reload adds/removes sections | Window height re-fits to the new content | Manual |
| TS-7 | `config.yaml.example` content | No `height` line present | Manual |

## Code Quality Verification
- Format: `gofmt -l -w .` (expects no diff on a clean tree)

## SPEC.md Compliance

### Success Criteria
| ID | Criterion | How to Verify |
|----|-----------|---------------|
| SC-1 | Window height follows content | TS-3, TS-6 |
| SC-2 | Monitor-height cap | TS-4 |
| SC-3 | `height` key silently ignored | TS-1 |
| SC-4 | Example config has no height | TS-7 |
| SC-5 | Build/test/format pass | Build & Test Verification above |

### Functional Requirements Coverage
| Requirement | Tasks | Verification |
|-------------|-------|--------------|
| FR1 | task0001 | TS-3 |
| FR2 | task0001 | TS-3, TS-6 |
| FR3 | task0001 | TS-4 |
| FR4 | task0001 | TS-1, TS-2, TS-7 |
| NFR1 | task0001 | `go test ./...` runs headlessly (TS-1, TS-2) |
| NFR2 | task0001 | TS-5 + width behavior unchanged (TS-2) |

## Manual Testing (E2E Not Possible)

GUI behavior is manually verified per test/README.md (no automated GUI tests).

- [ ] TS-3: run the app, use a section whose output line count changes;
      observe the window growing and shrinking
- [ ] TS-4: configure a section producing more lines than fit the monitor;
      observe the cap and clipping
- [ ] TS-5: run with zero sections; observe the 48px minimum frame and
      drag-to-move
- [ ] TS-6: while running, edit config.yaml to add/remove a section;
      observe the height re-fit
- [ ] TS-7: inspect config.yaml.example

## Verification Summary
| Category | Items | Automated | E2E | Manual |
|----------|-------|-----------|-----|--------|
| Build/Format | 2 | 2 | 0 | 0 |
| Unit tests | 2 | 2 | 0 | 0 |
| GUI behavior | 5 | 0 | 0 | 5 |
