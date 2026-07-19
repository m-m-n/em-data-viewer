# Verification Document: App Icon

## Overview

**Feature**: app-icon / **SPEC.md**: `feature-docs/app-icon/SPEC.md` /
**IMPLEMENTATION.md**: `feature-docs/app-icon/IMPLEMENTATION.md`

## Build Verification

- Command: `go build ./...`
- Expected: exit code 0, no errors

## Test Verification

- Command: `go test ./...`
- Coverage target: existing tests keep passing (no new Go logic is added by
  this feature; no coverage delta expected)

### Test Scenarios from SPEC.md

| ID | Scenario | Expected Result | Test Type |
|----|----------|-----------------|-----------|
| TS-1 | `go build ./...` after all merges | exit 0 | Integration |
| TS-2 | `go test ./...` after all merges | all pass | Unit |
| TS-3 | `make dpkg`; list deb contents | deb contains `usr/share/icons/hicolor/{48x48,64x64,128x128,256x256}/apps/em-data-viewer.png`, `usr/share/icons/hicolor/scalable/apps/em-data-viewer.svg`, `usr/share/applications/em-data-viewer.desktop` with the FR4 key set | Integration |
| TS-4 | Extract deb control archive | `postinst` and `postrm` present, executable, containing icon-cache and desktop-database refresh | Integration |
| TS-5 | Run `make icons` with rsvg-convert masked from PATH | non-zero exit, error message names rsvg-convert | Integration |
| TS-6 | After full `make dpkg`, run `git status --porcelain` | no generated artifact appears outside git-ignored paths | Integration |
| TS-7 | Re-run `make icons` twice | second run exits 0, PNGs overwritten | Integration |

## Code Quality Verification

- Format: `gofmt -l -w .` (no diff expected on committed files)

## SPEC.md Compliance

### Success Criteria

| ID | Criterion | How to Verify |
|----|-----------|---------------|
| SC-1 | All FRs implemented and tested | TS-1..TS-7 + M-1..M-2 all pass |
| SC-2 | No binary file added to git | TS-6 + `git ls-files` shows no new binary |

### Functional Requirements Coverage

| Requirement | Tasks | Verification |
|-------------|-------|--------------|
| FR1 | task0001 | TS-3 (scalable SVG shipped from assets/icon.svg); file tracked by git |
| FR2 | task0001 | TS-3, TS-5, TS-7 |
| FR3 | task0001 | TS-3 |
| FR4 | task0001 | TS-3 |
| FR5 | task0002 | TS-1 + M-1 |
| FR6 | task0001 | TS-4 |
| NFR1 | task0001 | TS-6 |
| NFR2 | task0002 | TS-2 (no gotk4 import added to logic packages) |

## E2E Testing

- Command (workflow.yaml `e2e_test_command`): `make dpkg`
- [ ] TS-3 / TS-4 / TS-6 are evaluated on this run's output.

## Manual Testing (E2E Not Possible)

GUI behavior is verified manually per project convention.

- [ ] M-1: after installing the built deb, the app list shows "eM Data
      Viewer" with the icon; launching shows the icon on the
      window/taskbar (X11).
- [ ] M-2: モックとの目視照合 — rendered icon matches the design mockup
      `feature-docs/app-icon/design/mockups/screen-app-icon.html` (open in
      a browser; compare the 48/64/128/256 strip).

## Verification Summary

| Category | Items | Automated | E2E | Manual |
|----------|-------|-----------|-----|--------|
| Build/Test | TS-1, TS-2 | 2 | 0 | 0 |
| Packaging | TS-3..TS-7 | 2 (TS-5, TS-7) | 3 | 0 |
| GUI | M-1, M-2 | 0 | 0 | 2 |
