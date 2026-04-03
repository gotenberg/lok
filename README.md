# lok

Go library providing CGO bindings to [LibreOfficeKit](https://docs.libreoffice.org/libreofficekit.html) for document-to-PDF conversion. It loads LibreOffice as an in-process shared library via `dlopen`, eliminating the need for Python, UNO sockets, or external process management.

Used by [Gotenberg](https://github.com/gotenberg/gotenberg).

## Prerequisites

- Go 1.26+
- LibreOffice (runtime)
- `libreofficekit-dev` (build-time headers)
- CGO enabled (`CGO_ENABLED=1`)

```bash
apt-get install -y libreofficekit-dev
```

## Usage

```go
import "github.com/gotenberg/lok/pkg/lok"

office, err := lok.Init("/usr/lib/libreoffice/program")
if err != nil {
    log.Fatal(err)
}
defer office.Close()

doc, err := office.LoadDocument("input.docx")
if err != nil {
    log.Fatal(err)
}
defer doc.Close()

err = doc.SaveAs("output.pdf", "pdf", "")
if err != nil {
    log.Fatal(err)
}
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).
