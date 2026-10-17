package extract

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danielededo/cedolo/internal/layout"
	"github.com/danielededo/cedolo/internal/money"
	"github.com/danielededo/cedolo/internal/payslip"
	"github.com/danielededo/cedolo/internal/pdftext"
	"github.com/danielededo/cedolo/internal/synth"
)

// checkAgainst compares an extraction with the synthetic payslip it was
// generated from: header, every item with its side and taxonomy key, the
// totals, and that every total was verified by a cross-check.
func checkAgainst(t *testing.T, got []payslip.Payslip, want synth.Payslip, s synth.Style) {
	t.Helper()
	if len(got) != 1 {
		t.Fatalf("got %d payslips, want 1", len(got))
	}
	p := got[0]
	if p.Year != want.Year || p.Month != int(want.Month) || p.Kind != payslip.Monthly {
		t.Errorf("period = %d/%d %s, want %d/%d monthly", p.Month, p.Year, p.Kind, want.Month, want.Year)
	}
	if p.CCNL != want.CCNL || p.Level != want.Level {
		t.Errorf("CCNL %q level %q, want %q %q", p.CCNL, p.Level, want.CCNL, want.Level)
	}
	wantCode := ""
	if s == synth.StyleGrid {
		wantCode = "H011"
	}
	if p.CCNLCode != wantCode {
		t.Errorf("CCNL code %q, want %q", p.CCNLCode, wantCode)
	}
	if len(p.Items) != len(want.Items) {
		t.Fatalf("got %d items, want %d: %+v", len(p.Items), len(want.Items), p.Items)
	}
	for i, w := range want.Items {
		g := p.Items[i]
		side := payslip.Earning
		if w.Deduction {
			side = payslip.Deduction
		}
		if g.Code != w.Code || g.Description != w.Description || g.Cents != w.Amount || g.Side != side || g.Key != w.Key {
			t.Errorf("item %d = %+v, want code %s %q %d %s %s", i, g, w.Code, w.Description, w.Amount, side, w.Key)
		}
		if g.Quantity != w.Quantity {
			t.Errorf("item %d quantity = %q, want %q", i, g.Quantity, w.Quantity)
		}
		// The grid style folds deduction rates into the description and the
		// signed style prints only one reference value per line.
		printed := !(s == synth.StyleGrid && w.Deduction) && !(s == synth.StyleSigned && w.Quantity != "")
		if w.Base != "" && printed && g.Base != w.Base {
			t.Errorf("item %d base = %q, want %q", i, g.Base, w.Base)
		}
	}
	gross, deductions, net := want.Totals()
	for _, c := range []struct {
		name string
		got  payslip.Amount
		want int64
	}{
		{"gross", p.Gross, gross},
		{"deductions", p.Deductions, deductions},
		{"net", p.Net, net},
		{"INPS base", p.INPSBase, want.INPSBase},
		{"IRPEF base", p.IRPEFBase, want.IRPEFBase},
		{"TFR", p.TFR, want.TFRAccrued},
	} {
		if c.got.Cents != c.want || c.got.Status != payslip.Verified {
			t.Errorf("%s = %d (%s), want %d verified", c.name, c.got.Cents, c.got.Status, c.want)
		}
		if c.got.Source == nil {
			t.Errorf("%s has no source", c.name)
		}
	}
	if len(p.Leave) != len(want.Leave) {
		t.Fatalf("got %d leave balances, want %d: %+v", len(p.Leave), len(want.Leave), p.Leave)
	}
	for i, w := range want.Leave {
		g := p.Leave[i]
		// Vertical labels of several words are read word by word, so the
		// label may be the kind's last word only ("FEST." of "EX FEST.").
		// The spaced style, like the payslips it imitates, prints no unit.
		unitless := s == synth.StyleSpaced
		if string(g.Kind) != w.Key || !strings.Contains(w.Kind, g.Label) || g.Label == "" || (g.Unit == "") != unitless {
			t.Errorf("leave %d = %s %q unit %q, want %s %q", i, g.Kind, g.Label, g.Unit, w.Key, w.Kind)
		}
		for name, pair := range map[string]struct {
			got  *int64
			want string
		}{"previous": {g.Previous, w.Previous}, "accrued": {g.Accrued, w.Accrued}, "used": {g.Used, w.Used}, "remaining": {g.Remaining, w.Remaining}} {
			wv, _ := money.Parse(pair.want)
			if pair.got == nil || *pair.got != wv {
				t.Errorf("leave %s %s = %v, want %d", w.Key, name, pair.got, wv)
			}
		}
	}
	balanceChecks := 0
	for _, ch := range p.Checks {
		if !ch.OK {
			t.Errorf("check %s failed: %s", ch.Name, ch.Detail)
		}
		if strings.HasPrefix(ch.Name, "leave_balance_") {
			balanceChecks++
		}
	}
	if balanceChecks != len(want.Leave) {
		t.Errorf("got %d leave balance checks, want %d", balanceChecks, len(want.Leave))
	}
}

func TestEveryStyleFromLayout(t *testing.T) {
	for _, s := range synth.Styles() {
		t.Run(s.String(), func(t *testing.T) {
			want := synth.ExampleFor(s)
			checkAgainst(t, Document(synth.Layout(want, s)), want, s)
		})
	}
}

// TestEveryStyleFromPDF runs the whole chain: synthetic PDF, PDFium text
// layer, extraction.
func TestEveryStyleFromPDF(t *testing.T) {
	for _, s := range synth.Styles() {
		t.Run(s.String(), func(t *testing.T) {
			want := synth.ExampleFor(s)
			path := filepath.Join(t.TempDir(), s.String()+".pdf")
			f, err := os.Create(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := synth.WritePDF(f, want, s); err != nil {
				t.Fatal(err)
			}
			f.Close()
			doc, err := pdftext.PDFium{}.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			checkAgainst(t, Document(doc), want, s)
		})
	}
}

func TestInconsistentNetIsFlagged(t *testing.T) {
	doc := synth.Layout(synth.Example(), synth.StyleClassic)
	for i, tok := range doc.Pages[0].Tokens {
		if tok.Text == "1.667,40" {
			doc.Pages[0].Tokens[i].Text = "1.676,40"
		}
	}
	p := Document(doc)[0]
	if p.Net.Status != payslip.Inconsistent {
		t.Errorf("net status = %s, want inconsistent", p.Net.Status)
	}
	if p.Gross.Status != payslip.Verified {
		t.Errorf("gross status = %s, want verified by the item sum", p.Gross.Status)
	}
}

func TestMissingTotalsAreInferred(t *testing.T) {
	doc := synth.Layout(synth.Example(), synth.StyleClassic)
	var kept []layout.Token
	drop := map[string]bool{"TOTALE": true, "NETTO": true, "2.194,50": true, "527,10": true, "1.667,40": true}
	for _, tok := range doc.Pages[0].Tokens {
		if !drop[tok.Text] {
			kept = append(kept, tok)
		}
	}
	doc.Pages[0].Tokens = kept
	p := Document(doc)[0]
	gross, deductions, net := synth.Example().Totals()
	for name, c := range map[string]struct {
		a    payslip.Amount
		want int64
	}{"gross": {p.Gross, gross}, "deductions": {p.Deductions, deductions}, "net": {p.Net, net}} {
		if c.a.Status != payslip.Inferred || c.a.Cents != c.want {
			t.Errorf("%s = %d (%s), want %d inferred", name, c.a.Cents, c.a.Status, c.want)
		}
	}
}

func TestMultiplePayslipsInOneDocument(t *testing.T) {
	oct := synth.Example()
	nov := synth.Example()
	nov.Month++
	a := synth.Layout(oct, synth.StyleClassic).Pages[0]
	b := synth.Layout(nov, synth.StyleSigned).Pages[0]
	b.Number = 2
	got := Document(layout.Document{Pages: []layout.Page{a, b}})
	if len(got) != 2 || got[0].Month != 10 || got[1].Month != 11 {
		t.Fatalf("got %d payslips: %+v", len(got), got)
	}
	if got[1].Pages[0] != 2 {
		t.Errorf("second payslip pages = %v", got[1].Pages)
	}
}

func TestThirteenthMonthKind(t *testing.T) {
	p := synth.Example()
	doc := synth.Layout(p, synth.StyleClassic)
	doc.Pages[0].Tokens = append(doc.Pages[0].Tokens, layout.Token{Text: "TREDICESIMA", X: 300, Y: 40, W: 50, H: 9})
	if got := Document(doc)[0].Kind; got != payslip.Thirteenth {
		t.Errorf("kind = %s, want 13th", got)
	}
}

func TestNumericPeriod(t *testing.T) {
	page := layout.Page{Number: 1, Width: 595, Height: 842, Tokens: []layout.Token{
		{Text: "Stampato", X: 40, Y: 20, W: 30, H: 9}, {Text: "27/10/2026", X: 80, Y: 20, W: 40, H: 9},
		{Text: "Mese", X: 40, Y: 40, W: 20, H: 9}, {Text: "retribuito", X: 65, Y: 40, W: 40, H: 9},
		{Text: "09/2026", X: 40, Y: 52, W: 30, H: 9},
	}}
	y, m, _ := findPeriod(buildLines(page))
	if y != 2026 || m != 9 {
		t.Errorf("period = %d/%d, want 9/2026", m, y)
	}
}

// Two complete payslips of the same month in one file, as when the same
// month is issued twice or two employers' payslips are joined, stay two;
// a payslip continued on a second page stays one.
func TestSamePeriodPayslipsInOneFile(t *testing.T) {
	a := synth.Layout(synth.ExampleFor(synth.StyleClassic), synth.StyleClassic)
	b := synth.Layout(synth.ExampleFor(synth.StyleGrid), synth.StyleGrid)
	b.Pages[0].Number = 2
	got := Document(layout.Document{Pages: []layout.Page{a.Pages[0], b.Pages[0]}})
	if len(got) != 2 {
		t.Fatalf("got %d payslips, want 2", len(got))
	}
	for _, p := range got {
		if p.Net.Status != payslip.Verified {
			t.Errorf("page %v: net %s", p.Pages, p.Net.Status)
		}
	}

	// The first page without its totals, the second with them only.
	first, second := a.Pages[0], a.Pages[0]
	second.Number = 2
	first.Tokens, second.Tokens = nil, nil
	for _, tok := range a.Pages[0].Tokens {
		if tok.Y < 290 { // above the net pay line
			first.Tokens = append(first.Tokens, tok)
		}
	}
	second.Tokens = append(second.Tokens, a.Pages[0].Tokens...)
	if got := Document(layout.Document{Pages: []layout.Page{first, second}}); len(got) != 1 {
		t.Errorf("continuation page split off: %d payslips", len(got))
	}
}
