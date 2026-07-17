# Verification Document: window-drag-keep-above

## Overview
**Feature**: window-drag-keep-above / **SPEC.md**: `feature-docs/window-drag-keep-above/SPEC.md` / **IMPLEMENTATION.md**: `feature-docs/window-drag-keep-above/IMPLEMENTATION.md`

## Build Verification
- Command: `go build ./...`
- Expected: exit code 0, no errors

## Test Verification
- Command: `go test ./...`
- Coverage: logic additions (config default/validation/write-back, app routing, ui state derivation) unit-tested; GTK/X11 excluded per test/README.md

### Test Scenarios from SPEC.md
| ID | Scenario | Expected Result | Test Type |
|----|----------|-----------------|-----------|
| TS-1 | keep_above absent → true; false → false; non-bool → validation error | Defaulting + validation | Unit |
| TS-2 | UpdateKeepAbove flips only the key, preserves other keys and comments, creates key when absent, errors cleanly | Comment-preserving atomic write-back | Unit |
| TS-3 | startup/reload routes keep_above to ui seam; toggle → immediate apply + write-back call off main loop | App wiring | Unit |
| TS-4 | menu state derivation (check/sensitive/hint) from (enabled, supported) | Pure-function correctness | Unit |
| TS-5 | drag gesture registered window-level, capture phase, button 1 only | Constructive check | Unit |

## Code Quality Verification
- Format: `gofmt -l -w .` (expect no output)

## SPEC.md Compliance

### Success Criteria
| ID | Criterion | How to Verify |
|----|-----------|---------------|
| SC-1 | FR1-FR4 implemented | Coverage below + M-1..M-3 |
| SC-2 | TS-1..TS-5 pass | `go test ./...` |
| SC-3 | Manual scenarios verified | M-1..M-3 |
| SC-4 | gofmt clean; existing tests unaffected | format + full suite |

### Functional Requirements Coverage
| Requirement | Tasks | Verification |
|-------------|-------|--------------|
| FR1 | task0002 | TS-5, M-1 |
| FR2 | task0001, task0003 | TS-1, TS-3, M-2 |
| FR3 | task0001, task0002, task0003 | TS-2, TS-3, M-2 |
| FR4 | task0002 | TS-4, M-2, M-3 |
| NFR1 | task0002 | M-2, M-3 |
| NFR2 | task0001, task0002, task0003 | TS-1..TS-5 run headlessly |

## E2E Testing
None (GUI verified manually).

## Manual Testing (E2E Not Possible)
- [ ] M-1: Wayland と X11 の両方で、時計テキストの上・stream テキストの上・余白のそれぞれから左ドラッグしてウィンドウが動く。右クリックは従来どおりメニューが開く
- [ ] M-2: `GDK_BACKEND=x11` で起動 → デフォルトで他ウィンドウより手前に維持される。メニューの「最前面に固定」トグルで即時に解除/復帰し、config ファイルの keep_above が更新される（コメント保持）
- [ ] M-3: Wayland ネイティブ起動 → メニュー項目は表示されるが無効化（WM の「常に手前に表示」の案内付き）。クラッシュなし。起動が config の keep_above を書き換えない

## Verification Summary
| Category | Items | Automated | E2E | Manual |
|----------|-------|-----------|-----|--------|
| Build | 1 | 1 | 0 | 0 |
| Unit tests | 5 (TS-1..5) | 5 | 0 | 0 |
| Format | 1 | 1 | 0 | 0 |
| Manual | 3 (M-1..3) | 0 | 0 | 3 |
