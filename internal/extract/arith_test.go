package extract

import (
	"testing"

	"github.com/danielededo/cedolo/internal/payslip"
)

func item(desc string, cents int64, side payslip.Side, key string, y float64) payslip.Item {
	return payslip.Item{Description: desc, Cents: cents, Side: side, Key: key, Source: payslip.Source{Page: 1, Y: y}}
}

// pay is a table of three items, read as earnings 2.100,00 and deductions
// 300,00, followed by the rows given.
func pay(rows ...payslip.Item) payslip.Payslip {
	p := payslip.Payslip{Items: []payslip.Item{
		item("PAGA BASE", 200000, payslip.Earning, "earnings.base_salary", 10),
		item("INDENNITA", 10000, payslip.Earning, "earnings.allowance", 20),
		item("RITENUTE", 30000, payslip.Deduction, "tax.irpef", 30),
	}}
	p.Items = append(p.Items, rows...)
	for _, a := range []*payslip.Amount{&p.Gross, &p.Deductions, &p.Net, &p.SocialContributions, &p.Taxes, &p.RoundingPrevious, &p.RoundingCurrent} {
		*a = payslip.Amount{Status: payslip.Missing}
	}
	return p
}

func TestTotalsByArithmetic(t *testing.T) {
	p := pay(
		item("XUVO KAPE", 210000, payslip.Earning, "earnings.other", 40),
		item("XUVO RIDA", 30000, payslip.Deduction, "deductions.other", 50),
		item("LEMI BOZA", 180000, payslip.Deduction, "deductions.other", 60),
		item("FERIE 12,00", 800, payslip.Earning, "earnings.leave", 70),
	)
	totalsByArithmetic(&p)
	if p.Gross.Cents != 210000 || p.Deductions.Cents != 30000 || p.Net.Cents != 180000 {
		t.Fatalf("totals = %d, %d, %d", p.Gross.Cents, p.Deductions.Cents, p.Net.Cents)
	}
	if len(p.Items) != 3 {
		t.Errorf("the table should end at the totals, kept %d items", len(p.Items))
	}
	if len(p.Checks) != 1 || p.Checks[0].Name != "totals_by_arithmetic" {
		t.Errorf("checks = %+v", p.Checks)
	}
	validate(&p)
	for name, a := range map[string]payslip.Amount{"gross": p.Gross, "deductions": p.Deductions, "net": p.Net} {
		if a.Status != payslip.Verified {
			t.Errorf("%s is %s, not verified by the checks", name, a.Status)
		}
	}
}

// Totals printed on one line, the net pay as the row's amount.
func TestTotalsByArithmeticOnOneLine(t *testing.T) {
	p := pay(item("2.100,00 300,00", 180000, payslip.Earning, "earnings.other", 40))
	totalsByArithmetic(&p)
	if p.Gross.Cents != 210000 || p.Deductions.Cents != 30000 || p.Net.Cents != 180000 || len(p.Items) != 3 {
		t.Fatalf("totals = %d, %d, %d with %d items", p.Gross.Cents, p.Deductions.Cents, p.Net.Cents, len(p.Items))
	}
}

// The net pay found by its label is the second relation.
func TestTotalsByArithmeticWithLabelledNet(t *testing.T) {
	p := pay(item("XUVO KAPE", 210000, payslip.Earning, "earnings.other", 40))
	p.Net = payslip.Amount{Cents: 180000, Status: payslip.Extracted}
	totalsByArithmetic(&p)
	if p.Gross.Cents != 210000 || len(p.Items) != 3 {
		t.Fatalf("gross = %d with %d items", p.Gross.Cents, len(p.Items))
	}
}

// One sum alone is never enough: the row stays an item.
func TestTotalsByArithmeticNeedsASecondRelation(t *testing.T) {
	for name, p := range map[string]payslip.Payslip{
		"no net": pay(item("XUVO KAPE", 210000, payslip.Earning, "earnings.other", 40)),
		"a net that does not add up": pay(
			item("XUVO KAPE", 210000, payslip.Earning, "earnings.other", 40),
			item("LEMI BOZA", 170000, payslip.Deduction, "deductions.other", 50)),
		"a labelled net that does not add up": func() payslip.Payslip {
			p := pay(item("XUVO KAPE", 210000, payslip.Earning, "earnings.other", 40))
			p.Net = payslip.Amount{Cents: 175000, Status: payslip.Extracted}
			return p
		}(),
	} {
		before := len(p.Items)
		totalsByArithmetic(&p)
		if p.Gross.Found() || len(p.Items) != before || len(p.Checks) != 0 {
			t.Errorf("%s: read gross %d, %d items left", name, p.Gross.Cents, len(p.Items))
		}
	}
}

// A recognized item never becomes a total, and an unrecognized one that is
// not the sum above it stays an item.
func TestTotalsByArithmeticKeepsItems(t *testing.T) {
	p := pay(
		item("PREMIO XUVO", 50000, payslip.Earning, "earnings.other", 40),
		item("TOTALE", 260000, payslip.Earning, "earnings.base_salary", 50),
	)
	p.Net = payslip.Amount{Cents: 230000, Status: payslip.Extracted}
	totalsByArithmetic(&p)
	if p.Gross.Found() || len(p.Items) != 5 {
		t.Errorf("read gross %d, %d items left", p.Gross.Cents, len(p.Items))
	}
}

// Totals found by their labels are kept: the arithmetic only fills those
// missing.
func TestTotalsByArithmeticKeepsLabels(t *testing.T) {
	p := pay(
		item("XUVO KAPE", 210000, payslip.Earning, "earnings.other", 40),
		item("LEMI BOZA", 180000, payslip.Deduction, "deductions.other", 50),
	)
	p.Gross = payslip.Amount{Cents: 210000, Status: payslip.Extracted, Source: &payslip.Source{Page: 1, Y: 99}}
	totalsByArithmetic(&p)
	if p.Gross.Source.Y != 99 || p.Net.Cents != 180000 {
		t.Errorf("gross from y %v, net %d", p.Gross.Source.Y, p.Net.Cents)
	}
}
