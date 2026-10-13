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

// validate fills totals that can be computed, runs the cross-checks, and sets
// the status of each total: verified when a check confirms it, inconsistent
// when a check contradicts it, extracted when no check applies.
func validate(p *payslip.Payslip) {
	earnings, deductions := p.Sum(payslip.Earning), p.Sum(payslip.Deduction)
	hasItems := len(p.Items) > 0

	// Item sums against printed totals.
	sumCheck := func(name, what string, total *payslip.Amount, sum int64) {
		switch {
		case total.Found() && hasItems:
			ok := total.Cents == sum
			p.Checks = append(p.Checks, payslip.Check{Name: name, OK: ok,
				Detail: fmt.Sprintf("items add up to %s, printed %s is %s", money.Format(sum), what, money.Format(total.Cents))})
			mark(total, ok)
		case !total.Found() && hasItems:
			*total = payslip.Amount{Cents: sum, Status: payslip.Inferred}
		}
	}
	sumCheck("earnings_sum", "gross", &p.Gross, earnings)
	sumCheck("deductions_sum", "total deductions", &p.Deductions, deductions)

	// Gross minus deductions against net.
	if p.Gross.Found() && p.Deductions.Found() {
		expected := p.Gross.Cents - p.Deductions.Cents
		if p.Net.Found() {
			ok := p.Net.Cents == expected
			p.Checks = append(p.Checks, payslip.Check{Name: "net_balance", OK: ok,
				Detail: fmt.Sprintf("gross minus deductions is %s, printed net is %s", money.Format(expected), money.Format(p.Net.Cents))})
			mark(&p.Net, ok)
			if ok {
				mark(&p.Gross, true)
				mark(&p.Deductions, true)
			}
		} else {
			p.Net = payslip.Amount{Cents: expected, Status: payslip.Inferred}
		}
	}

	// INPS contribution against its taxable base and the known rates.
	if inps, n := p.SumKey("deductions.inps"); n > 0 && p.INPSBase.Found() {
		ok := false
		for _, rate := range inpsRates {
			if abs(inps-roundDiv(p.INPSBase.Cents*rate, 10000)) <= 1 {
				ok = true
				p.Checks = append(p.Checks, payslip.Check{Name: "inps_rate", OK: true,
					Detail: fmt.Sprintf("INPS %s is %s%% of the taxable base %s", money.Format(inps), money.Format(rate), money.Format(p.INPSBase.Cents))})
				break
			}
		}
		if ok {
			mark(&p.INPSBase, true)
		} else {
			p.Checks = append(p.Checks, payslip.Check{Name: "inps_rate", OK: false,
				Detail: fmt.Sprintf("INPS %s is not a usual rate of the taxable base %s; this can be normal for special contracts or contributions above the yearly threshold", money.Format(inps), money.Format(p.INPSBase.Cents))})
		}
	}

	// IRPEF taxable base is usually the INPS base minus the INPS contribution.
	if inps, n := p.SumKey("deductions.inps"); n > 0 && p.INPSBase.Found() && p.IRPEFBase.Found() {
		expected := p.INPSBase.Cents - inps
		ok := p.IRPEFBase.Cents == expected
		detail := fmt.Sprintf("INPS base minus INPS contribution is %s, printed IRPEF base is %s", money.Format(expected), money.Format(p.IRPEFBase.Cents))
		if !ok {
			detail += "; a difference is normal when pension fund contributions or other deductible items exist"
		}
		p.Checks = append(p.Checks, payslip.Check{Name: "irpef_base", OK: ok, Detail: detail})
		if ok {
			mark(&p.IRPEFBase, true)
		}
	}

	// Monthly TFR accrual is the month's eligible pay divided by 13.5.
	if p.TFR.Found() {
		base := p.INPSBase
		if !base.Found() {
			base = p.Gross
		}
		if base.Found() {
			expected := roundDiv(base.Cents*10, 135)
			ok := abs(p.TFR.Cents-expected) <= 1
			detail := fmt.Sprintf("TFR %s, the taxable base divided by 13.5 is %s", money.Format(p.TFR.Cents), money.Format(expected))
			if !ok {
				detail += "; a difference is normal when the pay used for TFR differs from the taxable base"
			}
			p.Checks = append(p.Checks, payslip.Check{Name: "tfr_ratio", OK: ok, Detail: detail})
			if ok {
				mark(&p.TFR, true)
			}
		}
	}

	for _, a := range []*payslip.Amount{&p.Gross, &p.Deductions, &p.Net, &p.INPSBase, &p.IRPEFBase, &p.TFR} {
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
