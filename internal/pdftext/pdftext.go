// Package pdftext converts PDF files into a layout.Document for the CLI. Two
// readers are available behind the same interface, as ADR 0004 plans: PDFium
// run as WebAssembly inside the Go binary (the default), and poppler's
// pdftotext as an independent reference. The browser will use pdf.js instead
// and never imports this package.
package pdftext

import (
	"errors"
	"fmt"
	"sort"

	"github.com/danielededo/cedolo/internal/layout"
)

// Reader extracts positioned text from a PDF file.
type Reader interface {
	Name() string
	ReadFile(path string) (layout.Document, error)
	// ReadFileWithPassword opens a PDF protected by a password.
	ReadFileWithPassword(path, password string) (layout.Document, error)
}

var (
	// ErrPassword means the PDF needs a password, or the one given is wrong.
	ErrPassword = errors.New("the PDF is protected by a password")
	// ErrDamaged means the file is not a PDF or is damaged.
	ErrDamaged = errors.New("the file is not a PDF or is damaged")
)

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
