# Verification Document: Default Output Font Size 11pt

## Overview

**Feature**: default-output-size-11pt / **SPEC.md**:
`feature-docs/default-output-size-11pt/SPEC.md` / **IMPLEMENTATION.md**:
`feature-docs/default-output-size-11pt/IMPLEMENTATION.md`

## Build Verification

- Command: `go build ./...`
- Expected: exit code 0, no errors

## Test Verification

- Command: `go test ./...`
- Coverage target: existing coverage maintained (value-change feature)

### Test Scenarios from SPEC.md

| ID | Scenario | Expected Result | Test Type |
|----|----------|-----------------|-----------|
| TS-1 | Generate section CSS with no style overrides | Output text font size is 11pt | Unit |
| TS-2 | Generate section CSS with no style overrides | Label font size is 9pt (unchanged) | Unit |
| TS-3 | Generate section CSS with a user font override of size 12 | Output text font size is 12pt (override wins) | Unit |
| TS-4 | Inspect the output font token in design-system/tokens.yaml | Token size states 11pt, matching the code default | Manual |

## Code Quality Verification

- Format: `gofmt -l -w .` (expect no diff after run)

## SPEC.md Compliance

### Success Criteria

| ID | Criterion | How to Verify |
|----|-----------|---------------|
| SC-1 | All functional requirements implemented and tested | TS-1〜TS-4 pass |
| SC-2 | `go test ./...` passes | Test run exit code 0 |
| SC-3 | tokens.yaml and code default agree on 11pt | TS-4 inspection |

### Functional Requirements Coverage

| Requirement | Tasks | Verification |
|-------------|-------|--------------|
| FR1 | task0001 | TS-1, TS-2 |
| FR2 | task0001 | TS-4 |
| NFR1 | task0001 | TS-3 |

## Manual Testing (E2E Not Possible)

- [ ] TS-4: `design-system/tokens.yaml` の monospace 出力トークンの size が
      11pt であることを確認する
- [ ] 実機で data-viewer を起動し、フォント未指定セクションの出力テキストが
      従来よりわずかに小さく（11pt で）表示されることを目視確認する
      （GUI は自動テスト対象外 — test/README.md）

## Verification Summary

| Category | Items | Automated | E2E | Manual |
|----------|-------|-----------|-----|--------|
| Build | 1 | 1 | 0 | 0 |
| Test scenarios | 4 | 3 | 0 | 1 |
| Format | 1 | 1 | 0 | 0 |
