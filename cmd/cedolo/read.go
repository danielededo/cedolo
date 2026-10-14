package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/term"

	"github.com/danielededo/cedolo/internal/extract"
	"github.com/danielededo/cedolo/internal/insights"
	"github.com/danielededo/cedolo/internal/layout"
	"github.com/danielededo/cedolo/internal/payslip"
	"github.com/danielededo/cedolo/internal/pdftext"
)

// passwordEnv names the environment variable that can hold the password of
// protected payslips, for scripts; people at a terminal are asked instead.
const passwordEnv = "CEDOLO_PDF_PASSWORD"

// warnings receives notes about files that could not be read.
var warnings io.Writer = os.Stderr

// askPassword asks for the password of a protected file. It reads from the
// terminal without echo, and fails when there is no terminal.
var askPassword = func(path string) (string, error) {
	tty, err := os.Open("/dev/tty")
	if err != nil {
		return "", errors.New("no terminal to ask for it")
	}
	defer tty.Close()
	if !term.IsTerminal(int(tty.Fd())) {
		return "", errors.New("no terminal to ask for it")
	}
	fmt.Fprintf(warnings, "Password for %s (not shown): ", filepath.Base(path))
	pw, err := term.ReadPassword(int(tty.Fd()))
	fmt.Fprintln(warnings)
	return string(pw), err
}

// opener reads PDFs, asking for passwords when needed. Passwords that worked
// are tried first on the next protected file, since payslips from the same
// employer usually share one (often the codice fiscale); they are kept in
// memory only.
type opener struct {
	r     pdftext.Reader
	known []string
}

func newOpener(readerName string) (*opener, error) {
	r, err := pdftext.ByName(readerName)
	if err != nil {
		return nil, err
	}
	return &opener{r: r}, nil
}

func (o *opener) read(path string) (layout.Document, error) {
	doc, err := o.r.ReadFile(path)
	if !errors.Is(err, pdftext.ErrPassword) {
		return doc, err
	}
	candidates := append([]string(nil), o.known...)
	if env := os.Getenv(passwordEnv); env != "" {
		candidates = append(candidates, env)
	}
	for _, pw := range candidates {
		if doc, err = o.r.ReadFileWithPassword(path, pw); err == nil {
			return doc, nil
		}
	}
	for range 3 {
		pw, askErr := askPassword(path)
		if askErr != nil {
			return doc, fmt.Errorf("%w; set %s or run cedolo in a terminal", pdftext.ErrPassword, passwordEnv)
		}
		if doc, err = o.r.ReadFileWithPassword(path, pw); err == nil {
			o.known = append(o.known, pw)
			return doc, nil
		}
		fmt.Fprintln(warnings, "cedolo: wrong password")
	}
	return doc, err
}

// readAll extracts every payslip from the given files. A file that cannot be
// read is reported and skipped; it is an error only when no payslip is read
// at all.
func readAll(readerName string, paths []string) ([]payslip.Payslip, error) {
	o, err := newOpener(readerName)
	if err != nil {
		return nil, err
	}
	var all []payslip.Payslip
	var failed int
	for _, path := range paths {
		slips, err := readPayslips(o, path)
		if err != nil {
			failed++
			fmt.Fprintf(warnings, "cedolo: %s: %v (skipped)\n", path, err)
			continue
		}
		if len(slips) == 0 {
			fmt.Fprintf(warnings, "cedolo: %s: no payslip recognized\n", path)
		}
		all = append(all, slips...)
	}
	if len(all) == 0 && failed > 0 {
		return nil, errors.New("no file could be read")
	}
	return all, nil
}

// readPayslips reads one file; a panic in the extractor becomes an error, so
// one odd layout does not stop the others.
func readPayslips(o *opener, path string) (slips []payslip.Payslip, err error) {
	doc, err := o.read(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("internal error while reading the payslip: %v", r)
		}
	}()
	return extract.Document(doc), nil
}

// dedupe drops repeated payslips and says so.
func dedupe(slips []payslip.Payslip) []payslip.Payslip {
	kept, dropped := insights.Dedupe(slips)
	if dropped > 0 {
		fmt.Fprintf(warnings, "cedolo: %d repeated payslip(s) ignored\n", dropped)
	}
	return kept
}
