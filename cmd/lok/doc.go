// Lok converts documents to PDF using LibreOfficeKit.
//
// Single-shot mode (default):
//
//	lok --input-path input.docx --output-path output.pdf
//
// Long-running mode:
//
//	lok --long-running --libreoffice-program-path /usr/lib/libreoffice/program
//
// In long-running mode, lok reads JSON requests from stdin (one per line)
// and writes JSON responses to stdout, reusing a single LibreOffice instance.
package main
