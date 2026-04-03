package lok

import "errors"

// Sentinel errors for LibreOfficeKit operations.
var (
	// ErrInitFailed indicates that LibreOfficeKit could not be initialized.
	ErrInitFailed = errors.New("failed to initialize LibreOfficeKit")

	// ErrLoadFailed indicates that a document could not be loaded.
	ErrLoadFailed = errors.New("failed to load document")

	// ErrSaveFailed indicates that a document could not be saved.
	ErrSaveFailed = errors.New("failed to save document")

	// ErrInvalidPDFFormat indicates an unsupported PDF format value.
	ErrInvalidPDFFormat = errors.New("invalid PDF format")

	// ErrOfficeDestroyed indicates the [Office] instance has already been
	// destroyed.
	ErrOfficeDestroyed = errors.New("office instance has been destroyed")

	// ErrDocumentDestroyed indicates the [Document] has already been destroyed.
	ErrDocumentDestroyed = errors.New("document has been destroyed")
)
