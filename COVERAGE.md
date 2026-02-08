# Test coverage guide

## Current state

- **Total coverage** (with all internal tests): ~44–50% when repositories pass; lower when one suite fails.
- **High-coverage packages**: `errors` (90.7%), `validation` (95.9%), `worker` (100%), `middleware` (75.3%), `integrations/northwind` (68.3%), `services` (67.8%).

## Reaching 80%+ coverage

### 1. Run tests without CGO (Windows / CI)

Tests use **pure-Go SQLite** ([glebarez/sqlite](https://github.com/glebarez/sqlite)) so you don’t need a C compiler:

- `internal/database/test_helper.go` – uses `github.com/glebarez/sqlite`
- `internal/repositories/transfer_repository_test.go` – uses glebarez
- `internal/models/transfer_test.go` – uses glebarez

Ensure dependencies are resolved:

```powershell
go mod tidy
go test ./internal/... -coverprofile=coverage.out -covermode=atomic
```

### 2. If one repository test is flaky

`TestTransferRepositoryTestSuite/TestFindByID_ExistingTransfer` can fail when the full suite runs (e.g. parallelism or decimal handling). Run repository tests with package parallelism 1:

```powershell
go test ./internal/repositories/... -coverprofile=coverage.out -p 1 -count=1
```

Or run the full internal suite with `-p 1`:

```powershell
go test ./internal/... -coverprofile=coverage.out -covermode=atomic -p 1 -count=1
```

### 3. Generate and open the HTML report

```powershell
go tool cover -html coverage.out -o coverage.html
# open coverage.html in a browser
go tool cover -func coverage.out   # summary by function
```

### 4. Optional: enable CGO for maximum compatibility

If you have a C toolchain (e.g. MinGW on Windows), you can keep using the original SQLite driver:

```powershell
$env:CGO_ENABLED = "1"
go test ./internal/... -coverprofile=coverage.out -covermode=atomic
```

With CGO, `gorm.io/driver/sqlite` (mattn/go-sqlite3) is used wherever it’s still imported.

### 5. Pushing toward 80%+

- **Validation** and **worker** are already well covered (95.9% and 100%).
- To raise the **overall** percentage:
  - Add tests in **handlers** (currently ~52%) and **config** (~47%).
  - Add edge-case and error-path tests in **services** and **repositories**.
  - Use the HTML report (`coverage.html`) to find red (uncovered) lines and add targeted tests.

### Quick commands

| Goal              | Command |
|-------------------|--------|
| Run all internal tests with coverage | `go test ./internal/... -coverprofile=coverage.out -covermode=atomic -count=1` |
| Run with less parallelism (more stable) | Add `-p 1` to the command above |
| Show total and per-function coverage | `go tool cover -func coverage.out` |
| Open HTML report | `go tool cover -html coverage.out -o coverage.html` |
