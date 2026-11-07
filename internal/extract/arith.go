package extract

import (
	"fmt"
	"slices"
	"strings"

	"github.com/danielededo/cedolo/internal/money"
	"github.com/danielededo/cedolo/internal/payslip"
)

// maxFollowing bounds how many numbers after the gross pay are tried as
// the deductions and the net pay.
const maxFollowing = 4

// number is an amount printed in the item table, in reading order, with
// where it was read when it was a row's amount.
type number struct {
	cents  int64
	source *payslip.Source
}

// totalsByArithmetic recognizes totals that the item table read as items
// because their labels are unknown (ADR 0011, first step). A table ends at
// its totals, found by their labels; when they are worded otherwise, the
// table runs on and their rows become unrecognized items, with whatever
// follows them. Such a row is the gross pay when its amount is, to the cent,
// the sum of the earnings above it, and the payslip's arithmetic confirms it
// with a second relation: the net pay that follows, or the net pay found by
// its label, is the earnings less what is withheld. A single sum is never
// enough. The table then ends at that row, and the totals read from it take
// the place of those no label gave.
func totalsByArithmetic(p *payslip.Payslip) {
	for i, it := range p.Items {
		if !strings.HasSuffix(it.Key, ".other") || i < 2 {
			continue
		}
		above := p.Items[:i]
		var earnings, taxable, deductions, absences int64
		nEarnings := 0
		for _, a := range above {
			switch a.Side {
			case payslip.Earning:
				earnings += a.Cents
				nEarnings++
				if !payslip.OutsideGross(a.Key) {
					taxable += a.Cents
				}
			case payslip.Deduction:
				deductions += a.Cents
				if a.Key == "deductions.absence" {
					absences += a.Cents
				}
			}
		}
		if nEarnings == 0 || earnings <= 0 {
			continue
		}
		// The gross pay as validate accepts it.
		grosses := []int64{earnings, taxable}
		if absences != 0 {
			grosses = append(grosses, earnings-absences, taxable-absences)
		}
		nums := numbers(p.Items[i:], it.Source.Page)
		own := len(rowNumbers(it))
		gi := slices.IndexFunc(nums[:own], func(n number) bool { return slices.Contains(grosses, n.cents) })
		if gi < 0 {
			continue
		}
		gross := nums[gi]

		// Everything withheld: the item deductions, and the contributions
		// and taxes printed in boxes of their own.
		withheld := deductions
		for _, a := range []payslip.Amount{p.SocialContributions, p.Taxes} {
			if a.Found() {
				withheld += a.Cents
			}
		}
		roundings := []int64{0}
		for _, a := range []payslip.Amount{p.RoundingPrevious, p.RoundingCurrent} {
			if a.Found() && a.Cents != 0 {
				roundings = append(roundings, a.Cents, -a.Cents)
			}
		}
		isNet := func(v, withheldAs int64) bool {
			return slices.ContainsFunc(roundings, func(r int64) bool { return earnings-withheldAs+r == v })
		}

		rest := nums[gi+1 : min(gi+1+maxFollowing, len(nums))]
		var ded, net *number
		for k := range rest {
			n := &rest[k]
			switch {
			case ded == nil && net == nil && n.cents > 0 && (n.cents == deductions || n.cents == withheld):
				ded = n
			case net == nil && (isNet(n.cents, deductions) || isNet(n.cents, withheld) || ded != nil && isNet(n.cents, ded.cents)):
				net = n
			}
		}
		// The second relation: a net pay that adds up, read here or by its
		// label.
		labelled := p.Net.Found() && (isNet(p.Net.Cents, deductions) || isNet(p.Net.Cents, withheld) ||
			p.Deductions.Found() && isNet(p.Net.Cents, p.Deductions.Cents))
		if net == nil && !labelled {
			continue
		}

		read := []string{"gross pay " + money.Format(gross.cents)}
		set := func(a *payslip.Amount, n *number, name string) {
			if n == nil || a.Found() {
				return
			}
			*a = payslip.Amount{Cents: n.cents, Status: payslip.Extracted, Source: n.source}
			if name != "" {
				read = append(read, name+" "+money.Format(n.cents))
			}
		}
		if p.Gross.Found() {
			read = read[:0]
		}
		set(&p.Gross, &gross, "")
		set(&p.Deductions, ded, "total deductions")
		set(&p.Net, net, "net pay")
		// The table ends at the totals: what follows them on the page are
		// more totals, taxable bases, TFR and leave, not pay items.
		kept := slices.Clone(p.Items[:i])
		for _, later := range p.Items[i:] {
			if later.Source.Page != it.Source.Page {
				kept = append(kept, later)
			}
		}
		p.Items = kept
		if len(read) > 0 {
			p.Checks = append(p.Checks, payslip.Check{Name: "totals_by_arithmetic", OK: true,
				Detail: fmt.Sprintf("%s read without a known label: the gross pay is the sum of the items above it, and the net pay is the earnings less what is withheld", strings.Join(read, ", "))})
		}
		return
	}
}

// rowNumbers lists the amounts printed in an item's row: those its
// description holds, as when totals share a line, then its own.
func rowNumbers(it payslip.Item) []number {
	var out []number
	for _, w := range strings.Fields(it.Description) {
		if money.IsAmount(w) {
			if v, err := money.Parse(w); err == nil {
				out = append(out, number{cents: v})
			}
		}
	}
	src := it.Source
	return append(out, number{cents: it.Cents, source: &src})
}

// numbers lists, in reading order, the amounts of the items on one page.
// The items of one row share its description, so its amounts are listed
// once.
func numbers(items []payslip.Item, page int) []number {
	var out []number
	var prev *payslip.Item
	for k := range items {
		it := &items[k]
		if it.Source.Page != page {
			continue
		}
		if prev != nil && prev.Description == it.Description && prev.Source.Y == it.Source.Y {
			src := it.Source
			out = append(out, number{cents: it.Cents, source: &src})
		} else {
			out = append(out, rowNumbers(*it)...)
		}
		prev = it
	}
	return out
}
