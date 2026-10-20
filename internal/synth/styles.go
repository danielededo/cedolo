package synth

import (
	"fmt"
	"time"

	"github.com/danielededo/cedolo/internal/money"
)

// Style selects a payslip layout. The styles imitate traits found across
// Italian payroll software without copying any vendor's layout: column order,
// separate or signed amount columns, totals beside their labels or in a grid
// below them, and different wording for the same items.
type Style int

const (
	// StyleClassic prints earnings and deductions in two columns and each
	// total on the same line as its label.
	StyleClassic Style = iota
	// StyleSigned prints a single amount column where deductions carry a
	// trailing minus, with mixed-case labels.
	StyleSigned
	// StyleGrid puts deductions before earnings, uses different column names,
	// and prints totals and taxable bases in label cells with values below.
	StyleGrid
	// StyleSpaced places every character on its own with extra spacing, so
	// that text layers report words a character at a time. Its totals sit
	// under small labels, with a statistics column printed just beside them
	// on another baseline and a row of year-to-date values below; the INPS
	// contributions and the IRPEF appear only as totals.
	StyleSpaced
)

// Styles lists every style.
func Styles() []Style { return []Style{StyleClassic, StyleSigned, StyleGrid, StyleSpaced} }

func (s Style) String() string {
	switch s {
	case StyleClassic:
		return "classic"
	case StyleSigned:
		return "signed"
	case StyleGrid:
		return "grid"
	case StyleSpaced:
		return "spaced"
	}
	return fmt.Sprintf("Style(%d)", int(s))
}

// ParseStyle returns the style with the given name.
func ParseStyle(name string) (Style, error) {
	for _, s := range Styles() {
		if s.String() == name {
			return s, nil
		}
	}
	return 0, fmt.Errorf("unknown style %q", name)
}

var italianMonths = [...]string{"", "GENNAIO", "FEBBRAIO", "MARZO", "APRILE", "MAGGIO", "GIUGNO",
	"LUGLIO", "AGOSTO", "SETTEMBRE", "OTTOBRE", "NOVEMBRE", "DICEMBRE"}

func render(p Payslip, s Style) *canvas {
	c := &canvas{}
	switch s {
	case StyleSigned:
		renderSigned(c, p)
	case StyleGrid:
		renderGrid(c, p)
	case StyleSpaced:
		c.tracking = 0.1
		renderSpaced(c, p)
	default:
		renderClassic(c, p)
	}
	c.text(40, 40, 7, false, fmt.Sprintf("Documento sintetico generato da cedolo il %s. Dati inventati.",
		time.Date(p.Year, p.Month, 27, 0, 0, 0, 0, time.UTC).Format("02/01/2006")))
	return c
}

func renderClassic(c *canvas, p Payslip) {
	const (
		colCode, colDesc                 = 40.0, 80.0
		colQty, colBase, colEarn, colDed = 330.0, 400.0, 480.0, 555.0
	)
	c.text(colCode, 800, 14, true, "CEDOLINO PAGA")
	c.text(colCode, 780, 9, true, p.Employer)
	c.text(400, 780, 9, false, "PERIODO")
	c.text(450, 780, 9, true, fmt.Sprintf("%s %d", italianMonths[p.Month], p.Year))

	c.text(colCode, 750, 8, false, "DIPENDENTE")
	c.text(colCode, 738, 9, true, p.Employee)
	c.text(250, 750, 8, false, "MATRICOLA")
	c.text(250, 738, 9, false, p.EmployeeNumber)
	c.text(330, 750, 8, false, "QUALIFICA")
	c.text(330, 738, 9, false, p.Qualification)
	c.text(410, 750, 8, false, "LIVELLO")
	c.text(410, 738, 9, false, p.Level)
	if p.EmployeeCode != "" {
		c.text(460, 750, 8, false, "CODICE FISCALE")
		c.text(460, 738, 9, false, p.EmployeeCode)
	}
	if p.EmployerCode != "" {
		c.text(250, 780, 9, false, "P.IVA "+p.EmployerCode)
	}
	c.text(colCode, 722, 8, false, "CCNL")
	c.text(colDesc, 722, 9, false, p.CCNL)

	y := 695.0
	c.text(colCode, y, 8, true, "COD")
	c.text(colDesc, y, 8, true, "DESCRIZIONE")
	c.textRight(colQty, y, 8, true, "QUANTITÀ")
	c.textRight(colBase, y, 8, true, "BASE")
	c.textRight(colEarn, y, 8, true, "COMPETENZE")
	c.textRight(colDed, y, 8, true, "TRATTENUTE")
	c.rule(colCode, y-4, colDed)

	y -= 18
	for _, it := range p.Items {
		c.text(colCode, y, 9, false, it.Code)
		c.text(colDesc, y, 9, false, it.Description)
		if it.Quantity != "" {
			c.textRight(colQty, y, 9, false, it.Quantity)
		}
		if it.Base != "" {
			c.textRight(colBase, y, 9, false, it.Base)
		}
		col := colEarn
		if it.Deduction {
			col = colDed
		}
		c.textRight(col, y, 9, false, money.Format(it.Amount))
		y -= 14
	}

	gross, deductions, net := p.Totals()
	y -= 6
	c.rule(colCode, y+10, colDed)
	c.text(colDesc, y, 9, true, "TOTALE COMPETENZE")
	c.textRight(colEarn, y, 9, true, money.Format(gross))
	y -= 14
	c.text(colDesc, y, 9, true, "TOTALE TRATTENUTE")
	c.textRight(colDed, y, 9, true, money.Format(deductions))
	y -= 20
	c.text(colDesc, y, 11, true, "NETTO DEL MESE")
	c.textRight(colDed, y, 11, true, money.Format(net))

	y -= 30
	c.text(colCode, y, 8, false, "IMPONIBILE INPS")
	c.textRight(220, y, 8, false, money.Format(p.INPSBase))
	c.text(260, y, 8, false, "IMPONIBILE IRPEF")
	c.textRight(colEarn, y, 8, false, money.Format(p.IRPEFBase))
	y -= 16
	c.text(colCode, y, 8, true, "TFR")
	c.text(colDesc, y, 8, false, "QUOTA MATURATA NEL MESE")
	c.textRight(colEarn, y, 8, false, money.Format(p.TFRAccrued))

	y -= 24
	c.text(colCode, y, 8, true, "ORE")
	c.textRight(260, y, 8, true, "RES. PREC.")
	c.textRight(330, y, 8, true, "MATURATE")
	c.textRight(400, y, 8, true, "GODUTE")
	c.textRight(480, y, 8, true, "RESIDUE")
	for _, l := range p.Leave {
		y -= 12
		c.text(colCode, y, 8, false, l.Kind)
		c.textRight(260, y, 8, false, l.Previous)
		c.textRight(330, y, 8, false, l.Accrued)
		c.textRight(400, y, 8, false, l.Used)
		c.textRight(480, y, 8, false, l.Remaining)
	}
}

func renderSigned(c *canvas, p Payslip) {
	const colCode, colDesc, colRef, colAmount = 40.0, 90.0, 430.0, 555.0
	c.text(colCode, 800, 13, true, "Busta paga")
	c.text(colCode, 782, 9, false, "Ditta: "+p.Employer)
	c.text(330, 782, 9, false, fmt.Sprintf("Periodo di paga: %s %d", titleCase(italianMonths[p.Month]), p.Year))
	c.text(colCode, 766, 9, false, "Dipendente: "+p.Employee)
	c.text(330, 766, 9, false, "Livello: "+p.Level)
	c.text(colCode, 752, 9, false, "Contratto: CCNL "+p.CCNL)

	y := 725.0
	c.text(colCode, y, 8, true, "Voce")
	c.text(colDesc, y, 8, true, "Descrizione")
	c.textRight(colRef, y, 8, true, "Rif.")
	c.textRight(colAmount, y, 8, true, "Importo")
	c.rule(colCode, y-4, colAmount)
	y -= 16
	for _, it := range p.Items {
		c.text(colCode, y, 9, false, it.Code)
		c.text(colDesc, y, 9, false, it.Description)
		if ref := firstNonEmpty(it.Quantity, it.Base); ref != "" {
			c.textRight(colRef, y, 9, false, ref)
		}
		amount := money.Format(it.Amount)
		if it.Deduction {
			amount += "-"
		}
		c.textRight(colAmount, y, 9, false, amount)
		y -= 13
	}

	gross, deductions, net := p.Totals()
	y -= 10
	for _, row := range []struct {
		label string
		cents int64
	}{
		{"Totale lordo", gross},
		{"Totale ritenute", deductions},
		{"Netto in busta", net},
	} {
		c.text(330, y, 9, true, row.label)
		c.textRight(colAmount, y, 9, true, money.Format(row.cents))
		y -= 14
	}
	y -= 16
	c.text(colCode, y, 8, false, "Imponibile previdenziale")
	c.textRight(250, y, 8, false, money.Format(p.INPSBase))
	c.text(330, y, 8, false, "Imponibile fiscale")
	c.textRight(colAmount, y, 8, false, money.Format(p.IRPEFBase))
	y -= 14
	c.text(colCode, y, 8, false, "Quota TFR maturata")
	c.textRight(250, y, 8, false, money.Format(p.TFRAccrued))
	y -= 24
	for _, l := range p.Leave {
		c.text(colCode, y, 8, true, l.Kind+" (ore)")
		c.text(130, y, 8, false, "res. prec. "+l.Previous)
		c.text(230, y, 8, false, "maturate "+l.Accrued)
		c.text(330, y, 8, false, "godute "+l.Used)
		c.text(430, y, 8, false, "residue "+l.Remaining)
		y -= 12
	}
}

func renderGrid(c *canvas, p Payslip) {
	const (
		colCode, colDesc               = 40.0, 85.0
		colQty, colBase, colDed, colEa = 320.0, 390.0, 470.0, 555.0
	)
	c.text(colCode, 805, 12, true, "PROSPETTO PAGA")
	cells := func(y float64, labels, values []string) {
		for i := range labels {
			x := colCode + float64(i)*130
			c.text(x, y, 7, false, labels[i])
			c.text(x, y-11, 9, true, values[i])
		}
	}
	cells(785, []string{"AZIENDA", "PERIODO DI RETRIBUZIONE", "LIVELLO", "CCNL APPLICATO"},
		[]string{p.Employer, fmt.Sprintf("%s %d", italianMonths[p.Month], p.Year), p.Level + "° LIV.", p.CCNL})
	cells(755, []string{"COGNOME E NOME", "MATRICOLA", "QUALIFICA"},
		[]string{p.Employee, p.EmployeeNumber, p.Qualification})

	y := 715.0
	c.text(colCode, y, 7, true, "CODICE")
	c.text(colDesc, y, 7, true, "DESCRIZIONE VOCE")
	c.textRight(colQty, y, 7, true, "ORE/GG")
	c.textRight(colBase, y, 7, true, "DATO BASE")
	c.textRight(colDed, y, 7, true, "RITENUTE")
	c.textRight(colEa, y, 7, true, "SPETTANZE")
	y -= 15
	for _, it := range p.Items {
		c.text(colCode, y, 8, false, it.Code)
		c.text(colDesc, y, 8, false, it.Description)
		if it.Quantity != "" {
			c.textRight(colQty, y, 8, false, it.Quantity)
		}
		// Deduction rates are part of the description in this style.
		if it.Base != "" && !it.Deduction {
			c.textRight(colBase, y, 8, false, it.Base)
		}
		col := colEa
		if it.Deduction {
			col = colDed
		}
		c.textRight(col, y, 8, false, money.Format(it.Amount))
		y -= 12
	}

	gross, deductions, net := p.Totals()
	var rounding int64
	for _, it := range p.Items {
		if it.Key == "rounding" {
			rounding = it.Amount
		}
	}
	y -= 20
	grid := func(y float64, labels []string, values []int64) {
		for i := range labels {
			x := colCode + float64(i)*130
			c.text(x, y, 7, false, labels[i])
			c.textRight(x+120, y-12, 9, true, money.Format(values[i]))
		}
	}
	grid(y, []string{"TOTALE SPETTANZE", "TOTALE RITENUTE", "ARROTONDAMENTO", "NETTO DEL MESE"},
		[]int64{gross, deductions, rounding, net})
	y -= 36
	grid(y, []string{"IMPONIBILE INPS", "IMPONIBILE IRPEF", "TFR DEL MESE"},
		[]int64{p.INPSBase, p.IRPEFBase, p.TFRAccrued})

	// Leave balances in side-by-side blocks, one per kind, with the kind
	// printed vertically at the block's left edge and each measure label
	// qualified on the line just below, next to the unit and the value.
	y -= 40
	lefts := []float64{70, 170, 270}
	for _, row := range []struct {
		measure, qualifier string
		value              func(Leave) string
	}{
		{"RESIDUO", "A.P.", func(l Leave) string { return l.Previous }},
		{"MATURATO", "A.C.", func(l Leave) string { return l.Accrued }},
		{"GODUTO", "A.C.", func(l Leave) string { return l.Used }},
		{"RESIDUO", "TOT.", func(l Leave) string { return l.Remaining }},
	} {
		for i, l := range p.Leave {
			c.text(lefts[i], y, 6, false, row.measure)
			c.text(lefts[i], y-3.5, 6, false, row.qualifier)
			c.text(lefts[i]+24, y-3.5, 6, false, "O")
			c.textRight(lefts[i]+80, y-3.5, 6, false, row.value(l))
		}
		y -= 12
	}
	for i, l := range p.Leave {
		c.textUp(lefts[i]-4, y+6, 6, l.Kind)
	}
}

func renderSpaced(c *canvas, p Payslip) {
	const (
		colCode, colDesc                 = 27.0, 60.0
		colQty, colBase, colEarn, colDed = 250.0, 320.0, 400.0, 470.0
		colTotals, colStats              = 359.0, 425.0
	)
	c.text(colCode, 805, 10, true, "CEDOLINO PAGA")
	cells := func(y float64, labels, values []string) {
		for i := range labels {
			x := colCode + float64(i)*130
			c.text(x, y, 5, false, labels[i])
			c.text(x, y-10, 7, true, values[i])
		}
	}
	cells(785, []string{"AZIENDA", "PERIODO DI RETRIBUZIONE", "LIVELLO", "CCNL"},
		[]string{p.Employer, fmt.Sprintf("%s %d", italianMonths[p.Month], p.Year), p.Level, p.CCNL})
	cells(760, []string{"COGNOME E NOME", "MATRICOLA", "QUALIFICA"},
		[]string{p.Employee, p.EmployeeNumber, p.Qualification})

	y := 725.0
	c.text(colCode, y, 6, true, "COD.")
	c.text(colDesc, y, 6, true, "DESCRIZIONE")
	c.textRight(colQty, y, 6, true, "QUANTITÀ")
	c.textRight(colBase, y, 6, true, "BASE")
	c.textRight(colEarn, y, 6, true, "COMPETENZE")
	c.textRight(colDed, y, 6, true, "TRATTENUTE")
	y -= 14
	for _, it := range p.Items {
		c.text(colCode, y, 7, false, it.Code)
		c.text(colDesc, y, 7, false, it.Description)
		if it.Quantity != "" {
			c.textRight(colQty, y, 7, false, it.Quantity)
		}
		if it.Base != "" {
			c.textRight(colBase, y, 7, false, it.Base)
		}
		col := colEarn
		if it.Deduction {
			col = colDed
		}
		c.textRight(col, y, 7, false, money.Format(it.Amount))
		y -= 12
	}

	// Totals: small labels with their values in the cell below, and on
	// the right a statistics column whose numbers sit a little lower than
	// the labels beside them.
	gross, deductions, net := p.Totals()
	y -= 24
	c.text(colTotals, y+14, 5, false, "STATISTICHE")
	c.text(colStats, y+14, 5, false, "ORE LAVORATE")
	// The statistics number of a row sits either a little lower than its
	// labels, on their line, or low enough to make a line of its own with
	// its top just above the labels' bottom.
	drop := 2.5
	row := func(labels []string, xs []float64, values []string, stats string) {
		for i := range labels {
			c.text(xs[i], y, 5, false, labels[i])
			c.text(xs[i]+15, y-14, 8, true, values[i])
		}
		c.text(colStats, y-drop, 8, false, stats)
		c.text(colStats, y-14, 8, false, stats)
		y -= 24
	}
	row([]string{"TOTALE LORDO", "IMPON. CONTR. SOC.", "TOT. CONTR. SOC."}, []float64{colCode, 110, colTotals},
		[]string{money.Format(gross), money.Format(p.INPSBase), money.Format(p.Contributions)}, "8,00")
	row([]string{"IMP. IRPEF", "TOT. TRAT. IRPEF"}, []float64{214, colTotals},
		[]string{money.Format(p.IRPEFBase), money.Format(p.Tax)}, "7,50")
	drop = 6.5
	row([]string{"ARR. PREC.", "TOT. TRATTENUTE"}, []float64{273, colTotals},
		[]string{"0,00", money.Format(deductions)}, "8,00")
	row([]string{"ARR. ATTUALE", "NETTO BUSTA"}, []float64{273, colTotals},
		[]string{"0,00", money.Format(net)}, "6,00")
	row([]string{"TFR MESE"}, []float64{colTotals}, []string{money.Format(p.TFRAccrued)}, "8,00")

	// Leave: one label per kind and measure, values right-aligned below.
	x := colCode
	for _, l := range p.Leave {
		for _, m := range []struct{ label, value string }{
			{"A.P.", l.Previous}, {"MAT.", l.Accrued}, {"GOD.", l.Used}, {"RES.", l.Remaining},
		} {
			c.text(x, y, 5, false, l.Kind+" "+m.label)
			c.textRight(x+38, y-12, 7, false, m.value)
			x += 42
		}
	}

	// Year-to-date values, under the same headings as the month's.
	y -= 34
	for i, label := range []string{"IMPONIBILE INPS", "IMPONIBILE IRPEF", "IRPEF PAGATA"} {
		c.text(110+float64(i)*100, y, 5, false, label)
	}
	c.text(colCode, y-6, 5, false, "PROGRES.")
	// "ANNUI" ends the row label, its top aligned with the values'.
	c.text(colCode, y-12+ascent*(8-5), 5, false, "ANNUI")
	for i, v := range []int64{p.INPSBase * 10, p.IRPEFBase * 10, p.Tax * 10} {
		c.text(120+float64(i)*100, y-12, 8, true, money.Format(v))
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	for i := 1; i < len(r); i++ {
		if r[i] >= 'A' && r[i] <= 'Z' {
			r[i] += 'a' - 'A'
		}
	}
	return string(r)
}
