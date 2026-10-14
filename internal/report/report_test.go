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
	if len(d.Tiles) != 3 || d.Tiles[0].Delta == "" {
		t.Errorf("tiles = %+v", d.Tiles)
	}
	if !strings.HasPrefix(d.Tiles[1].Delta, "+") {
		t.Errorf("gross grew, delta = %q", d.Tiles[1].Delta)
	}
	if len(d.Leave) == 0 {
		t.Error("no leave rows")
	}
	if len(d.Items) == 0 || d.Items[0].Months != 3 {
		t.Errorf("items = %+v", d.Items)
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
			`<svg id="withheld"`,
			`class="status verified"`,
			html.EscapeString(i18n.T(l, "report.privacy")),
			html.EscapeString(i18n.T(l, "report.disclaimer")),
		} {
			if !strings.Contains(page, want) {
				t.Errorf("%s: page lacks %q", l, want)
			}
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
