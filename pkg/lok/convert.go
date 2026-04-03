package lok

import "fmt"

// Convert loads a document, applies [Options], and exports to PDF. The office
// must be initialized via [Init]. The caller is responsible for serializing
// calls to Convert if needed (e.g., via an external queue or supervisor).
//
// The conversion pipeline:
//  1. Load the document (with password if provided).
//  2. Apply landscape orientation if requested.
//  3. Update indexes if requested.
//  4. Export to PDF with filter options built from opts.
func Convert(office *Office, inputPath, outputPath string, opts Options) error {
	var doc *Document
	var err error

	if opts.Password != "" {
		doc, err = office.LoadDocumentWithOptions(inputPath, opts.Password)
	} else {
		doc, err = office.LoadDocument(inputPath)
	}

	if err != nil {
		return err
	}

	defer doc.Close()

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

	switch opts.ExportMethod {
	case ExportViaUnoCommand:
		return doc.ExportPDFViaUnoCommand(outputPath, filterOptions)
	default:
		return doc.SaveAs(outputPath, "pdf", filterOptions)
	}
}
