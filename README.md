# Payroll Batch (Go)

A payroll batch-processing program migrated from COBOL to Go. The repository includes an `employees.txt` data file, which appears to be the batch input. Confirm this against the code before relying on it.

> **Status:** The build passed and output was compared with the original COBOL program, but **the target unit tests are currently failing**. See [Known Limitations](#known-limitations) and [Manual Review Required](#manual-review-required) before treating this branch as production-ready.

---

## Tech Stack

| Component | Technology |
|---|---|
| Language | Go |
| Framework | None (Go standard library / standard toolchain) |
| Dependency management | Go modules (`go.mod`) |

---

## Prerequisites

- A Go toolchain compatible with the version declared in `go.mod`
- A POSIX shell (`sh`/`bash`). The install and test commands use shell conditionals.
- Network access for `go mod download` / `go mod tidy`, unless your module cache is already populated

---

## Getting Started

Run these steps in order from a fresh clone.

### 1. Install dependencies and build

```sh
go mod download && if [ -d /app/cmd/app ]; then mkdir -p /app/bin && go build -o /app/bin/app ./cmd/app; else echo 'No ./cmd/app source present; skipping build (only go.mod exists)'; fi
```

Notes on this command:

- It checks for the **absolute path** `/app/cmd/app` and writes the binary to `/app/bin/app`. It assumes the repository is checked out at `/app`, for example inside a container. If your clone lives elsewhere, the directory check fails and the build step is skipped with the "skipping build" message.
- It builds only `./cmd/app`. It does **not** build `./cmd/payroll`.

### 2. Resolve module dependencies

This command was run successfully during migration and is required on a fresh clone:

```sh
go mod tidy
```

### 3. Configure environment variables

Set the variables listed in [Environment Variables](#environment-variables). For example:

```sh
export DATABASE_URL=...
export PORT=...
export JWT_SECRET=...
```

### 4. Database setup

No database setup command was detected for this project. See [Manual Review Required](#manual-review-required) regarding `DATABASE_URL`.

### 5. Run

No run command was detected for this project. The repository contains two entry points, `cmd/app/main.go` and `cmd/payroll/main.go`. You need to determine which one is the payroll batch entry point and how it consumes `employees.txt`. See [Manual Review Required](#manual-review-required).

---

## Running Tests

```sh
if [ -n "$(go list ./... 2>/dev/null)" ]; then go test ./...; else echo 'No Go packages to test'; fi
```

⚠️ **The target unit tests currently fail.** Expect a non-zero exit from this command until the failures are investigated and fixed.

---

## Environment Variables

These variables were detected during migration. Their exact usage has not been verified. Check `internal/config/config.go` and the entry points to confirm which are actually read and whether they are required.

| Variable | Description | Required |
|---|---|---|
| `DATABASE_URL` | Database connection string | Unverified |
| `PORT` | Port number, if the application listens on one | Unverified |
| `JWT_SECRET` | Secret used for JWT signing/verification | Unverified |

`PORT` and `JWT_SECRET` are unusual for a batch payroll job. They may come from template code rather than the migrated payroll logic.

---

## Architecture Overview

```
.
├── cmd/
│   ├── app/
│   │   └── main.go        # Entry point; built by the install command to /app/bin/app
│   └── payroll/
│       └── main.go        # Entry point; not built by the install command
├── internal/
│   └── config/
│       └── config.go      # Configuration package (internal to this module)
├── employees.txt          # Data file (likely batch input)
└── go.mod                 # Go module definition
```

- **`cmd/`** follows the standard Go layout, with one subdirectory per executable. There are two executables, `app` and `payroll`.
- **`internal/config`** holds configuration code. Go's `internal/` convention means it can only be imported from within this module.
- **`employees.txt`** is a flat data file kept alongside the code. This is consistent with the file-oriented processing typical of COBOL batch programs.

---

## Migration Notes

- **Source:** COBOL, with no framework.
- **Target:** Go, using the standard toolchain.
- **Scope:** 1 of 1 module migrated. No modules were flagged as unmigratable.
- **Verification performed:**
  - ✅ Build: passed
  - ❌ Target unit tests: failed
  - ✅ Behavior compared with the original: passed
- **Confidence:** The migration tool self-assessed its confidence at 93%. This figure is an estimate, not a measured result.
- **Dependency resolution:** `go mod tidy` was required after the initial install to resolve module dependencies.
- The program is now a compiled Go binary instead of a COBOL program. Any job scheduling, JCL, or runtime wrappers around the original COBOL program are not part of this repository and must be recreated separately if needed.

---

## Known Limitations

- **Unit tests fail.** `go test ./...` does not currently pass. The cause has not been documented.
- **No run command.** No verified command exists for running the batch job.
- **Install command assumes `/app`.** The build only happens when the repository is located at `/app`, and only `./cmd/app` is built.
- **Second entry point is not built.** `cmd/payroll/main.go` is never built by the install command.
- **No DB setup.** No database setup step was detected, even though `DATABASE_URL` was flagged as required.

---

## Manual Review Required

Verify the following before relying on this migration:

1. **Failing unit tests.** Run the test command, identify the failures, and determine whether they come from migration defects or from incorrect tests.
2. **`cmd/app/main.go` vs `cmd/payroll/main.go`.** Decide which is the real payroll entry point. If one is leftover scaffolding, remove it. Update the install/build command if it targets the wrong one.
3. **`internal/config/config.go`.** Confirm which environment variables are actually read. Pay particular attention to whether `PORT`, `JWT_SECRET`, and `DATABASE_URL` are needed by a batch payroll job.
4. **`employees.txt`.** Confirm it is the input file, how the program locates it (path, flag, or working directory), and that its record layout matches the original COBOL record definitions. Check field widths, implied decimals, and signs.
5. **Numeric behavior.** COBOL fixed-point decimal arithmetic and rounding can differ from Go numeric types. The behavior comparison passed, but check that the compared cases covered edge cases such as rounding boundaries, negative values, and large totals.
6. **Install command paths.** Replace the hard-coded `/app` paths if the project will be built outside a container rooted at `/app`.