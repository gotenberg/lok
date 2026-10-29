package lok

import "errors"

// Sentinel errors for LibreOfficeKit operations.
var (
	// ErrInitFailed indicates that LibreOfficeKit could not be initialized.
	ErrInitFailed = errors.New("failed to initialize LibreOfficeKit")

	// ErrLoadFailed indicates that a document could not be loaded.
	ErrLoadFailed = errors.New("failed to load document")

	// ErrInvalidPassword indicates that LibreOffice rejected the password
	// given for an encrypted document. It is returned wrapped together with
	// [ErrLoadFailed]. Without a password, loading an encrypted document
	// fails with [ErrLoadFailed] alone: the load runs in batch mode, which
	// never asks for a password.
	ErrInvalidPassword = errors.New("wrong document password")

	// ErrSaveFailed indicates that a document could not be saved.
	ErrSaveFailed = errors.New("failed to save document")

	// ErrOfficeDestroyed indicates the [Office] instance has already been
	// destroyed.
	ErrOfficeDestroyed = errors.New("office instance has been destroyed")

	// ErrDocumentDestroyed indicates the [Document] has already been destroyed.
	ErrDocumentDestroyed = errors.New("document has been destroyed")
)
