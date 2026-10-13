package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/danielededo/cedolo/internal/anonymize"
	"github.com/danielededo/cedolo/internal/pdftext"
	"github.com/danielededo/cedolo/internal/pii"
)

func runAnonymize(args []string, out, info io.Writer) error {
	fs := flag.NewFlagSet("anonymize", flag.ContinueOnError)
	readerName := fs.String("reader", "pdfium", "text reader: pdfium or poppler")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("anonymize: exactly one file expected")
	}
	r, err := pdftext.ByName(*readerName)
	if err != nil {
		return err
	}
	doc, err := r.ReadFile(fs.Arg(0))
	if err != nil {
		return err
	}
	anon, st := anonymize.Layout(doc)
	data, err := json.MarshalIndent(anon, "", "  ")
	if err != nil {
		return err
	}
	// Masking every digit should make this impossible; check anyway before
	// printing anything.
	if m := pii.Find(string(data)); len(m) > 0 {
		return fmt.Errorf("anonymize: output still contains a %s; nothing was printed", m[0].Kind)
	}
	if _, err := out.Write(append(data, '\n')); err != nil {
		return err
	}
	fmt.Fprintf(info, "cedolo: kept %d tokens of known payroll vocabulary, masked %d. Review the output before sharing it.\n", st.Kept, st.Masked)
	return nil
}
