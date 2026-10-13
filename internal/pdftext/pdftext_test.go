package pdftext

import (
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danielededo/cedolo/internal/layout"
	"github.com/danielededo/cedolo/internal/money"
	"github.com/danielededo/cedolo/internal/synth"
)

func writeExample(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "example.pdf")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := synth.WritePDF(f, synth.Example()); err != nil {
		t.Fatal(err)
	}
	return path
}

func lineTexts(doc layout.Document) []string {
	var out []string
	for _, l := range doc.Pages[0].Lines(2) {
		out = append(out, l.Text())
	}
	return out
}

// checkExample verifies that a reader recovers the synthetic payslip: the
// expected lines in order, and every amount right-aligned on its column.
func checkExample(t *testing.T, r Reader) {
	doc, err := r.ReadFile(writeExample(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Pages) != 1 || doc.Pages[0].Width != 595 || doc.Pages[0].Height != 842 {
		t.Fatalf("unexpected pages: %+v", doc.Pages)
	}
	got := strings.Join(lineTexts(doc), "\n")
	gross, deductions, net := synth.Example().Totals()
	for _, want := range []string{
		"CEDOLINO PAGA",
		"ESEMPIO SINTETICO SRL PERIODO OTTOBRE 2026",
		"0250 STRAORDINARIO 25% 10,00 14,45 144,50",
		"0300 INDENNITÀ DI FUNZIONE 50,00",
		"9001 CONTRIBUTO IVS 9,19% 201,67",
		"TOTALE COMPETENZE " + money.Format(gross),
		"TOTALE TRATTENUTE " + money.Format(deductions),
		"NETTO DEL MESE " + money.Format(net),
		"PERMESSI ROL 6,00 0,00 42,00",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("%s: missing line %q in:\n%s", r.Name(), want, got)
		}
	}
	for _, tok := range doc.Pages[0].Tokens {
		if tok.Text == "2.000,00" && math.Abs(tok.Right()-480) > 0.5 {
			t.Errorf("%s: earnings amount right edge at %.2f, want 480", r.Name(), tok.Right())
		}
		if tok.Text == "201,67" && math.Abs(tok.Right()-555) > 0.5 {
			t.Errorf("%s: deduction amount right edge at %.2f, want 555", r.Name(), tok.Right())
		}
	}
}

func TestPDFiumReadsSyntheticPayslip(t *testing.T) {
	checkExample(t, PDFium{})
}

func TestPopplerReadsSyntheticPayslip(t *testing.T) {
	if _, err := exec.LookPath("pdftotext"); err != nil {
		t.Skip("pdftotext not installed")
	}
	checkExample(t, Poppler{})
}

// TestReadersAgree checks that both readers find the same words at the same
// positions, within half a point.
func TestReadersAgree(t *testing.T) {
	if _, err := exec.LookPath("pdftotext"); err != nil {
		t.Skip("pdftotext not installed")
	}
	path := writeExample(t)
	a, err := PDFium{}.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Poppler{}.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	ag := Compare(a, b, 0.5)
	if ag.OnlyA != 0 || ag.OnlyB != 0 || ag.Matched != a.TokenCount() {
		t.Errorf("readers disagree: %+v", ag)
	}
}

func TestCompareCountsMismatches(t *testing.T) {
	a := layout.Document{Pages: []layout.Page{{Tokens: []layout.Token{
		{Text: "NETTO", X: 10, Y: 10}, {Text: "1,00", X: 100, Y: 10}, {Text: "X", X: 0, Y: 0},
	}}}}
	b := layout.Document{Pages: []layout.Page{{Tokens: []layout.Token{
		{Text: "NETTO", X: 10.3, Y: 10}, {Text: "1,00", X: 200, Y: 10},
	}}}}
	ag := Compare(a, b, 0.5)
	if ag.Matched != 1 || ag.OnlyA != 2 || ag.OnlyB != 1 {
		t.Errorf("Compare = %+v", ag)
	}
	if r := ag.Ratio(); r != 0.25 {
		t.Errorf("Ratio = %v, want 0.25", r)
	}
}

func TestByName(t *testing.T) {
	for _, n := range Names() {
		r, err := ByName(n)
		if err != nil || r.Name() != n {
			t.Errorf("ByName(%q) = %v, %v", n, r, err)
		}
	}
	if _, err := ByName("ocr"); err == nil {
		t.Error("ByName accepted an unknown reader")
	}
}
