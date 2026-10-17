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
		{"IRPEF A CREDITO -MOD.730-", payslip.Earning, "tax.return"},
		{"ADD.REG.CREDITO -MOD.730-", payslip.Earning, "tax.return"},
		{"IRPEF A DEBITO MOD. 730 1A RATA", payslip.Deduction, "tax.return"},
		{"Rimborso assistenza fiscale", payslip.Earning, "tax.return"},
		{"FONDO PENSIONE FON.TE C/DIP", payslip.Deduction, "deductions.pension_fund"},
		{"TRATTENUTA SINDACALE", payslip.Deduction, "deductions.union"},
		{"ARROTONDAMENTO MESE CORRENTE", payslip.Earning, "rounding"},
		{"Arrot. mese precedente", payslip.Deduction, "rounding"},
		{"RATEO TREDICESIMA", payslip.Earning, "earnings.thirteenth"},
		{"BUONI PASTO", payslip.Earning, "meal_vouchers"},
		{"REC. ASSOGGETTAMENTO PASTI", payslip.Deduction, "meal_vouchers"},
		{"NOTA SPESE OTTOBRE", payslip.Earning, "earnings.reimbursement"},
		{"ESON.CONTR.IVS L.234/21", payslip.Earning, "earnings.contribution_relief"},
		{"LAVORO ORDIN.(mens.)", payslip.Earning, "earnings.base_salary"},
		{"RATA ADD.COMUN.A.P.", payslip.Deduction, "tax.surcharge.municipal"},
		{"ACCONTO ADD.COMUNALE", payslip.Deduction, "tax.surcharge.municipal"},
		{"RATA ADD.REG. A.P.", payslip.Deduction, "tax.surcharge.regional"},
		{"STRAORDINARIO FESTIVO", payslip.Earning, "earnings.overtime"},
		{"GETTONE PRESENZA", payslip.Earning, OtherEarning},
		{"Storno T.F.R. a F.P.C.", payslip.Info, "tfr.accrued"},
		{"Contr. FPC dipendente", payslip.Info, "pension_fund.contribution"},
		{"CONTRIBUZIONE TFR", payslip.Info, OtherInfo},
		{"ESONERO CTR - TFR PREV.C.", payslip.Info, OtherInfo},
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
	for _, k := range []string{OtherEarning, OtherDeduction, OtherInfo, "tax.irpef"} {
		if _, ok := Lookup(k); !ok {
			t.Errorf("Lookup(%q) failed", k)
		}
	}
}

func TestAdditionalMonthItems(t *testing.T) {
	for desc, key := range map[string]string{
		"13ª MENSILITÀ": "earnings.thirteenth", "XIII MENSILITA'": "earnings.thirteenth", "13^ MENS.": "earnings.thirteenth",
		"RATEI 13MA": "earnings.thirteenth", "14ª MENSILITÀ": "earnings.fourteenth", "XIV MENSILITA'": "earnings.fourteenth",
		"14esima": "earnings.fourteenth",
	} {
		if got := Classify(desc, payslip.Earning); got != key {
			t.Errorf("%q = %s, want %s", desc, got, key)
		}
	}
}
