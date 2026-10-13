// Package pdftext converts PDF files into a layout.Document for the CLI. Two
// readers are available behind the same interface, as ADR 0004 plans: PDFium
// run as WebAssembly inside the Go binary (the default), and poppler's
// pdftotext as an independent reference. The browser will use pdf.js instead
// and never imports this package.
package pdftext

import (
	"fmt"
	"sort"

	"github.com/danielededo/cedolo/internal/layout"
)

// Reader extracts positioned text from a PDF file.
type Reader interface {
	Name() string
	ReadFile(path string) (layout.Document, error)
}

var readers = map[string]Reader{
	"pdfium":  PDFium{},
	"poppler": Poppler{},
}

// ByName returns the reader called name ("pdfium" or "poppler").
func ByName(name string) (Reader, error) {
	r, ok := readers[name]
	if !ok {
		return nil, fmt.Errorf("unknown reader %q (available: %v)", name, Names())
	}
	return r, nil
}

// Names lists the available reader names.
func Names() []string {
	names := make([]string, 0, len(readers))
	for n := range readers {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
