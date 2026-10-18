package extract

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/danielededo/cedolo/internal/money"
	"github.com/danielededo/cedolo/internal/payslip"
)

// maxReadings bounds the distinct values tried for each total, so that the
// search stays small.
const maxReadings = 3

// total is one of the totals the cross-checks tie together, with the amounts
// it could be read as, most preferred first.
type total struct {
	name     string
	field    func(*payslip.Payslip) *payslip.Amount
	readings []payslip.Amount
}

// resolve validates the payslip under the preferred reading of each total
// and, when a cross-check fails, under the other readings: the first
// combination, closest to the preferred readings, under which every check
// passes wins (ADR 0009). A combination that only passes more checks is not
// enough, since a single sum matched by chance is likelier than all of them.
// Without a winner, the payslip is left as validate leaves it.
func resolve(p *payslip.Payslip, totals []total) {
	// The first choice is the value read today, which may be missing.
	choices := make([][]payslip.Amount, len(totals))
	for k, t := range totals {
		choices[k] = []payslip.Amount{*t.field(p)}
		for _, r := range t.readings {
			if len(choices[k]) < maxReadings && !slices.ContainsFunc(choices[k], func(a payslip.Amount) bool { return a.Found() && a.Cents == r.Cents }) {
				choices[k] = append(choices[k], r)
			}
		}
	}
	try := func(pick []int) (q payslip.Payslip, passed, applied int) {
		q = *p
		q.Checks = slices.Clone(p.Checks)
		for k, t := range totals {
			if pick[k] > 0 {
				a := choices[k][pick[k]]
				a.Alternative = true
				*t.field(&q) = a
			}
		}
		validate(&q)
		for _, c := range q.Checks[len(p.Checks):] {
			if !c.Advisory {
				applied++
				if c.OK {
					passed++
				}
			}
		}
		return q, passed, applied
	}

	best, passed, applied := try(make([]int, len(totals)))
	if passed < applied {
		for _, pick := range combinations(choices)[1:] {
			q, n, _ := try(pick)
			if n < applied {
				continue
			}
			best = q
			for k, i := range pick {
				if i == 0 {
					continue
				}
				detail := fmt.Sprintf("%s read as %s rather than %s, the reading under which the totals add up",
					totals[k].name, money.Format(choices[k][i].Cents), money.Format(choices[k][0].Cents))
				if !choices[k][0].Found() {
					detail = fmt.Sprintf("%s read as %s, farther from its label than usual or from boxes of its own, the reading under which the totals add up",
						totals[k].name, money.Format(choices[k][i].Cents))
				}
				best.Checks = append(best.Checks, payslip.Check{Name: "total_reading", OK: true, Detail: detail})
			}
			break
		}
	}
	*p = best
}

// combinations lists every choice of one reading per total, as indexes,
// ordered by how far they stray from the preferred readings: fewer and
// nearer alternatives first. The first is the preferred reading of each.
func combinations(choices [][]payslip.Amount) [][]int {
	out := [][]int{make([]int, len(choices))}
	for k, c := range choices {
		n := len(out)
		for i := 1; i < len(c); i++ {
			for _, pick := range out[:n] {
				next := slices.Clone(pick)
				next[k] = i
				out = append(out, next)
			}
		}
	}
	strays := func(pick []int) (n, sum int) {
		for _, i := range pick {
			if i > 0 {
				n++
				sum += i
			}
		}
		return n, sum
	}
	slices.SortStableFunc(out, func(a, b []int) int {
		an, as := strays(a)
		bn, bs := strays(b)
		return cmp.Or(cmp.Compare(an, bn), cmp.Compare(as, bs))
	})
	return out
}
