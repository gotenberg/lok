package lok

// DocumentType represents a LibreOfficeKit document type, matching the
// LibreOfficeKitDocumentType enum.
type DocumentType int

const (
	// TextDocument represents Writer documents (.docx, .odt, .doc, .rtf).
	TextDocument DocumentType = 0

	// SpreadsheetDocument represents Calc documents (.xlsx, .ods, .xls, .csv).
	SpreadsheetDocument DocumentType = 1

	// PresentationDocument represents Impress documents (.pptx, .odp, .ppt).
	PresentationDocument DocumentType = 2

	// DrawingDocument represents Draw documents (.odg).
	DrawingDocument DocumentType = 3
)

// String returns the human-readable name of the document type.
func (dt DocumentType) String() string {
	switch dt {
	case TextDocument:
		return "text"
	case SpreadsheetDocument:
		return "spreadsheet"
	case PresentationDocument:
		return "presentation"
	case DrawingDocument:
		return "drawing"
	default:
		return "unknown"
	}
}

// IsValid reports whether the document type is a recognized value.
func (dt DocumentType) IsValid() bool {
	return dt >= TextDocument && dt <= DrawingDocument
}
