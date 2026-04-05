# lok

Go library and CLI providing CGO bindings to [LibreOfficeKit](https://docs.libreoffice.org/libreofficekit.html) for document-to-PDF conversion. It loads LibreOffice as an in-process shared library via `dlopen`, eliminating the need for Python or UNO sockets.

> [!WARNING]
> This package is a work in progress. The API is unstable and may change without notice. It is planned to replace [unoconverter](https://github.com/gotenberg/unoconverter) in [Gotenberg](https://github.com/gotenberg/gotenberg) v9.

## Prerequisites

Build and runtime dependencies (Debian/Ubuntu):

```bash
apt-get install -y \
  gcc \
  libc6-dev \
  libreoffice-core \
  libreoffice-writer \
  libreoffice-calc \
  libreoffice-impress \
  libreofficekit-dev \
  fontconfig \
  fonts-liberation
```

- **Go 1.26+** with `CGO_ENABLED=1`
- **gcc, libc6-dev**: C compiler and headers for CGO
- **libreoffice-core, -writer, -calc, -impress**: LibreOffice runtime
- **libreofficekit-dev**: C headers for the LibreOfficeKit API
- **fontconfig, fonts-liberation**: fonts for document rendering

## CLI

The `cmd/lok` binary converts a single document to PDF:

```bash
go build -o lok ./cmd/lok

lok \
  --input-path input.docx \
  --output-path output.pdf \
  --libreoffice-program-path /usr/lib/libreoffice/program \
  --landscape \
  --quality 50 \
  --page-ranges "1-3"
```

Run `lok --help` for the full list of flags (printer descriptor, PDF export filter, viewer preferences).

### Long-running mode

In long-running mode, the binary keeps a single LibreOffice instance alive and reads JSON requests from stdin:

```bash
lok --long-running --libreoffice-program-path /usr/lib/libreoffice/program
```

```json
{ "inputPath": "/tmp/in.docx", "outputPath": "/tmp/out.pdf", "landscape": true }
```

```json
{ "success": true }
```

This avoids the ~1-2s `lok_init` overhead per conversion.

## Recommended usage

**Prefer the CLI binary over embedding the library directly in your Go process.**

LibreOffice loads as a shared library (`dlopen`) and has characteristics that make in-process embedding difficult:

- **Memory leaks.** LibreOffice leaks ~0.5 MiB per conversion. `destroy()` + `lok_init()` does not fully reclaim memory because the shared library stays loaded without `dlclose`. The only reliable way to reclaim memory is process death.
- **No cancellation.** LibreOfficeKit calls are blocking C functions. Go contexts, timeouts, and cancellation do not interrupt them. A hung conversion will block the goroutine (and the underlying OS thread) indefinitely.
- **Signal handler conflicts.** LibreOffice installs signal handlers without `SA_ONSTACK`, which can crash the Go runtime. This requires `GODEBUG=asyncpreemptoff=1` as a workaround.
- **Not thread-safe.** A single LibreOfficeKit instance cannot be used from multiple goroutines concurrently, even for operations on different documents.

The `cmd/lok` binary sidesteps all of these: the caller spawns it as a subprocess, can kill it on timeout, and gets full memory reclamation on process exit. For batch workloads, the `--long-running` mode avoids re-initialization overhead while still allowing the caller to kill and restart the process when needed.

## Library usage

For cases where direct embedding is acceptable (short-lived processes, controlled environments):

```go
import "github.com/gotenberg/lok/pkg/lok"

office, err := lok.Init("/usr/lib/libreoffice/program")
if err != nil {
    log.Fatal(err)
}
defer office.Close()

opts := lok.DefaultOptions()
opts.Landscape = true
opts.Quality = 50

err = lok.Convert(office, "input.docx", "output.pdf", opts)
if err != nil {
    log.Fatal(err)
}
```

The `Lifecycle` type automates memory trimming between conversions:

```go
lc, err := lok.NewLifecycle(lok.LifecycleConfig{
    ProgramPath:  "/usr/lib/libreoffice/program",
    TrimInterval: 10,
})
if err != nil {
    log.Fatal(err)
}
defer lc.Close()

err = lc.Convert("input.docx", "output.pdf", lok.DefaultOptions())
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).
