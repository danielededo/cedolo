package extract

import (
	"fmt"
	"math"
	"strings"

	"github.com/danielededo/cedolo/internal/layout"
	"github.com/danielededo/cedolo/internal/money"
	"github.com/danielededo/cedolo/internal/payslip"
)

// Leave balances are printed in one of three shapes, all handled here:
//
//   - rows:       a header of measures (maturato, goduto, residuo) and one row
//     per kind of leave;
//   - columns:    a header of kinds (ferie, permessi, ex festività) and one row
//     per measure;
//   - inline:     one line per kind with measure and value pairs.

type measure int

const (
	mPrevious measure = iota
	mAccrued
	mUsed
	mRemaining
)

type phrase[T any] struct {
	words string
	value T
}

// Longer phrases come first so that "residuo ap" is a previous balance and not
// a remaining one, and "permessi rol" is matched as a whole.
var leaveKinds = []phrase[payslip.LeaveKind]{
	{"ex festivita", payslip.FormerHolidays}, {"festivita soppresse", payslip.FormerHolidays},
	{"fest soppresse", payslip.FormerHolidays}, {"ex fest", payslip.FormerHolidays},
	{"banca ore", payslip.HoursBank},
	{"permessi rol", payslip.ReducedHours}, {"riduzione orario", payslip.ReducedHours},
	{"permessi", payslip.ReducedHours}, {"rol", payslip.ReducedHours}, {"par", payslip.ReducedHours},
	{"ferie", payslip.Holiday},
}

var leaveMeasures = []phrase[measure]{
	{"residuo anno precedente", mPrevious}, {"residuo ap", mPrevious}, {"res ap", mPrevious},
	{"residuo prec", mPrevious}, {"res prec", mPrevious}, {"anno precedente", mPrevious},
	{"residuo iniziale", mPrevious}, {"riporto", mPrevious},
	{"maturato", mAccrued}, {"maturate", mAccrued}, {"maturati", mAccrued}, {"maturata", mAccrued},
	{"spettanti", mAccrued}, {"spettante", mAccrued},
	{"goduto", mUsed}, {"godute", mUsed}, {"goduti", mUsed}, {"goduta", mUsed},
	{"fruito", mUsed}, {"fruite", mUsed}, {"fruiti", mUsed}, {"usufruito", mUsed},
	{"da godere", mRemaining}, {"residuo", mRemaining}, {"residue", mRemaining}, {"residui", mRemaining},
	{"saldo", mRemaining}, {"rimanenti", mRemaining},
}

// match is a phrase found in a line, with the horizontal extent of its tokens.
type match[T any] struct {
	value      T
	label      string
	x0, x1     float64
	firstWord  int
	afterWord  int // index of the first word after the phrase
	afterToken int // index of the first token after the phrase
}

// scan finds non-overlapping phrases in a line, left to right.
func scan[T any](l line, phrases []phrase[T]) []match[T] {
	var out []match[T]
	for i := 0; i < len(l.words); {
		found := false
		for _, p := range phrases {
			pw := strings.Fields(p.words)
			if i+len(pw) > len(l.words) {
				continue
			}
			ok := true
			for k := range pw {
				if l.words[i+k].text != pw[k] {
					ok = false
					break
				}
			}
			if !ok {
				continue
			}
			first, last := l.words[i].tok, l.words[i+len(pw)-1].tok
			var label []string
			for t := first; t <= last; t++ {
				label = append(label, l.toks[t].Text)
			}
			out = append(out, match[T]{value: p.value, label: strings.Join(label, " "),
				x0: l.toks[first].X, x1: l.toks[last].Right(),
				firstWord: i, afterWord: i + len(pw), afterToken: last + 1})
			i += len(pw)
			found = true
			break
		}
		if !found {
			i++
		}
	}
	return out
}

func distinct[T comparable](ms []match[T]) int {
	seen := map[T]bool{}
	for _, m := range ms {
		seen[m.value] = true
	}
	return len(seen)
}

// leaveUnit reads the unit from a header or label line. A header naming both
// units ("giorni/ore") leaves the unit unknown.
func leaveUnit(l line) string {
	hours, days := l.has("ore"), l.has("giorni", "gg")
	switch {
	case hours && !days:
		return "hours"
	case days && !hours:
		return "days"
	}
	return ""
}

// findLeave extracts leave balances from the lines of one page.
func findLeave(lines []line) []payslip.Leave {
	var out []payslip.Leave
	byKind := map[payslip.LeaveKind]int{}
	add := func(lv payslip.Leave) {
		if lv.Previous == nil && lv.Accrued == nil && lv.Used == nil && lv.Remaining == nil {
			return
		}
		if _, dup := byKind[lv.Kind]; dup {
			return
		}
		byKind[lv.Kind] = len(out)
		out = append(out, lv)
	}
	for i, l := range lines {
		kinds := scan(l, leaveKinds)
		measures := scan(l, leaveMeasures)
		switch {
		case distinct(measures) >= 2 && (len(kinds) == 0 || kinds[0].firstWord > 0):
			// A header of measures: rows below start with a kind.
			for _, lv := range leaveRows(lines[i+1:], measures, leaveUnit(l)) {
				add(lv)
			}
		case distinct(kinds) >= 2 && len(measures) == 0:
			// A header of kinds: rows below start with a measure.
			for _, lv := range leaveColumns(lines[i+1:], kinds, leaveUnit(l)) {
				add(lv)
			}
		case len(kinds) > 0 && kinds[0].firstWord == 0 && distinct(measures) >= 2:
			add(leaveInline(l, kinds[0], measures))
		}
	}
	return out
}

func leaveRows(lines []line, header []match[measure], unit string) []payslip.Leave {
	var out []payslip.Leave
	for _, l := range nextBlock(lines) {
		kinds := scan(l, leaveKinds)
		if len(kinds) == 0 || kinds[0].firstWord != 0 {
			break
		}
		lv := payslip.Leave{Kind: kinds[0].value, Label: kinds[0].label, Unit: unit}
		for _, t := range l.toks[kinds[0].afterToken:] {
			v, ok := quantity(t.Text)
			if !ok {
				continue
			}
			setMeasure(&lv, nearest(t, header), v)
		}
		out = append(out, lv)
	}
	return out
}

func leaveColumns(lines []line, header []match[payslip.LeaveKind], unit string) []payslip.Leave {
	out := make([]payslip.Leave, len(header))
	for i, h := range header {
		out[i] = payslip.Leave{Kind: h.value, Label: h.label, Unit: unit}
	}
	for _, l := range nextBlock(lines) {
		ms := scan(l, leaveMeasures)
		if len(ms) == 0 || ms[0].firstWord != 0 {
			break
		}
		for _, t := range l.toks[ms[0].afterToken:] {
			v, ok := quantity(t.Text)
			if !ok {
				continue
			}
			setMeasure(&out[nearestIndex(t, header)], ms[0].value, v)
		}
	}
	return out
}

func leaveInline(l line, kind match[payslip.LeaveKind], measures []match[measure]) payslip.Leave {
	lv := payslip.Leave{Kind: kind.value, Label: kind.label, Unit: leaveUnit(l)}
	for _, m := range measures {
		for _, t := range l.toks[m.afterToken:] {
			if v, ok := quantity(t.Text); ok {
				setMeasure(&lv, m.value, v)
				break
			}
		}
	}
	return lv
}

// nextBlock returns the lines that directly follow, stopping at a vertical
// gap that ends the table.
func nextBlock(lines []line) []line {
	for i := 1; i < len(lines); i++ {
		if lines[i].y-lines[i-1].y > 3*math.Max(lines[i-1].h, 6) {
			return lines[:i]
		}
	}
	return lines
}

func nearest[T any](t layout.Token, header []match[T]) T {
	return header[nearestIndex(t, header)].value
}

// nearestIndex returns the header phrase closest to a value, comparing right
// edges and centers since values are usually right-aligned under headers
// that may be centered.
func nearestIndex[T any](t layout.Token, header []match[T]) int {
	best, bestDist := 0, math.Inf(1)
	for i, h := range header {
		d := math.Min(math.Abs(t.Right()-h.x1), math.Abs(center(t)-(h.x0+h.x1)/2))
		if d < bestDist {
			best, bestDist = i, d
		}
	}
	return best
}

func setMeasure(lv *payslip.Leave, m measure, v int64) {
	p := &v
	switch m {
	case mPrevious:
		lv.Previous = p
	case mAccrued:
		lv.Accrued = p
	case mUsed:
		lv.Used = p
	case mRemaining:
		lv.Remaining = p
	}
}

// quantity parses hours or days as printed ("14,00", "8", "-3,50") into
// hundredths.
func quantity(s string) (int64, bool) {
	if !numericToken.MatchString(s) || strings.HasSuffix(s, "%") {
		return 0, false
	}
	v, err := money.Parse(s)
	return v, err == nil
}

// checkLeave verifies that each fully printed balance adds up.
func checkLeave(p *payslip.Payslip) {
	for _, lv := range p.Leave {
		if lv.Previous == nil || lv.Accrued == nil || lv.Used == nil || lv.Remaining == nil {
			continue
		}
		expected := *lv.Previous + *lv.Accrued - *lv.Used
		ok := expected == *lv.Remaining
		p.Checks = append(p.Checks, payslip.Check{Name: "leave_balance_" + string(lv.Kind), OK: ok,
			Detail: fmt.Sprintf("%s: previous %s plus accrued %s minus used %s is %s, printed remaining is %s",
				lv.Label, money.Format(*lv.Previous), money.Format(*lv.Accrued), money.Format(*lv.Used),
				money.Format(expected), money.Format(*lv.Remaining))})
	}
}
