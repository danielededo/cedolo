package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/danielededo/cedolo/internal/ccnl"
	"github.com/danielededo/cedolo/internal/i18n"
)

func runCCNL(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("ccnl", flag.ContinueOnError)
	search := fs.Bool("search", false, "search contracts by name, sector or signatory instead of by code")
	lang := fs.String("lang", "it", "language: it or en")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return errors.New("ccnl: give a CNEL code, or -search and some words")
	}
	l, err := i18n.Parse(*lang)
	if err != nil {
		return err
	}
	defer fmt.Fprintf(out, "\n%s\n", i18n.T(l, "ccnl.source", ccnl.Source, i18n.Date(l, ccnl.AsOf())))
	if *search {
		found := ccnl.Search(strings.Join(fs.Args(), " "), 10)
		if len(found) == 0 {
			fmt.Fprintln(out, i18n.T(l, "ccnl.no_results"))
			return nil
		}
		fmt.Fprintln(out, i18n.T(l, "ccnl.suggestions"))
		for _, c := range found {
			fmt.Fprintf(out, "  %s  %s\n", c.Code, c.Title)
			if len(c.Employers) > 0 {
				fmt.Fprintf(out, "        %s: %s\n", i18n.T(l, "ccnl.employers"), strings.Join(c.Employers, ", "))
			}
		}
		return nil
	}
	for i, code := range fs.Args() {
		if i > 0 {
			fmt.Fprintln(out)
		}
		c, ok := ccnl.Lookup(code)
		if !ok {
			fmt.Fprintln(out, i18n.T(l, "ccnl.not_found", strings.ToUpper(code), i18n.Date(l, ccnl.AsOf())))
			continue
		}
		fmt.Fprintf(out, "%s  %s\n", c.Code, c.Title)
		for _, row := range [][2]string{
			{i18n.T(l, "ccnl.sectors"), strings.Join(c.Sectors, ", ")},
			{i18n.T(l, "ccnl.period"), period(l, c)},
			{i18n.T(l, "ccnl.employers"), strings.Join(c.Employers, ", ")},
			{i18n.T(l, "ccnl.unions"), strings.Join(c.Unions, ", ")},
		} {
			if row[1] != "" {
				fmt.Fprintf(out, "  %s: %s\n", row[0], row[1])
			}
		}
	}
	return nil
}

// period describes the validity of a contract's latest agreement.
func period(l i18n.Lang, c ccnl.Contract) string {
	switch {
	case c.From != "" && c.Until != "":
		return i18n.T(l, "ccnl.from_until", i18n.Date(l, c.From), i18n.Date(l, c.Until))
	case c.From != "":
		return i18n.T(l, "ccnl.from", i18n.Date(l, c.From))
	}
	return ""
}
