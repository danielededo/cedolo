// Command cedolo reads Italian payslips on the local machine. This early
// version exposes the text layer only: it inspects PDFs, prints their
// positioned text, and writes synthetic payslips for testing.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"
	"time"
	"unicode"

	"github.com/danielededo/cedolo/internal/layout"
	"github.com/danielededo/cedolo/internal/money"
	"github.com/danielededo/cedolo/internal/pdftext"
	"github.com/danielededo/cedolo/internal/synth"
)

const usage = `cedolo reads Italian payslips on your machine. Nothing is sent anywhere.

Usage:
  cedolo inspect [-reader pdfium|poppler|both] FILE...
      Report whether each PDF has a readable text layer. Prints counts only,
      never names, amounts or file names, so the output is safe to share.
  cedolo extract [-json] [-reader pdfium|poppler] FILE...
      Read payslips and print their data, item by item, with the outcome of
      the cross-checks. The output contains your payslip data: keep it local.
  cedolo compare [-lang it|en] [-json] [-reader pdfium|poppler] FILE...
      Read payslips of several months and list what changed from one month
      to the next, with the most likely explanation. The output contains your
      payslip data: keep it local.
  cedolo report [-lang it|en] [-o FILE] [-reader pdfium|poppler] FILE...
      Write a self-contained HTML report (default: report.html) with charts
      over time, what changed, leave balances and a glossary of the items.
      It opens offline in any browser. It contains your payslip data: keep
      it private.
  cedolo glossary [-lang it|en] [WORDS...]
      Explain the items printed on payslips and the words around them, such
      as imponibile or conguaglio; with words, only the matching entries.
  cedolo ccnl [-lang it|en] CODE...
  cedolo ccnl [-lang it|en] -search WORDS...
      Show a national collective agreement by its CNEL code (sector,
      latest agreement, signatories), or search contracts by name. The list
      of contracts is built into cedolo: nothing is sent anywhere.
  cedolo anonymize [-reader pdfium|poppler] [-amounts keep|mask] [-o FILE.pdf] FILE
      Make a copy of a payslip to report a layout that cedolo reads badly.
      Everything but known payroll labels is masked: names, companies,
      addresses, identifiers, dates and codes. Amounts are kept unless
      -amounts mask. With -o it writes a new PDF rebuilt from that text,
      holding nothing of the original file; otherwise it prints the text as
      JSON. Review it before sharing.
  cedolo layout [-reader pdfium|poppler] [-redact] FILE
      Print the positioned text of a PDF as JSON. Without -redact the output
      contains personal data: keep it on your machine.
  cedolo synth [-style classic|signed|grid|spaced] [-month 1-12] [-password PW] [-rotate 90] [-o FILE]
      Write a synthetic payslip PDF with invented data (default: synthetic.pdf).
  cedolo version
      Print the version.

Payslips protected by a password are opened after asking for it in the
terminal, or with the password in the CEDOLO_PDF_PASSWORD environment
variable. A file that cannot be read is reported and skipped.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	var err error
	switch cmd, args := os.Args[1], os.Args[2:]; cmd {
	case "inspect":
		err = runInspect(args, os.Stdout)
	case "extract":
		err = runExtract(args, os.Stdout)
	case "compare":
		err = runCompare(args, os.Stdout)
	case "report":
		err = runReport(args, os.Stderr)
	case "glossary":
		err = runGlossary(args, os.Stdout)
	case "ccnl":
		err = runCCNL(args, os.Stdout)
	case "anonymize":
		err = runAnonymize(args, os.Stdout, os.Stderr)
	case "layout":
		err = runLayout(args, os.Stdout, os.Stderr)
	case "synth":
		err = runSynth(args)
	case "version":
		fmt.Println(version())
	case "help", "-h", "--help":
		fmt.Print(usage)
	default:
		err = fmt.Errorf("unknown command %q", cmd)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "cedolo:", err)
		os.Exit(1)
	}
}

// buildVersion is set by the release build (-ldflags "-X main.buildVersion=v1.2.3").
var buildVersion string

func version() string {
	if buildVersion != "" {
		return buildVersion
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		return info.Main.Version
	}
	return "(devel)"
}

func runInspect(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("inspect", flag.ContinueOnError)
	readerName := fs.String("reader", "pdfium", "text reader: pdfium, poppler, or both")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return errors.New("inspect: no files given")
	}
	names := []string{*readerName}
	if *readerName == "both" {
		names = pdftext.Names()
	}
	for i, path := range fs.Args() {
		// Files are numbered, not named: file names often contain a person's
		// name or a date, and this output is meant to be shareable.
		fmt.Fprintf(out, "file %d\n", i+1)
		docs := make([]layout.Document, 0, len(names))
		for _, n := range names {
			r, err := pdftext.ByName(n)
			if err != nil {
				return err
			}
			doc, err := r.ReadFile(path)
			if err != nil {
				fmt.Fprintf(out, "  %s: error: %s\n", n, scrub(err.Error(), path))
				continue
			}
			docs = append(docs, doc)
			s := stats(doc)
			fmt.Fprintf(out, "  %s: pages %d, tokens %d, lines %d, amounts %d, unreadable characters %d\n",
				n, s.pages, s.tokens, s.lines, s.amounts, s.unreadable)
			fmt.Fprintf(out, "  %s: %s\n", n, s.verdict())
		}
		if len(docs) == 2 {
			ag := pdftext.Compare(docs[0], docs[1], 2)
			fmt.Fprintf(out, "  agreement: %.1f%% (matched %d, only %s %d, only %s %d, max offset %.2fpt)\n",
				100*ag.Ratio(), ag.Matched, names[0], ag.OnlyA, names[1], ag.OnlyB, ag.MaxOffset)
		}
	}
	return nil
}

type docStats struct {
	pages, tokens, lines, amounts, unreadable, chars int
}

func stats(d layout.Document) docStats {
	s := docStats{pages: len(d.Pages)}
	for _, p := range d.Pages {
		s.lines += len(p.Lines(2))
		for _, t := range p.Tokens {
			s.tokens++
			if money.IsAmount(t.Text) {
				s.amounts++
			}
			for _, r := range t.Text {
				s.chars++
				if r == unicode.ReplacementChar || unicode.IsControl(r) || unicode.In(r, unicode.Co) {
					s.unreadable++
				}
			}
		}
	}
	return s
}

func (s docStats) verdict() string {
	switch {
	case s.tokens == 0:
		return "no text layer found; the file is probably a scan, which is not supported yet"
	case float64(s.unreadable) > 0.05*float64(s.chars):
		return "text layer present but many characters cannot be decoded (font encoding problem)"
	case s.amounts == 0:
		return "text layer readable but no amounts recognized"
	default:
		return "text layer readable"
	}
}

// scrub removes the file path from error messages so inspect output never
// contains it.
func scrub(msg, path string) string {
	return strings.ReplaceAll(msg, path, "FILE")
}

func runLayout(args []string, out, warn io.Writer) error {
	fs := flag.NewFlagSet("layout", flag.ContinueOnError)
	readerName := fs.String("reader", "pdfium", "text reader: pdfium or poppler")
	redact := fs.Bool("redact", false, "mask letters and digits, keeping positions")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("layout: exactly one file expected")
	}
	o, err := newOpener(*readerName)
	if err != nil {
		return err
	}
	doc, err := o.read(fs.Arg(0))
	if err != nil {
		return err
	}
	if *redact {
		doc = doc.Redact()
	} else {
		fmt.Fprintln(warn, "cedolo: this output contains the document's personal data; do not share it, use -redact instead")
	}
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(doc)
}

func runSynth(args []string) error {
	fs := flag.NewFlagSet("synth", flag.ContinueOnError)
	outPath := fs.String("o", "synthetic.pdf", "output file")
	styleName := fs.String("style", "classic", "layout style: classic, signed, grid, or spaced")
	password := fs.String("password", "", "protect the PDF with this password")
	rotate := fs.Int("rotate", 0, "turn the page by 90, 180 or 270 degrees")
	month := fs.Int("month", 10, "month of the payslip, 1 to 12")
	person := fs.String("person", "", "worker's name, surname first, printed with an invented codice fiscale (classic style)")
	company := fs.String("employer", "", "employer's name, printed with an invented VAT number (classic style)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *month < 1 || *month > 12 {
		return fmt.Errorf("synth: -month must be 1 to 12, not %d", *month)
	}
	style, err := synth.ParseStyle(*styleName)
	if err != nil {
		return err
	}
	f, err := os.Create(*outPath)
	if err != nil {
		return err
	}
	opts := synth.Options{Password: *password, Rotate: *rotate}
	p := synth.ExampleFor(style)
	p.Month = time.Month(*month)
	if *person != "" {
		p.Employee, p.EmployeeCode = strings.ToUpper(*person), synth.CodiceFiscale(*person)
	}
	if *company != "" {
		// The VAT number follows the name, so that two employers differ.
		var digits int
		for _, r := range *company {
			digits = (digits*31 + int(r)) % 10000000000
		}
		p.Employer, p.EmployerCode = strings.ToUpper(*company), synth.VATNumber(fmt.Sprintf("%010d", digits))
	}
	if err := synth.WritePDFWith(f, p, style, opts); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
