package insights

import (
	"strings"
	"testing"
	"time"

	"github.com/danielededo/cedolo/internal/extract"
	"github.com/danielededo/cedolo/internal/i18n"
	"github.com/danielededo/cedolo/internal/payslip"
	"github.com/danielededo/cedolo/internal/synth"
)

// read extracts a synthetic payslip, as the CLI would from a PDF.
func read(t *testing.T, p synth.Payslip) payslip.Payslip {
	t.Helper()
	got := extract.Document(synth.Layout(p, synth.StyleClassic))
	if len(got) != 1 {
		t.Fatalf("got %d payslips", len(got))
	}
	return got[0]
}

func codes(cs []Change) map[string]int {
	out := map[string]int{}
	for _, c := range cs {
		out[c.Code]++
	}
	return out
}

func TestCompareDetectsChanges(t *testing.T) {
	// October without surcharges; in November a municipal surcharge
	// advance starts, overtime grows a lot, the level changes and leave is
	// used.
	oct := synth.Example()
	var kept []synth.Item
	for _, it := range oct.Items {
		if !strings.HasPrefix(it.Key, "tax.surcharge") {
			kept = append(kept, it)
		}
	}
	oct.Items = kept
	nov := oct
	nov.Items = append([]synth.Item(nil), oct.Items...)
	nov.Leave = append([]synth.Leave(nil), oct.Leave...)
	nov.Month = time.November
	nov.Level = "3"
	nov.Items[1].Amount = 80000
	nov.Items = append(nov.Items, synth.Item{Code: "9170", Description: "ACCONTO ADDIZIONALE COMUNALE", Amount: 1200, Deduction: true})
	nov.Leave[0].Used, nov.Leave[0].Remaining = "16,00", "88,00"

	slips := []payslip.Payslip{read(t, nov), read(t, oct)} // out of order on purpose
	got := Compare(i18n.IT, slips)
	if len(got) != 2 || got[0].Month != 10 || got[1].Month != 11 {
		t.Fatalf("periods = %+v", got)
	}
	if len(got[0].Changes) != 0 {
		t.Errorf("first payslip has changes: %+v", got[0].Changes)
	}
	c := codes(got[1].Changes)
	for _, want := range []string{"level", "net", "gross", "leave"} {
		if c[want] != 1 {
			t.Errorf("want one %q change, got %v", want, got[1].Changes)
		}
	}
	var surcharge, net string
	for _, ch := range got[1].Changes {
		if ch.Code == "item_new" {
			surcharge = ch.String()
		}
		if ch.Code == "net" {
			net = ch.String()
		}
	}
	if !strings.Contains(surcharge, "Addizionale comunale") || !strings.Contains(surcharge, "a rate") {
		t.Errorf("surcharge change = %q", surcharge)
	}
	if !strings.Contains(net, "straordinari") {
		t.Errorf("net change should mention overtime: %q", net)
	}
	if got[1].Label != "novembre 2026" {
		t.Errorf("label = %q", got[1].Label)
	}
}

func TestCompareQuietWhenStable(t *testing.T) {
	oct := synth.Example()
	nov := synth.Example()
	nov.Month = time.November
	got := Compare(i18n.EN, []payslip.Payslip{read(t, oct), read(t, nov)})
	if len(got[1].Changes) != 0 {
		t.Errorf("identical months produced changes: %+v", got[1].Changes)
	}
}

func TestGapAndExtraMonths(t *testing.T) {
	jan := payslip.Payslip{Year: 2026, Month: 1, Kind: payslip.Monthly}
	apr := payslip.Payslip{Year: 2026, Month: 4, Kind: payslip.Monthly}
	dec13 := payslip.Payslip{Year: 2025, Month: 12, Kind: payslip.Thirteenth}
	dec := payslip.Payslip{Year: 2025, Month: 12, Kind: payslip.Monthly}
	unknown := payslip.Payslip{Kind: payslip.Monthly}
	got := Compare(i18n.EN, []payslip.Payslip{apr, unknown, dec13, jan, dec})
	var labels []string
	for _, p := range got {
		labels = append(labels, p.Label)
	}
	want := "December 2025|December 2025, thirteenth month|January 2026|April 2026|unknown period"
	if strings.Join(labels, "|") != want {
		t.Errorf("order = %v", labels)
	}
	if c := codes(got[3].Changes); c["gap"] != 1 {
		t.Errorf("April should report a gap: %+v", got[3].Changes)
	}
	if len(got[1].Changes) != 0 || len(got[2].Changes) != 0 {
		t.Errorf("unexpected changes: %+v %+v", got[1].Changes, got[2].Changes)
	}
}

func TestInconsistentPayslipIsFlagged(t *testing.T) {
	p := payslip.Payslip{Year: 2026, Month: 5, Kind: payslip.Monthly, Net: payslip.Amount{Status: payslip.Inconsistent}}
	got := Compare(i18n.EN, []payslip.Payslip{p})
	if c := codes(got[0].Changes); c["unchecked"] != 1 {
		t.Errorf("changes = %+v", got[0].Changes)
	}
}

func TestCCNLChangeSuggestsDifferentEmployers(t *testing.T) {
	a := payslip.Payslip{Year: 2026, Month: 9, Kind: payslip.Monthly, CCNL: "COMMERCIO"}
	b := payslip.Payslip{Year: 2026, Month: 10, Kind: payslip.Monthly, CCNL: "METALMECCANICO"}
	got := Compare(i18n.EN, []payslip.Payslip{a, b})
	if len(got[1].Changes) != 1 || got[1].Changes[0].Code != "ccnl" || len(got[1].Changes[0].Why) != 1 {
		t.Errorf("changes = %+v", got[1].Changes)
	}
}

func TestDedupe(t *testing.T) {
	oct := read(t, synth.Example())
	nov := synth.Example()
	nov.Month = time.November
	slips := []payslip.Payslip{oct, read(t, nov), oct}
	got, dropped := Dedupe(slips)
	if dropped != 1 || len(got) != 2 || got[1].Month != 11 {
		t.Errorf("Dedupe = %d payslips, %d dropped", len(got), dropped)
	}
	if len(slips) != 3 {
		t.Error("Dedupe changed its input")
	}
}

func TestAdditionalMonthWithoutMonth(t *testing.T) {
	fourteenth := payslip.Payslip{Year: 2025, Kind: payslip.Fourteenth, Net: payslip.Amount{Cents: 100, Status: payslip.Verified}}
	if got := Label(i18n.IT, fourteenth); got != "Quattordicesima 2025" {
		t.Errorf("label = %q", got)
	}
	june := payslip.Payslip{Year: 2025, Month: 6, Kind: payslip.Monthly}
	august := payslip.Payslip{Year: 2025, Month: 8, Kind: payslip.Monthly}
	slips := []payslip.Payslip{august, fourteenth, june}
	Sort(slips)
	if slips[0].Month != 6 || slips[1].Kind != payslip.Fourteenth || slips[2].Month != 8 {
		t.Errorf("order = %+v", slips)
	}
	if kept, dropped := Dedupe([]payslip.Payslip{fourteenth, fourteenth}); len(kept) != 1 || dropped != 1 {
		t.Errorf("the same fourteenth twice: kept %d, dropped %d", len(kept), dropped)
	}
}

func TestPercentFollowsTheLanguage(t *testing.T) {
	for _, c := range []struct {
		l      i18n.Lang
		v      float64
		d      int
		signed bool
		want   string
	}{
		{i18n.IT, -30.26, 1, true, "-30,3"}, {i18n.IT, 9.19, 2, false, "9,19"},
		{i18n.EN, 2.5, 1, true, "+2.5"}, {i18n.EN, 12.5, 1, false, "12.5"},
	} {
		if got := percent(c.l, c.v, c.d, c.signed); got != c.want {
			t.Errorf("percent(%s, %v) = %q, want %q", c.l, c.v, got, c.want)
		}
	}
}

// TestPreviousOfSameEmployer compares a month with the latest one of the same
// person and employer, a payslip without codes with any, and nothing across
// people (ADR 0010).
func TestPreviousOfSameEmployer(t *testing.T) {
	slip := func(month int, person, employer string) payslip.Payslip {
		return payslip.Payslip{Year: 2026, Month: month, Kind: payslip.Monthly,
			Person: payslip.Party{ID: person}, Employer: payslip.Party{ID: employer}}
	}
	slips := []payslip.Payslip{slip(1, "A", "X"), slip(2, "A", "Y"), slip(3, "A", "X"), slip(4, "B", "X"), slip(5, "", "")}
	want := []int{-1, -1, 0, -1, 3}
	for i, w := range want {
		got := Previous(slips, i)
		if (w < 0) != (got == nil) || (got != nil && got != &slips[w]) {
			t.Errorf("Previous(%d) = %v, want %d", i, got, w)
		}
	}
}
