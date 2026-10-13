package extract

import (
	"strings"
	"testing"

	"github.com/danielededo/cedolo/internal/layout"
	"github.com/danielededo/cedolo/internal/payslip"
)

// page builds a layout from rows of cells. Each cell is "x:text" where x is
// the left edge in points, or "x>text" for text right-aligned on x. Words are
// given a width of 5 points per character. It lets tests describe layouts
// that the synthetic generator does not produce.
func page(rows ...[]string) layout.Page {
	p := layout.Page{Number: 1, Width: 595, Height: 842}
	for r, cells := range rows {
		y := 40 + float64(r)*14
		for _, c := range cells {
			right := strings.Contains(c, ">")
			sep := ":"
			if right {
				sep = ">"
			}
			pos, text, _ := strings.Cut(c, sep)
			var x float64
			for _, ch := range pos {
				x = x*10 + float64(ch-'0')
			}
			words := strings.Fields(text)
			width := 5 * float64(len([]rune(text)))
			if right {
				x -= width
			}
			for _, w := range words {
				ww := 5 * float64(len([]rune(w)))
				p.Tokens = append(p.Tokens, layout.Token{Text: w, X: x, Y: y, W: ww, H: 9})
				x += ww + 5
			}
		}
	}
	return p
}

// TestForeignLayout covers traits none of the synthetic styles has: headers
// centered over wide columns, euro signs, labels ending with colons, a
// numeric period, no item codes, an unrecognized item, a pension fund
// deduction, and totals printed on the left of their values with a large gap.
func TestForeignLayout(t *testing.T) {
	pg := page(
		[]string{"40:ACME SINTETICA SPA", "380:Mese retribuito:"},
		[]string{"380:03/2026"},
		[]string{"40:Livello: 5S", "200:CCNL: METALMECCANICO INDUSTRIA"},
		[]string{},
		[]string{"40:Descrizione", "300:Ore", "390:Competenze", "480:Trattenute"},
		[]string{"40:Minimo contrattuale", "320>173", "450>€ 1.900,00"},
		[]string{"40:Superminimo individuale", "450>€ 120,00"},
		[]string{"40:Buoni pasto sostitutivi", "450>€ 80,00"},
		[]string{"40:Contributo IVS a carico dipendente", "540>€ 192,02"},
		[]string{"40:Fondo pensione COMETA", "540>€ 24,24"},
		[]string{"40:Irpef", "540>€ 310,00"},
		[]string{},
		[]string{"40:Totale competenze:", "450>€ 2.100,00"},
		[]string{"40:Totale trattenute:", "540>€ 526,26"},
		[]string{"40:Netto a pagare:", "540>€ 1.573,74"},
		[]string{"40:Imponibile contributivo:", "250>2.089,45"},
	)
	got := Document(layout.Document{Pages: []layout.Page{pg}})
	if len(got) != 1 {
		t.Fatalf("got %d payslips", len(got))
	}
	p := got[0]
	if p.Year != 2026 || p.Month != 3 {
		t.Errorf("period = %d/%d, want 3/2026", p.Month, p.Year)
	}
	if p.Level != "5S" || p.CCNL != "METALMECCANICO INDUSTRIA" {
		t.Errorf("level %q CCNL %q", p.Level, p.CCNL)
	}
	wantKeys := []string{"earnings.base_salary", "earnings.superminimum", "earnings.other",
		"deductions.inps", "deductions.pension_fund", "tax.irpef"}
	if len(p.Items) != len(wantKeys) {
		t.Fatalf("got %d items: %+v", len(p.Items), p.Items)
	}
	for i, k := range wantKeys {
		if p.Items[i].Key != k {
			t.Errorf("item %d %q key = %s, want %s", i, p.Items[i].Description, p.Items[i].Key, k)
		}
	}
	if p.Items[0].Code != "" || p.Items[0].Quantity != "173" {
		t.Errorf("first item = %+v", p.Items[0])
	}
	for name, a := range map[string]payslip.Amount{"gross": p.Gross, "deductions": p.Deductions, "net": p.Net} {
		if a.Status != payslip.Verified {
			t.Errorf("%s status = %s", name, a.Status)
		}
	}
	if p.Net.Cents != 157374 {
		t.Errorf("net = %d", p.Net.Cents)
	}
	// 192,02 is 9,19% of 2.089,45.
	if p.INPSBase.Status != payslip.Verified {
		t.Errorf("INPS base status = %s; checks %+v", p.INPSBase.Status, p.Checks)
	}
}

// TestUnsignedSingleColumn covers a single amount column without signs: the
// side comes from what the item is.
func TestUnsignedSingleColumn(t *testing.T) {
	pg := page(
		[]string{"40:Periodo: Gennaio 2026"},
		[]string{"40:Voce", "90:Descrizione", "500:Importo"},
		[]string{"40:1000", "90:Retribuzione", "540>1.500,00"},
		[]string{"40:8000", "90:Contributi INPS", "540>137,85"},
		[]string{"40:8100", "90:Ritenute IRPEF", "540>200,00"},
		[]string{},
		[]string{"300:Netto in busta", "540>1.162,15"},
	)
	p := Document(layout.Document{Pages: []layout.Page{pg}})[0]
	if len(p.Items) != 3 || p.Items[1].Side != payslip.Deduction || p.Items[2].Side != payslip.Deduction {
		t.Fatalf("items = %+v", p.Items)
	}
	if p.Net.Status != payslip.Verified || p.Gross.Status != payslip.Inferred {
		t.Errorf("net %s, gross %s", p.Net.Status, p.Gross.Status)
	}
}
