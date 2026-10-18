// Package extract reads payslip data from a text layout without knowing which
// payroll software produced it. Italian payslips share a legally required
// content and a common structure (header, a table of items with earnings and
// deductions, totals, taxable bases, TFR), so the extractor recognizes labels
// from a dictionary of synonyms, derives table columns from the table header,
// classifies items through the taxonomy, and then cross-checks the arithmetic
// to decide how far each value can be trusted. Vendor-specific tuning can be
// layered on top later; it is not required for a first reading.
package extract

import (
	"github.com/danielededo/cedolo/internal/ccnl"
	"github.com/danielededo/cedolo/internal/layout"
	"github.com/danielededo/cedolo/internal/payslip"
)

// Document extracts every payslip in a document. A page that names a pay
// period different from the previous page starts a new payslip; pages without
// a period continue the current one.
func Document(doc layout.Document) []payslip.Payslip {
	doc = layout.JoinSpaced(doc)
	type group struct {
		pages       []layout.Page
		year, month int
		kind        payslip.Kind // set when the period itself names an additional month
		closed      bool         // a page already printed the net pay
	}
	var groups []group
	for _, p := range doc.Pages {
		lines := buildLines(p)
		year, month, kind := findPeriod(lines)
		n := len(groups)
		switch {
		// A new period starts a new payslip; so does a page with a period
		// after a payslip that already printed its net pay, as when two
		// payslips of the same month share a file. Continuation pages repeat
		// the header but leave the net to the last page.
		case n == 0, (month != 0 || kind != "") && (month != groups[n-1].month || year != groups[n-1].year ||
			kind != groups[n-1].kind || groups[n-1].closed):
			groups = append(groups, group{pages: []layout.Page{p}, year: year, month: month, kind: kind})
		default:
			groups[n-1].pages = append(groups[n-1].pages, p)
		}
		if _, ok := amountField(lines, p.Width, netLabels); ok {
			groups[len(groups)-1].closed = true
		}
	}
	out := make([]payslip.Payslip, 0, len(groups))
	for _, g := range groups {
		p := extractPayslip(g.pages, g.year, g.month)
		if g.kind != "" {
			p.Kind = g.kind
		}
		if len(p.Items) == 0 && len(p.Leave) == 0 && !p.Gross.Found() && !p.Deductions.Found() && !p.Net.Found() {
			// No pay item, total or leave balance: another document, such
			// as a tax return, whose dates only look like a pay period.
			continue
		}
		out = append(out, p)
	}
	return out
}

func extractPayslip(pages []layout.Page, year, month int) payslip.Payslip {
	p := payslip.Payslip{Year: year, Month: month, Kind: payslip.Monthly}
	var all [][]line
	for _, pg := range pages {
		p.Pages = append(p.Pages, pg.Number)
		all = append(all, buildLines(pg))
	}

	first := all[0]
	headerEnd := len(first)
	for i, pg := range all {
		t, ok := findTable(pg, pages[i].Width)
		if !ok {
			continue
		}
		if i == 0 {
			headerEnd = t.headerLine
		}
		p.Items = append(p.Items, t.rows(pg, pages[i].Number)...)
	}

	header := first[:headerEnd]
	for _, l := range header {
		if l.has(thirteenthWords...) {
			p.Kind = payslip.Thirteenth
		}
		if l.has(fourteenthWords...) {
			p.Kind = payslip.Fourteenth
		}
	}
	// "Mensilità aggiuntiva" alone: the fourteenth is paid in June or July,
	// the thirteenth in December.
	if p.Kind == payslip.Monthly {
		for _, l := range header {
			if l.has(additionalWords...) {
				p.Kind = payslip.Thirteenth
				if p.Month == 6 || p.Month == 7 {
					p.Kind = payslip.Fourteenth
				}
			}
		}
	}
	p.CCNL = textField(header, pages[0].Width, "ccnl", "contratto")
	if c, ok := ccnl.FindCode(p.CCNL); ok {
		p.CCNLCode = c.Code
	}
	p.Level = levelField(header, pages[0].Width)

	// Totals usually close the payslip, so later pages are searched first.
	find := func(labels []string) payslip.Amount {
		for i := len(all) - 1; i >= 0; i-- {
			if a, ok := amountField(all[i], pages[i].Width, labels); ok {
				return a
			}
		}
		return payslip.Amount{Status: payslip.Missing}
	}
	readings := func(labels []string) []payslip.Amount {
		var out []payslip.Amount
		for i := len(all) - 1; i >= 0; i-- {
			out = append(out, amountReadings(all[i], pages[i].Width, labels, true)...)
		}
		return out
	}
	p.Gross = find(grossLabels)
	p.Deductions = find(deductionLabels)
	p.Net = find(netLabels)
	p.INPSBase = find(inpsBaseLabels)
	p.IRPEFBase = find(irpefBaseLabels)
	p.TFR = find(tfrLabels)
	p.TFRBase = find(tfrBaseLabels)
	addFamilyAllowance(&p, find(familyAllowanceLabels))
	if !p.TFR.Found() {
		// Some layouts print the month's TFR only as a neutral item, such as
		// its transfer to a pension fund.
		for _, it := range p.Items {
			if it.Key == "tfr.accrued" {
				src := it.Source
				p.TFR = payslip.Amount{Cents: p.TFR.Cents + it.Cents, Status: payslip.Extracted, Source: &src}
			}
		}
	}
	p.SocialContributions = find(socialLabels)
	p.Taxes = find(taxLabels)
	p.INPS = find(inpsLabels)
	p.RoundingPrevious = find(roundingPreviousLabels)
	p.RoundingCurrent = find(roundingCurrentLabels)

	for _, pg := range all {
		for _, lv := range findLeave(pg) {
			if !hasLeave(p.Leave, lv.Kind) {
				p.Leave = append(p.Leave, lv)
			}
		}
	}

	// The totals the cross-checks tie together, with every reading of each.
	resolve(&p, []total{
		{"gross pay", func(p *payslip.Payslip) *payslip.Amount { return &p.Gross }, readings(grossLabels)},
		{"total deductions", func(p *payslip.Payslip) *payslip.Amount { return &p.Deductions }, readings(deductionLabels)},
		{"net pay", func(p *payslip.Payslip) *payslip.Amount { return &p.Net }, readings(netLabels)},
		{"contributions", func(p *payslip.Payslip) *payslip.Amount { return &p.SocialContributions }, readings(socialLabels)},
		{"taxes", func(p *payslip.Payslip) *payslip.Amount { return &p.Taxes }, append(readings(taxLabels), otherTaxes(find)...)},
		{"previous rounding", func(p *payslip.Payslip) *payslip.Amount { return &p.RoundingPrevious }, readings(roundingPreviousLabels)},
		{"current rounding", func(p *payslip.Payslip) *payslip.Amount { return &p.RoundingCurrent }, readings(roundingCurrentLabels)},
	})
	checkLeave(&p)
	return p
}

// otherTaxes returns the income tax withheld in boxes of its own, the
// separate taxation of TFR and the IRPEF adjustment: both added together,
// then each alone. They are only alternative readings of the month's taxes.
func otherTaxes(find func([]string) payslip.Amount) []payslip.Amount {
	sep, adj := find(separateTaxLabels), find(taxAdjustmentLabels)
	var out []payslip.Amount
	if sep.Found() && adj.Found() {
		out = append(out, payslip.Amount{Cents: sep.Cents + adj.Cents, Status: payslip.Extracted, Source: sep.Source})
	}
	for _, a := range []payslip.Amount{sep, adj} {
		if a.Found() {
			out = append(out, a)
		}
	}
	return out
}

// addFamilyAllowance adds the family allowance printed outside the item
// table as an earning, when the table has none and the printed gross is the
// items plus the allowance.
func addFamilyAllowance(p *payslip.Payslip, a payslip.Amount) {
	if !a.Found() || a.Cents <= 0 || !p.Gross.Found() || p.Gross.Cents != p.Sum(payslip.Earning)+a.Cents {
		return
	}
	if _, n := p.SumKey("earnings.family_allowance"); n > 0 {
		return
	}
	p.Items = append(p.Items, payslip.Item{Description: "Importo assegno", Cents: a.Cents, Side: payslip.Earning,
		Key: "earnings.family_allowance", Source: *a.Source})
}

func hasLeave(ls []payslip.Leave, k payslip.LeaveKind) bool {
	for _, l := range ls {
		if l.Kind == k {
			return true
		}
	}
	return false
}
