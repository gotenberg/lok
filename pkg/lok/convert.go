package lok

import "os"

// Convert loads a document, applies [Options], and exports to PDF. The office
// must be initialized via [Init]. The caller is responsible for serializing
// calls to Convert if needed (e.g., via an external queue or supervisor).
//
// The conversion pipeline:
//  1. Build load options from password and macro settings.
//  2. Load the document.
//  3. Apply page geometry (landscape, paper format/size) and update indexes via
//     a Basic macro if requested. LibreOfficeKit cannot do this through a UNO
//     dispatch in headless mode.
//  4. Export to PDF with filter options built from opts.
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

	loadOpts := BuildLoadOptions(opts)

	var doc *Document
	var err error

	if loadOpts != "" {
		doc, err = office.LoadDocumentWithOptions(inputPath, loadOpts)
	} else {
		doc, err = office.LoadDocument(inputPath)
	}

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
