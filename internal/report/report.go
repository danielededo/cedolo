// Package report renders payslips as one self-contained HTML page: a summary
// of the latest month, charts over time, the changes worth a look, leave
// balances and a glossary of the items found. The page loads nothing from
// the network and runs no code but a small inline hover layer.
package report

import (
	_ "embed"
	"fmt"
	"html/template"
	"io"
	"slices"
	"strings"

	"github.com/danielededo/cedolo/internal/ccnl"
	"github.com/danielededo/cedolo/internal/i18n"
	"github.com/danielededo/cedolo/internal/insights"
	"github.com/danielededo/cedolo/internal/money"
	"github.com/danielededo/cedolo/internal/payslip"
)

//go:embed report.html.tmpl
var pageTemplate string

var page = template.Must(template.New("report").Parse(pageTemplate))

// Status summarizes how far a payslip's totals were cross-checked.
type Status string

const (
	Verified     Status = "verified"
	Unverified   Status = "unverified"
	Inconsistent Status = "inconsistent"
)

// Tile is a headline number for the latest month.
type Tile struct {
	Label string
	Value string
	Delta string // change from the previous month, empty when unknown
}

// Row is one payslip in the month-by-month table.
type Row struct {
	Period        string
	Gross         string
	Deductions    string
	Net           string
	Contributions string
	Taxes         string
	Status        Status
	StatusText    string
	Changes       []insights.Change
}

// LeaveRow is one leave balance.
type LeaveRow struct {
	Label, Previous, Accrued, Used, Remaining string
}

// ItemRow is one taxonomy key found in the payslips.
type ItemRow struct {
	Key, Label, LabelIT string
	Months              int
}

// Contract describes the contract of the latest payslip that names one.
type Contract struct {
	Printed string // as printed on the payslip
	Code    string
	Title   string
	Rows    [][2]string // label, value
	Note    string      // expiry, or why the code is unknown
	Source  string
}

// Data is everything the page shows; Render builds it with Build.
type Data struct {
	Lang     string
	T        map[string]string
	Range    string
	Latest   string
	Tiles    []Tile
	Trend    template.HTML
	Withheld template.HTML
	Rows     []Row
	Changed  bool
	Leave    []LeaveRow
	Contract *Contract
	Items    []ItemRow
	Version  string
}

// Render writes the report for the payslips. The slice is sorted in place.
func Render(w io.Writer, l i18n.Lang, slips []payslip.Payslip, version string) error {
	return page.Execute(w, Build(l, slips, version))
}

// Build computes the page content. The slice is sorted in place.
func Build(l i18n.Lang, slips []payslip.Payslip, version string) Data {
	periods := insights.Compare(l, slips) // sorts slips
	d := Data{Lang: string(l), T: texts(l), Version: version}
	if len(slips) == 0 {
		return d
	}
	d.Range = i18n.T(l, "report.range", len(slips), periods[0].Label, periods[len(periods)-1].Label)

	var monthly []int
	for i, p := range slips {
		if p.Kind == payslip.Monthly && p.Month != 0 {
			monthly = append(monthly, i)
		}
		st := status(p)
		d.Rows = append(d.Rows, Row{
			Period:        periods[i].Label,
			Gross:         amount(p.Gross),
			Deductions:    amount(p.Deductions),
			Net:           amount(p.Net),
			Contributions: optional(contributions(p)),
			Taxes:         optional(taxes(p)),
			Status:        st,
			StatusText:    i18n.T(l, "report."+string(st)),
			Changes:       periods[i].Changes,
		})
		if len(periods[i].Changes) > 0 {
			d.Changed = true
		}
	}

	if len(monthly) > 0 {
		cur := slips[monthly[len(monthly)-1]]
		var prev *payslip.Payslip
		if len(monthly) > 1 {
			prev = &slips[monthly[len(monthly)-2]]
		}
		d.Latest = i18n.T(l, "report.latest", insights.Label(l, cur))
		for _, t := range []struct {
			key string
			f   func(payslip.Payslip) payslip.Amount
		}{
			{"report.net", func(p payslip.Payslip) payslip.Amount { return p.Net }},
			{"report.gross", func(p payslip.Payslip) payslip.Amount { return p.Gross }},
			{"report.deductions", func(p payslip.Payslip) payslip.Amount { return p.Deductions }},
		} {
			a := t.f(cur)
			tile := Tile{Label: i18n.T(l, t.key), Value: amount(a)}
			if prev != nil && a.Found() && t.f(*prev).Found() {
				tile.Delta = i18n.T(l, "report.vs_previous", signed(a.Cents-t.f(*prev).Cents))
			}
			d.Tiles = append(d.Tiles, tile)
		}

		labels := make([]string, len(monthly))
		gross := series{Name: i18n.T(l, "report.gross"), Color: "var(--series-1)"}
		net := series{Name: i18n.T(l, "report.net"), Color: "var(--series-2)"}
		contrib := series{Name: i18n.T(l, "report.contrib"), Color: "var(--series-1)"}
		tax := series{Name: i18n.T(l, "report.taxes"), Color: "var(--series-2)"}
		other := series{Name: i18n.T(l, "report.other"), Color: "var(--series-3)"}
		for i, idx := range monthly {
			p := slips[idx]
			labels[i] = shortLabel(l, p)
			gross.Values = append(gross.Values, p.Gross.Cents)
			net.Values = append(net.Values, p.Net.Cents)
			c, t := contributions(p), taxes(p)
			contrib.Values = append(contrib.Values, max(c, 0))
			tax.Values = append(tax.Values, max(t, 0))
			other.Values = append(other.Values, max(p.Deductions.Cents-max(c, 0)-max(t, 0), 0))
		}
		d.Trend = template.HTML(lineChart("trend", labels, []series{gross, net}))
		d.Withheld = template.HTML(stackedBars("withheld", labels, []series{contrib, tax, other}))

		// Leave balances: the latest payslip that prints any.
		for _, idx := range slices.Backward(monthly) {
			if len(slips[idx].Leave) == 0 {
				continue
			}
			for _, lv := range slips[idx].Leave {
				d.Leave = append(d.Leave, LeaveRow{
					Label:     lv.Label,
					Previous:  quantity(lv.Previous, lv.Unit, l),
					Accrued:   quantity(lv.Accrued, lv.Unit, l),
					Used:      quantity(lv.Used, lv.Unit, l),
					Remaining: quantity(lv.Remaining, lv.Unit, l),
				})
			}
			break
		}
	}

	d.Contract = contract(l, slips)

	months := map[string]int{}
	for _, p := range slips {
		seen := map[string]bool{}
		for _, it := range p.Items {
			if it.Key != "" && !seen[it.Key] {
				seen[it.Key] = true
				months[it.Key]++
			}
		}
	}
	for key, n := range months {
		d.Items = append(d.Items, ItemRow{Key: key, Label: i18n.ItemLabel(l, key), LabelIT: i18n.ItemLabel(i18n.IT, key), Months: n})
	}
	slices.SortFunc(d.Items, func(a, b ItemRow) int {
		if a.Months != b.Months {
			return b.Months - a.Months
		}
		if a.Label < b.Label {
			return -1
		}
		if a.Label > b.Label {
			return 1
		}
		return 0
	})
	return d
}

func texts(l i18n.Lang) map[string]string {
	m := map[string]string{}
	for _, k := range i18n.Keys(l) {
		m[k] = i18n.T(l, k)
	}
	return m
}

func status(p payslip.Payslip) Status {
	all := true
	for _, a := range []payslip.Amount{p.Gross, p.Deductions, p.Net} {
		if a.Status == payslip.Inconsistent {
			return Inconsistent
		}
		if a.Status != payslip.Verified {
			all = false
		}
	}
	if all {
		return Verified
	}
	return Unverified
}

func amount(a payslip.Amount) string {
	if !a.Found() {
		return "–"
	}
	return money.Format(a.Cents) + " €"
}

func optional(c int64) string {
	if c == 0 {
		return "–"
	}
	return money.Format(c) + " €"
}

func signed(c int64) string {
	if c > 0 {
		return "+" + money.Format(c) + " €"
	}
	return money.Format(c) + " €"
}

func quantity(q *int64, unit string, l i18n.Lang) string {
	if q == nil {
		return "–"
	}
	s := money.Format(*q)
	if unit != "" {
		s += " " + i18n.T(l, "unit."+unit)
	}
	return s
}

// shortLabel names a month on a chart axis: "gen 25", "Jan 25".
func shortLabel(l i18n.Lang, p payslip.Payslip) string {
	name := []rune(i18n.Month(l, p.Month))
	if len(name) > 3 {
		name = name[:3]
	}
	return string(name) + " " + twoDigits(p.Year%100)
}

func twoDigits(n int) string {
	return string([]byte{byte('0' + n/10), byte('0' + n%10)})
}

// contributions returns social contributions withheld: the summary box when
// printed, otherwise the INPS items.
func contributions(p payslip.Payslip) int64 {
	if p.SocialContributions.Found() {
		return p.SocialContributions.Cents
	}
	s, _ := p.SumKey("deductions.inps")
	return s
}

// taxes returns income taxes withheld: the summary box when printed,
// otherwise the IRPEF and surcharge items.
func taxes(p payslip.Payslip) int64 {
	if p.Taxes.Found() {
		return p.Taxes.Cents
	}
	var sum int64
	for _, k := range []string{"tax.irpef", "tax.surcharge.regional", "tax.surcharge.municipal"} {
		s, _ := p.SumKey(k)
		sum += s
	}
	return sum
}

// contract describes the contract of the latest payslip that names one.
func contract(l i18n.Lang, slips []payslip.Payslip) *Contract {
	for _, p := range slices.Backward(slips) {
		if p.CCNL == "" && p.CCNLCode == "" {
			continue
		}
		out := &Contract{Source: i18n.T(l, "ccnl.source", ccnl.Source, i18n.Date(l, ccnl.AsOf()))}
		if p.CCNL != "" {
			out.Printed = i18n.T(l, "ccnl.printed", p.CCNL)
		}
		c, ok := ccnl.Lookup(p.CCNLCode)
		if !ok {
			out.Note = i18n.T(l, "ccnl.no_code")
			return out
		}
		out.Code, out.Title = c.Code, c.Title
		period := ""
		switch {
		case c.From != "" && c.Until != "":
			period = i18n.T(l, "ccnl.from_until", i18n.Date(l, c.From), i18n.Date(l, c.Until))
		case c.From != "":
			period = i18n.T(l, "ccnl.from", i18n.Date(l, c.From))
		}
		for _, r := range [][2]string{
			{i18n.T(l, "ccnl.code"), c.Code},
			{i18n.T(l, "ccnl.sectors"), strings.Join(c.Sectors, ", ")},
			{i18n.T(l, "ccnl.period"), period},
			{i18n.T(l, "ccnl.employers"), strings.Join(c.Employers, ", ")},
			{i18n.T(l, "ccnl.unions"), strings.Join(c.Unions, ", ")},
		} {
			if r[1] != "" {
				out.Rows = append(out.Rows, r)
			}
		}
		// Expired before the payslip's month.
		if c.Until != "" && p.Month != 0 && c.Until < fmt.Sprintf("%04d-%02d-01", p.Year, p.Month) {
			out.Note = i18n.T(l, "ccnl.expired", i18n.Date(l, c.Until))
		}
		return out
	}
	return nil
}
