# Verification Document: Section Flow Layout

## Overview
**Feature**: section-flow-layout / **SPEC.md**: `feature-docs/section-flow-layout/SPEC.md` / **IMPLEMENTATION.md**: `feature-docs/section-flow-layout/IMPLEMENTATION.md`

## Build Verification
- Command: `go build ./...`
- Expected: exit code 0, no errors (benign gotk4 cgo warnings are known and acceptable)

## Test Verification
- Command: `go test ./...`
- Coverage target: no reduction against the current suite; new stylesheet behavior covered

### Test Scenarios from SPEC.md
| ID | Scenario | Expected Result | Test Type |
|----|----------|-----------------|-----------|
| TS-1 | Generated base stylesheet includes the flow-container child-wrapper reset | Rule with zero padding and zero margin present | Unit |
| TS-2 | Existing `internal/ui` style/merge tests | Pass unchanged | Unit |
| TS-3 | Full suite `go test ./...` | All packages pass | Unit |

## Code Quality Verification
- Format: `gofmt -l -w .` (expected: no files rewritten)

## SPEC.md Compliance

### Success Criteria
| ID | Criterion | How to Verify |
|----|-----------|---------------|
| SC-1 | Narrow sections share rows at width 800; overflow wraps | Manual M-1 |
| SC-2 | Visual order equals config order | Manual M-1 |
| SC-3 | Reload / update / styling / error behaviors unchanged | TS-2, TS-3, Manual M-2, M-3 |
| SC-4 | Build, tests, gofmt clean | Build/Test/Format commands above |

### Functional Requirements Coverage
| Requirement | Tasks | Verification |
|-------------|-------|--------------|
| FR1 | task0001 | Manual M-1 |
| FR2 | task0001 | Manual M-1, Edge E-1 |
| FR3 | task0001 | Manual M-1; tokens wording check (M-4) |
| FR4 | task0001 | TS-1, Manual M-3 |
| FR5 | task0001 | TS-2, TS-3, Manual M-2, M-3 |
| NFR1 | task0001 | go.mod unchanged (M-5) |
| NFR2 | task0001 | Build/Test/Format commands (SC-4) |

## Manual Testing (E2E Not Possible)
- [ ] M-1: Run the app with window width 800 and the sensor config;
      narrow sections (co2 / pressure / disk-usage / uptime 等) share
      rows; sections that do not fit wrap to the next row; order follows
      config order left→right, top→bottom.
- [ ] M-2: Edit config.yaml while running (remove/add a section); layout
      rebuilds without crash or leftover widgets.
- [ ] M-3: A section with a failing command shows the error style; a
      section with a `style` override renders it; no selection/hover
      highlight appears when clicking sections.
- [ ] M-4: tokens.yaml / tokens.html state セクション間 横 2 unit / 縦 0.
- [ ] M-5: go.mod / go.sum have no new dependencies.
- [ ] E-1: With a narrow window, a wide section (wide-status) occupies
      its own row and its output wraps as before.
- [ ] E-2: A config with zero sections renders an empty window without
      error.

## Verification Summary
| Category | Items | Automated | E2E | Manual |
|----------|-------|-----------|-----|--------|
| Build/Format | 2 | 2 | 0 | 0 |
| Unit tests | 3 | 3 | 0 | 0 |
| Manual/Edge | 7 | 0 | 0 | 7 |
