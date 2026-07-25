# Verification Document: Whole-Window Opacity Toggle

## Overview

**Feature**: whole-window-opacity-toggle /
**SPEC.md**: `feature-docs/whole-window-opacity-toggle/SPEC.md` /
**IMPLEMENTATION.md**: `feature-docs/whole-window-opacity-toggle/IMPLEMENTATION.md`

## Build Verification

- Command: `PATH="$HOME/go/compiler/bin:$PATH" make build`
- Expected: exit code 0, no errors

## Test Verification

- Command: `PATH="$HOME/go/compiler/bin:$PATH" make test`
- Coverage target: no numeric project target; every acceptance criterion of
  both tasks must be covered by at least one test in `internal/ui`.

### Test Scenarios from SPEC.md

| ID | Scenario | Expected Result | Test Type |
|----|----------|-----------------|-----------|
| TS-1 | Merge a config that sets neither opacity key | Primary opacity `"1.0"`, alternate opacity `"0.45"` | Unit |
| TS-2 | Override each opacity key individually | The overridden key changes; the other stays at its default | Unit |
| TS-3 | Token/runtime drift guard over the two opacity tokens | Guard passes when token file and defaults agree, fails on a deliberate mismatch | Unit |
| TS-4 | Parse well-formed decimals in range, including `0.0` and `1.0` | The parsed value is returned | Unit |
| TS-5 | Parse unparsable input (empty string, non-numeric text) | The supplied fallback is returned | Unit |
| TS-6 | Parse out-of-range numbers (below 0.0, above 1.0) | The supplied fallback is returned (not clamped) | Unit |
| TS-7 | Select the effective opacity for each value of the active flag | Primary value when false, alternate value when true; input map unmutated | Unit |
| TS-8 | Flip the active pattern repeatedly from a fresh frame | Starts primary, alternates primary → alternate → primary; merged style map untouched | Unit |
| TS-9 | Sample config documents both opacity keys and round-trips through the merge path | Both keys present in the sample and effective after merging | Unit |
| TS-10 | Alternate-background key removed; a style map still carrying it is merged | Key absent from the defaults table, selection helper gone, merge succeeds and ignores the key | Unit |
| TS-11 | `Ctrl+T` match rule against the full modifier matrix | Accepts `t`/`T` with Control only; rejects Control+Alt, Control+Super, Control+other key, `t` without Control | Unit |
| TS-12 | Effective opacity a frame would apply, per active pattern | Primary configured value when primary is active, alternate configured value when alternate is active | Unit |
| TS-13 | Re-apply settings with new values while the alternate pattern is active | Alternate stays active and the newly configured alternate value becomes effective | Integration |
| TS-14 | Stylesheet built with each value of the active pattern | Identical output for both; window background is the configured primary background in both | Integration |

## Code Quality Verification

- Format: `PATH="$HOME/go/compiler/bin:$PATH" gofmt -l -w .`
- Static analysis: none configured beyond the compiler and `go test`.

## SPEC.md Compliance

### Success Criteria

| ID | Criterion | How to Verify |
|----|-----------|---------------|
| SC-1 | All functional requirements implemented and tested | Requirements coverage table below; all listed TS pass |
| SC-2 | All test scenarios pass | `make test` exit 0 with TS-1..TS-14 covered |
| SC-3 | The background-alt mechanism, token, docs and tests are fully removed | TS-10 plus a repository-wide search finding no remaining reference |
| SC-4 | A pre-change config file starts with an unchanged appearance | TS-1 (primary default `1.0` leaves the composited result unchanged) plus M-4 |
| SC-5 | Dimming covers text, not just the background | M-1 (manual — the only way to observe composited rendering) |

### Functional Requirements Coverage

| Requirement | Tasks | Verification |
|-------------|-------|--------------|
| FR1 | task0001 | TS-1, TS-2, TS-9 |
| FR2 | task0001 | TS-1 |
| FR3 | task0001 | TS-3 |
| FR4 | task0001 | TS-4, TS-5, TS-6 |
| FR5 | task0001, task0002 | TS-7, TS-12, TS-14 |
| FR6 | task0002 | TS-8 |
| FR7 | task0002 | TS-12, TS-14, M-3 |
| FR8 | task0002 | TS-11, M-5 |
| FR9 | task0002 | TS-13 |
| FR10 | task0001, task0002 | TS-10, TS-14 |
| NFR1 | task0001 | TS-1, TS-9, TS-10, M-4 |
| NFR2 | task0001, task0002 | TS-7, TS-8, TS-12 |
| NFR3 | task0002 | TS-14, M-3 |

## E2E Testing

No E2E framework exists in this project (`test/README.md` records that GUI
behavior is verified manually). No automatable E2E scenarios are defined.

## Manual Testing (E2E Not Possible)

These require a running GUI session and a compositor, so they cannot run in an
unattended batch execution. They are left for the human evaluator.

- [ ] M-1: With the widget running and focused, press `Ctrl+T` — the entire
      window fades: background, frame border, section labels and command
      output text all become translucent together, and the window behind is
      readable through them.
- [ ] M-2: Press `Ctrl+T` again — the window returns to the primary
      appearance.
- [ ] M-3: Across both presses, the window does not move or change size, and
      section output keeps updating.
- [ ] M-4: Start the widget with a config file written before this change —
      the initial appearance is indistinguishable from the previous release.
- [ ] M-5: With the window focused, press `Ctrl+Alt+T` — the compositor's own
      binding fires and the widget does not toggle.
- [ ] M-6: Set the alternate opacity to `0.0`, press `Ctrl+T` — the window
      disappears visually, and pressing `Ctrl+T` again brings it back.
- [ ] M-7: Edit the alternate opacity in the config file while the alternate
      pattern is active — the new level takes effect without restarting and
      without the window reverting to the primary pattern.

## Performance / Security Verification

Not applicable. The feature adds no I/O beyond reading two additional values
from the already-loaded config file, and no data crosses a trust boundary.

## Verification Summary

| Category | Items | Automated | E2E | Manual |
|----------|-------|-----------|-----|--------|
| Test scenarios | 14 | 14 | 0 | 0 |
| Success criteria | 5 | 4 | 0 | 2 (SC-4 partly, SC-5 fully) |
| Manual checks | 7 | 0 | 0 | 7 |
