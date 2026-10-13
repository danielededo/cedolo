package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/danielededo/cedolo/internal/i18n"
	"github.com/danielededo/cedolo/internal/report"
)

func runReport(args []string, stderr io.Writer) error {
	fs := flag.NewFlagSet("report", flag.ContinueOnError)
	lang := fs.String("lang", "it", "language: it or en")
	outPath := fs.String("o", "report.html", "output file")
	readerName := fs.String("reader", "pdfium", "text reader: pdfium or poppler")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return errors.New("report: no files given")
	}
	l, err := i18n.Parse(*lang)
	if err != nil {
		return err
	}
	slips, err := readAll(*readerName, fs.Args())
	if err != nil {
		return err
	}
	if len(slips) == 0 {
		return errors.New("report: no payslips found")
	}
	// The report holds personal data: readable by its owner only.
	f, err := os.OpenFile(*outPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if err := report.Render(f, l, slips, version()); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	fmt.Fprintf(stderr, "wrote %s (%d payslips). It contains your payslip data: keep it private.\n", *outPath, len(slips))
	return nil
}
