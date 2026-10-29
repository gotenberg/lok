package lok

import "os"

// Convert loads a document, applies [Options], and exports to PDF. The office
// must be initialized via [Init]. The caller is responsible for serializing
// calls to Convert if needed (e.g., via an external queue or supervisor).
//
// Convert loads the document with the password and macro settings, then
// applies page geometry (landscape, paper format and size) and refreshes
// indexes through a Basic macro when requested, because LibreOfficeKit cannot
// do either through a UNO dispatch in headless mode. It exports to PDF with
// the filter options built from opts.
func Convert(office *Office, inputPath, outputPath string, opts Options) error {
	// SinglePageSheets starts each sheet's single page at the workbook's saved
	// scroll position, truncating everything above and to the left of it. Load
	// a copy with that position reset to the top-left cell instead.
	// See https://github.com/gotenberg/gotenberg/issues/1222.
	if opts.SinglePageSheets {
		sanitized := resetCalcScrollPosition(inputPath)
		if sanitized != inputPath {
			defer func() { _ = os.Remove(sanitized) }()
			inputPath = sanitized
		}
	}

	doc, err := office.LoadDocumentWithPassword(inputPath, BuildLoadOptions(opts), opts.Password)
	if err != nil {
		return err
	}

	defer doc.Close()

	g, geometryNeeded := resolveGeometry(opts)
	if geometryNeeded || opts.UpdateIndexes {
		if err = office.prepareDocument(g.landscape, g.width, g.height, opts.UpdateIndexes); err != nil {
			return err
		}
	}

	filterOptions := BuildFilterOptions(opts)

	return doc.SaveAs(outputPath, "pdf", filterOptions)
}
