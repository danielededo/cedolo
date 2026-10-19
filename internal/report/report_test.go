package report

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"html"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/danielededo/cedolo/internal/extract"
	"github.com/danielededo/cedolo/internal/i18n"
	"github.com/danielededo/cedolo/internal/insights"
	"github.com/danielededo/cedolo/internal/payslip"
	"github.com/danielededo/cedolo/internal/synth"
)

// months extracts a run of synthetic monthly payslips, overtime growing
// each month so that the charts have something to show.
func months(t *testing.T, n int) []payslip.Payslip {
	t.Helper()
	var out []payslip.Payslip
	for i := range n {
		p := synth.Example()
		p.Items = append([]synth.Item(nil), p.Items...)
		p.Month = time.Month(1 + i)
		p.Items[1].Amount += int64(i) * 5000
		got := extract.Document(synth.Layout(p, synth.StyleClassic))
		if len(got) != 1 {
			t.Fatalf("got %d payslips", len(got))
		}
		out = append(out, got[0])
	}
	return out
}

func TestBuild(t *testing.T) {
	slips := months(t, 3)
	slips[0], slips[2] = slips[2], slips[0] // out of order on purpose
	d := Build(i18n.EN, slips, "test")
	if len(d.Rows) != 3 || !strings.HasPrefix(d.Rows[0].Period, "January") {
		t.Fatalf("rows = %+v", d.Rows)
	}
	for _, r := range d.Rows {
		if r.Status != Verified {
			t.Errorf("%s: status %s", r.Period, r.Status)
		}
	}
	if len(d.Months) != 3 || d.Selected != 2 || !strings.HasPrefix(d.Months[2].Label, "March") {
		t.Fatalf("months = %+v, selected %d", d.Months, d.Selected)
	}
	m := d.Months[2]
	if m.Net == "–" || !strings.HasPrefix(m.Gross, "out of ") || !strings.Contains(m.Delta, "+") || m.Short != "Mar 26" {
		t.Errorf("latest month = %+v", m)
	}
	if d.Months[0].Delta != "" {
		t.Errorf("first month has a delta: %q", d.Months[0].Delta)
	}
	if d.Trend == "" {
		t.Error("no trend for three months")
	}
	var width float64
	keys := map[string]bool{}
	for _, p := range m.Parts {
		width += p.Width
		keys[p.Key] = true
		if p.Text == "" || p.Amount == "" {
			t.Errorf("part %+v", p)
		}
	}
	if !keys["net"] || !keys["contrib"] || !keys["taxes"] || width < 99.9 || width > 100.1 {
		t.Errorf("parts %+v sum to %.2f%%", m.Parts, width)
	}
	lines := len(m.Earnings) + len(m.SetAside)
	for _, p := range m.Parts {
		lines += len(p.Lines)
	}
	if lines != len(slips[2].Items) {
		t.Errorf("%d lines for %d items", lines, len(slips[2].Items))
	}
	for _, l := range m.Earnings {
		if l.Description == "" || l.Text == "" || l.Label == "" {
			t.Errorf("line %+v", l)
		}
	}
	if len(m.Leave) == 0 || m.Leave[0].Remaining == "" || !strings.HasPrefix(m.Leave[0].Moves, "before ") {
		t.Errorf("leave rows in the month = %+v", m.Leave)
	}
	if len(d.Terms) != len(i18n.Terms) || d.Terms[0].Text == "" {
		t.Errorf("terms = %+v", d.Terms)
	}
}

func TestRender(t *testing.T) {
	for _, l := range []i18n.Lang{i18n.IT, i18n.EN} {
		var b bytes.Buffer
		if err := Render(&b, l, months(t, 4), "test"); err != nil {
			t.Fatal(err)
		}
		page := b.String()
		for _, want := range []string{
			`<html lang="` + string(l) + `">`,
			"Content-Security-Policy",
			`<svg id="trend"`,
			`<section class="month" data-month="3">`,
			`data-month="0" hidden>`,
			`class="status verified"`,
			html.EscapeString(i18n.T(l, "report.privacy")),
			html.EscapeString(i18n.T(l, "report.disclaimer")),
			html.EscapeString(i18n.T(l, "report.where")),
			html.EscapeString(i18n.ItemText(l, "tax.irpef")),
			html.EscapeString(i18n.T(l, "term.year_end.text")),
		} {
			if !strings.Contains(page, want) {
				t.Errorf("%s: page lacks %q", l, want)
			}
		}
		if strings.Count(page, `aria-pressed="true">`) != 1 || strings.Count(page, `<section class="month"`) != 4 {
			t.Errorf("%s: the picker must select exactly one of four months", l)
		}
		// Self-contained: nothing is fetched from anywhere.
		for _, bad := range []string{"http://", "https://", "src=", "@import", "url("} {
			if strings.Contains(page, bad) {
				t.Errorf("%s: page contains %q", l, bad)
			}
		}
	}
}

func TestRenderEmpty(t *testing.T) {
	var b bytes.Buffer
	if err := Render(&b, i18n.EN, nil, ""); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(b.String(), "<svg") {
		t.Error("charts drawn without payslips")
	}
}

func TestNiceMax(t *testing.T) {
	for _, c := range []struct{ in, top, step int64 }{
		{0, 10000, 2500},
		{230000, 300000, 100000},
		{190000, 200000, 50000},
		{345678, 400000, 100000},
		{99, 100, 25},
	} {
		top, step := niceMax(c.in)
		if top != c.top || step != c.step {
			t.Errorf("niceMax(%d) = %d, %d; want %d, %d", c.in, top, step, c.top, c.step)
		}
	}
}

func TestTicks(t *testing.T) {
	for in, want := range map[int64]string{0: "0 €", 50000: "500 €", 100000: "1.000 €", 250000: "2.500 €", 2500000: "25.000 €"} {
		if got := euros(in); got != want {
			t.Errorf("euros(%d) = %q, want %q", in, got, want)
		}
	}
	if got := whole(9600); got != "96" {
		t.Errorf("whole(9600) = %q", got)
	}
}

func TestContract(t *testing.T) {
	slips := months(t, 2)
	slips[1].CCNL = "H011 COMMERCIO TERZIARIO"
	slips[1].CCNLCode = "H011"
	slips[1].Year = 2030 // after the agreement in the list expires
	d := Build(i18n.EN, slips, "")
	c := d.Contract
	if c == nil || c.Code != "H011" || c.Title == "" || len(c.Rows) < 3 {
		t.Fatalf("contract = %+v", c)
	}
	if !strings.Contains(c.Note, "expired") {
		t.Errorf("note = %q", c.Note)
	}

	slips = months(t, 1)
	slips[0].CCNL = "COMMERCIO"
	slips[0].CCNLCode = ""
	c = Build(i18n.IT, slips, "").Contract
	if c == nil || c.Code != "" || !strings.Contains(c.Note, "cedolo ccnl -search") {
		t.Errorf("contract without code = %+v", c)
	}
}

// The inline script must render verbatim, or pages embedding the report
// with ScriptHash in their policy would block it.
func TestScriptHash(t *testing.T) {
	var b bytes.Buffer
	if err := Render(&b, i18n.EN, months(t, 2), ""); err != nil {
		t.Fatal(err)
	}
	_, rest, _ := strings.Cut(b.String(), "<script>")
	script, _, ok := strings.Cut(rest, "</script>")
	if !ok || strings.Count(b.String(), "<script") != 1 {
		t.Fatal("the report should have exactly one inline script")
	}
	sum := sha256.Sum256([]byte(script))
	if want := "'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"; ScriptHash() != want {
		t.Errorf("ScriptHash() = %s, rendered script hashes to %s", ScriptHash(), want)
	}
}

func TestTellingChanges(t *testing.T) {
	cs := []insights.Change{{Code: "leave"}, {Code: "gross"}, {Code: "net"}, {Code: "item_new"}, {Code: "unchecked"}}
	top, rest := telling(cs)
	if len(top) != 2 || top[0].Code != "unchecked" || top[1].Code != "net" {
		t.Errorf("top = %+v", top)
	}
	if len(rest) != 3 || rest[0].Code != "item_new" || rest[2].Code != "leave" {
		t.Errorf("rest = %+v", rest)
	}
	if top, rest := telling(cs[:1]); len(top) != 1 || rest != nil {
		t.Errorf("one change: %+v %+v", top, rest)
	}
	if cs[0].Code != "leave" {
		t.Error("input reordered")
	}
}

func TestMonthSummary(t *testing.T) {
	d := Build(i18n.EN, months(t, 3), "test")
	if got := d.Months[0].Summary; !strings.HasPrefix(got, "Of every 100 € of gross pay you receive ") {
		t.Errorf("first month summary = %q", got)
	}
	// Overtime grows by 50 € a month and drives the net up.
	want := "Compared with February 2026 you receive "
	if got := d.Months[2].Summary; !strings.HasPrefix(got, want) || !strings.Contains(got, "more, mostly because of “"+i18n.ItemLabel(i18n.EN, "earnings.overtime")+"” (+50,00 €).") {
		t.Errorf("latest month summary = %q", got)
	}

	item := func(key string, side payslip.Side, cents int64) payslip.Item {
		return payslip.Item{Key: key, Side: side, Cents: cents}
	}
	slip := func(month int, net int64, items ...payslip.Item) payslip.Payslip {
		return payslip.Payslip{Year: 2026, Month: month, Kind: payslip.Monthly,
			Net: payslip.Amount{Cents: net, Status: payslip.Verified}, Items: items}
	}
	prev := slip(1, 150000, item("earnings.base_salary", payslip.Earning, 200000), item("tax.irpef", payslip.Deduction, 50000))
	for _, c := range []struct {
		name string
		cur  payslip.Payslip
		want string
	}{
		{"same", slip(2, 150000, prev.Items...), "You receive the same as in January 2026."},
		{"tax up", slip(2, 140000, item("earnings.base_salary", payslip.Earning, 200000), item("tax.irpef", payslip.Deduction, 60000)),
			"Compared with January 2026 you receive 100,00 € less, mostly because of “" + i18n.ItemLabel(i18n.EN, "tax.irpef") + "” (-100,00 €)."},
		{"no single cause", slip(2, 151000, item("earnings.base_salary", payslip.Earning, 200400), item("earnings.allowance", payslip.Earning, 300),
			item("earnings.overtime", payslip.Earning, 300), item("tax.irpef", payslip.Deduction, 50000)),
			"Compared with January 2026 you receive 10,00 € more."},
	} {
		if got, _ := compared(i18n.EN, prev, c.cur); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}

func TestMonthSummaryItalianPreposition(t *testing.T) {
	prev := payslip.Payslip{Year: 2026, Month: 8, Kind: payslip.Monthly, Net: payslip.Amount{Cents: 100, Status: payslip.Verified}}
	cur := prev
	cur.Month = 9
	if got, _ := compared(i18n.IT, prev, cur); got != "Ricevi quanto ad agosto 2026." {
		t.Errorf("got %q", got)
	}
	prev.Month, cur.Month = 9, 10
	if got, _ := compared(i18n.IT, prev, cur); got != "Ricevi quanto a settembre 2026." {
		t.Errorf("got %q", got)
	}
}

// TestLatestAdditionalMonthOpensFirst checks that a thirteenth month after
// the last ordinary month is the one shown and highlighted first.
func TestLatestAdditionalMonthOpensFirst(t *testing.T) {
	slips := months(t, 12)[9:] // October to December
	thirteenth := slips[2]
	thirteenth.Kind = payslip.Thirteenth
	slips = append(slips, thirteenth)
	d := Build(i18n.IT, slips, "test")
	if d.Selected != len(d.Months)-1 || d.Months[d.Selected].Short != "13ª 26" {
		t.Fatalf("selected %d (%q), want the thirteenth month", d.Selected, d.Months[d.Selected].Short)
	}
	var b bytes.Buffer
	if err := Render(&b, i18n.IT, slips, "test"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), `aria-pressed="true">13ª 26</button>`) {
		t.Error("the thirteenth month is not highlighted in the picker")
	}
}

func TestTrendAndLeaveCharts(t *testing.T) {
	slips := months(t, 3)
	thirteenth := slips[2]
	thirteenth.Kind = payslip.Thirteenth
	slips = append(slips, thirteenth)
	d := Build(i18n.IT, slips, "test")
	// One column per payslip, the thirteenth month included, each a hit
	// target that opens its month; drawn wide and narrow, for phones.
	trend := string(d.Trend)
	if n := strings.Count(trend, `class="col-hit"`); n != 8 {
		t.Errorf("%d columns, want 4 in each size", n)
	}
	if !strings.Contains(trend, `class="chart stack wide"`) || !strings.Contains(trend, `id="trend-narrow" class="chart stack narrow" viewBox="0 0 360 240"`) {
		t.Errorf("trend is not drawn in both sizes")
	}
	if !strings.Contains(trend, "13ª 26") || !strings.Contains(trend, "3.000 €") {
		t.Errorf("trend misses the thirteenth month or euro ticks")
	}
	// The synthetic months print holidays and ROL; the thirteenth month
	// repeats December and is left out of the lines.
	if len(d.LeaveCharts) != 2 {
		t.Fatalf("leave charts = %+v", d.LeaveCharts)
	}
	if c := d.LeaveCharts[0]; c.Label != "FERIE" || c.Now != "96,00 ore" || strings.Contains(string(c.SVG), "13ª") {
		t.Errorf("holiday chart = %q %q", c.Label, c.Now)
	}
	if d := Build(i18n.IT, slips[:1], "test"); d.Trend != "" || len(d.LeaveCharts) != 0 {
		t.Error("charts for a single payslip")
	}
}

func TestRereadTotalsAreNoted(t *testing.T) {
	p := payslip.Payslip{Year: 2026, Month: 3, Kind: payslip.Monthly,
		Net:   payslip.Amount{Cents: 150000, Status: payslip.Verified, Alternative: true},
		Taxes: payslip.Amount{Cents: 30000, Status: payslip.Verified, Alternative: true}}
	want := "To make the totals add up, cedolo read some amounts from a less usual place on the payslip, or from boxes of their own (Net pay, Taxes): compare them with the payslip."
	if got := reread(i18n.EN, p); got != want {
		t.Errorf("reread = %q", got)
	}
	p.Net.Alternative, p.Taxes.Alternative = false, false
	if got := reread(i18n.EN, p); got != "" {
		t.Errorf("reread without alternatives = %q", got)
	}
}

func TestYearsTable(t *testing.T) {
	a := func(c int64) payslip.Amount { return payslip.Amount{Cents: c, Status: payslip.Verified} }
	slip := func(year, month int, kind payslip.Kind, gross, net, tfr int64) payslip.Payslip {
		return payslip.Payslip{Year: year, Month: month, Kind: kind, Gross: a(gross), Net: a(net), TFR: a(tfr),
			SocialContributions: a(gross / 10), Taxes: a(gross / 5)}
	}
	slips := []payslip.Payslip{
		slip(2026, 1, payslip.Monthly, 200000, 140000, 14000),
		slip(2025, 11, payslip.Monthly, 200000, 140000, 14000),
		slip(2025, 12, payslip.Monthly, 200000, 160000, 14000),
		slip(2025, 12, payslip.Thirteenth, 200000, 150000, 14000),
	}
	got := years(i18n.EN, slips)
	if got == nil || strings.Join(got.Years, ",") != "2025*,2026*" || !got.Partial {
		t.Fatalf("years = %+v", got)
	}
	cells := map[string]string{}
	for _, r := range got.Rows {
		cells[r.Label] = strings.Join(r.Cells, " | ")
	}
	for label, want := range map[string]string{
		"Payslips":                    "2 + 1 | 1",
		"Gross pay":                   "6.000,00 € | 2.000,00 €",
		"Taxes":                       "1.200,00 € | 400,00 €",
		"Net pay":                     "4.500,00 € | 1.400,00 €",
		"Average monthly net":         "1.500,00 € | 1.400,00 €",
		"Net of every 100 € of gross": "75 € | 70 €",
	} {
		if cells[label] != want {
			t.Errorf("%s = %q, want %q", label, cells[label], want)
		}
	}
	if years(i18n.EN, []payslip.Payslip{{}}) != nil {
		t.Error("undated payslips make a table")
	}
}

func TestVariablePayAndTFR(t *testing.T) {
	slips := months(t, 3)
	d := Build(i18n.EN, slips, "test")
	// The synthetic payslips carry overtime and an allowance besides the
	// base salary, and print the month's TFR.
	if !strings.Contains(string(d.Variable), `id="variable"`) || !strings.Contains(string(d.Variable), "Overtime, bonuses and allowances") {
		t.Errorf("variable = %.200s", d.Variable)
	}
	if !strings.Contains(string(d.TFR), `id="tfr"`) || d.TFRTotal == "" || d.TFRSpan != "accrued over 3 payslips" {
		t.Errorf("tfr = %.200s, total %q, span %q", d.TFR, d.TFRTotal, d.TFRSpan)
	}
	// Without overtime, bonuses or allowances there is nothing to chart.
	for i := range slips {
		var base []payslip.Item
		for _, it := range slips[i].Items {
			if it.Side != payslip.Earning || it.Key == "earnings.base_salary" {
				base = append(base, it)
			}
		}
		slips[i].Items = base
	}
	if v := variablePay(i18n.EN, slips); v != "" {
		t.Error("variable pay drawn for ordinary pay only")
	}
	if svg, _, _ := tfrChart(i18n.EN, slips[:1]); svg != "" {
		t.Error("TFR chart drawn for one payslip")
	}
}

func TestTabs(t *testing.T) {
	var b bytes.Buffer
	if err := Render(&b, i18n.EN, months(t, 2), "test"); err != nil {
		t.Fatal(err)
	}
	page := b.String()
	if n := strings.Count(page, `role="tab" id=`); n != 3 {
		t.Errorf("%d tabs, want 3", n)
	}
	for _, want := range []string{`id="panel-month" role="tabpanel" aria-labelledby="tab-month">`,
		`id="panel-time" role="tabpanel" aria-labelledby="tab-time" hidden>`} {
		if !strings.Contains(page, want) {
			t.Errorf("page misses %s", want)
		}
	}
	// A single payslip has nothing over time: one page, no tabs.
	b.Reset()
	if err := Render(&b, i18n.EN, months(t, 1), "test"); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(b.String(), `role="tab" id=`) {
		t.Error("tabs for a single payslip")
	}
}

func TestToldChangesAreLeftOut(t *testing.T) {
	changes := []insights.Change{
		{Code: "net", Text: "net", Why: []string{"overtime"}},
		{Code: "item_new", Key: "earnings.overtime", Text: "overtime"},
		{Code: "item_new", Key: "earnings.bonus", Text: "bonus"},
		{Code: "leave", Text: "leave"},
	}
	rest, why := told(changes, "earnings.overtime")
	if len(rest) != 2 || rest[0].Text != "bonus" || rest[1].Text != "leave" || len(why) != 1 || why[0] != "overtime" {
		t.Errorf("rest = %+v, why = %v", rest, why)
	}
}

// TestPensionFundInContributions covers payroll software that prints your
// own pension fund contribution inside the contributions box, beside INPS:
// it is no contribution to INPS, so it moves to the other deductions, and
// it explains a change of the net.
func TestPensionFundInContributions(t *testing.T) {
	a := func(c int64) payslip.Amount { return payslip.Amount{Cents: c, Status: payslip.Verified} }
	slip := func(month int, fund, net int64) payslip.Payslip {
		return payslip.Payslip{Year: 2026, Month: month, Kind: payslip.Monthly,
			Items: []payslip.Item{
				{Key: "earnings.base_salary", Side: payslip.Earning, Cents: 300000},
				{Description: "Contr. FPC dipendente", Key: "pension_fund.employee", Side: payslip.Info, Cents: fund},
				{Description: "Contr. FPC ditta", Key: "pension_fund.contribution", Side: payslip.Info, Cents: 6000},
				{Description: "Storno T.F.R. a F.P.C.", Key: "tfr.accrued", Side: payslip.Info, Cents: 20000},
			},
			Gross: a(300000), Net: a(net), INPS: a(27570), SocialContributions: a(27570 + fund), Taxes: a(60000),
			Deductions: a(27570 + fund + 60000), TFR: a(20000)}
	}
	aug, sep := slip(8, 3000, 209430), slip(9, 70000, 142430)
	if got := contributions(sep); got != 27570 {
		t.Errorf("contributions = %d, want the INPS part only", got)
	}
	if got := split(sep); got[3] != 70000 {
		t.Errorf("other deductions = %d, want the pension fund contribution", got[3])
	}
	d := Build(i18n.EN, []payslip.Payslip{aug, sep}, "test")
	m := d.Months[1]
	if !strings.Contains(m.Summary, "Your pension fund contribution") {
		t.Errorf("summary = %q", m.Summary)
	}
	var lines []string
	for _, pt := range m.Parts {
		for _, l := range pt.Lines {
			lines = append(lines, pt.Key+":"+l.Description)
		}
	}
	if !slices.Contains(lines, "other:Contr. FPC dipendente") || len(m.SetAside) != 2 {
		t.Errorf("lines = %v, set aside = %+v", lines, m.SetAside)
	}
	// What went into the fund: the month's three parts, and over time a
	// chart in their place of the TFR's, all of which went to the fund.
	if m.Fund != "To your pension fund this month: 960,00 € (700,00 € from you, 60,00 € from the employer, 200,00 € of TFR)." {
		t.Errorf("fund = %q", m.Fund)
	}
	if d.Fund == "" || d.FundTotal != "1.250,00 €" || d.FundSpan != "paid in over 2 payslips" || d.TFR != "" {
		t.Errorf("fund chart %q %q, TFR chart %.40q", d.FundTotal, d.FundSpan, d.TFR)
	}
	// The year counts against the deductible limit your contribution and
	// the employer's, not the TFR.
	var deductible []string
	for _, r := range d.Years.Rows {
		if r.Label == "Pension fund deductible used" {
			deductible = r.Cells
		}
	}
	if !slices.Equal(deductible, []string{"850,00 €"}) || !d.Years.Deductible || d.Years.Over {
		t.Errorf("deductible = %v, over = %v", deductible, d.Years.Over)
	}
	if over := Build(i18n.EN, []payslip.Payslip{aug, slip(9, 530000, 142430)}, "test"); !over.Years.Over {
		t.Errorf("no word of a year over the limit")
	}
	if deductibleLimit(2025) != 516457 {
		t.Errorf("2025 limit = %d", deductibleLimit(2025))
	}
	// The chart starts at the first payslip that paid in, and its columns
	// still lead to their own months.
	jul := payslip.Payslip{Year: 2026, Month: 7, Kind: payslip.Monthly,
		Items: []payslip.Item{{Key: "earnings.base_salary", Side: payslip.Earning, Cents: 300000}},
		Gross: a(300000), Net: a(212430), INPS: a(27570), SocialContributions: a(27570), Taxes: a(60000),
		Deductions: a(87570), TFR: a(20000)}
	d = Build(i18n.EN, []payslip.Payslip{jul, aug, sep}, "test")
	fund := string(d.Fund)
	if strings.Contains(fund, "Jul") || strings.Count(fund, `class="col-hit" data-i="1"`) != 2 || strings.Contains(fund, `data-i="0"`) {
		t.Errorf("fund chart from July: %.300s", fund)
	}
	if d.FundTotal != "1.250,00 €" {
		t.Errorf("fund total = %q", d.FundTotal)
	}
	// One payslip from the first payment is not a chart.
	if d = Build(i18n.EN, []payslip.Payslip{jul, sep}, "test"); d.Fund != "" {
		t.Errorf("fund chart from one payslip")
	}
	// A box that cannot hold the contribution beside INPS keeps it all.
	sep.SocialContributions = a(27570)
	if got := contributions(sep); got != 27570 || pensionInContributions(sep) != 0 {
		t.Errorf("contributions = %d", got)
	}
}
