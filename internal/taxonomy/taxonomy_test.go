package taxonomy

import (
	"testing"

	"github.com/danielededo/cedolo/internal/payslip"
)

func TestNormalize(t *testing.T) {
	for in, want := range map[string]string{
		"Add.le Reg.":            "add le reg",
		"INDENNITÀ DI FUNZIONE":  "indennita di funzione",
		"  CTR. F.P.L.D. 9,19% ": "ctr f p l d 9 19",
	} {
		if got := Normalize(in); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestClassify(t *testing.T) {
	cases := []struct {
		desc string
		side payslip.Side
		want string
	}{
		{"RETRIBUZIONE ORDINARIA", payslip.Earning, "earnings.base_salary"},
		{"Paga base", payslip.Earning, "earnings.base_salary"},
		{"STRAORD. FERIALE 25%", payslip.Earning, "earnings.overtime"},
		{"IND. FUNZIONE", payslip.Earning, "earnings.allowance"},
		{"Indennità di funzione", payslip.Earning, "earnings.allowance"},
		{"CONTRIBUTO IVS", payslip.Deduction, "deductions.inps"},
		{"CTR. F.P.L.D. 9,19%", payslip.Deduction, "deductions.inps"},
		{"Contributi INPS c/dipendente", payslip.Deduction, "deductions.inps"},
		{"RITENUTA IRPEF", payslip.Deduction, "tax.irpef"},
		{"IRPEF NETTA", payslip.Deduction, "tax.irpef"},
		{"Add.le regionale IRPEF", payslip.Deduction, "tax.surcharge.regional"},
		{"ADD. COM. RATA", payslip.Deduction, "tax.surcharge.municipal"},
		{"Trattamento integrativo L.21/2020", payslip.Earning, "tax.credit"},
		{"FONDO PENSIONE FON.TE C/DIP", payslip.Deduction, "deductions.pension_fund"},
		{"TRATTENUTA SINDACALE", payslip.Deduction, "deductions.union"},
		{"ARROTONDAMENTO MESE CORRENTE", payslip.Earning, "rounding"},
		{"Arrot. mese precedente", payslip.Deduction, "rounding"},
		{"RATEO TREDICESIMA", payslip.Earning, "earnings.thirteenth"},
		{"BUONI PASTO", payslip.Earning, OtherEarning},
		{"PIGNORAMENTO", payslip.Deduction, OtherDeduction},
		// An earnings keyword printed as a deduction is not forced into the
		// earnings category.
		{"PAGA BASE", payslip.Deduction, OtherDeduction},
	}
	for _, c := range cases {
		if got := Classify(c.desc, c.side); got != c.want {
			t.Errorf("Classify(%q, %s) = %q, want %q", c.desc, c.side, got, c.want)
		}
	}
}

func TestEntriesAreWellFormed(t *testing.T) {
	seen := map[string]bool{}
	for _, e := range Entries {
		if seen[e.Key] {
			t.Errorf("duplicate key %s", e.Key)
		}
		seen[e.Key] = true
		if e.LabelEN == "" || e.LabelIT == "" || len(e.Keywords) == 0 {
			t.Errorf("incomplete entry %s", e.Key)
		}
		for _, kw := range e.Keywords {
			if Normalize(kw) != kw {
				t.Errorf("%s: keyword %q is not normalized", e.Key, kw)
			}
		}
	}
	for _, k := range []string{OtherEarning, OtherDeduction, "tax.irpef"} {
		if _, ok := Lookup(k); !ok {
			t.Errorf("Lookup(%q) failed", k)
		}
	}
}
