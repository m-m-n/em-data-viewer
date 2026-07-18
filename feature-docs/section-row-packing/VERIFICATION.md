# Verification Document: Section Row Packing

## Overview
**Feature**: section-row-packing / **SPEC.md**: `feature-docs/section-row-packing/SPEC.md` / **IMPLEMENTATION.md**: `feature-docs/section-row-packing/IMPLEMENTATION.md`

## Build Verification
- Command: `go build ./...`
- Expected: exit code 0, no errors (benign gotk4 cgo warnings are known and acceptable)

## Test Verification
- Command: `go test ./...`
- Coverage target: no reduction against the current suite; packing logic fully covered

### Test Scenarios from SPEC.md
| ID | Scenario | Expected Result | Test Type |
|----|----------|-----------------|-----------|
| TS-1 | packRows greedy packing (width-6 example shape) | Rows [1 2 2], [2 2 2], … | Unit |
| TS-2 | packRows boundaries: exact fit stays, +1 wraps, gap counted | Matches greedy rule | Unit |
| TS-3 | packRows: oversize own row; avail<=0 one per row; empty input no rows | Matches rule | Unit |
| TS-4 | Stylesheet contains no flowboxchild rule | Absent | Unit |
| TS-5 | Existing internal/ui tests (incl. gap constants) | Pass | Unit |
| TS-6 | Full suite `go test ./...` | All packages pass | Unit |

## Code Quality Verification
- Format: `gofmt -l -w .` (expected: no files rewritten)

## SPEC.md Compliance

### Success Criteria
| ID | Criterion | How to Verify |
|----|-----------|---------------|
| SC-1 | Rows hold variable numbers of natural-width sections, no column alignment | Manual M-1 |
| SC-2 | Visual order equals config order | Manual M-1 |
| SC-3 | Window resize re-packs rows | Manual M-2 |
| SC-4 | Reload / update / styling / error behaviors unchanged | TS-5, TS-6, Manual M-3 |
| SC-5 | Build, tests, gofmt clean | Build/Test/Format commands above |

### Functional Requirements Coverage
| Requirement | Tasks | Verification |
|-------------|-------|--------------|
| FR1 | task0001 | Manual M-1 |
| FR2 | task0001 | TS-1, TS-2, TS-3 |
| FR3 | task0001 | TS-3, Manual E-1 |
| FR4 | task0001 | Manual M-2, M-3 (trigger wiring); TS-1..TS-3 (recompute correctness) |
| FR5 | task0001 | TS-4, TS-5; tokens unchanged (M-4) |
| FR6 | task0001 | TS-5, TS-6, Manual M-3, E-2 |
| NFR1 | task0001 | go.mod/go.sum unchanged (M-5) |
| NFR2 | task0001 | Build/Test/Format commands (SC-5) |

## Manual Testing (E2E Not Possible)
- [ ] M-1: With the sensor config, rows hold multiple natural-width
      sections; column positions are NOT aligned across rows; order
      follows config order left→right, top→bottom.
- [ ] M-2: Resizing the window (wider/narrower) re-packs rows within a
      moment.
- [ ] M-3: Config reload rebuilds the layout without crash; section
      errors and per-section style overrides render unchanged.
- [ ] M-4: design-system/tokens.yaml / tokens.html are unchanged by this
      feature.
- [ ] M-5: go.mod / go.sum have no new dependencies.
- [ ] E-1: A section wider than the window occupies its own row and its
      output wraps.
- [ ] E-2: A config with zero sections renders an empty window without
      error.

## Verification Summary
| Category | Items | Automated | E2E | Manual |
|----------|-------|-----------|-----|--------|
| Build/Format | 2 | 2 | 0 | 0 |
| Unit tests | 6 | 6 | 0 | 0 |
| Manual/Edge | 7 | 0 | 0 | 7 |
