package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/danielededo/cedolo/internal/i18n"
	"github.com/danielededo/cedolo/internal/insights"
)

func runCompare(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("compare", flag.ContinueOnError)
	asJSON := fs.Bool("json", false, "print JSON instead of text")
	lang := fs.String("lang", "it", "language: it or en")
	readerName := fs.String("reader", "pdfium", "text reader: pdfium or poppler")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return errors.New("compare: no files given")
	}
	l, err := i18n.Parse(*lang)
	if err != nil {
		return err
	}
	slips, err := readAll(*readerName, fs.Args())
	if err != nil {
		return err
	}
	slips = dedupe(slips)
	periods := insights.Compare(l, slips)
	if *asJSON {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(periods)
	}
	for _, p := range periods {
		fmt.Fprintln(out, p.Label)
		if len(p.Changes) == 0 {
			fmt.Fprintln(out, "  -")
		}
		for _, c := range p.Changes {
			fmt.Fprintf(out, "  %s\n", c.Text)
			for _, w := range c.Why {
				fmt.Fprintf(out, "      %s\n", w)
			}
		}
	}
	return nil
}
