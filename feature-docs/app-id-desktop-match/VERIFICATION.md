# Verification Document: Application ID / .desktop Filename Match

## Overview

**Feature**: app-id-desktop-match / **SPEC.md**: `feature-docs/app-id-desktop-match/SPEC.md` /
**IMPLEMENTATION.md**: `feature-docs/app-id-desktop-match/IMPLEMENTATION.md`

## Build Verification

- Command: `make build`
- Expected: exit code 0

## Test Verification

- Command: `make test`
- Coverage target: existing tests keep passing

### Test Scenarios from SPEC.md

| ID | Scenario | Expected Result | Test Type |
|----|----------|-----------------|-----------|
| TS-1 | `make build` / `make test` | both exit 0 | Integration |
| TS-2 | grep the source tree for `org.dataviewer.Widget` | no occurrence outside feature-docs | Integration |
| TS-3 | `make dpkg`; list deb contents | contains `usr/share/applications/io.github.m_m_n.EmDataViewer.desktop`; NO `em-data-viewer.desktop` | Integration |
| TS-4 | Extract the shipped .desktop | content byte-identical to previous nine lines (incl. `Icon=em-data-viewer`, `StartupWMClass=em-data-viewer`), mode 644 | Integration |
| TS-5 | Deb payload structure | hicolor icons / postinst / postrm unchanged | Integration |
| TS-6 | grep icon-name usages (Makefile hicolor targets, gresource XML, windowIconName) | all still `em-data-viewer` | Integration |

## Code Quality Verification

- Format: `gofmt -l -w .` (no diff expected on committed files)

## SPEC.md Compliance

### Success Criteria

| ID | Criterion | How to Verify |
|----|-----------|---------------|
| SC-1 | All FRs implemented and tested | TS-1..TS-6 + M-1 pass |

### Functional Requirements Coverage

| Requirement | Tasks | Verification |
|-------------|-------|--------------|
| FR1 | task0001 | TS-1, TS-2 |
| FR2 | task0001 | TS-3, TS-4 |
| NFR1 | task0001 | TS-6 |
| NFR2 | task0001 | TS-5 |

## E2E Testing

- Command (workflow.yaml `e2e_test_command`): `make dpkg`
- [ ] TS-3 / TS-4 / TS-5 are evaluated on this run's output.

## Manual Testing (E2E Not Possible)

- [ ] M-1: on the Wayland session, reinstall the deb, relaunch the app;
      the dock shows the app icon (gear gone).

## Verification Summary

| Category | Items | Automated | E2E | Manual |
|----------|-------|-----------|-----|--------|
| Build/Test | TS-1, TS-2 | 2 | 0 | 0 |
| Packaging | TS-3..TS-6 | 1 (TS-6) | 3 | 0 |
| GUI | M-1 | 0 | 0 | 1 |
