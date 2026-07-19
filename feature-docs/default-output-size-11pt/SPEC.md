# Feature: Default Output Font Size 11pt

## Overview

Change the default font size of section output text (command output, monospace)
from 12pt to 11pt. The label default (9pt) and all config override mechanisms
stay unchanged.

## Objectives

- Render output text at 11pt when the user config specifies no font size
- Keep the design token SSOT (design-system/tokens.yaml) in sync with the code default

## User Stories

### US1: Default output text renders at 11pt
As a widget user, I want the output text to default to 11pt, so that sections
render at my preferred size without per-section config.

**Acceptance Criteria:**
- [ ] With no `style.font` / `output-size` in config, generated section CSS contains `font-size: 11pt`
- [ ] `label-size` default stays `9pt`
- [ ] A config override (e.g. `font: "monospace 12"`) still wins over the default

## Technical Requirements

### Functional Requirements
- **FR1:** `SectionStyleDefaults["output-size"]` in `internal/ui/style.go` is changed from `"12pt"` to `"11pt"`.
- **FR2:** The output font token in `design-system/tokens.yaml` (the entry with `family: "monospace"`) has its `size` updated from `"12pt"` to `"11pt"`.

### Non-Functional Requirements
- **NFR1 - Compatibility:** Config override behavior (`style.font`, `output-size`, `label-size`) is unchanged; only the default value changes.

## Implementation Approach

### Architecture

No structural change. Two value edits:

- `internal/ui/style.go`: `SectionStyleDefaults` map, key `output-size`
- `design-system/tokens.yaml`: output font token `size`

### Dependencies

**Internal Dependencies:**
- `internal/ui/style.go` CSS generation consumes `SectionStyleDefaults`

**External Dependencies:**
- None

### File Structure

```
internal/ui/style.go            # default value change
internal/ui/style_test.go       # add/adjust default-size assertion if needed
design-system/tokens.yaml       # token sync
```

## Test Scenarios

### Unit Tests
- [ ] Test 1: Default section CSS contains `font-size: 11pt` for output text
- [ ] Test 2: Default section CSS still contains `font-size: 9pt` for labels
- [ ] Test 3: With `font: "monospace 12"` override, output CSS contains `font-size: 12pt`

### Integration Tests
- [ ] None (GUI verified manually per test/README.md)

### E2E Tests
**Existing E2E tests**: None
**Run command**: Not detected
- [ ] Not applicable

### Edge Cases
- [ ] None (constant value change)

### Performance Tests
- [ ] Not applicable

## Security Considerations

Not applicable (no input handling changes).

## Error Handling

Not applicable (no new error paths).

## Success Criteria

- [ ] All functional requirements are implemented and tested
- [ ] `go test ./...` passes
- [ ] `design-system/tokens.yaml` and `SectionStyleDefaults` agree on 11pt

## Open Questions

None.

## References

- REQUIREMENTS.md: feature-docs/default-output-size-11pt/REQUIREMENTS.md
- internal/ui/style.go: SectionStyleDefaults
- design-system/tokens.yaml: font tokens
