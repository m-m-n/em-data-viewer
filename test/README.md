# Test Instructions for AI Agents

This document provides guidelines for AI agents when writing and executing tests.

## Test Framework

Go standard `testing` package (go test).

## Test Execution

### Unit Tests
```bash
go test ./...
```

### Integration Tests (if applicable)
None. GUI behavior is verified manually.

### E2E Tests (if applicable)
None.

## Test File Organization

Test files live next to the code under test (`foo.go` → `foo_test.go`),
in the same package.

## Writing Tests

### Test Naming Conventions

- Test functions: `TestXxx(t *testing.T)` where `Xxx` describes the unit and behavior
- Subtests: `t.Run("case description", ...)`

### Test Structure

Table-driven tests are preferred for functions with multiple input/output cases.

## Adding New Tests

Add cases to the existing table when testing the same function.
Create a new `_test.go` file next to new source files.

## E2E Test Guidelines (if applicable)

Not applicable. GTK/GUI layers are excluded from automated tests;
keep logic (config parsing, markup conversion, process management state)
in GUI-free packages so it stays testable with `go test`.

## Common Patterns

- Logic packages must not import gotk4 so they can be tested headlessly.
- Use `t.TempDir()` for file-watching / config tests.
