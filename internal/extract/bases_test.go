package extract

import (
	"testing"

	"github.com/danielededo/cedolo/internal/layout"
	"github.com/danielededo/cedolo/internal/payslip"
)

// The synthetic example's figures: INPS 201,67 is 9,19% of the INPS base
// 2.194,50, the IRPEF base 1.992,83 is the base less INPS, and the TFR
// 162,56 is the base divided by 13.5.
const (
	exINPS  = 20167
	exBase  = 219450
	exIRPEF = 199283
	exTFR   = 16256
)

// basesSlip is a payslip whose only item is the INPS contribution, printed
// on the first row of page 1, with no total or base read.
func basesSlip() payslip.Payslip {
	p := payslip.Payslip{Items: []payslip.Item{{Description: "CONTRIBUTO IVS", Key: "deductions.inps",
		Side: payslip.Deduction, Cents: exINPS, Source: payslip.Source{Page: 1, X: 500, Y: 40}}}}
	for _, a := range []*payslip.Amount{&p.Gross, &p.Deductions, &p.Net, &p.INPSBase, &p.IRPEFBase, &p.TFR, &p.TFRBase,
		&p.SocialContributions, &p.Taxes, &p.INPS, &p.RoundingPrevious, &p.RoundingCurrent} {
		*a = payslip.Amount{Status: payslip.Missing}
	}
	return p
}

// basesPage prints the INPS item's row, then the rows given, under
// invented labels.
func basesPage(rows ...[]string) layout.Page {
	return page(append([][]string{{"40:9001 CONTRIBUTO IVS", "540>201,67"}}, rows...)...)
}

func linesOf(pages ...layout.Page) [][]line {
	var out [][]line
	for _, pg := range pages {
		out = append(out, buildLines(pg))
	}
	return out
}

func labelled(cents int64) payslip.Amount {
	return payslip.Amount{Cents: cents, Status: payslip.Extracted, Source: &payslip.Source{Page: 1, X: 1, Y: 999}}
}

func checksNamed(p payslip.Payslip, name string) int {
	n := 0
	for _, c := range p.Checks {
		if c.Name == name {
			n++
		}
	}
	return n
}

func TestBasesAllHidden(t *testing.T) {
	p := basesSlip()
	basesByArithmetic(&p, linesOf(basesPage(
		[]string{"40:BIKO MAKA", "250>2.194,50", "300:LUVE PODA", "450>1.992,83"},
		[]string{"40:ZEFI RUMA", "250>162,56"},
	)))
	if p.INPSBase.Cents != exBase || p.IRPEFBase.Cents != exIRPEF || p.TFR.Cents != exTFR {
		t.Fatalf("INPS base %d, IRPEF base %d, TFR %d", p.INPSBase.Cents, p.IRPEFBase.Cents, p.TFR.Cents)
	}
	if n := checksNamed(p, "bases_by_arithmetic"); n != 1 {
		t.Errorf("%d bases_by_arithmetic checks, want 1: %+v", n, p.Checks)
	}
	validate(&p)
	for name, a := range map[string]payslip.Amount{"INPS base": p.INPSBase, "IRPEF base": p.IRPEFBase, "TFR": p.TFR} {
		if a.Status != payslip.Verified {
			t.Errorf("%s is %s, not verified", name, a.Status)
		}
	}
}

func TestBasesINPSHidden(t *testing.T) {
	p := basesSlip()
	p.IRPEFBase, p.TFR = labelled(exIRPEF), labelled(exTFR)
	basesByArithmetic(&p, linesOf(basesPage([]string{"40:BIKO MAKA", "250>2.194,50"})))
	if p.INPSBase.Cents != exBase || p.INPSBase.Source == nil {
		t.Errorf("INPS base %d", p.INPSBase.Cents)
	}
}

func TestBasesTFRHidden(t *testing.T) {
	p := basesSlip()
	p.INPSBase, p.IRPEFBase = labelled(exBase), labelled(exIRPEF)
	basesByArithmetic(&p, linesOf(basesPage([]string{"40:ZEFI RUMA", "250>162,56"})))
	if p.TFR.Cents != exTFR {
		t.Errorf("TFR %d", p.TFR.Cents)
	}
}

func TestBasesIRPEFHidden(t *testing.T) {
	p := basesSlip()
	p.INPSBase, p.TFR = labelled(exBase), labelled(exTFR)
	basesByArithmetic(&p, linesOf(basesPage([]string{"40:LUVE PODA", "250>1.992,83"})))
	if p.IRPEFBase.Cents != exIRPEF {
		t.Errorf("IRPEF base %d", p.IRPEFBase.Cents)
	}
}

// R1 alone is never enough.
func TestBasesRateAlone(t *testing.T) {
	p := basesSlip()
	basesByArithmetic(&p, linesOf(basesPage([]string{"40:BIKO MAKA", "250>2.194,50", "450>1.500,00"})))
	if p.INPSBase.Found() || len(p.Checks) != 0 {
		t.Errorf("INPS base %d read from the rate alone", p.INPSBase.Cents)
	}
}

func TestBasesAmbiguous(t *testing.T) {
	p := basesSlip()
	p.INPSBase = labelled(exBase)
	basesByArithmetic(&p, linesOf(basesPage([]string{"40:ZEFI RUMA", "250>162,56", "450>162,57"})))
	if p.TFR.Found() {
		t.Errorf("TFR %d read from two different amounts", p.TFR.Cents)
	}
}

// January's year-to-date row repeats the month's amount.
func TestBasesSameAmountTwice(t *testing.T) {
	p := basesSlip()
	p.INPSBase = labelled(exBase)
	basesByArithmetic(&p, linesOf(basesPage(
		[]string{"40:LUVE PODA", "250>1.992,83"},
		[]string{"40:PROGR. LUVE PODA", "250>1.992,83"},
	)))
	if p.IRPEFBase.Cents != exIRPEF || p.IRPEFBase.Source == nil || p.IRPEFBase.Source.Y != 54 {
		t.Errorf("IRPEF base %d from %+v, want the first", p.IRPEFBase.Cents, p.IRPEFBase.Source)
	}
}

// A labelled base that is no known rate of the contribution confirms
// nothing.
func TestBasesNeedConfirmedBase(t *testing.T) {
	p := basesSlip()
	p.INPSBase = labelled(200000)
	basesByArithmetic(&p, linesOf(basesPage([]string{"40:ZEFI RUMA", "250>148,15", "450>1.798,33"})))
	if p.TFR.Found() || p.IRPEFBase.Found() {
		t.Errorf("TFR %d, IRPEF base %d read from an unconfirmed base", p.TFR.Cents, p.IRPEFBase.Cents)
	}
}

func TestBasesKeepLabels(t *testing.T) {
	p := basesSlip()
	p.INPSBase, p.IRPEFBase, p.TFR = labelled(exBase), labelled(190000), labelled(16000)
	basesByArithmetic(&p, linesOf(basesPage([]string{"40:LUVE PODA", "250>1.992,83", "450>162,56"})))
	if p.IRPEFBase.Cents != 190000 || p.TFR.Cents != 16000 || len(p.Checks) != 0 {
		t.Errorf("labels replaced: IRPEF base %d, TFR %d", p.IRPEFBase.Cents, p.TFR.Cents)
	}
}

func TestBasesNoContribution(t *testing.T) {
	p := basesSlip()
	p.Items = nil
	basesByArithmetic(&p, linesOf(basesPage(
		[]string{"40:BIKO MAKA", "250>2.194,50", "450>1.992,83"},
		[]string{"40:ZEFI RUMA", "250>162,56"},
	)))
	if p.INPSBase.Found() || p.IRPEFBase.Found() || p.TFR.Found() {
		t.Error("read without a contribution")
	}
}

func TestBasesSkipNegativeAndZero(t *testing.T) {
	p := basesSlip()
	got := pageAmounts(&p, linesOf(basesPage([]string{"40:BIKO", "250>201,67-", "350>0,00", "450>12,00"})))
	if len(got) != 1 || got[0].cents != 1200 {
		t.Errorf("candidates %+v, want only 12,00", got)
	}
}

func TestBasesSkipsItemRows(t *testing.T) {
	p := basesSlip()
	p.INPSBase = labelled(exBase)
	pg := basesPage([]string{"40:9100 RITENUTA", "540>1.992,83"})
	p.Items = append(p.Items, payslip.Item{Key: "tax.irpef", Side: payslip.Deduction, Cents: exIRPEF,
		Source: payslip.Source{Page: 1, X: 500, Y: 54}})
	basesByArithmetic(&p, linesOf(pg))
	if p.IRPEFBase.Found() {
		t.Error("an item's amount read as the IRPEF base")
	}
}

func TestBasesAcrossPages(t *testing.T) {
	p := basesSlip()
	p.INPSBase = labelled(exBase)
	second := page(
		[]string{"40:9100 RITENUTA", "540>1.992,83"},
		[]string{"40:ZEFI RUMA", "250>162,56"},
	)
	second.Number = 2
	p.Items = append(p.Items, payslip.Item{Key: "tax.irpef", Side: payslip.Deduction, Cents: exIRPEF,
		Source: payslip.Source{Page: 2, X: 500, Y: 40}})
	basesByArithmetic(&p, linesOf(basesPage(), second))
	if p.TFR.Cents != exTFR || p.TFR.Source == nil || p.TFR.Source.Page != 2 {
		t.Errorf("TFR %d from %+v, want page 2", p.TFR.Cents, p.TFR.Source)
	}
	if p.IRPEFBase.Found() {
		t.Error("an item's amount on page 2 read as the IRPEF base")
	}
}

// A TFR base read by its label is the base of the TFR, as validate takes
// it: a candidate that is the INPS base divided by 13.5 is not the TFR,
// nor does it confirm an INPS base.
func TestBasesTFRBaseLabelled(t *testing.T) {
	p := basesSlip()
	p.INPSBase, p.TFRBase = labelled(exBase), labelled(230000)
	basesByArithmetic(&p, linesOf(basesPage([]string{"40:ZEFI RUMA", "250>162,56"})))
	if p.TFR.Found() {
		t.Errorf("TFR %d read against the INPS base, not the labelled TFR base", p.TFR.Cents)
	}

	p = basesSlip()
	p.TFRBase = labelled(230000)
	basesByArithmetic(&p, linesOf(basesPage([]string{"40:BIKO MAKA", "250>2.194,50", "450>162,56"})))
	if p.INPSBase.Found() {
		t.Errorf("INPS base %d confirmed by a TFR that is not on the TFR base", p.INPSBase.Cents)
	}

	p = basesSlip()
	p.INPSBase, p.TFRBase = labelled(exBase), labelled(230000)
	basesByArithmetic(&p, linesOf(basesPage([]string{"40:ZEFI RUMA", "250>170,37"})))
	if p.TFR.Cents != 17037 {
		t.Errorf("TFR %d, want 170,37, the TFR base divided by 13.5", p.TFR.Cents)
	}
}
