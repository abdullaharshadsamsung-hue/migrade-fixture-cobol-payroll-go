# Payroll Batch (Go port of COBOL PAYROLL)

A Go port of the COBOL `PAYROLL` batch program. This is a **command-line batch job**, not a server. It reads employee records from `employees.txt` in the current working directory, writes its report (including a total) to stdout, and exits with a status code. If a file operation fails, it writes an error to stderr, exits with status `1`, and does not print the total.

The repository contains a sample input file, `employees.txt`, at its root.

## Migration status

| Item | Status |
|---|---|
| Source → target | COBOL (no framework) → Go (standard library) |
| Modules migrated | 1 / 1 |
| Build | Passed |
| Unit tests (`go test ./...`) | Passed |
| Behavior compared with the original | Passed (see scope below) |
| Confidence | 97%. This is the migration tool's own estimate, not a measured result |

The behavior comparison ran the program with no arguments and empty stdin. Both the original and the Go port returned exit status `0`. No other input scenarios were compared.

## Tech stack

- **Language:** Go. The module declares `go 1.23`. The migration was built and tested with the Go 1.25 toolchain (`golang:1.25-alpine`).
- **Module name:** `migrated-app`
- **Dependencies:** Standard library for the entry points and config shown. `go.mod` also requires `github.com/spf13/viper v1.19.0` and its indirect dependencies.
- **Database:** None required to run the batch job.

## Prerequisites

- Go 1.25 (the toolchain used during migration; the module itself requires ≥ 1.23)
- A POSIX shell (the commands below use `sh`/`bash` syntax)

## Getting started

Run all commands from the root of a fresh clone.

### 1. Download dependencies and build

```sh
go mod download
mkdir -p bin
go build -o bin/app ./cmd/app
```

This produces the binary `bin/app`.

### 2. Environment setup

The batch job does not need any environment variables. The entry point in `cmd/app/main.go` does not load `internal/config`. See [Environment variables](#environment-variables) for the variables that package defines.

### 3. Database setup

No database setup step is required.

### 4. Run

Run the program from the repository root so it can find `employees.txt`:

```sh
./bin/app
echo "exit status: $?"
```

There is no port to connect to. The program reads its input, writes its report to stdout, and exits.

**First run to try:** This is the verified case. Run it with no arguments and empty stdin:

```sh
./bin/app < /dev/null
echo "exit status: $?"
```

Expected: exit status `0`, which matches the original COBOL program.

To run it without building a binary first:

```sh
go run ./cmd/app
# or the equivalent entry point:
go run ./cmd/payroll
```

## Running tests

```sh
go test ./...
```

This runs:

- `internal/payroll/payroll_test.go`, the tests for the payroll logic.
- `cmd/payroll/main_test.go`, smoke tests for the CLI wrapper. They run in an empty temporary directory and check two things: that `run` does not panic, and that two runs produce the same exit code and output. They do not check report contents.

## Environment variables

These variables are read by `config.Load()` in `internal/config/config.go`. Neither entry point (`cmd/app/main.go`, `cmd/payroll/main.go`) calls `config.Load()`, so they have no effect on the batch run.

| Variable | Required | Default | Read by | Notes |
|---|---|---|---|---|
| `DATABASE_URL` | No | empty | `internal/config` | Not used by the batch job |
| `PORT` | No | `8080` | `internal/config` | Not used. The program does not listen on any port |
| `JWT_SECRET` | No | empty | `internal/config` | Not used by the batch job |

## Architecture overview

```
cmd/
  app/main.go          Entry point built as bin/app
  payroll/main.go      Entry point for the "payroll" command (same code as cmd/app)
  payroll/main_test.go CLI smoke tests (no panic, deterministic output)
internal/
  payroll/payroll.go       Go translation of the COBOL PAYROLL program
  payroll/payroll_test.go  Tests for the payroll logic
  config/config.go         Env-based config loader (not wired into the entry points)
employees.txt          Sample input file read from the working directory
go.mod / go.sum        Module definition (module "migrated-app")
```

**How a run works:**

1. `main()` calls `run(os.Args[1:])` and exits with the returned code.
2. `run` creates a context that is cancelled on `SIGINT` or `SIGTERM`. It wraps stdout in a buffered writer.
3. All program logic is in `Main(ctx, args, stdin, stdout, stderr)` in `internal/payroll/payroll.go`, which returns the exit code.
4. After `Main` returns, stdout is flushed. If the flush fails and the code was `0`, the exit code becomes `1`.

`cmd/app` and `cmd/payroll` contain identical `run`/`main` logic. Both delegate to `internal/payroll`.

## Migration notes

- **File error reporting changed.** When `employees.txt` (or another file the program uses) cannot be opened, read, written, or closed, the Go port does the following:
  - It writes a Go-style wrapped error to stderr, for example `payroll: open employees.txt: ...`.
  - It exits with status `1`.
  - It does not print the total.

  The original program showed the GnuCOBOL runtime's own file-error message instead.

  **Reason:** The exact text of a COBOL runtime file error depends on the runtime and is not part of the program's contract. The parts that are kept are the ones the contract covers: the abort with a non-zero status and the missing total.
- **No source files were left untranslated.**

## Known limitations

- **Behavior comparison was narrow.** The only scenario compared against the original was a no-argument run with empty stdin. Those runs were checked for exit status (`0`). Other inputs, malformed records, and error paths were not compared against the original.
- **CLI tests are smoke tests.** `cmd/payroll/main_test.go` checks only for panics and determinism, not output correctness.
- **Unused config package.** `internal/config` defines `DATABASE_URL`, `PORT`, and `JWT_SECRET`, but nothing in the entry points uses them. The program does not listen on port 8080 or any other port.
- **Duplicate entry points.** `cmd/app` and `cmd/payroll` duplicate each other.

## Manual review required

- **`internal/payroll/payroll.go`:** Check calculations, rounding, and numeric formatting against the original COBOL. Pay particular attention to fixed-point `PIC` fields, which do not map directly to Go types. Use inputs beyond the single verified scenario.
- **`internal/payroll/payroll_test.go`:** Confirm the tests cover the original's record layouts, edge cases, and the total line.
- **Error path:** Confirm that downstream consumers do not parse stderr text. The wording of file errors changed (see Migration notes).
- **`internal/config/config.go`:** Decide whether to wire this package in or remove it. It currently has no effect on the batch job.
- **`go.mod`:** It declares `github.com/spf13/viper`. Confirm this dependency is actually imported; remove it if not.
- **`cmd/app` vs `cmd/payroll`:** Pick one entry point to keep and remove the duplicate.