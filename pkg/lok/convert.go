package lok

import "fmt"

// Convert loads a document, applies [Options], and exports to PDF. The office
// must be initialized via [Init]. The caller is responsible for serializing
// calls to Convert if needed (e.g., via an external queue or supervisor).
//
// The conversion pipeline:
//  1. Build load options from password and macro settings.
//  2. Load the document.
//  3. Apply page geometry (landscape, paper format/size) to the page styles
//     via a Basic macro if any geometry option is set.
//  4. Update indexes if requested.
//  5. Export to PDF with filter options built from opts.
func Convert(office *Office, inputPath, outputPath string, opts Options) error {
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

	// Apply page geometry. LibreOfficeKit cannot change a page style through a
	// dispatch, so a Basic macro sets orientation and size on the loaded
	// document before export.
	if g, needed := resolveGeometry(opts); needed {
		if err = office.applyGeometry(g.landscape, g.width, g.height); err != nil {
			return err
		}
	}

	if opts.UpdateIndexes {
		err = doc.PostUnoCommand(".uno:UpdateAllIndexes", "")
		if err != nil {
			return fmt.Errorf("%w: update indexes: %s", ErrSaveFailed, err)
		}
	}

	filterOptions := BuildFilterOptions(opts)

	// Any value other than the experimental UNO path uses saveAs, including
	// unrecognized ExportMethod values.
	if opts.ExportMethod == ExportViaUnoCommand {
		return doc.ExportPDFViaUnoCommand(outputPath, filterOptions)
	}

	return doc.SaveAs(outputPath, "pdf", filterOptions)
}
