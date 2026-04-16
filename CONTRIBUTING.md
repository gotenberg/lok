# Contributing to lok

Go library providing CGO bindings to LibreOfficeKit for document-to-PDF conversion. Loads LibreOffice as an in-process shared library via `dlopen`. Two rules override everything else: **CGO safety** (every `C.CString` has a matching `C.free` in a defer, no dangling C pointers) and **thread safety** (LOK calls are serialized, no concurrent access to `LibreOfficeKit` or `LibreOfficeKitDocument` handles).

- Module: `github.com/gotenberg/lok`
- Go: 1.26.0+
- Docker (all build dependencies are installed in the Docker image)

## Quick start

```bash
make build-test       # Build the Docker image for testing
make fmt              # Format Go code + tidy modules
make prettify         # Prettier formatting for non-Go files
make lint             # golangci-lint (strict)
make lint-prettier    # Prettier check for non-Go files
make test-unit        # Unit tests in Docker
make test-integration # Integration tests in Docker
make godoc            # Browse docs at http://localhost:6060
```

Run `make fmt && make lint` before submitting code. Run `make prettify && make lint-prettier` for non-Go files. Zero warnings policy.

## Project layout

```
cmd/lok/                       CLI binary (single-shot and long-running modes)
pkg/lok/                       Public API (Office, Document, Convert, Lifecycle)
  internal/cgo/                CGO bindings (Go <-> C wrapper, lokbridge.h/.c)
test/integration/              Integration tests + testdata fixtures
Dockerfile                     Docker image for testing
```

Import path: `github.com/gotenberg/lok/pkg/lok`.

## Coding rules

### Encapsulation

`pkg/lok/` is the public surface. No type from `pkg/lok/internal/` may appear in a public signature. Map internal types to public types before returning to callers.

### Error handling

- Never assign errors to `_` unless explicitly justified.
- Wrap and propagate every error: `fmt.Errorf("description: %w", err)`.
- Match errors with `errors.Is`, never `strings.Contains`.
- Sentinel errors live in `errors.go` per package.
- LOK errors: after a failed LOK call, retrieve the error message via `getError()`, wrap it with the appropriate sentinel, and free the C string via `freeError()`.

### CGO safety

- Every `C.CString()` must have a matching `C.free(unsafe.Pointer(...))` in a defer on the same function.
- Never store C pointers in Go structs beyond the CGO wrapper types in `pkg/lok/internal/cgo/`. The public `Office` and `Document` types hold the internal wrapper, not raw C pointers.
- LOK's `saveAs` returns 0 on failure (inverted C convention). Document and test this.
- LOK's `freeError` must deallocate strings returned by `getError`, not Go's `C.free`. The C bridge checks for `freeError` availability and falls back to `free()` on older LibreOffice versions.
- Do not use Go's `init()` for LOK initialization. All initialization is explicit via `Init()`.

### Thread safety

LibreOfficeKit is not thread-safe. A single `LibreOfficeKit` handle must not be used from multiple goroutines concurrently, even for operations on different documents.

The public `Office` type uses a `sync.Mutex` to serialize individual CGO calls. For a full conversion workflow (load, manipulate, save, close), the caller must hold its own higher-level lock.

### Memory management

LibreOffice leaks memory over time (~0.5 MiB per conversion). `TrimMemory(0)` releases per-document caches. `TrimMemory(2000)` is aggressive: joins threads, releases VCL caches. Available since LibreOffice 7.6, no-op on older versions.

`destroy()` + `lok_init()` does not fully reclaim memory because the shared library stays loaded (`dlopen` without `dlclose`). Container-level restarts are the nuclear fallback. The `Lifecycle` type automates trim scheduling.

## Documentation rules

### Tone

- Short, declarative sentences. Say what it does, then stop.
- Lead with the action. "Converts the document to PDF", not "This function converts the document to PDF".
- Active voice. "lok serializes access", not "Access is serialized by lok".
- No em dashes. Use a period, colon, or comma.

### Godoc

Every exported type and function has a Godoc comment starting with its identifier name:

```go
// Office wraps a LibreOfficeKit instance for document conversion.
type Office struct { ... }

// Init loads LibreOffice from the given program directory.
func Init(programPath string) (*Office, error)
```

Each package has a `doc.go` with a `// Package foo ...` comment.

Reference identifiers with `[Name]` brackets for pkg.go.dev linking:

```go
// Convert loads a document, applies [Options], and exports to PDF.
// Returns [ErrSaveFailed] if the export fails.
```

### Code comments

- Explain _why_, not _what_.
- No numbered step comments. No noise comments that restate the code.
- Reference LibreOffice specifics where relevant (`// LOK saveAs returns 0 on failure, unlike typical C convention.`).
- Mark debt with `// TODO: [context]`.

## Testing

All tests run inside Docker because the package requires CGO and `libreofficekit-dev` headers. Build the image first with `make build-test`.

Unit tests use `_test.go` convention in the same package. Integration tests use `//go:build integration` and live in `test/integration/` with fixtures in `test/integration/testdata/`.

## Pull requests

Plan non-trivial changes before coding. Present the problem, approach, affected files, and testing strategy.

### Checklist

- [ ] No `pkg/lok/internal` types leak into public signatures
- [ ] All errors wrapped and propagated; no silenced errors without justification
- [ ] Every `C.CString` has a matching `C.free` in a defer; no dangling C pointers
- [ ] LOK calls are serialized; no concurrent access to handles
- [ ] Linting: `make fmt && make lint` passes with zero warnings
- [ ] No dead code, no commented-out code, no TODO without context
- [ ] Public types and functions have Godoc comments
- [ ] New behavior is covered; no regressions

### Commits

[Conventional Commits](https://www.conventionalcommits.org/): `feat:`, `fix:`, `refactor:`, `test:`, `docs:`, `chore:`.

Stage specific files. Never `git add -A` or `git add .`.
