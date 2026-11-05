package synth_test

import (
	"testing"

	"github.com/danielededo/cedolo/internal/extract"
	"github.com/danielededo/cedolo/internal/payslip"
	"github.com/danielededo/cedolo/internal/synth"
)

// TestYearReadsBack reads every payslip of the example year as the web app
// would, and finds its totals, checked, and its worker.
func TestYearReadsBack(t *testing.T) {
	year := synth.Year(2025)
	if len(year) != 12 {
		t.Fatalf("%d payslips", len(year))
	}
	nets := map[int64]bool{}
	for _, p := range year {
		got := extract.Document(synth.Layout(p, synth.StyleClassic))
		if len(got) != 1 {
			t.Fatalf("%s %d: %d payslips read", p.Month, p.Year, len(got))
		}
		r := got[0]
		gross, deductions, net := p.Totals()
		if r.Gross.Cents != gross || r.Deductions.Cents != deductions || r.Net.Cents != net || r.Net.Status != payslip.Verified {
			t.Errorf("%s: read %d/%d/%d (%s), want %d/%d/%d", p.Month, r.Gross.Cents, r.Deductions.Cents, r.Net.Cents, r.Net.Status, gross, deductions, net)
		}
		if r.Person.Name != "Anna Bianchi" || int(p.Month) != r.Month || r.Year != 2025 {
			t.Errorf("%s: person %q, period %d/%d", p.Month, r.Person.Name, r.Month, r.Year)
		}
		nets[net] = true
	}
	if len(nets) < 6 {
		t.Errorf("only %d different net pays: the example should move", len(nets))
	}
}
