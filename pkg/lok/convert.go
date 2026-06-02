package lok

import "fmt"

// Convert loads a document, applies [Options], and exports to PDF. The office
// must be initialized via [Init]. The caller is responsible for serializing
// calls to Convert if needed (e.g., via an external queue or supervisor).
//
// The conversion pipeline:
//  1. Build load options from password and macro settings.
//  2. Load the document.
//  3. Apply printer descriptor properties (landscape, paper format/size)
//     via .uno:Printer if any are set.
//  4. Apply page style fallback for landscape if needed.
//  5. Update indexes if requested.
//  6. Export to PDF with filter options built from opts.
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

	// Apply printer descriptor properties (orientation, paper format/size).
	printerProps := BuildPrinterProps(opts)
	if printerProps != "" {
		err = doc.PostUnoCommand(".uno:Printer", printerProps)
		if err != nil {
			return fmt.Errorf("%w: printer descriptor: %s", ErrSaveFailed, err)
		}
	}

	// The printer descriptor orientation governs the print/UNO-export path. The
	// default saveAs path takes orientation from the page style instead, so set
	// it via .uno:AttributePageSize as well.
	if opts.Landscape {
		err = doc.SetLandscape(true)
		if err != nil {
			return fmt.Errorf("%w: landscape: %s", ErrSaveFailed, err)
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
