# Verification Document: App Icon Binary Embedding

## Overview

**Feature**: app-icon-embed / **SPEC.md**: `feature-docs/app-icon-embed/SPEC.md` /
**IMPLEMENTATION.md**: `feature-docs/app-icon-embed/IMPLEMENTATION.md`

## Build Verification

- Command: `make build`
- Expected: exit code 0; produces `internal/ui/icon.gresource` and the
  binary, starting from a tree where the bundle is absent

## Test Verification

- Command: `make test`
- Coverage target: existing tests keep passing; new pure-data tests (if
  any) pass

### Test Scenarios from SPEC.md

| ID | Scenario | Expected Result | Test Type |
|----|----------|-----------------|-----------|
| TS-1 | Clean tree (no bundle): `make build` | exit 0, bundle + binary produced | Integration |
| TS-2 | Clean tree (no bundle): `make test` | exit 0, all tests pass | Integration |
| TS-3 | Re-run `make resources` twice | second run exit 0, outputs overwritten | Integration |
| TS-4 | `make resources` with `glib-compile-resources` masked from PATH | non-zero exit, error names the tool | Integration |
| TS-5 | `make resources` with `rsvg-convert` masked from PATH | non-zero exit, error names the tool | Integration |
| TS-6 | After full `make build`, `git status --porcelain` | no generated artifact appears (bundle ignored) | Integration |
| TS-7 | `gresource list internal/ui/icon.gresource` | exactly six entries `/net/laser5/em-data-viewer/icons/{S}x{S}/apps/em-data-viewer.png`, S ∈ {16,32,48,64,128,256} | Integration |
| TS-8 | `make dpkg`; inspect payload | hicolor icons / .desktop / postinst / postrm structure identical to app-icon feature output | Integration |

## Code Quality Verification

- Format: `gofmt -l -w .` (no diff expected on committed files)

## SPEC.md Compliance

### Success Criteria

| ID | Criterion | How to Verify |
|----|-----------|---------------|
| SC-1 | All FRs implemented and tested | TS-1..TS-8 + M-1 pass |
| SC-2 | No binary file added to git | TS-6 + `git ls-files` gains no binary |

### Functional Requirements Coverage

| Requirement | Tasks | Verification |
|-------------|-------|--------------|
| FR1 | task0001 | TS-1, TS-3, TS-4, TS-5, TS-7 |
| FR2 | task0001 | TS-1, TS-2 |
| FR3 | task0001 | TS-2 + M-1 (+ code review for the log-and-continue failure path) |
| FR4 | task0001 | TS-6 |
| NFR1 | task0001 | TS-6 |
| NFR2 | task0001 | TS-8 |

## E2E Testing

- Command (workflow.yaml `e2e_test_command`): `make dpkg`
- [ ] TS-8 is evaluated on this run's output.

## Manual Testing (E2E Not Possible)

- [ ] M-1: with hicolor icons NOT installed (deb uninstalled), launch the
      `make build` binary; the dock/taskbar and window show the app icon
      (X11).

## Verification Summary

| Category | Items | Automated | E2E | Manual |
|----------|-------|-----------|-----|--------|
| Build/Test | TS-1, TS-2, TS-3 | 3 | 0 | 0 |
| Tooling failure | TS-4, TS-5 | 2 | 0 | 0 |
| Artifacts | TS-6, TS-7 | 2 | 0 | 0 |
| Packaging | TS-8 | 0 | 1 | 0 |
| GUI | M-1 | 0 | 0 | 1 |
