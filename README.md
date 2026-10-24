# Payroll Batch (Go port of COBOL PAYROLL)

This is a Go port of a COBOL payroll batch program. It is a command-line batch job, not a server. It runs once, writes its report to standard output, and exits with a status code. The program reads employee records from `employees.txt` and prints a payroll total. If a file operation fails, it aborts with a non-zero status and does not print the total.

Migration status:

| Item | Status |
|---|---|
| Modules migrated | 1/1 |
| Build | passed |
| Target unit tests | passed |
| Behavior compared with the original COBOL program | passed |
| Confidence | 97%, self-assessed by the migration model. This is not a measured result. |

## Tech stack

| Component | Before | After |
|---|---|---|
| Language | COBOL (GnuCOBOL runtime) | Go (module `migrated-app`, `go 1.23` minimum in `go.mod`) |
| Framework | none | Go standard library |
| Dependencies | n/a | `github.com/spf13/viper v1.19.0` plus its indirect dependencies (see `go.mod` / `go.sum`) |

## Prerequisites

- Go 1.25. The migration was built and tested with the `golang:1.25-alpine` image. `go.mod` declares `go 1.23` as the minimum.
- A POSIX shell, for the commands below.

No database is needed to run the batch job (see [Environment variables](#environment-variables)).

## Getting started

Run all commands from the root of a fresh clone.

### 1. Install dependencies and build

```sh
go mod download
mkdir -p bin
go build -o bin/app ./cmd/app
```

### 2. Environment setup

Running the batch job needs no environment variables. The configuration loader in `internal/config/config.go` reads `DATABASE_URL`, `PORT` and `JWT_SECRET`, but neither entry point (`cmd/app/main.go`, `cmd/payroll/main.go`) imports it. See [Environment variables](#environment-variables).

### 3. Database setup

None. No database setup step was required or run during migration.

### 4. Run

Run the binary from the repository root so that it finds `employees.txt`:

```sh
./bin/app
```

The program does not listen on any port. It processes the input, writes to stdout, and exits.

### A first run to try

This invocation was verified to behave the same as the original program:

```
RUN (no arguments) -> 0 with body {"args": [], "stdin": ""}
```

The equivalent shell command runs with no arguments and empty stdin, and expects exit status `0`:

```sh
./bin/app < /dev/null
echo "exit status: $?"
```

## Running tests

```sh
go test ./...
```

The tests are:

- `cmd/payroll/main_test.go`: smoke tests for the entry point.
  - `TestRunDoesNotPanic` checks that `run(nil)` does not panic and returns a non-negative exit code.
  - `TestRunDeterministic` checks that two runs produce identical output and exit codes.
  - Both tests run in an empty temporary directory, so they do not use the repository's `employees.txt`.
- `internal/payroll/payroll_test.go`: unit tests for the payroll logic.

## Environment variables

`internal/config.Load()` reads these variables. Neither entry point calls `config.Load()`, so they have no effect on the batch run unless `internal/payroll` uses them (see [Manual review required](#manual-review-required)).

| Variable | Default | Used by | Notes |
|---|---|---|---|
| `DATABASE_URL` | empty | `internal/config` | No database access is part of the batch run. |
| `PORT` | `8080` | `internal/config` | The program starts no network listener. |
| `JWT_SECRET` | empty | `internal/config` | No authentication exists in the batch flow. |

## Architecture overview

```
cmd/app/main.go              Entry point built by the install step (bin/app)
cmd/payroll/main.go          Second entry point, functionally identical to cmd/app
cmd/payroll/main_test.go     Smoke tests for run()
internal/payroll/payroll.go  Port of the COBOL PAYROLL program logic
internal/payroll/payroll_test.go
internal/config/config.go    Env-based config loader (DATABASE_URL, PORT, JWT_SECRET)
employees.txt                Employee input data file
go.mod / go.sum              Module definition and dependency checksums
```

Execution flow, which is the same in both entry points:

1. `main()` calls `run(os.Args[1:])` and exits with the returned code.
2. `run` creates a context that is cancelled on `SIGINT`/`SIGTERM`.
3. `run` wraps stdout in a buffered writer and calls the payroll entry function in `internal/payroll/payroll.go` with `(ctx, args, os.Stdin, out, os.Stderr)`.
4. Buffered stdout is flushed. If the flush fails and the payroll entry function returned `0`, the exit code becomes `1`.

All business logic lives in `internal/payroll`. The `cmd/` packages are thin wrappers.

## Migration notes

Source: COBOL (GnuCOBOL runtime, no framework). Target: Go, standard library.

**File error reporting changed.**

- **New behavior:** When `employees.txt` (or any file) cannot be opened, read, written or closed, the program prints a Go-style wrapped error to stderr and exits with status `1`. For example:

  ```
  payroll: open employees.txt: ...
  ```

- **Old behavior:** The original printed the GnuCOBOL runtime's own file-error message.
- **Unchanged:** The total is still not printed when a file operation fails.
- **Reason:** The exact text of a COBOL runtime file error depends on the runtime and is not part of the program's contract. Only two things were kept:
  - the abort with a non-zero status;
  - the absence of the total.

No source files were left untranslated, and no components were classified as unmigratable.

## Known limitations

- **Not a server.** The migration tooling described the run step as "listens on port 8080", but the entry code starts no listener. `PORT` and the other variables in `internal/config` are not used by the entry points.
- **Two identical entry points.** `cmd/app` and `cmd/payroll` contain the same `run`/`main` code. Only `cmd/app` is built by the install step, and only `cmd/payroll` has tests.
- **Error text differs from the original.** File-error messages are worded differently from GnuCOBOL's (see [Migration notes](#migration-notes)). Any script that parses stderr from the original program needs updating.
- **Limited verification.** Behavior comparison with the original was verified only for the no-argument, empty-stdin invocation listed above.

## Manual review required

- **`internal/payroll/payroll.go`.** Confirm the following match the original COBOL program:
  - the arithmetic (rounding and decimal precision of pay calculations);
  - the record layout parsing for `employees.txt`;
  - the output formatting.
- **`employees.txt`.** Confirm the file format and location (relative to the working directory) match what production jobs will supply.
- **File-error paths.** Exercise missing, unreadable and unwritable file cases. Confirm each one exits with status `1` and prints no total.
- **`internal/config/config.go` and the `viper` dependency.**
  - The config package uses only `os`, yet `go.mod` requires `github.com/spf13/viper`.
  - Determine whether `internal/payroll` uses either one.
  - If neither is used, consider removing the config package and the dependency.
- **`cmd/app` vs `cmd/payroll`.** Decide which entry point to keep, and make sure the kept one is both built and tested.
- **`cmd/payroll/main_test.go`.** These tests only check for panics and determinism, not output values. Add tests that compare output against known-good results from the original program.