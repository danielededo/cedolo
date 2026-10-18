package report

import (
	"cmp"
	"html/template"
	"slices"
	"strconv"

	"github.com/danielededo/cedolo/internal/i18n"
	"github.com/danielededo/cedolo/internal/money"
	"github.com/danielededo/cedolo/internal/payslip"
)

// YearTable sums the payslips by calendar year: one column per year, the
// latest last, one row per measure.
type YearTable struct {
	Years   []string // "2025", with an asterisk when the year is incomplete
	Rows    []YearRow
	Partial bool // some year has fewer than twelve monthly payslips
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
		year                            int
		monthly, extra                  int
		gross, contrib, taxes, net, tfr int64
		monthlyNet                      int64
		netMonths                       int
	}
	var sums []*sum
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
		if p.TFR.Found() {
			s.tfr += p.TFR.Cents
		}
		s.contrib += contributions(p)
		s.taxes += taxes(p)
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
	row("report.year.share", func(s *sum) string {
		if s.gross <= 0 || s.net <= 0 {
			return "–"
		}
		return strconv.FormatInt(roundDiv(s.net*100, s.gross), 10) + " €"
	})
	row("report.year.tfr", func(s *sum) string { return optional(s.tfr) })
	return t
}

// The groups the earnings chart stacks, in order from the base up.
var (
	earningGroups = []string{"ordinary", "variable", "additional", "other"}
	groupOf       = map[string]string{
		"earnings.base_salary": "ordinary", "earnings.superminimum": "ordinary", "earnings.seniority": "ordinary",
		"earnings.contingency": "ordinary", "earnings.leave": "ordinary", "earnings.sick_pay": "ordinary",
		"earnings.overtime": "variable", "earnings.bonus": "variable", "earnings.allowance": "variable",
		"earnings.thirteenth": "additional", "earnings.fourteenth": "additional",
	}
)

// earningsMix charts what the earnings of every payslip are made of: the
// ordinary pay, the variable part, the additional months and the rest. It
// is left out when every payslip is ordinary pay only, as it would say
// nothing the trend does not.
func earningsMix(l i18n.Lang, slips []payslip.Payslip) string {
	labels := make([]string, len(slips))
	ss := make([]series, len(earningGroups))
	for g, key := range earningGroups {
		ss[g] = series{Name: i18n.T(l, "report.mix."+key), Color: partColors[g]}
	}
	total := series{Name: i18n.T(l, "report.mix.total")}
	beyond := false
	for i, p := range slips {
		labels[i] = shortLabel(l, p)
		sums := make([]int64, len(earningGroups))
		for _, it := range p.Items {
			if it.Side != payslip.Earning {
				continue
			}
			g := slices.Index(earningGroups, cmp.Or(groupOf[it.Key], "other"))
			sums[g] += it.Cents
		}
		var sum int64
		for g, v := range sums {
			v = max(v, 0)
			ss[g].Values = append(ss[g].Values, v)
			sum += v
			if g > 0 && v > 0 {
				beyond = true
			}
		}
		total.Values = append(total.Values, sum)
	}
	if !beyond {
		return ""
	}
	return stackChart("mix", labels, ss, total)
}

// tfrChart follows the TFR accrued in the payslips, added up month by month.
// It needs two payslips that print it.
func tfrChart(l i18n.Lang, slips []payslip.Payslip) (svg template.HTML, total string) {
	var labels []string
	s := series{Name: i18n.T(l, "report.tfr.series"), Color: "var(--series-1)"}
	var sum int64
	for _, p := range slips {
		if !p.TFR.Found() {
			continue
		}
		sum += p.TFR.Cents
		labels = append(labels, shortLabel(l, p))
		s.Values = append(s.Values, sum)
	}
	if len(s.Values) < 2 {
		return "", ""
	}
	return template.HTML(lineChart("tfr", labels, []series{s}, lineOptions{
		width: chartW, height: chartH, tick: euros, value: moneyValue, endLabels: true,
	})), moneyValue(sum)
}

// roundDiv divides rounding half away from zero.
func roundDiv(a, b int64) int64 {
	if (a < 0) != (b < 0) {
		return -((-a + b/2) / b)
	}
	return (a + b/2) / b
}
