package report

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"html"
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
	if len(d.Leave) == 0 {
		t.Error("no leave rows")
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

func TestCompact(t *testing.T) {
	for in, want := range map[int64]string{0: "0", 50000: "500", 100000: "1K", 150000: "1,5K", 2500000: "25K"} {
		if got := compact(in); got != want {
			t.Errorf("compact(%d) = %q, want %q", in, got, want)
		}
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
		if got := compared(i18n.EN, prev, c.cur); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}

func TestMonthSummaryItalianPreposition(t *testing.T) {
	prev := payslip.Payslip{Year: 2026, Month: 8, Kind: payslip.Monthly, Net: payslip.Amount{Cents: 100, Status: payslip.Verified}}
	cur := prev
	cur.Month = 9
	if got := compared(i18n.IT, prev, cur); got != "Ricevi quanto ad agosto 2026." {
		t.Errorf("got %q", got)
	}
	prev.Month, cur.Month = 9, 10
	if got := compared(i18n.IT, prev, cur); got != "Ricevi quanto a settembre 2026." {
		t.Errorf("got %q", got)
	}
}
