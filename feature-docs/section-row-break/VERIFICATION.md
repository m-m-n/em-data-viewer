# Verification Document: Section Row Break

## Overview

**Feature**: section-row-break /
**SPEC.md**: `feature-docs/section-row-break/SPEC.md` /
**IMPLEMENTATION.md**: `feature-docs/section-row-break/IMPLEMENTATION.md`

## Build Verification

- Command: `go build ./...`
- Expected: exit code 0, no errors

## Test Verification

- Command: `go test ./...`
- Expected: all packages pass

### Test Scenarios from SPEC.md

| ID | Scenario | Expected Result | Test Type |
|----|----------|-----------------|-----------|
| TS-1 | Load a config with `- type: break` between sections | Load succeeds; break present at the right position in the sections slice | Unit |
| TS-2 | Load a config with `type: unknown-value` | Validation error naming the value | Unit |
| TS-3 | Break entry carrying extra fields (command etc.) | Loads without error; stays a break | Unit |
| TS-4 | Config with two break entries | No duplicate-key error | Unit |
| TS-5 | App conversions with breaks present | Runner section list and rehydration list contain no break entries | Unit |
| TS-6 | Segment packing with a break where width remains | Row ends at the break; the following segment packs by width as before | Unit |
| TS-7 | Leading / trailing / consecutive breaks; breaks-only config | No empty rows produced | Unit |
| TS-8 | Divider counts over a break-produced assignment | Equal to a natural-wrap assignment of the same shape | Unit |

## Code Quality Verification

- Format: `gofmt -l -w .` (expected: no diffs)

## SPEC.md Compliance

### Success Criteria

| ID | Criterion | How to Verify |
|----|-----------|---------------|
| SC-1 | Break forces a new row at its position | TS-6; Manual M-1 |
| SC-2 | Segments wrap by width as before | TS-6 |
| SC-3 | Empty segments produce no rows | TS-7 |
| SC-4 | Break boundary renders like a natural wrap | TS-8; Manual M-1 |
| SC-5 | Unknown `type` value is a config error | TS-2 |

### Functional Requirements Coverage

| Requirement | Tasks | Verification |
|-------------|-------|--------------|
| FR1 | task0001 | TS-1, TS-2, TS-3 |
| FR2 | task0001 | TS-4, TS-5 |
| FR3 | task0001 | TS-6 (assignment reflects config break positions) |
| FR4 | task0001 | TS-6, TS-7 |
| FR5 | task0001 | TS-8 |
| NFR1 | task0001 | go.mod unchanged (no new dependencies) |
| NFR2 | task0001 | Build / test / format commands above |
| NFR3 | task0001 | New tests run headlessly via `go test ./...` |

## Manual Testing (E2E Not Possible)

- [ ] M-1: With a wide window and a config containing breaks, rows split
      at the break positions and the boundary (hdivider, spacing) looks
      identical to a natural width-driven wrap.

## Verification Summary

| Category | Items | Automated | E2E | Manual |
|----------|-------|-----------|-----|--------|
| Build / Quality | 2 | 2 | 0 | 0 |
| Test scenarios | 8 | 8 | 0 | 0 |
| Manual | 1 | 0 | 0 | 1 |
