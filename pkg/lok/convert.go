package lok

import "os"

// Convert loads a document, applies [Options], and exports to PDF. The office
// must be initialized via [Init]. The caller is responsible for serializing
// calls to Convert if needed (e.g., via an external queue or supervisor).
//
// Convert loads the document with the password and macro settings, then
// prepares it through a Basic macro, because LibreOfficeKit cannot change a
// document through a UNO dispatch in headless mode: it hides tracked changes
// unless [Options.ShowTrackedChanges] is set, and applies page geometry
// (landscape, paper format and size), index refresh and the Calc header
// suppression when requested. It exports to PDF with the filter options built
// from opts.
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

	// The macro always runs: hiding tracked changes is the default.
	g := resolveGeometry(opts)
	err = office.prepareDocument(preparation{
		landscape:         g.landscape,
		width:             g.width,
		height:            g.height,
		updateIndexes:     opts.UpdateIndexes,
		disableCalcHeader: opts.DisableCalcHeader,
		showChanges:       opts.ShowTrackedChanges,
	})
	if err != nil {
		return err
	}

	filterOptions := BuildFilterOptions(opts)

	return doc.SaveAs(outputPath, "pdf", filterOptions)
}
