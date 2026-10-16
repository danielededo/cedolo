// Package report renders payslips as one self-contained HTML page. It shows
// one payslip at a time, the latest first: the net pay, where the gross pay
// goes, the printed items with their explanation one tap away, and what
// changed. Pay over time, the month table, leave, the contract and the
// payslip's vocabulary follow, closed until opened. The page loads nothing
// from the network and runs no code but a small inline script.
package report

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
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

// Month is one payslip as the page shows it.
type Month struct {
	Label      string // "ottobre 2026"
	Short      string // "ott 26", for the month picker
	Status     Status
	StatusText string
	Net        string
	Gross      string // "of 2.194,50 € gross pay"
	Delta      string // change of the net from the previous month, empty when unknown
	Parts      []Part // where the gross pay goes
	Earnings   []Line // what makes up the gross pay
	SetAside   []Line // TFR and other amounts that do not change this month's pay
	GrossTotal string // the gross pay, next to Earnings
	AsideTotal string // the sum of SetAside
	Changes    []insights.Change
}

// Part is one share of the gross pay: the net or a kind of deduction.
type Part struct {
	Key, Label, Amount, Text string
	Color                    string
	Width                    float64 // percent of the bar
	Lines                    []Line
}

// Line is one printed item with its explanation.
type Line struct {
	Description, Label, Amount, Text string
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

// Term is a glossary word printed around the items.
type Term struct {
	Label, Text string
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
	Months   []Month
	Selected int // index in Months of the month shown first
	Trend    template.HTML
	Rows     []Row
	Leave    []LeaveRow
	Contract *Contract
	Terms    []Term
	Version  string
}

// ScriptHash returns the Content-Security-Policy source that allows the
// report's inline script, such as "'sha256-…'", for pages that embed the
// report and so pass their own policy on to it.
func ScriptHash() string {
	// Hash the script as rendered: html/template drops its comments.
	var page bytes.Buffer
	if err := Render(&page, i18n.IT, nil, ""); err != nil {
		panic(err)
	}
	_, rest, _ := strings.Cut(page.String(), "<script>")
	script, _, _ := strings.Cut(rest, "</script>")
	sum := sha256.Sum256([]byte(script))
	return "'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"
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
	if len(slips) == 1 {
		d.Range = i18n.T(l, "report.range.one", periods[0].Label)
	} else {
		d.Range = i18n.T(l, "report.range", len(slips), periods[0].Label, periods[len(periods)-1].Label)
	}

	var monthly []int
	var prev *payslip.Payslip // previous ordinary month
	for i, p := range slips {
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
		m := month(l, p, periods[i])
		if p.Kind == payslip.Monthly && p.Month != 0 {
			if prev != nil && p.Net.Found() && prev.Net.Found() {
				m.Delta = i18n.T(l, "report.vs_previous", signed(p.Net.Cents-prev.Net.Cents))
			}
			prev = &slips[i]
			monthly = append(monthly, i)
			d.Selected = i
		}
		d.Months = append(d.Months, m)
	}
	if len(monthly) == 0 {
		d.Selected = len(slips) - 1
	}

	if len(monthly) > 1 {
		labels := make([]string, len(monthly))
		gross := series{Name: i18n.T(l, "report.gross"), Color: "var(--series-1)"}
		net := series{Name: i18n.T(l, "report.net"), Color: "var(--series-2)"}
		for i, idx := range monthly {
			p := slips[idx]
			labels[i] = shortLabel(l, p)
			gross.Values = append(gross.Values, p.Gross.Cents)
			net.Values = append(net.Values, p.Net.Cents)
		}
		d.Trend = template.HTML(lineChart("trend", labels, []series{gross, net}))
	}

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

	d.Contract = contract(l, slips)
	for _, term := range i18n.Terms {
		d.Terms = append(d.Terms, Term{Label: i18n.T(l, "term."+term), Text: i18n.T(l, "term."+term+".text")})
	}
	return d
}

// month builds the view of one payslip.
func month(l i18n.Lang, p payslip.Payslip, per insights.Period) Month {
	st := status(p)
	m := Month{
		Label:      per.Label,
		Short:      shortLabel(l, p),
		Status:     st,
		StatusText: i18n.T(l, "report."+string(st)),
		Net:        amount(p.Net),
		Changes:    per.Changes,
	}
	if p.Gross.Found() {
		m.Gross = i18n.T(l, "report.of_gross", amount(p.Gross))
		m.GrossTotal = amount(p.Gross)
	}
	line := func(it payslip.Item) Line {
		return Line{Description: it.Description, Label: i18n.ItemLabel(l, it.Key), Amount: money.Format(it.Cents) + " €", Text: i18n.ItemText(l, it.Key)}
	}
	var contrib, tax, other []Line
	for _, it := range p.Items {
		switch {
		case it.Side == payslip.Earning:
			m.Earnings = append(m.Earnings, line(it))
		case it.Side == payslip.Info:
			m.SetAside = append(m.SetAside, line(it))
		case it.Key == "deductions.inps":
			contrib = append(contrib, line(it))
		case strings.HasPrefix(it.Key, "tax.") && it.Key != "tax.credit":
			tax = append(tax, line(it))
		default:
			other = append(other, line(it))
		}
	}
	var aside int64
	for _, it := range p.Items {
		if it.Side == payslip.Info {
			aside += it.Cents
		}
	}
	if aside != 0 {
		m.AsideTotal = money.Format(aside) + " €"
	}
	c, t := max(contributions(p), 0), max(taxes(p), 0)
	o := max(p.Deductions.Cents-c-t, 0)
	parts := []struct {
		key   string
		cents int64
		color string
		lines []Line
	}{
		{"net", max(p.Net.Cents, 0), "var(--series-1)", nil},
		{"contrib", c, "var(--series-2)", contrib},
		{"taxes", t, "var(--series-3)", tax},
		{"other", o, "var(--series-4)", other},
	}
	var total int64
	for _, pt := range parts {
		total += pt.cents
	}
	if total == 0 {
		return m
	}
	for _, pt := range parts {
		if pt.cents == 0 && len(pt.lines) == 0 {
			continue
		}
		m.Parts = append(m.Parts, Part{
			Key:    pt.key,
			Label:  i18n.T(l, "report.part."+pt.key),
			Amount: money.Format(pt.cents) + " €",
			Text:   i18n.T(l, "report.part."+pt.key+".text"),
			Color:  pt.color,
			Width:  float64(pt.cents) * 100 / float64(total),
			Lines:  pt.lines,
		})
	}
	return m
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

// shortLabel names a month on a chart axis or the month picker: "gen 25",
// "Jan 25", and "13ª 25" or "13th 25" for an additional month.
func shortLabel(l i18n.Lang, p payslip.Payslip) string {
	switch p.Kind {
	case payslip.Thirteenth, payslip.Fourteenth:
		return i18n.T(l, "report.short."+string(p.Kind)) + " " + twoDigits(p.Year%100)
	}
	if p.Month == 0 {
		return "?"
	}
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
