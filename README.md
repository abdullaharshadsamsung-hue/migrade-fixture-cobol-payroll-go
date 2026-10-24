# Payroll Batch (Go port of COBOL `PAYROLL`)

A Go port of a COBOL payroll batch program. It reads fixed-width employee records from `employees.txt` in the current working directory and calculates each employee's gross pay, including overtime. It writes one pay line per employee to an output file and prints the grand total to stdout.

This is a **command-line batch program, not a server**. It runs, processes the input file and exits. It does not listen on a port.

## Verification status

| Check | Status |
|---|---|
| Build | Passed |
| Target unit tests | **Not run** |
| Behavior compared with the original COBOL program | Passed. The program was run with no arguments and empty stdin, and exited with status 0 like the original. |

Migration confidence is the migration model's self-assessment (96%). It is not a measured result. Modules migrated: 1/1.

## Tech stack

- Go (module `migrated-app`, `go 1.23` minimum in `go.mod`)
- Standard library only on the execution path (`bufio`, `context`, `os`, `os/signal`, `syscall`)
- `github.com/spf13/viper v1.19.0` is listed in `go.mod`, but none of the code shown here imports it. See [Manual review required](#manual-review-required).

## Prerequisites

- Go 1.25. The migration was built and checked with the `golang:1.25-alpine` toolchain. `go.mod` declares Go 1.23 as the minimum.
- A POSIX shell (the commands below use `mkdir -p`).

No database is required.

## Getting started

Run all commands from the root of a fresh clone.

### 1. Download dependencies

```sh
go mod download
```

### 2. Build

```sh
mkdir -p bin && go build -o bin/app ./cmd/app
```

### 3. Environment setup

The payroll program does not need any environment variables. See [Environment variables](#environment-variables).

### 4. Database setup

None. The program reads and writes local files only.

### 5. Run

The program reads `employees.txt` from the **current working directory**, so run it from the repository root, where a sample `employees.txt` is checked in:

```sh
./bin/app
```

There is no port to connect to. The program processes the file, prints a `TOTAL` line to stdout and exits.

### A first run to try

The verified case is a run with no arguments and empty stdin. The migrated program exited with status `0`, matching the original:

```sh
./bin/app < /dev/null
echo "exit status: $?"
```

On success stdout contains one line of the form:

```
TOTAL      1950.42
```

The amount depends on the contents of `employees.txt`. Per-employee pay lines go to the output file, not to stdout.

### Alternative entry point

`cmd/payroll/main.go` is functionally identical to `cmd/app/main.go`. If you prefer that command name, you can build it the same way:

```sh
go build -o bin/payroll ./cmd/payroll
```

## Running tests

```sh
go test ./...
```

**Unit tests have not been run as part of this migration.** Expect `./cmd/payroll` to fail to compile under `go test`. See [Known limitations](#known-limitations).

## Environment variables

`internal/config/config.go` reads the variables below. However, neither `cmd/app/main.go` nor `cmd/payroll/main.go` imports `internal/config`, so **the payroll program does not use any of them**. They came from the migration scaffolding.

| Variable | Default | Read by | Used by payroll run? |
|---|---|---|---|
| `DATABASE_URL` | empty | `internal/config` | No |
| `PORT` | `8080` | `internal/config` | No (no server is started) |
| `JWT_SECRET` | empty | `internal/config` | No |

## Architecture overview

```
cmd/app/main.go              Entry point built by the install step (bin/app)
cmd/payroll/main.go          Identical entry point under the name "payroll"
cmd/payroll/main_test.go     Tests for the payroll logic (see Known limitations)
internal/payroll/payroll.go  Ported COBOL logic: parsing, pay calculation, formatting, file I/O
internal/payroll/payroll_test.go
internal/config/config.go    Env-based config loader (currently unused)
employees.txt                Sample input file
go.mod / go.sum              Module definition
```

### Entry points

Both `main.go` files do the same thing:

1. Create a context that is cancelled on `SIGINT`/`SIGTERM`.
2. Call `payroll.Main(ctx, args, os.Stdin, out, os.Stderr)`, where `out` is a buffered stdout.
3. Flush stdout. If the flush fails and the program would otherwise have succeeded, the exit status becomes `1`.
4. Exit with the returned code.

### Payroll logic (`internal/payroll`)

The behavior below is taken from the test suite.

**Input record** (one per line, fixed-width, CRLF tolerated):

```
IIII HHH RRRRR
```

| Field | Meaning |
|---|---|
| `IIII` | Employee ID, 4 characters |
| `HHH` | Hours worked, 3 digits |
| `RRRRR` | Hourly rate in cents, 5 digits |

Parsing behaves like the COBOL fixed-width fields:
- Short lines are padded.
- Characters past column 14 are ignored.
- Spaces inside numeric fields decode as zeros.
- Non-numeric input does not panic and never decodes to a negative value.

**Pay calculation (`PayCents`):**
- Hours up to 40 are paid at the rate.
- Hours above 40 are paid at 1.5× the rate.
- Half cents round up. For example, 41 h at 1¢ gives 42¢.
- Integer cents are used throughout, with no overflow at the maximum input (999 h × 99999¢).

**Output:**
- **Output file:** one line per input record, in input order, written as `ID` + space + pay. The pay is right-aligned in 10 characters with 2 decimals, for example `0001    1000.00`. An existing output file is overwritten. The file name is the `outputPath` constant in `internal/payroll/payroll.go`.
- **stdout:** a single `TOTAL` line. The total is right-aligned in 12 characters, for example `TOTAL      1950.42`. For empty input it is `TOTAL          0.00`.

**Failures:**
- If `employees.txt` is missing, the run fails and no output file is created.
- Read errors, write errors and context cancellation abort the run, and no total is printed.

## Migration notes

Original: COBOL (GnuCOBOL runtime), no framework. Target: Go, standard library.

### Changes from the original

- **File error messages.**
  - **What changed:** when a file cannot be opened, read, written or closed, the Go program prints a Go-style wrapped error to stderr and exits with status `1`, for example `payroll: open employees.txt: ...`. The original printed the GnuCOBOL runtime's own file-error message instead. In both versions the total is not printed on failure.
  - **Why:** the exact text of a COBOL runtime file error depends on the runtime and is not part of the program's contract. Only the abort with a non-zero status and the missing total were kept.

### Source files not translated

None.

## Known limitations

- **`cmd/payroll/main_test.go` does not match its directory.** This is very likely a compile error under `go test ./...`.
  - The file declares `package payroll`, while `cmd/payroll/main.go` in the same directory declares `package main`. Go rejects two package names in one directory when building tests.
  - The tests also reference identifiers that `cmd/payroll/main.go` does not define: `ParseRecord`, `Record`, `PayCents`, `FormatPay`, `FormatTotal`, `PayLine`, `Run`, `runMain`, `inputPath` and `outputPath`. These appear to belong to `internal/payroll`.
- **Unit tests have not been run.** No test results exist for either test file.
- **Scaffolding that is not part of the program.** The original setup notes said the app "listens on port 8080" and needed `DATABASE_URL`, `PORT` and `JWT_SECRET`. That does not match the code: no server is started, no database is used, and `internal/config` is never imported.
- **Narrow behavioral comparison.** The comparison with the original covered runs with no arguments and empty stdin. Other cases have not been compared against the COBOL binary, such as different `employees.txt` contents, a missing input file or I/O errors.

## Manual review required

- **`cmd/payroll/main_test.go`:** move it to `internal/payroll` (or change it to `package main` with matching code), or delete it if it duplicates `internal/payroll/payroll_test.go`. Then run `go test ./...`.
- **`internal/payroll/payroll.go`:**
  - Confirm that the overtime rule (1.5× above 40 h), the half-cent rounding and the field widths match the original COBOL `PICTURE` clauses.
  - Confirm the output file name (`outputPath`) is the one downstream consumers expect.
- **`cmd/app/main.go` vs `cmd/payroll/main.go`:** the two are duplicates. Decide which entry point to keep.
- **`internal/config/config.go` and the `viper` requirement in `go.mod`:** both appear unused. Consider removing them and running `go mod tidy`.
- **Error output:** check that any scripts or schedulers that parsed GnuCOBOL file-error messages from stderr now rely only on the non-zero exit status.