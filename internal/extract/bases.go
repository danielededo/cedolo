package extract

import (
	"fmt"
	"math"
	"strings"

	"github.com/danielededo/cedolo/internal/money"
	"github.com/danielededo/cedolo/internal/payslip"
)

// basesByArithmetic reads the INPS taxable base, the IRPEF taxable base and
// the month's TFR that no known label gave, from the relations among the
// amounts printed on the page (ADR 0011, second step). With c the employee's
// INPS contribution and b the INPS base:
//
//   - R1: c is one of the known INPS rates of b;
//   - R2: the TFR is b divided by 13.5, as is or net of the 0.5% contribution,
//     or the TFR base instead of b when its label gives it;
//   - R3: the IRPEF base is exactly b less c.
//
// An amount is read as the INPS base when R1 and at least one of R2 or R3
// hold, never on R1 alone; the TFR and the IRPEF base only from an INPS base
// that R1 confirms. When amounts of different value fit the same role, the
// role stays unknown. Values found by their labels are never replaced.
func basesByArithmetic(p *payslip.Payslip, all [][]line) {
	c, ok := inpsContribution(p)
	if !ok || c <= 0 {
		return
	}
	cands := pageAmounts(p, all)
	matching := func(holds func(int64) bool) []number {
		var out []number
		for _, n := range cands {
			if holds(n.cents) {
				out = append(out, n)
			}
		}
		return out
	}
	var read []string

	// A TFR base read by its label is the base of the TFR, as validate
	// takes it; the TFR then says nothing about the INPS base.
	tfrFits := func(b int64) bool {
		if p.TFRBase.Found() {
			return false
		}
		if p.TFR.Found() {
			if _, ok := tfrHolds(p.TFR.Cents, b); ok {
				return true
			}
		}
		return len(matching(func(t int64) bool { _, ok := tfrHolds(t, b); return ok })) > 0
	}
	irpefFits := func(b int64) bool {
		return p.IRPEFBase.Found() && p.IRPEFBase.Cents == b-c || len(matching(func(i int64) bool { return i == b-c })) > 0
	}
	if !p.INPSBase.Found() {
		bs := matching(func(b int64) bool {
			_, ok := rateHolds(c, b)
			return ok && (tfrFits(b) || irpefFits(b))
		})
		if b, ok := single(bs); ok {
			setRead(&p.INPSBase, b)
			rate, _ := rateHolds(c, b.cents)
			var why []string
			if tfrFits(b.cents) {
				why = append(why, "the TFR is it divided by 13.5")
			}
			if irpefFits(b.cents) {
				why = append(why, "the IRPEF base is it less INPS")
			}
			read = append(read, fmt.Sprintf("INPS base %s: INPS %s is %s%% of it, and %s",
				money.Format(b.cents), money.Format(c), money.Format(rate), strings.Join(why, " and ")))
		}
	}

	if !p.INPSBase.Found() {
		record(p, read)
		return
	}
	b := p.INPSBase.Cents
	if _, ok := rateHolds(c, b); !ok {
		record(p, read)
		return
	}
	if !p.TFR.Found() {
		on, name := b, "INPS base"
		if p.TFRBase.Found() {
			on, name = p.TFRBase.Cents, "TFR base"
		}
		if t, ok := single(matching(func(t int64) bool { _, ok := tfrHolds(t, on); return ok })); ok {
			setRead(&p.TFR, t)
			how := "divided by 13.5"
			if net, _ := tfrHolds(t.cents, on); net {
				how += ", net of the 0.5% contribution"
			}
			read = append(read, fmt.Sprintf("TFR %s: the %s %s %s", money.Format(t.cents), name, money.Format(on), how))
		}
	}
	if !p.IRPEFBase.Found() {
		if i, ok := single(matching(func(i int64) bool { return i == b-c })); ok {
			setRead(&p.IRPEFBase, i)
			read = append(read, fmt.Sprintf("IRPEF base %s: the INPS base %s less INPS %s",
				money.Format(i.cents), money.Format(b), money.Format(c)))
		}
	}
	record(p, read)
}

// single returns the first of the amounts when they all have the same
// value, as when a year-to-date row repeats the month's.
func single(ns []number) (number, bool) {
	if len(ns) == 0 {
		return number{}, false
	}
	for _, n := range ns[1:] {
		if n.cents != ns[0].cents {
			return number{}, false
		}
	}
	return ns[0], true
}

func setRead(a *payslip.Amount, n number) {
	*a = payslip.Amount{Cents: n.cents, Status: payslip.Extracted, Source: n.source}
}

func record(p *payslip.Payslip, read []string) {
	if len(read) == 0 {
		return
	}
	p.Checks = append(p.Checks, payslip.Check{Name: "bases_by_arithmetic", OK: true,
		Detail: "read without a known label: " + strings.Join(read, "; ")})
}

// pageAmounts lists, in reading order, the positive amounts printed on the
// payslip's pages, except those of the item rows and those already read as
// a total, a base or a box.
func pageAmounts(p *payslip.Payslip, all [][]line) []number {
	var used []*payslip.Source
	for _, a := range []payslip.Amount{p.Gross, p.Deductions, p.Net, p.INPSBase, p.IRPEFBase, p.TFR, p.TFRBase,
		p.SocialContributions, p.Taxes, p.INPS, p.RoundingPrevious, p.RoundingCurrent} {
		if a.Found() && a.Source != nil {
			used = append(used, a.Source)
		}
	}
	var out []number
	for _, pg := range all {
		for _, l := range pg {
			for _, t := range l.toks {
				if !money.IsAmount(t.Text) {
					continue
				}
				v, err := money.Parse(t.Text)
				if err != nil || v <= 0 {
					continue
				}
				src := sourceOf(t, l.page)
				if onItemRow(p.Items, src) || isUsed(used, src) {
					continue
				}
				out = append(out, number{cents: v, source: &src})
			}
		}
	}
	return out
}

func onItemRow(items []payslip.Item, s payslip.Source) bool {
	for _, it := range items {
		if it.Source.Page == s.Page && math.Abs(it.Source.Y-s.Y) < 1 {
			return true
		}
	}
	return false
}

func isUsed(used []*payslip.Source, s payslip.Source) bool {
	for _, u := range used {
		if u.Page == s.Page && math.Abs(u.X-s.X) < 0.5 && math.Abs(u.Y-s.Y) < 0.5 {
			return true
		}
	}
	return false
}
