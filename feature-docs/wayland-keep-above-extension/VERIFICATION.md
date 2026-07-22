# Verification Document: Wayland Keep-Above GNOME Shell Extension

## Overview

**Feature**: wayland-keep-above-extension /
**SPEC.md**: `feature-docs/wayland-keep-above-extension/SPEC.md` /
**IMPLEMENTATION.md**: `feature-docs/wayland-keep-above-extension/IMPLEMENTATION.md`

## Build Verification

- Command: `go build ./...`
- Expected: exit code 0, no errors (NFR2 — the Go application is untouched)

## Test Verification

- Command: `go test ./...`
- Coverage target: no change from baseline (this feature adds no Go code;
  the GJS extension is excluded from automated tests per project policy)

### Test Scenarios from SPEC.md

| ID | Scenario | Expected Result | Test Type |
|----|----------|-----------------|-----------|
| TS-1 | Run `go test ./...` after the feature lands | Exit 0; identical result to baseline | Unit (existing suite) |
| TS-2 | Inspect `contrib/gnome-shell-extension/keep-above@em-data-viewer.m-m-n.github.io/`: metadata.json parses as JSON and declares uuid `keep-above@em-data-viewer.m-m-n.github.io` + shell-version `["50"]`; extension.js is ESM with a default-exported extension class and no non-GNOME imports | All checks hold | Static inspection |
| TS-3 | Inspect the Makefile: `install-extension` / `uninstall-extension` targets exist, are `.PHONY`, probe for `gnome-extensions`, copy/remove the pinned UUID directory, and existing target recipes are unmodified | All checks hold | Static inspection |

## Code Quality Verification

- Format: `gofmt -l -w .` (no Go files should change)
- Static analysis: none configured

## SPEC.md Compliance

### Success Criteria

| ID | Criterion | How to Verify |
|----|-----------|---------------|
| SC-1 | FR1–FR6 implemented | TS-2, TS-3, M-1..M-6 |
| SC-2 | Manual scenarios pass on GNOME Shell 50 / Wayland | M-1..M-7 |
| SC-3 | Go build/tests unaffected | TS-1, `go build ./...` |

### Functional Requirements Coverage

| Requirement | Tasks | Verification |
|-------------|-------|--------------|
| FR1 | task0001 | TS-2 |
| FR2 | task0001, task0003 | M-2, M-6, M-8 (+ late-id / either-id edge cases) |
| FR3 | task0001 | M-3 |
| FR4 | task0001 | M-4 |
| FR5 | task0002, task0003 | TS-3, M-1 |
| FR6 | task0002 | TS-3, M-5 |
| NFR1 | task0001 | TS-2, M-2 |
| NFR2 | task0001, task0002 | TS-1, M-7 |

## E2E Testing

None (project has no E2E infrastructure; GUI behavior is verified
manually).

## Manual Testing (E2E Not Possible)

Environment: GNOME Shell 50 / Wayland session, em-data-viewer installed.

- [ ] M-1: `make install-extension` copies the extension and enables it;
      a fresh install prints the re-login guidance and exits 0
- [ ] M-2: After re-login, launching em-data-viewer yields an
      always-on-top window (Alt+Space menu shows "Always on Top" checked)
- [ ] M-3: With em-data-viewer already running, enabling the extension
      sets the existing window above
- [ ] M-4: Disabling the extension reverts the window (no longer above)
- [ ] M-5: `make uninstall-extension` disables the extension and removes
      its directory; running it again (not installed) exits 0
- [ ] M-6: Other applications' windows are never set above
- [ ] M-7: `go test ./...` and `make build` behave exactly as before
- [ ] M-8: A window presenting the target app id from a different owning
      executable (spoofed id) is not set above
- [ ] Edge: a matching window whose app id arrives late is still set above
- [ ] Edge: `make install-extension` run twice in a row succeeds with no
      stale files
- [ ] Edge: manually un-setting "Always on Top" via Alt+Space is not
      re-applied for that existing window
- [ ] Edge: a running instance whose binary was replaced on disk
      (` (deleted)` proc marker) still passes process verification
- [ ] Edge: enable failure with the uuid still known to the Shell fails
      `install-extension` even when the install directory was absent

## Verification Summary

| Category | Items | Automated | E2E | Manual |
|----------|-------|-----------|-----|--------|
| Build/Test/Format | TS-1 + build + format | 3 | 0 | 0 |
| Static inspection | TS-2, TS-3 | 0 | 0 | 2 |
| Manual scenarios | M-1..M-8 + 4 edges | 0 | 0 | 12 |
