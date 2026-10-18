package extract

import (
	"fmt"
	"strings"
	"testing"

	"github.com/danielededo/cedolo/internal/layout"
	"github.com/danielededo/cedolo/internal/payslip"
	"github.com/danielededo/cedolo/internal/synth"
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
// numeric period, no item codes, meal vouchers, a pension fund
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
	wantKeys := []string{"earnings.base_salary", "earnings.superminimum", "meal_vouchers",
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

// TestLeaveInDaysWithoutPrevious covers a leave table in days whose header is
// centered over wider columns, with no previous balance, and an item table
// line about used holidays that must not be mistaken for a balance.
func TestLeaveInDaysWithoutPrevious(t *testing.T) {
	pg := page(
		[]string{"40:Periodo: Maggio 2026"},
		[]string{"40:Descrizione", "390:Competenze", "480:Trattenute"},
		[]string{"40:Ferie godute", "450>120,00"},
		[]string{},
		[]string{},
		[]string{},
		[]string{"40:Giorni", "200:Spettanti", "300:Fruiti", "400:Da godere"},
		[]string{"40:Ferie", "250>2,17", "330>1,00", "450>12,50"},
		[]string{"40:Ex festività", "250>0,33", "330>0", "450>3,00"},
	)
	p := Document(layout.Document{Pages: []layout.Page{pg}})[0]
	if len(p.Leave) != 2 {
		t.Fatalf("leave = %+v", p.Leave)
	}
	h := p.Leave[0]
	if h.Kind != payslip.Holiday || h.Unit != "days" || h.Previous != nil || *h.Accrued != 217 || *h.Used != 100 || *h.Remaining != 1250 {
		t.Errorf("holiday = %+v", h)
	}
	if f := p.Leave[1]; f.Kind != payslip.FormerHolidays || *f.Used != 0 || *f.Remaining != 300 {
		t.Errorf("former holidays = %+v", f)
	}
	for _, ch := range p.Checks {
		if strings.HasPrefix(ch.Name, "leave_balance") {
			t.Errorf("unexpected balance check without a previous balance: %+v", ch)
		}
	}
	if len(p.Items) != 1 || p.Items[0].Key != "earnings.leave" {
		t.Errorf("items = %+v", p.Items)
	}
}

func TestLeaveBalanceMismatchFails(t *testing.T) {
	doc := synth.Layout(synth.Example(), synth.StyleClassic)
	for i, tok := range doc.Pages[0].Tokens {
		if tok.Text == "96,00" {
			doc.Pages[0].Tokens[i].Text = "69,00"
		}
	}
	p := Document(doc)[0]
	for _, ch := range p.Checks {
		if ch.Name == "leave_balance_holiday" && ch.OK {
			t.Errorf("mismatched balance passed: %s", ch.Detail)
		}
	}
}

func TestAmbiguousLeaveUnit(t *testing.T) {
	pg := page(
		[]string{"40:Giorni/Ore", "200:Maturato", "300:Goduto"},
		[]string{"40:Ferie", "250>2,17", "330>1,00"},
	)
	p := Document(layout.Document{Pages: []layout.Page{pg}})[0]
	if len(p.Leave) != 1 || p.Leave[0].Unit != "" {
		t.Errorf("leave = %+v", p.Leave)
	}
}

// find returns the first item with the given key.
func find(t *testing.T, items []payslip.Item, key string) payslip.Item {
	t.Helper()
	for _, it := range items {
		if it.Key == key {
			return it
		}
	}
	t.Fatalf("no item %s in %+v", key, items)
	return payslip.Item{}
}

func wantStatus(t *testing.T, name string, a payslip.Amount, cents int64, status payslip.Status) {
	t.Helper()
	if a.Cents != cents || a.Status != status {
		t.Errorf("%s = %d (%s), want %d (%s)", name, a.Cents, a.Status, cents, status)
	}
}

// TestSummaryBoxesWithCalendar covers, with invented values, a layout with
// contributions and taxes printed in summary boxes below the item table
// rather than as items; a "totale trattenute" that is only the item table
// total; a column of neutral items that are not pay; a presence calendar
// printed beside the table, whose day codes and hours sit next to the
// totals; roundings carried from the previous month and made this month;
// and leave balances in stacked blocks.
func TestSummaryBoxesWithCalendar(t *testing.T) {
	pg := page(
		[]string{"40:PERIODO DI RETRIBUZIONE", "460:MARZO 2026"},
		[]string{"25:LIVELLO 6"},
		[]string{},
		[]string{"29:VOCE", "108:DESCRIZIONE", "211:ORE/GG./NUM.", "251:VAL.UNIT./NEUTRA", "303:COMPETENZE",
			"352:TRATTENUTE", "396:Gio", "419:OL"},
		[]string{"26:0001", "50:RETRIBUZIONE ORDINARIA", "240>168,00", "290>13,09524", "345>2.200,00", "391:01Ma", "440>8,00"},
		[]string{"26:0120", "50:BUONI PASTO ELETTRONICI", "240>20,00", "290>1,00000", "345>20,00", "391:02Me", "440>8,00"},
		[]string{"26:0121", "50:REC. BUONI PASTO", "240>20,00", "290>1,00000", "395>20,00", "391:03Gi", "440>8,00"},
		[]string{"50:--------- VOCI NEUTRE ----------", "391:04Ve", "440>8,00"},
		[]string{"26:0900", "50:Contr. FPC dipendente", "200>1,00%", "290>22,00", "391:05Sa"},
		[]string{"26:0901", "50:Storno T.F.R. a F.P.C.", "200>100,00%", "290>153,34", "391:06Do"},
		[]string{"34:IMP.LE F.P.L.D. 2.220,00", "341:TOTALE COMPETENZE", "391:07Lu", "440>8,00"},
		[]string{"34:CTR. F.P.L.D. 204,02", "352:2.220,00", "391:08Ma", "440>8,00"},
		[]string{"34:CTR. FONDO PENS. 22,00", "341:TOTALE TRATTENUTE", "391:09Me", "440>8,00"},
		[]string{"34:RETRIBUZIONE PER T.F.R. 2.220,00"},
		[]string{"32:IMPONIBILE IRPEF 1.993,98", "357:20,00", "391:10Gi"},
		[]string{"340:TOTALE TRATT.SOCIALI"},
		[]string{"352:226,02"},
		[]string{"338:TOTALE TRATT.FISCALE"},
		[]string{"357:300,00"},
		[]string{"343:ARROTOND. PRECED."},
		[]string{"367:0,40"},
		[]string{"342:ARROTOND. ATTUALE"},
		[]string{"367:0,62"},
		[]string{"354:NETTO"},
		[]string{"352:1.674,20"},
	)
	p := Document(layout.Document{Pages: []layout.Page{pg}})[0]
	if p.Month != 3 || p.Year != 2026 {
		t.Errorf("period %d/%d", p.Month, p.Year)
	}
	// Three pay items, then two information items from the neutral column.
	if len(p.Items) != 5 {
		t.Fatalf("items = %+v", p.Items)
	}
	if it := find(t, p.Items, "pension_fund.contribution"); it.Side != payslip.Info || it.Cents != 2200 {
		t.Errorf("pension contribution = %+v", it)
	}
	if it := find(t, p.Items, "tfr.accrued"); it.Side != payslip.Info || it.Cents != 15334 {
		t.Errorf("TFR transfer = %+v", it)
	}
	// TFR from the neutral item, verified as the TFR base divided by 13.5
	// net of the 0.5 percent contribution.
	wantStatus(t, "TFR", p.TFR, 15334, payslip.Verified)
	wantStatus(t, "TFR base", p.TFRBase, 222000, payslip.Verified)
	if it := find(t, p.Items, "earnings.base_salary"); it.Cents != 220000 || it.Description != "RETRIBUZIONE ORDINARIA" || it.Quantity != "168,00" {
		t.Errorf("base salary = %+v", it)
	}
	if p.Items[2].Key != "meal_vouchers" || p.Items[2].Side != payslip.Deduction {
		t.Errorf("voucher recovery = %+v", p.Items[2])
	}
	wantStatus(t, "gross", p.Gross, 222000, payslip.Verified)
	wantStatus(t, "deductions", p.Deductions, 54602, payslip.Verified)
	wantStatus(t, "net", p.Net, 167420, payslip.Verified)
	wantStatus(t, "social contributions", p.SocialContributions, 22602, payslip.Verified)
	wantStatus(t, "taxes", p.Taxes, 30000, payslip.Verified)
	wantStatus(t, "previous rounding", p.RoundingPrevious, 40, payslip.Verified)
	wantStatus(t, "current rounding", p.RoundingCurrent, 62, payslip.Verified)
	wantStatus(t, "INPS base", p.INPSBase, 222000, payslip.Verified)
	for _, c := range p.Checks {
		if !c.OK && !c.Advisory {
			t.Errorf("check %s failed: %s", c.Name, c.Detail)
		}
	}
}

// TestGrandTotalAndTaxableGross covers, with invented values, a layout with
// a statistical data column beside the pay columns; a printed gross that
// leaves out expense reimbursements and contribution relief; a "totale
// trattenute" that is everything withheld, including the previous rounding;
// labels such as "totale trattenute irpef t.s." that must not be read as
// "totale trattenute"; a current rounding label printed on two stacked
// lines; and leave balances in a row of "kind measure" cells.
func TestGrandTotalAndTaxableGross(t *testing.T) {
	pg := page(
		[]string{"27:MESE RETRIBUITO", "380:LIVELLO"},
		[]string{"25:GIUGNO 2026", "380:C3"},
		[]string{},
		[]string{"27:CODICE", "57:DESCRIZIONE VOCE", "219:ORE/GIORNI", "272:BASE", "344:COMPETENZE", "423:TRATTENUTE", "502:DATI STATISTICI"},
		[]string{"25:0100", "55:LAVORO ORDIN.(mens.)", "260>21,00", "320>2.300,00", "410>2.300,00"},
		[]string{"25:0200", "55:RATA ADD.REG. A.P.", "490>25,00"},
		[]string{"25:0300", "55:NOTA SPESE OTTOBRE", "410>45,00"},
		[]string{"25:0400", "55:ESON.CONTR.IVS L.234/21", "410>40,00"},
		[]string{"25:0500", "55:CONTRIBUZIONE TFR", "570>170,00"},
		[]string{},
		[]string{"27:TOTALE LORDO", "111:IMPON. CONTR. SOC.", "488:TOTALE CONTRIBUTI SOCIALI"},
		[]string{"100>2.300,00", "180>2.300,00", "570>211,37"},
		[]string{"139:IRPEF", "488:TOTALE TRATTENUTE IRPEF T.S."},
		[]string{},
		[]string{"272:IMPONIBILE IRPEF", "356:IRPEF LORDA", "488:TOTALE TRATTENUTE IRPEF"},
		[]string{"340>2.088,63", "570>350,00"},
		[]string{"362:ARROTOND. PRECED.", "422:TRATTENUTE CORPO", "488:TOTALE TRATTENUTE"},
		[]string{"410>0,30", "480>25,00", "570>586,67"},
		[]string{"428:ARROTONDAMENTO", "488:NETTO BUSTA"},
		[]string{},
		[]string{"27:FERIE A.P.", "77:FERIE MAT.", "121:FERIE GOD.", "166:FERIE RES.", "211:PERMESSI A.P.", "257:PERMESSI MAT.",
			"393:ROL A.P.", "438:ROL MAT.", "484:ROL GOD.", "529:ROL RES."},
		[]string{"70>40,00", "115>14,67", "160>8,00", "205>46,67", "485>9,33", "525>4,00", "570>5,33"},
	)
	// The current rounding: "ATTUALE" printed just below "ARROTONDAMENTO",
	// and the values one line further down.
	var labelY float64
	for _, tok := range pg.Tokens {
		if tok.Text == "ARROTONDAMENTO" {
			labelY = tok.Y
		}
	}
	pg.Tokens = append(pg.Tokens,
		layout.Token{Text: "ATTUALE", X: 442, Y: labelY + 4.3, W: 35, H: 9},
		layout.Token{Text: "0,67", X: 457, Y: labelY + 14, W: 20, H: 9},
		layout.Token{Text: "1.799,00", X: 523, Y: labelY + 14, W: 40, H: 9},
	)
	p := Document(layout.Document{Pages: []layout.Page{pg}})[0]
	if p.Month != 6 || p.Level != "C3" {
		t.Errorf("month %d level %q", p.Month, p.Level)
	}
	// Four pay items and one information item from the statistical column.
	if len(p.Items) != 5 || p.Items[4].Side != payslip.Info || p.Items[4].Key != "info.other" {
		t.Fatalf("items = %+v (the statistical column must not count as pay)", p.Items)
	}
	find(t, p.Items, "earnings.reimbursement")
	find(t, p.Items, "earnings.contribution_relief")
	if it := find(t, p.Items, "tax.surcharge.regional"); it.Side != payslip.Deduction || it.Cents != 2500 {
		t.Errorf("regional surcharge = %+v", it)
	}
	wantStatus(t, "gross", p.Gross, 230000, payslip.Verified)
	wantStatus(t, "deductions", p.Deductions, 58667, payslip.Verified)
	wantStatus(t, "net", p.Net, 179900, payslip.Verified)
	wantStatus(t, "social contributions", p.SocialContributions, 21137, payslip.Verified)
	wantStatus(t, "taxes", p.Taxes, 35000, payslip.Verified)
	wantStatus(t, "current rounding", p.RoundingCurrent, 67, payslip.Verified)
	wantStatus(t, "IRPEF base", p.IRPEFBase, 208863, payslip.Verified)
	for _, c := range p.Checks {
		if !c.OK && !c.Advisory {
			t.Errorf("check %s failed: %s", c.Name, c.Detail)
		}
	}
	if len(p.Leave) != 2 {
		t.Fatalf("leave = %+v", p.Leave)
	}
	h, r := p.Leave[0], p.Leave[1]
	if h.Kind != payslip.Holiday || *h.Previous != 4000 || *h.Accrued != 1467 || *h.Used != 800 || *h.Remaining != 4667 {
		t.Errorf("holiday = %+v", h)
	}
	if r.Kind != payslip.ReducedHours || r.Previous != nil || *r.Accrued != 933 || *r.Used != 400 || *r.Remaining != 533 {
		t.Errorf("rol = %+v", r)
	}
}

// TestLeaveColumns covers leave tables with one column per kind and one row
// per measure.
func TestLeaveColumns(t *testing.T) {
	pg := page(
		[]string{"40:ORE", "200:FERIE", "300:PERMESSI", "400:EX FEST."},
		[]string{"40:RESIDUO AP", "250>90,00", "350>36,00", "450>8,00"},
		[]string{"40:MATURATO", "250>14,00", "350>6,00", "450>2,67"},
		[]string{"40:GODUTO", "250>8,00", "350>0,00", "450>0,00"},
		[]string{"40:SALDO", "250>96,00", "350>42,00", "450>10,67"},
	)
	p := Document(layout.Document{Pages: []layout.Page{pg}})[0]
	if len(p.Leave) != 3 || p.Leave[2].Kind != payslip.FormerHolidays || *p.Leave[2].Remaining != 1067 || p.Leave[0].Unit != "hours" {
		t.Fatalf("leave = %+v", p.Leave)
	}
	for _, c := range p.Checks {
		if strings.HasPrefix(c.Name, "leave_balance") && !c.OK {
			t.Errorf("%s: %s", c.Name, c.Detail)
		}
	}
}

// TestContractFields covers a contract printed with its CNEL code, and an
// "ORE CCNL" column (contractual hours) that is not the contract.
func TestContractFields(t *testing.T) {
	pg := page(
		[]string{"40:PERIODO DI RETRIBUZIONE", "460:MARZO 2026"},
		[]string{"25:CCNL APPLICATO", "120:H011 COMMERCIO TERZIARIO", "300:LIVELLO 4"},
		[]string{},
		[]string{"29:VOCE", "108:DESCRIZIONE", "303:COMPETENZE", "352:TRATTENUTE"},
		[]string{"26:0001", "50:RETRIBUZIONE ORDINARIA", "345>2.000,00"},
		[]string{"341:TOTALE COMPETENZE", "345>2.000,00"},
		[]string{"341:NETTO DEL MESE", "345>2.000,00"},
	)
	got := Document(layout.Document{Pages: []layout.Page{pg}})
	if len(got) != 1 {
		t.Fatalf("got %d payslips", len(got))
	}
	if got[0].CCNLCode != "H011" || !strings.Contains(got[0].CCNL, "COMMERCIO") {
		t.Errorf("CCNL = %q, code %q", got[0].CCNL, got[0].CCNLCode)
	}

	hours := page(
		[]string{"40:PERIODO DI RETRIBUZIONE", "460:MARZO 2026"},
		[]string{"25:LIVELLO", "80:ORE CCNL", "140:GG."},
		[]string{"25:4", "80:40,00", "140:26"},
		[]string{},
		[]string{"29:VOCE", "108:DESCRIZIONE", "303:COMPETENZE", "352:TRATTENUTE"},
		[]string{"26:0001", "50:RETRIBUZIONE ORDINARIA", "345>2.000,00"},
		[]string{"341:NETTO DEL MESE", "345>2.000,00"},
	)
	got = Document(layout.Document{Pages: []layout.Page{hours}})
	if len(got) != 1 || got[0].CCNL != "" || got[0].CCNLCode != "" {
		t.Errorf("ORE CCNL read as the contract: %+v", got)
	}

	// The same with the qualifier on the line above the label.
	stacked := page(
		[]string{"40:PERIODO DI RETRIBUZIONE", "460:MARZO 2026"},
		[]string{"25:LIVELLO", "140:GG."},
		[]string{"140:CCNL"},
		[]string{"25:4", "140:26"},
		[]string{},
		[]string{"29:VOCE", "108:DESCRIZIONE", "303:COMPETENZE", "352:TRATTENUTE"},
		[]string{"26:0001", "50:RETRIBUZIONE ORDINARIA", "345>2.000,00"},
		[]string{"341:NETTO DEL MESE", "345>2.000,00"},
	)
	got = Document(layout.Document{Pages: []layout.Page{stacked}})
	if len(got) != 1 || got[0].CCNL != "" {
		t.Errorf("GG. CCNL read as the contract: %q", got[0].CCNL)
	}
}

// additional builds a minimal payslip page whose header shows period and
// whose only item is described as item.
func additional(period []string, item string) layout.Page {
	return page(
		append([]string{"40:ACME SINTETICA SPA"}, period...),
		[]string{},
		[]string{"40:Descrizione", "390:Competenze", "480:Trattenute"},
		[]string{"40:" + item, "450>1.900,00"},
		[]string{"40:Contributo IVS", "540>174,61"},
		[]string{},
		[]string{"40:Totale competenze", "450>1.900,00"},
		[]string{"40:Totale trattenute", "540>174,61"},
		[]string{"40:Netto a pagare", "540>1.725,39"},
	)
}

// TestAdditionalMonths covers the ways payroll software marks the
// thirteenth and fourteenth month: words in the header, with ordinal
// marks or Roman numerals, a generic "additional month" and a period
// numbered 13 or 14.
func TestAdditionalMonths(t *testing.T) {
	for _, c := range []struct {
		name        string
		period      []string
		item        string
		year, month int
		kind        payslip.Kind
		key         string
	}{
		{"ordinal", []string{"300:DICEMBRE 2025", "420:13ª MENSILITÀ"}, "13ª MENSILITÀ", 2025, 12, payslip.Thirteenth, "earnings.thirteenth"},
		{"roman", []string{"300:DICEMBRE 2025", "420:XIII MENSILITA'"}, "XIII MENSILITA'", 2025, 12, payslip.Thirteenth, "earnings.thirteenth"},
		{"caret", []string{"300:Periodo 13/2025"}, "13^ MENSILITA'", 2025, 12, payslip.Thirteenth, "earnings.thirteenth"},
		{"fourteenth number", []string{"300:Periodo 14/2025"}, "14ma MENSILITA'", 2025, 0, payslip.Fourteenth, "earnings.fourteenth"},
		{"additional in July", []string{"300:LUGLIO 2025", "420:MENSILITA' AGGIUNTIVA"}, "RATEI 14MA", 2025, 7, payslip.Fourteenth, "earnings.fourteenth"},
		{"additional in December", []string{"300:DICEMBRE 2025", "420:MENSILITA' AGGIUNTIVA"}, "RATEO 13MA", 2025, 12, payslip.Thirteenth, "earnings.thirteenth"},
		{"ordinary December", []string{"300:DICEMBRE 2025"}, "Retribuzione", 2025, 12, payslip.Monthly, "earnings.base_salary"},
		{"year only", []string{"300:PERIODO DI RETRIBUZIONE", "460:13MA MENS. 2025"}, "GRATIFICA NATALIZIA", 2025, 12, payslip.Thirteenth, "earnings.thirteenth"},
		{"fourteenth year only", []string{"460:14ª MENSILITÀ 2025"}, "14ª MENSILITÀ", 2025, 0, payslip.Fourteenth, "earnings.fourteenth"},
	} {
		got := Document(layout.Document{Pages: []layout.Page{additional(c.period, c.item)}})
		if len(got) != 1 {
			t.Fatalf("%s: %d payslips", c.name, len(got))
		}
		p := got[0]
		if p.Year != c.year || p.Month != c.month || p.Kind != c.kind {
			t.Errorf("%s: %d/%d %s, want %d/%d %s", c.name, p.Month, p.Year, p.Kind, c.month, c.year, c.kind)
		}
		if len(p.Items) == 0 || p.Items[0].Key != c.key {
			t.Errorf("%s: items %+v, want %s first", c.name, p.Items, c.key)
		}
		if p.Net.Cents != 172539 {
			t.Errorf("%s: net %d", c.name, p.Net.Cents)
		}
	}
}

// TestThirteenthAfterDecember reads a file with the December payslip and
// the thirteenth month numbered 13: two payslips, not one.
func TestThirteenthAfterDecember(t *testing.T) {
	doc := layout.Document{Pages: []layout.Page{
		additional([]string{"300:Periodo 12/2025"}, "Retribuzione"),
		additional([]string{"300:Periodo 13/2025"}, "Tredicesima"),
	}}
	doc.Pages[1].Number = 2
	got := Document(doc)
	if len(got) != 2 || got[0].Kind != payslip.Monthly || got[1].Kind != payslip.Thirteenth || got[1].Month != 12 {
		t.Fatalf("got %+v", got)
	}
}

// TestTotalsUnderSmallLabels covers totals printed under labels in small
// type, abbreviated, with a statistics column whose numbers sit just beside
// the labels on another baseline, and a row of year-to-date values labeled
// on its left.
func TestTotalsUnderSmallLabels(t *testing.T) {
	label := func(x, y float64, s string) layout.Token {
		return layout.Token{Text: s, X: x, Y: y, W: 3 * float64(len(s)), H: 5}
	}
	value := func(x, y float64, s string) layout.Token {
		return layout.Token{Text: s, X: x, Y: y, W: 4 * float64(len(s)), H: 8}
	}
	pg := layout.Page{Number: 1, Width: 595, Height: 842, Tokens: []layout.Token{
		label(27, 470, "IMP."), label(214, 470, "IMP."), label(227, 470, "IRPEF"),
		label(359, 470, "TOT."), label(372, 470, "TRAT."), label(388, 470, "IRPEF"),
		value(425, 472, "0,00"),
		value(221, 484, "2.088,63"), value(384, 484, "350,00"), value(425, 484, "0,00"),
		// Here the statistics number is far enough below to make a line of
		// its own.
		label(359, 500, "TOT."), label(372, 500, "TRATTENUTE"),
		value(425, 504, "0,00"),
		value(384, 514, "586,67"), value(425, 514, "0,00"),
		label(245, 600, "IMPONIBILE"), label(278, 600, "IRPEF"),
		label(28, 606, "PROGRES."), label(33, 612, "ANNUI"),
		value(257, 610, "25.000,00"),
	}}
	lines := buildLines(pg)
	for _, c := range []struct {
		name   string
		labels []string
		want   int64
	}{
		{"deductions", deductionLabels, 58667},
		{"taxes", taxLabels, 35000},
		{"IRPEF base", irpefBaseLabels, 208863},
	} {
		a, ok := amountField(lines, pg.Width, c.labels)
		if !ok || a.Cents != c.want {
			t.Errorf("%s = %d (found %v), want %d", c.name, a.Cents, ok, c.want)
		}
	}
}

// TestNotAPayslip covers a document that has a date reading as a pay period
// and some amounts, but no pay item, total or leave balance, such as a tax
// return: it is not a payslip.
func TestNotAPayslip(t *testing.T) {
	pg := page(
		[]string{"40:PROSPETTO DI LIQUIDAZIONE", "300:Periodo 07/2017"},
		[]string{"40:REDDITO COMPLESSIVO", "450>25.000,00"},
		[]string{"40:IMPOSTA NETTA", "450>4.000,00"},
		[]string{"40:ADDIZ. REGIONALE IRPEF", "450>300,00"},
	)
	if got := Document(layout.Document{Pages: []layout.Page{pg}}); len(got) != 0 {
		t.Errorf("got %d payslips, want none: %+v", len(got), got)
	}
}

// TestTaxReturnRefundOutsideGross covers a tax return refund paid in the
// payslip: it is not pay, so the printed gross leaves it out.
func TestTaxReturnRefundOutsideGross(t *testing.T) {
	pg := page(
		[]string{"40:Periodo: Settembre 2026"},
		[]string{"40:Descrizione", "390:Competenze", "480:Trattenute"},
		[]string{"40:Retribuzione ordinaria", "450>2.000,00"},
		[]string{"40:IRPEF A CREDITO -MOD.730-", "450>300,00"},
		[]string{"40:Ritenuta IRPEF", "540>400,00"},
		[]string{},
		[]string{"40:Totale lordo", "450>2.000,00"},
		[]string{"40:Totale trattenute", "540>400,00"},
		[]string{"40:Netto del mese", "540>1.900,00"},
	)
	p := Document(layout.Document{Pages: []layout.Page{pg}})[0]
	if it := find(t, p.Items, "tax.return"); it.Side != payslip.Earning || it.Cents != 30000 {
		t.Errorf("refund = %+v", it)
	}
	wantStatus(t, "gross", p.Gross, 200000, payslip.Verified)
	wantStatus(t, "net", p.Net, 190000, payslip.Verified)
}

// TestTotalReadByArithmetic covers a number printed right after a total's
// label that is not its value: the value under the label is the one with
// which the payslip adds up, so it is chosen and marked (ADR 0009).
func TestTotalReadByArithmetic(t *testing.T) {
	pg := page(
		[]string{"40:ACME SINTETICA SPA", "380:Mese retribuito:"},
		[]string{"380:03/2026"},
		[]string{},
		[]string{"40:Descrizione", "390:Competenze", "480:Trattenute"},
		[]string{"40:Minimo contrattuale", "450>1.900,00"},
		[]string{"40:Superminimo individuale", "450>200,00"},
		[]string{"40:Contributo IVS a carico dipendente", "540>192,99"},
		[]string{"40:Irpef", "540>333,27"},
		[]string{},
		[]string{"40:Totale competenze", "200:Totale trattenute", "360:Netto", "470:1,00"},
		[]string{"130>2.100,00", "290>526,26", "430>1.573,74"},
	)
	got := Document(layout.Document{Pages: []layout.Page{pg}})
	if len(got) != 1 {
		t.Fatalf("got %d payslips", len(got))
	}
	p := got[0]
	wantStatus(t, "net", p.Net, 157374, payslip.Verified)
	wantStatus(t, "deductions", p.Deductions, 52626, payslip.Verified)
	if !p.Net.Alternative || p.Gross.Alternative || p.Deductions.Alternative {
		t.Errorf("alternative: net %v, gross %v, deductions %v", p.Net.Alternative, p.Gross.Alternative, p.Deductions.Alternative)
	}
	var noted bool
	for _, c := range p.Checks {
		noted = noted || c.Name == "total_reading" && strings.Contains(c.Detail, "net pay read as 1.573,74 rather than 1,00")
	}
	if !noted {
		t.Errorf("no total_reading check in %+v", p.Checks)
	}
}

func TestCombinationsPreferFewerAlternatives(t *testing.T) {
	two := make([]payslip.Amount, 2)
	three := make([]payslip.Amount, 3)
	got := combinations([][]payslip.Amount{three, two, make([]payslip.Amount, 1)})
	want := [][]int{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}, {2, 0, 0}, {1, 1, 0}, {2, 1, 0}}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("combinations = %v, want %v", got, want)
	}
}

// TestPartialAgreementKeepsPreferredReadings checks that an alternative
// reading which fixes one check while another still fails is not taken.
func TestPartialAgreementKeepsPreferredReadings(t *testing.T) {
	amount := func(c int64) payslip.Amount { return payslip.Amount{Cents: c, Status: payslip.Extracted} }
	p := payslip.Payslip{
		Items: []payslip.Item{{Side: payslip.Earning, Key: "earnings.base_salary", Cents: 200000},
			{Side: payslip.Deduction, Key: "tax.irpef", Cents: 50000}},
		Gross: amount(190000), Deductions: amount(50000), Net: amount(140000),
	}
	resolve(&p, []total{
		{"gross pay", func(p *payslip.Payslip) *payslip.Amount { return &p.Gross }, []payslip.Amount{amount(190000), amount(200000)}},
		{"net pay", func(p *payslip.Payslip) *payslip.Amount { return &p.Net }, []payslip.Amount{amount(140000)}},
	})
	if p.Gross.Cents != 190000 || p.Gross.Alternative {
		t.Errorf("gross = %d, alternative %v; want the preferred 190000", p.Gross.Cents, p.Gross.Alternative)
	}
	wantStatus(t, "net", p.Net, 140000, payslip.Inconsistent)
}
