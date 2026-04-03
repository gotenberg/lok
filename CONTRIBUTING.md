# Contributing to lok

**lok** is a Go library providing CGO bindings to LibreOfficeKit for document-to-PDF conversion. It loads LibreOffice as an in-process shared library via `dlopen`, eliminating the need for Python, UNO sockets, or external process management.

- Module: `github.com/gotenberg/lok`
- Go version: 1.23.0

## Getting Started

### Prerequisites

- Go 1.26.0+
- LibreOffice (runtime)
- `libreofficekit-dev` (build-time headers for CGO)
- CGO enabled (`CGO_ENABLED=1`)

Install build dependencies on Debian/Ubuntu:

```bash
apt-get install -y libreofficekit-dev
```

### Build

```bash
go build ./...
```

### Development Loop

```bash
make fmt              # Format Go code + tidy modules
make prettify         # Prettier formatting for non-Go files (markdown, YAML, JSON)
make lint             # golangci-lint (strict)
make lint-prettier    # Prettier check for non-Go files
make lint-todo        # Find TODO comments via godox
make test-unit        # Unit tests (go test -race ./...)
make godoc            # Browse docs at http://localhost:6060
```

Run `make fmt && make lint` before submitting code. Run `make prettify && make lint-prettier` for non-Go files. Zero warnings policy.

## Submitting a Pull Request

Non-trivial changes benefit from planning before coding. Present the problem statement, proposed approach, affected files, and testing strategy. Get alignment before investing in implementation.

### PR Checklist

- [ ] **Encapsulation:** no `pkg/lok/internal` types leak into public signatures.
- [ ] **Error handling:** all errors wrapped and propagated; no silenced errors without justification.
- [ ] **CGO safety:** every `C.CString` has a matching `C.free` in a defer; no dangling C pointers.
- [ ] **Thread safety:** LOK calls are serialized; no concurrent access to `LibreOfficeKit` or `LibreOfficeKitDocument` handles.
- [ ] **Linting:** `make fmt && make lint` passes with zero warnings.
- [ ] **Code quality:** no dead code, no commented-out code, no TODO without context.
- [ ] **Documentation:** public types and functions have Godoc comments.
- [ ] **Tests:** new behavior is covered; no regressions.
- [ ] **Scope:** changes are limited to what was planned.

### Commit Conventions

Follow [Conventional Commits](https://www.conventionalcommits.org/): `feat:`, `fix:`, `refactor:`, `test:`, `docs:`, `chore:`.

Stage specific files only. Never `git add -A` or `git add .`. Do not push unless explicitly asked.

## Core Principles

### Error Handling

- Never assign errors to `_` unless explicitly justified.
- Always wrap and propagate errors up the call stack.
- Use `errors.Is()` with sentinel variables, never `strings.Contains()`.
- Sentinel errors live in `errors.go` per package.
- LOK errors: after a failed LOK call, retrieve the error message via `getError()`, wrap it with the appropriate sentinel, and free the C string via `freeError()`.

### Encapsulation

Public API (`pkg/lok/`) must never accept or return types from `pkg/lok/internal/`. The CGO bridge in `pkg/lok/internal/cgo/` is invisible to consumers. Map internal types to public types before returning to callers.

### CGO Safety

- Every `C.CString()` must have a matching `C.free(unsafe.Pointer(...))` in a defer on the same function.
- Never store C pointers in Go structs beyond the CGO wrapper types in `pkg/lok/internal/cgo/`. The public `Office` and `Document` types in `pkg/lok/` hold the internal wrapper, not raw C pointers.
- LOK's `saveAs` returns 0 on failure (inverted C convention). Document and test this.
- LOK's `freeError` must be used to deallocate strings returned by `getError`, not Go's `C.free`. The C bridge checks for `freeError` availability and falls back to `free()` on older LibreOffice versions.
- Do not use Go's `init()` for LOK initialization. All initialization is explicit via `Init()`.

### Thread Safety

LibreOfficeKit is not thread-safe. A single `LibreOfficeKit` handle must not be used from multiple goroutines concurrently, even for operations on different documents.

The public `Office` type uses a `sync.Mutex` to serialize individual CGO calls. For a full conversion workflow (load, manipulate, save, close), the caller must hold its own higher-level lock. In Gotenberg, this is the queue/supervisor's responsibility.

### Memory Management

LibreOffice leaks memory over time (~0.5 MiB per conversion). The `TrimMemory` method asks LibreOffice to release caches:

- `TrimMemory(0)`: gentle, release per-document caches.
- `TrimMemory(2000)`: aggressive, join threads, release VCL caches.

`trimMemory` is available since LibreOffice 7.6. On older versions it is a no-op. The `Lifecycle` type automates trim scheduling.

`destroy()` + `lok_init()` does not fully reclaim memory because the shared library stays loaded (`dlopen` without `dlclose`). Container-level restarts are the nuclear fallback.

## Project Layout

```
lok/
├── go.mod
├── README.md
├── CONTRIBUTING.md
│
├── pkg/lok/
│   ├── doc.go                     Package documentation
│   ├── office.go                  Office API (Init, Close, LoadDocument, TrimMemory)
│   ├── document.go                Document API (SaveAs, Close, PostUnoCommand)
│   ├── options.go                 Options struct, BuildFilterOptions JSON builder
│   ├── errors.go                  Sentinel errors
│   ├── doctype.go                 DocumentType enum
│   ├── convert.go                 Convert orchestrator (load → configure → save)
│   ├── lifecycle.go               Lifecycle manager (trim scheduling, conversion counting)
│   │
│   └── internal/cgo/
│       ├── doc.go                 Package documentation
│       ├── cgo.go                 CGO bindings (Go ↔ C wrapper functions)
│       ├── lokbridge.h            C header wrapping LOK vtable calls
│       └── lokbridge.c            C implementation
│
└── testdata/                      Test fixture documents (.docx, .xlsx, .pptx, .odt)
```

Import path:

```go
import "github.com/gotenberg/lok/pkg/lok"
```

Usage:

```go
office, err := lok.Init("/usr/lib/libreoffice/program")
```

## Documentation

### Writing Style

- **Short, declarative sentences.** Say what it does, then stop.
- **Lead with the action.** "Converts the document to PDF" not "This function converts the document to PDF".
- **Active voice.** "lok serializes access" not "Access is serialized by lok".
- **No em dashes.** Use a period, colon, or comma instead.

### Godoc

All exported types and functions require Godoc comments. Start with the identifier name:

```go
// Office wraps a LibreOfficeKit instance for document conversion.
type Office struct { ... }

// Init loads LibreOffice from the given program directory.
func Init(programPath string) (*Office, error)
```

Each package has a `doc.go` with a `// Package foo ...` comment. There is no root-level `doc.go`.

Reference other identifiers with square brackets so pkg.go.dev renders them as links:

```go
// Convert loads a document, applies [Options], and exports to PDF.
// The office must be initialized via [Init].
// Returns [ErrSaveFailed] if the export fails.
```

### Code Comments

- Explain _why_, not _what_. The code shows what; the comment explains the non-obvious reasoning.
- No numbered step comments (`// 1. Do X`, `// 2. Do Y`).
- No noise comments that restate the code (`// Check if err is nil`).
- Reference LibreOffice specifics where relevant (`// LOK saveAs returns 0 on failure, unlike typical C convention.`).
- Mark technical debt with `// TODO: [context]`.

## Formatting and Imports

Run `make fmt` to enforce formatting. Import order enforced by golangci-lint: standard library, third-party, then `github.com/gotenberg/lok/...`.

## Testing

### Unit Tests

Tests use `_test.go` convention in the same package. Run with `make test-unit`.

### Integration Tests

Tests requiring LibreOffice use the `//go:build integration` build tag. They live alongside unit tests but are excluded from `make test-unit`. Run with:

```bash
go test -tags integration ./...
```

Integration tests skip automatically if test fixtures in `testdata/` are missing.