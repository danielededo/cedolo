package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/danielededo/cedolo/internal/extract"
	"github.com/danielededo/cedolo/internal/money"
	"github.com/danielededo/cedolo/internal/payslip"
	"github.com/danielededo/cedolo/internal/pdftext"
)

func runExtract(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("extract", flag.ContinueOnError)
	asJSON := fs.Bool("json", false, "print JSON instead of a table")
	readerName := fs.String("reader", "pdfium", "text reader: pdfium or poppler")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return errors.New("extract: no files given")
	}
	r, err := pdftext.ByName(*readerName)
	if err != nil {
		return err
	}
	var all []payslip.Payslip
	for _, path := range fs.Args() {
		doc, err := r.ReadFile(path)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		all = append(all, extract.Document(doc)...)
	}
	if *asJSON {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(all)
	}
	for i, p := range all {
		if i > 0 {
			fmt.Fprintln(out)
		}
		printPayslip(out, p)
	}
	return nil
}

func printPayslip(out io.Writer, p payslip.Payslip) {
	period := "unknown period"
	if p.Month > 0 {
		period = fmt.Sprintf("%s %d", time.Month(p.Month), p.Year)
	}
	fmt.Fprintf(out, "%s (%s)", period, p.Kind)
	if p.CCNL != "" {
		fmt.Fprintf(out, ", CCNL %s", p.CCNL)
	}
	if p.Level != "" {
		fmt.Fprintf(out, ", level %s", p.Level)
	}
	fmt.Fprintln(out)

	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', tabwriter.AlignRight)
	for _, row := range []struct {
		name string
		a    payslip.Amount
	}{
		{"gross", p.Gross}, {"deductions", p.Deductions}, {"net", p.Net},
		{"INPS base", p.INPSBase}, {"IRPEF base", p.IRPEFBase}, {"TFR accrued", p.TFR},
	} {
		value := "-"
		if row.a.Found() {
			value = money.Format(row.a.Cents)
		}
		fmt.Fprintf(w, "  %s\t%s\t  %s\t\n", row.name, value, row.a.Status)
	}
	w.Flush()

	fmt.Fprintln(out, "  items:")
	w = tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	for _, it := range p.Items {
		sign := "+"
		if it.Side == payslip.Deduction {
			sign = "-"
		}
		fmt.Fprintf(w, "    %s\t%s\t%s\t%s%s\n", it.Code, it.Description, it.Key, sign, money.Format(it.Cents))
	}
	w.Flush()

	fmt.Fprintln(out, "  checks:")
	for _, c := range p.Checks {
		mark := "ok  "
		if !c.OK {
			mark = "FAIL"
		}
		fmt.Fprintf(out, "    %s %s: %s\n", mark, c.Name, c.Detail)
	}
}
