package extract

import (
	"testing"

	"github.com/danielededo/cedolo/internal/payslip"
)

func TestRateHolds(t *testing.T) {
	if rate, ok := rateHolds(20167, 219450); !ok || rate != 919 {
		t.Errorf("rateHolds(201,67, 2.194,50) = %d, %v; want 919, true", rate, ok)
	}
	if _, ok := rateHolds(20000, 219450); ok {
		t.Error("200,00 is no known rate of 2.194,50")
	}
}

func TestTFRHolds(t *testing.T) {
	if net, ok := tfrHolds(16256, 219450); !ok || net {
		t.Errorf("tfrHolds(162,56, 2.194,50) = %v, %v; want gross, true", net, ok)
	}
	// 162,56 less the 0.5% contribution on the base.
	if net, ok := tfrHolds(roundDiv(219450*373, 5400), 219450); !ok || !net {
		t.Errorf("the net TFR = %v, %v; want net, true", net, ok)
	}
	if _, ok := tfrHolds(16000, 219450); ok {
		t.Error("160,00 is not 2.194,50 divided by 13.5")
	}
}

func TestINPSContribution(t *testing.T) {
	missing := payslip.Amount{Status: payslip.Missing}
	box := func(c int64) payslip.Amount { return payslip.Amount{Cents: c, Status: payslip.Extracted} }
	items := []payslip.Item{{Key: "deductions.inps", Side: payslip.Deduction, Cents: 20167}}
	for name, tc := range map[string]struct {
		p    payslip.Payslip
		want int64
		ok   bool
	}{
		"items first":        {payslip.Payslip{Items: items, INPS: box(1), SocialContributions: box(2)}, 20167, true},
		"then the INPS box":  {payslip.Payslip{INPS: box(1), SocialContributions: box(2)}, 1, true},
		"then contributions": {payslip.Payslip{INPS: missing, SocialContributions: box(2)}, 2, true},
		"none":               {payslip.Payslip{INPS: missing, SocialContributions: missing}, 0, false},
	} {
		if got, ok := inpsContribution(&tc.p); got != tc.want || ok != tc.ok {
			t.Errorf("%s: %d, %v; want %d, %v", name, got, ok, tc.want, tc.ok)
		}
	}
}
