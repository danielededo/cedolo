package extract

import (
	"fmt"

	"github.com/danielededo/cedolo/internal/money"
	"github.com/danielededo/cedolo/internal/payslip"
)

// INPS employee contribution rates in basis points: the general rate, the
// rate with the additional 0.30 percent for larger employers, and the
// apprentice rate. A rate outside this list is not an error, only unverified.
var inpsRates = []int64{919, 949, 584}

// outsideGross are item kinds that some payroll software leaves out of the
// printed gross because they are not taxable pay.
var outsideGross = map[string]bool{
	"earnings.reimbursement":       true,
	"earnings.contribution_relief": true,
	"tax.credit":                   true,
	"tax.return":                   true,
	"earnings.family_allowance":    true,
	"rounding":                     true,
}

// validate fills totals that can be computed, runs the cross-checks, and sets
// the status of each total: verified when a check confirms it, inconsistent
// when a check contradicts it, extracted when no check applies.
//
// Payroll software disagrees on what its totals mean: "totale trattenute" is
// the item table total in some layouts and everything withheld in others,
// and contributions and taxes may be items or summary boxes. Rather than
// guess per vendor, validate tries the readings that are known to occur and
// keeps the one under which the payslip's arithmetic holds.
func validate(p *payslip.Payslip) {
	earnings, tableDeductions := p.Sum(payslip.Earning), p.Sum(payslip.Deduction)
	hasItems := len(p.Items) > 0

	// Gross: all earnings, or the taxable ones, either possibly net of the
	// pay withheld for absences.
	if hasItems {
		var taxable int64
		for _, it := range p.Items {
			if it.Side == payslip.Earning && !outsideGross[it.Key] {
				taxable += it.Cents
			}
		}
		absences, _ := p.SumKey("deductions.absence")
		switch {
		case !p.Gross.Found():
			p.Gross = payslip.Amount{Cents: earnings, Status: payslip.Inferred}
		case p.Gross.Cents == earnings:
			p.Checks = append(p.Checks, payslip.Check{Name: "earnings_sum", OK: true,
				Detail: fmt.Sprintf("items add up to %s, printed gross is %s", money.Format(earnings), money.Format(p.Gross.Cents))})
			mark(&p.Gross, true)
		case p.Gross.Cents == taxable:
			p.Checks = append(p.Checks, payslip.Check{Name: "earnings_sum", OK: true,
				Detail: fmt.Sprintf("items other than reimbursements, contribution relief, tax credits, tax return refunds and family allowance add up to %s, printed gross is %s", money.Format(taxable), money.Format(p.Gross.Cents))})
			mark(&p.Gross, true)
		case absences != 0 && (p.Gross.Cents == earnings-absences || p.Gross.Cents == taxable-absences):
			p.Checks = append(p.Checks, payslip.Check{Name: "earnings_sum", OK: true,
				Detail: fmt.Sprintf("items add up to %s, less %s withheld for absences; printed gross is %s", money.Format(earnings), money.Format(absences), money.Format(p.Gross.Cents))})
			mark(&p.Gross, true)
		default:
			p.Checks = append(p.Checks, payslip.Check{Name: "earnings_sum", OK: false,
				Detail: fmt.Sprintf("items add up to %s, printed gross is %s", money.Format(earnings), money.Format(p.Gross.Cents))})
			mark(&p.Gross, false)
		}
	}
	base := p.Gross.Cents
	if hasItems {
		base = earnings
	}

	// Everything withheld, under each known reading of the printed totals.
	summary := tableDeductions
	for _, a := range []payslip.Amount{p.SocialContributions, p.Taxes} {
		if a.Found() {
			summary += a.Cents
		}
	}
	type reading struct {
		total       int64
		printedUsed bool // the printed total is everything withheld
		desc        string
	}
	var readings []reading
	if p.Deductions.Found() {
		readings = append(readings, reading{p.Deductions.Cents, true, "printed total deductions"})
	}
	if p.Deductions.Found() && p.Deductions.Cents != summary && (p.SocialContributions.Found() || p.Taxes.Found()) {
		readings = append(readings, reading{summary, false, "item deductions plus contributions and taxes"})
	}
	if !p.Deductions.Found() && hasItems {
		readings = append(readings, reading{summary, false, "item deductions plus contributions and taxes"})
	}

	// Net: earnings minus everything withheld, adjusted by the rounding
	// carried from the previous month and the rounding of this month.
	if p.Net.Found() && len(readings) > 0 {
		prev, curr := p.RoundingPrevious.Cents, p.RoundingCurrent.Cents
		var adjustments [][2]int64
		for _, a := range []int64{0, -1, 1} {
			for _, b := range []int64{0, 1, -1} {
				if (a != 0 && !p.RoundingPrevious.Found()) || (b != 0 && !p.RoundingCurrent.Found()) {
					continue
				}
				adjustments = append(adjustments, [2]int64{a, b})
			}
		}
		found := false
		for _, r := range readings {
			for _, adj := range adjustments {
				if base-r.total+adj[0]*prev+adj[1]*curr != p.Net.Cents {
					continue
				}
				found = true
				detail := fmt.Sprintf("earnings %s minus %s %s", money.Format(base), r.desc, money.Format(r.total))
				if adj[0] != 0 {
					detail += fmt.Sprintf(", %s previous rounding %s", sign(adj[0]), money.Format(prev))
				}
				if adj[1] != 0 {
					detail += fmt.Sprintf(", %s current rounding %s", sign(adj[1]), money.Format(curr))
				}
				detail += fmt.Sprintf(" gives the printed net %s", money.Format(p.Net.Cents))
				p.Checks = append(p.Checks, payslip.Check{Name: "net_balance", OK: true, Detail: detail})
				mark(&p.Net, true)
				for _, a := range []*payslip.Amount{&p.SocialContributions, &p.Taxes} {
					if !r.printedUsed {
						mark(a, true)
					}
				}
				if adj[0] != 0 {
					mark(&p.RoundingPrevious, true)
				}
				if adj[1] != 0 {
					mark(&p.RoundingCurrent, true)
				}
				if r.printedUsed {
					mark(&p.Deductions, true)
				} else {
					// The printed total, if any, was the item table total.
					p.Deductions = payslip.Amount{Cents: r.total, Status: payslip.Verified}
				}
				break
			}
			if found {
				break
			}
		}
		if !found {
			r := readings[0]
			p.Checks = append(p.Checks, payslip.Check{Name: "net_balance", OK: false,
				Detail: fmt.Sprintf("earnings %s minus %s %s is %s, printed net is %s", money.Format(base), r.desc,
					money.Format(r.total), money.Format(base-r.total), money.Format(p.Net.Cents))})
			mark(&p.Net, false)
			mark(&p.Deductions, false)
		}
	} else if !p.Net.Found() && len(readings) > 0 {
		p.Net = payslip.Amount{Cents: base - readings[0].total, Status: payslip.Inferred}
		if !p.Deductions.Found() {
			p.Deductions = payslip.Amount{Cents: readings[0].total, Status: payslip.Inferred}
		}
	} else if !p.Deductions.Found() && len(readings) > 0 {
		p.Deductions = payslip.Amount{Cents: readings[0].total, Status: payslip.Inferred}
	}

	// Total deductions against its parts: the item table alone, or with
	// contributions and taxes, possibly including the rounding carried from
	// the previous month.
	if p.Deductions.Found() && hasItems {
		withPrevious := p.RoundingPrevious.Found() && p.Deductions.Cents == summary+p.RoundingPrevious.Cents
		ok := p.Deductions.Cents == summary || p.Deductions.Cents == tableDeductions || withPrevious
		detail := fmt.Sprintf("item deductions add up to %s, with contributions and taxes %s; total deductions is %s",
			money.Format(tableDeductions), money.Format(summary), money.Format(p.Deductions.Cents))
		if withPrevious {
			detail += fmt.Sprintf(", including the previous rounding %s", money.Format(p.RoundingPrevious.Cents))
			mark(&p.RoundingPrevious, true)
		}
		p.Checks = append(p.Checks, payslip.Check{Name: "deductions_sum", OK: ok, Detail: detail})
		if ok && summary != tableDeductions {
			mark(&p.SocialContributions, true)
			mark(&p.Taxes, true)
		}
		if !ok {
			mark(&p.Deductions, false)
		}
	}

	// INPS contribution against its taxable base and the known rates. The
	// contribution is an item, a summary box, or failing both the total of
	// social contributions.
	inps, n := p.SumKey("deductions.inps")
	switch {
	case n > 0:
	case p.INPS.Found():
		inps, n = p.INPS.Cents, 1
	case p.SocialContributions.Found():
		inps, n = p.SocialContributions.Cents, 1
	}
	if n > 0 && p.INPSBase.Found() {
		ok := false
		for _, rate := range inpsRates {
			if abs(inps-roundDiv(p.INPSBase.Cents*rate, 10000)) <= 1 {
				ok = true
				p.Checks = append(p.Checks, payslip.Check{Name: "inps_rate", OK: true, Advisory: true,
					Detail: fmt.Sprintf("INPS %s is %s%% of the taxable base %s", money.Format(inps), money.Format(rate), money.Format(p.INPSBase.Cents))})
				break
			}
		}
		if ok {
			mark(&p.INPSBase, true)
		} else {
			p.Checks = append(p.Checks, payslip.Check{Name: "inps_rate", OK: false, Advisory: true,
				Detail: fmt.Sprintf("INPS %s is not a usual rate of the taxable base %s; this can be normal for special contracts or contributions above the yearly threshold", money.Format(inps), money.Format(p.INPSBase.Cents))})
		}
	}

	// IRPEF taxable base is usually the INPS base minus the INPS contribution.
	if n > 0 && p.INPSBase.Found() && p.IRPEFBase.Found() {
		expected := p.INPSBase.Cents - inps
		ok := p.IRPEFBase.Cents == expected
		detail := fmt.Sprintf("INPS base minus INPS contribution is %s, printed IRPEF base is %s", money.Format(expected), money.Format(p.IRPEFBase.Cents))
		if !ok {
			detail += "; a difference is normal when pension fund contributions or other deductible items exist"
		}
		p.Checks = append(p.Checks, payslip.Check{Name: "irpef_base", OK: ok, Advisory: true, Detail: detail})
		if ok {
			mark(&p.IRPEFBase, true)
		}
	}

	// Monthly TFR accrual is the month's eligible pay divided by 13.5, shown
	// either as is or net of the 0.5 percent contribution that the law
	// charges on it.
	if p.TFR.Found() {
		base := p.TFRBase
		for _, b := range []payslip.Amount{p.INPSBase, p.Gross} {
			if !base.Found() {
				base = b
			}
		}
		if base.Found() {
			b := base.Cents
			gross, net := roundDiv(b*2, 27), roundDiv(b*373, 5400)
			var detail string
			ok := true
			switch {
			case abs(p.TFR.Cents-gross) <= 2:
				detail = fmt.Sprintf("TFR %s is the base %s divided by 13.5", money.Format(p.TFR.Cents), money.Format(b))
			case abs(p.TFR.Cents-net) <= 2:
				detail = fmt.Sprintf("TFR %s is the base %s divided by 13.5, net of the 0.5%% contribution", money.Format(p.TFR.Cents), money.Format(b))
			default:
				ok = false
				detail = fmt.Sprintf("TFR %s, the base %s divided by 13.5 is %s (%s net of the 0.5%% contribution); a difference is normal when the pay used for TFR differs from this base",
					money.Format(p.TFR.Cents), money.Format(b), money.Format(gross), money.Format(net))
			}
			p.Checks = append(p.Checks, payslip.Check{Name: "tfr_ratio", OK: ok, Advisory: true, Detail: detail})
			if ok {
				mark(&p.TFR, true)
				if p.TFRBase.Found() {
					mark(&p.TFRBase, true)
				}
			}
		}
	}

	for _, a := range []*payslip.Amount{&p.Gross, &p.Deductions, &p.Net, &p.INPSBase, &p.IRPEFBase, &p.TFR, &p.TFRBase,
		&p.SocialContributions, &p.Taxes, &p.INPS, &p.RoundingPrevious, &p.RoundingCurrent} {
		if a.Status == "" {
			a.Status = payslip.Missing
		}
	}
}

// mark records the outcome of a check on a value read from the document. A
// failed check makes it inconsistent; a passed check verifies it unless
// another check already found it inconsistent.
func mark(a *payslip.Amount, ok bool) {
	if a.Status == payslip.Inferred || a.Status == payslip.Missing {
		return
	}
	switch {
	case !ok:
		a.Status = payslip.Inconsistent
	case a.Status != payslip.Inconsistent:
		a.Status = payslip.Verified
	}
}

func sign(v int64) string {
	if v < 0 {
		return "minus"
	}
	return "plus"
}

func abs(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

// roundDiv divides rounding half away from zero, as payroll software rounds
// to the cent.
func roundDiv(a, b int64) int64 {
	if (a < 0) != (b < 0) {
		return -((abs(a) + abs(b)/2) / abs(b))
	}
	return (abs(a) + abs(b)/2) / abs(b)
}
