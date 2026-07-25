# Verification Document: Window Opacity Toggle

## Overview

**Feature**: window-opacity-toggle /
**SPEC.md**: `feature-docs/window-opacity-toggle/SPEC.md` /
**IMPLEMENTATION.md**: `feature-docs/window-opacity-toggle/IMPLEMENTATION.md`

## Build Verification

- Command: `PATH="$HOME/go/compiler/bin:$PATH" make build`
- Expected: exit code 0, no errors. (The make target regenerates the embedded
  icon resource first; a bare `go build` fails on a fresh worktree because
  that generated file is untracked.)

## Test Verification

- Command: `PATH="$HOME/go/compiler/bin:$PATH" make test`
- Coverage target: no numeric project target; every acceptance criterion in
  both task plans must be covered by at least one test.

### Test Scenarios from SPEC.md

| ID | Scenario | Expected Result | Test Type |
|----|----------|-----------------|-----------|
| TS-1 | A user config sets `window.style.background_alt` | The merged window style carries the user value for that key | Unit |
| TS-2 | A user config omits `background_alt` | The merged window style carries `#1e1e1e99`, and `background` is unchanged from its current default | Unit |
| TS-3 | Token/runtime drift guard runs | `color.surface-alt` exists in `design-system/tokens.yaml` and equals the runtime `background_alt` default | Unit |
| TS-4 | The selection rule runs with each flag value | The window background is the primary value when inactive and the alternate value when active; all other style entries are identical and the input map is unmutated | Unit |
| TS-5 | A frame is freshly constructed | The primary pattern is active | Unit |
| TS-6 | The toggle is invoked repeatedly | The active pattern alternates primary → alternate → primary, and the built stylesheet's window background follows it | Unit |
| TS-7 | The key match rule is given assorted key/modifier combinations | Only `t` (either case) with control and without alt/super is accepted | Unit |
| TS-8 | Window settings are re-applied while the alternate pattern is active | The alternate pattern stays active and the newly supplied alternate value is rendered | Unit |
| TS-9 | The selection rule receives a map with no `background_alt` entry and the flag active | The primary background value is used; no empty value is emitted | Unit |
| TS-10 | The shipped example config is loaded | It loads without error with `background_alt` documented | Unit |

## Code Quality Verification

- Format: `PATH="$HOME/go/compiler/bin:$PATH" gofmt -l -w .` — expected to
  leave no files reformatted at verification time.
- Static analysis: none configured for this project beyond the compiler.

## SPEC.md Compliance

### Success Criteria

| ID | Criterion | How to Verify |
|----|-----------|---------------|
| SC-1 | All functional requirements implemented and covered by tests | Requirement coverage table below plus the test run |
| SC-2 | `make test` passes | Test Verification above |
| SC-3 | Existing configs without `background_alt` run unchanged | TS-2, TS-10, and manual check M-1 |
| SC-4 | Manual verification passes on the desktop | Manual Testing section below |

### Functional Requirements Coverage

| Requirement | Tasks | Verification |
|-------------|-------|--------------|
| FR1 | task0001 | TS-1 |
| FR2 | task0001 | TS-2 |
| FR3 | task0001 | TS-3 |
| FR4 | task0001, task0002 | TS-4, TS-9 |
| FR5 | task0002 | TS-5 |
| FR6 | task0002 | TS-6, M-2 |
| FR7 | task0002 | TS-7, M-4 |
| FR8 | task0002 | TS-8, M-5 |
| NFR1 | task0001 | TS-2, TS-10, M-1 |
| NFR2 | task0001, task0002 | TS-4, TS-7 (both run headlessly under `make test`) |
| NFR3 | task0002 | TS-6, M-3 |

## E2E Testing

The project has no E2E framework (`test/README.md`: GUI behavior is verified
manually). No E2E scenarios are defined for this feature.

## Manual Testing (E2E Not Possible)

- [ ] M-1: Start the app with the current example config — the window looks
      exactly as it did before this feature.
- [ ] M-2: Focus the window and press `Ctrl+T` — the background becomes
      visibly more transparent; pressing it again restores the original.
- [ ] M-3: During M-2, the window's position, width and height do not change
      and the content does not reflow or flicker.
- [ ] M-4: With the window focused, drag it, open the right-click context
      menu, and scroll over it — all behave as before.
- [ ] M-5: With the alternate pattern showing, edit and save the config file
      — after the live reload the alternate pattern is still showing.
- [ ] M-6: Set a distinctive `window.style.background_alt` value, restart, and
      confirm `Ctrl+T` selects exactly that value.

## Performance / Security Verification

Not applicable: no new I/O, no new external input, no new dependency. The
feature reads no data it did not already read.

## Verification Summary

| Category | Items | Automated | E2E | Manual |
|----------|-------|-----------|-----|--------|
| Test scenarios | 10 | 10 | 0 | 0 |
| Success criteria | 4 | 3 | 0 | 1 |
| Manual checks | 6 | 0 | 0 | 6 |
