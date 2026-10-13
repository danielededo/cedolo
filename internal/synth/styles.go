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
)

// Styles lists every style.
func Styles() []Style { return []Style{StyleClassic, StyleSigned, StyleGrid} }

func (s Style) String() string {
	switch s {
	case StyleClassic:
		return "classic"
	case StyleSigned:
		return "signed"
	case StyleGrid:
		return "grid"
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
	c.textRight(330, y, 8, true, "MATURATE")
	c.textRight(400, y, 8, true, "GODUTE")
	c.textRight(480, y, 8, true, "RESIDUE")
	for _, l := range p.Leave {
		y -= 12
		c.text(colCode, y, 8, false, l.Kind)
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
