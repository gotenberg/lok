// Package lok provides Go bindings to LibreOfficeKit for document-to-PDF
// conversion.
//
// lok loads LibreOffice as an in-process shared library via dlopen, eliminating
// the need for Python, UNO sockets, or external process management.
//
// Initialize with [Init], then use [Office.LoadDocument] and
// [Document.SaveAs] to convert documents.
package lok
