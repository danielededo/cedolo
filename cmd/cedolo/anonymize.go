package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/danielededo/cedolo/internal/anonymize"
)

func runAnonymize(args []string, out, info io.Writer) error {
	fs := flag.NewFlagSet("anonymize", flag.ContinueOnError)
	readerName := fs.String("reader", "pdfium", "text reader: pdfium or poppler")
	amounts := fs.String("amounts", "keep", "keep amounts so the totals can be checked, or mask them too: keep or mask")
	output := fs.String("o", "", "write a rebuilt PDF to this file instead of printing the text as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("anonymize: exactly one file expected")
	}
	if *amounts != "keep" && *amounts != "mask" {
		return fmt.Errorf("anonymize: -amounts must be keep or mask, not %q", *amounts)
	}
	if *output != "" && sameFile(*output, fs.Arg(0)) {
		return errors.New("anonymize: the output would overwrite the payslip")
	}
	o, err := newOpener(*readerName)
	if err != nil {
		return err
	}
	doc, err := o.read(fs.Arg(0))
	if err != nil {
		return err
	}
	anon, st := anonymize.Layout(doc, anonymize.Options{KeepAmounts: *amounts == "keep"})
	// Masking should make this impossible; check anyway before writing
	// anything.
	if leaks := anonymize.Leaks(doc, anon); len(leaks) > 0 {
		return fmt.Errorf("anonymize: the output would still contain %d personal value(s); nothing was written", len(leaks))
	}
	var data bytes.Buffer
	if *output != "" {
		err = anonymize.WritePDF(&data, anon)
	} else {
		var j []byte
		j, err = json.MarshalIndent(anon, "", "  ")
		data.Write(append(j, '\n'))
	}
	if err != nil {
		return err
	}
	if *output != "" {
		if err := os.WriteFile(*output, data.Bytes(), 0o600); err != nil {
			return err
		}
		fmt.Fprintf(info, "cedolo: wrote %s.\n", *output)
	} else if _, err := out.Write(data.Bytes()); err != nil {
		return err
	}
	fmt.Fprintf(info, "cedolo: kept %d tokens of known payroll vocabulary", st.Kept)
	if *amounts == "keep" {
		fmt.Fprint(info, " and numbers")
	}
	fmt.Fprintf(info, ", masked %d. Review it before sharing it.\n", st.Masked)
	if *amounts == "keep" {
		fmt.Fprintln(info, "cedolo: amounts are kept, and together they can identify you to someone who knows your pay; use -amounts mask if that matters.")
	}
	return nil
}

// sameFile tells whether two paths name the same file.
func sameFile(a, b string) bool {
	if fa, err := os.Stat(a); err == nil {
		if fb, err := os.Stat(b); err == nil {
			return os.SameFile(fa, fb)
		}
	}
	return filepath.Clean(a) == filepath.Clean(b)
}
