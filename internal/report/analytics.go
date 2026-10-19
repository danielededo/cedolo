package report

import (
	"cmp"
	"html/template"
	"slices"
	"strconv"
	"strings"

	"github.com/danielededo/cedolo/internal/i18n"
	"github.com/danielededo/cedolo/internal/money"
	"github.com/danielededo/cedolo/internal/payslip"
	"github.com/danielededo/cedolo/internal/taxonomy"
)

// YearTable sums the payslips by calendar year: one column per year, the
// latest last, one row per measure.
type YearTable struct {
	Years   []string // "2025", with an asterisk when the year is incomplete
	Rows    []YearRow
	Partial bool // some year has fewer than twelve monthly payslips
	// Deductible is set when the table has the row of the pension fund
	// deductible used; Over when some year goes above the limit.
	Deductible, Over bool
}

// YearRow is one measure over the years.
type YearRow struct {
	Label string
	Cells []string
}

// years sums the payslips by calendar year. It is nil without a dated
// payslip.
func years(l i18n.Lang, slips []payslip.Payslip) *YearTable {
	type sum struct {
		year                             int
		monthly, extra                   int
		gross, contrib, taxes, net, fund int64
		deductible                       int64
		monthlyNet                       int64
		netMonths                        int
	}
	var sums []*sum
	fundYears := false
	for _, p := range slips {
		if p.Year == 0 {
			continue
		}
		i := slices.IndexFunc(sums, func(s *sum) bool { return s.year == p.Year })
		if i < 0 {
			sums = append(sums, &sum{year: p.Year})
			i = len(sums) - 1
		}
		s := sums[i]
		if p.Kind == payslip.Monthly {
			s.monthly++
			if p.Net.Found() {
				s.monthlyNet += p.Net.Cents
				s.netMonths++
			}
		} else {
			s.extra++
		}
		if p.Gross.Found() {
			s.gross += p.Gross.Cents
		}
		if p.Net.Found() {
			s.net += p.Net.Cents
		}
		s.contrib += contributions(p)
		s.taxes += taxes(p)
		f := fundOf(p)
		s.fund += f.total()
		s.deductible += f.own + f.employer
		fundYears = fundYears || f.own > 0 || f.employer > 0
	}
	if len(sums) == 0 {
		return nil
	}
	slices.SortFunc(sums, func(a, b *sum) int { return cmp.Compare(a.year, b.year) })
	t := &YearTable{}
	row := func(key string, cell func(*sum) string) {
		r := YearRow{Label: i18n.T(l, key)}
		for _, s := range sums {
			r.Cells = append(r.Cells, cell(s))
		}
		t.Rows = append(t.Rows, r)
	}
	for _, s := range sums {
		y := strconv.Itoa(s.year)
		if s.monthly < 12 {
			y += "*"
			t.Partial = true
		}
		t.Years = append(t.Years, y)
	}
	row("report.year.months", func(s *sum) string {
		if s.extra > 0 {
			return i18n.T(l, "report.year.count_extra", s.monthly, s.extra)
		}
		return strconv.Itoa(s.monthly)
	})
	row("report.gross", func(s *sum) string { return optional(s.gross) })
	row("report.contrib", func(s *sum) string { return optional(s.contrib) })
	row("report.taxes", func(s *sum) string { return optional(s.taxes) })
	row("report.net", func(s *sum) string { return optional(s.net) })
	row("report.year.mean", func(s *sum) string {
		if s.netMonths == 0 {
			return "–"
		}
		return money.Format(roundDiv(s.monthlyNet, int64(s.netMonths))) + " €"
	})
	if fundYears {
		row("report.year.fund", func(s *sum) string { return optional(s.fund) })
		row("report.year.deductible", func(s *sum) string {
			t.Over = t.Over || s.deductible > deductibleLimit(s.year)
			return optional(s.deductible)
		})
		t.Deductible = true
	}
	row("report.year.share", func(s *sum) string {
		if s.gross <= 0 || s.net <= 0 {
			return "–"
		}
		return strconv.FormatInt(roundDiv(s.net*100, s.gross), 10) + " €"
	})
	return t
}

// deductibleLimit is the most that contributions to a supplementary
// pension fund, yours and the employer's, take off the taxable income in
// a year (d.lgs. 252/2005, art. 8 c. 4). The TFR paid into the fund is not
// counted. The 2026 budget law raised it from 5.164,57 € to 5.300 €.
func deductibleLimit(year int) int64 {
	if year >= 2026 {
		return 530000
	}
	return 516457
}

// variableKeys are the earnings that change from one month to the next.
var variableKeys = []string{"earnings.overtime", "earnings.bonus", "earnings.allowance"}

// variablePay charts the variable part of the earnings of every payslip:
// overtime, bonuses and allowances, on a scale of its own so that their
// changes show, where in the gross they are a sliver. It is left out when
// no payslip has any.
func variablePay(l i18n.Lang, slips []payslip.Payslip) string {
	labels := make([]string, len(slips))
	s := series{Name: i18n.T(l, "report.variable.series"), Color: "var(--series-2)"}
	found := false
	for i, p := range slips {
		labels[i] = shortLabel(l, p)
		var sum int64
		for _, it := range p.Items {
			if it.Side == payslip.Earning && slices.Contains(variableKeys, it.Key) {
				sum += it.Cents
			}
		}
		s.Values = append(s.Values, max(sum, 0))
		found = found || sum > 0
	}
	if !found {
		return ""
	}
	return stackCharts("variable", labels, []series{s}, s, 0)
}

// fund is what a payslip pays into the supplementary pension fund: your
// own contribution, the employer's, and the TFR sent to the fund.
type fund struct{ own, employer, tfr int64 }

func (f fund) total() int64 { return f.own + f.employer + f.tfr }

// fundOf reads a payslip's payments to the pension fund. Your contribution
// is an information item or a deduction item; the TFR sent to the fund is
// a TFR item that names the fund.
func fundOf(p payslip.Payslip) fund {
	var f fund
	f.own, _ = p.SumKey("pension_fund.employee")
	ded, _ := p.SumKey("deductions.pension_fund")
	f.own += ded
	f.employer, _ = p.SumKey("pension_fund.contribution")
	for _, it := range p.Items {
		if it.Key == "tfr.accrued" && toFund(it.Description) {
			f.tfr += it.Cents
		}
	}
	return f
}

// toFund reports whether a TFR item names a pension fund, as in "Storno
// T.F.R. a F.P.C.", "TFR a fondo" or "TFR previdenza complementare".
func toFund(desc string) bool {
	words := " " + taxonomy.Normalize(desc) + " "
	for _, w := range []string{" fpc ", " f p c ", " fondo ", " fondi ", " previdenza ", " fp "} {
		if strings.Contains(words, w) {
			return true
		}
	}
	return false
}

// fundChart charts what each payslip paid into the pension fund, in its
// three parts, under the total. The chart starts at the first payslip that
// paid in: the months before the fund would only be empty columns. It
// needs two payslips from there and a fund to which you or your employer
// contribute.
func fundChart(l i18n.Lang, slips []payslip.Payslip) (svg template.HTML, total, span string) {
	first := -1
	for i, p := range slips {
		if fundOf(p).total() > 0 {
			first = i
			break
		}
	}
	if first < 0 || len(slips)-first < 2 {
		return "", "", ""
	}
	var labels []string
	ss := []series{
		{Name: i18n.T(l, "report.fund.own"), Color: "var(--series-1)"},
		{Name: i18n.T(l, "report.fund.employer"), Color: "var(--series-2)"},
		{Name: i18n.T(l, "report.fund.tfr"), Color: "var(--series-3)"},
	}
	sum := series{Name: i18n.T(l, "report.fund.total")}
	var all int64
	n, contributes := 0, false
	for _, p := range slips[first:] {
		labels = append(labels, shortLabel(l, p))
		f := fundOf(p)
		for k, v := range []int64{f.own, f.employer, f.tfr} {
			ss[k].Values = append(ss[k].Values, max(v, 0))
		}
		sum.Values = append(sum.Values, f.total())
		all += f.total()
		if f.total() > 0 {
			n++
		}
		contributes = contributes || f.own > 0 || f.employer > 0
	}
	if !contributes {
		return "", "", ""
	}
	return template.HTML(stackCharts("fund", labels, ss, sum, first)), moneyValue(all), i18n.T(l, "report.fund.span", n)
}

// tfrChart follows the TFR accrued in the payslips and kept by the
// employer, added up month by month, as a small line under its total. It
// needs two payslips that print it; the TFR sent to a pension fund is in
// the fund's chart.
func tfrChart(l i18n.Lang, slips []payslip.Payslip) (svg template.HTML, total, span string) {
	var labels []string
	s := series{Name: i18n.T(l, "report.tfr.series"), Color: "var(--series-1)"}
	var sum int64
	for _, p := range slips {
		if !p.TFR.Found() {
			continue
		}
		sum += max(p.TFR.Cents-fundOf(p).tfr, 0)
		labels = append(labels, shortLabel(l, p))
		s.Values = append(s.Values, sum)
	}
	if len(s.Values) < 2 || sum == 0 {
		return "", "", ""
	}
	return template.HTML(lineChart("tfr", labels, []series{s}, lineOptions{
		width: 340, height: 170, tick: whole, value: moneyValue,
	})), moneyValue(sum), i18n.T(l, "report.tfr.span", len(s.Values))
}

// roundDiv divides rounding half away from zero.
func roundDiv(a, b int64) int64 {
	if (a < 0) != (b < 0) {
		return -((-a + b/2) / b)
	}
	return (a + b/2) / b
}
