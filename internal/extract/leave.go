package extract

import (
	"fmt"
	"math"
	"strings"

	"github.com/danielededo/cedolo/internal/layout"
	"github.com/danielededo/cedolo/internal/money"
	"github.com/danielededo/cedolo/internal/payslip"
	"github.com/danielededo/cedolo/internal/taxonomy"
)

// Leave balances are printed in one of three shapes, all handled here:
//
//   - rows:       a header of measures (maturato, goduto, residuo) and one row
//     per kind of leave;
//   - columns:    a header of kinds (ferie, permessi, ex festività) and one row
//     per measure;
//   - inline:     one line per kind with measure and value pairs;
//   - cells:      a row of "kind measure" labels with values in the cells
//     below ("FERIE MAT.", "FERIE GOD.", "PERMESSI RES.");
//   - stacked:    side-by-side blocks, one per kind, each a column of
//     measure labels qualified on the line below ("RESIDUO" over "A.P.").

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
	{"fest", payslip.FormerHolidays},
	{"banca ore", payslip.HoursBank}, {"b ore", payslip.HoursBank},
	{"permessi rol", payslip.ReducedHours}, {"riduzione orario", payslip.ReducedHours},
	{"permessi", payslip.Permits}, {"r o l", payslip.ReducedHours}, {"rol", payslip.ReducedHours},
	{"p a r", payslip.ReducedHours}, {"par", payslip.ReducedHours},
	{"flessibilita", payslip.Flexibility}, {"fless", payslip.Flexibility},
	{"ferie", payslip.Holiday},
}

var leaveMeasures = []phrase[measure]{
	{"residuo anno precedente", mPrevious}, {"residuo a p", mPrevious}, {"residuo ap", mPrevious},
	{"res a p", mPrevious}, {"res ap", mPrevious}, {"a p", mPrevious},
	{"residuo prec", mPrevious}, {"res prec", mPrevious}, {"anno precedente", mPrevious},
	{"residuo iniziale", mPrevious}, {"riporto", mPrevious},
	{"maturato", mAccrued}, {"maturate", mAccrued}, {"maturati", mAccrued}, {"maturata", mAccrued},
	{"spettanti", mAccrued}, {"spettante", mAccrued},
	{"goduto", mUsed}, {"godute", mUsed}, {"goduti", mUsed}, {"goduta", mUsed},
	{"fruito", mUsed}, {"fruite", mUsed}, {"fruiti", mUsed}, {"usufruito", mUsed},
	{"da godere", mRemaining}, {"residuo", mRemaining}, {"residue", mRemaining}, {"residui", mRemaining},
	{"saldo", mRemaining}, {"rimanenti", mRemaining},
	{"mat", mAccrued}, {"god", mUsed}, {"res", mRemaining},
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
	// A kind seen again (a later row of stacked blocks, a second table)
	// fills the measures still missing; it never overwrites one.
	add := func(lv payslip.Leave) {
		if lv.Previous == nil && lv.Accrued == nil && lv.Used == nil && lv.Remaining == nil {
			return
		}
		i, seen := byKind[lv.Kind]
		if !seen {
			byKind[lv.Kind] = len(out)
			out = append(out, lv)
			return
		}
		cur := &out[i]
		for _, f := range []struct{ dst, src **int64 }{
			{&cur.Previous, &lv.Previous}, {&cur.Accrued, &lv.Accrued}, {&cur.Used, &lv.Used}, {&cur.Remaining, &lv.Remaining},
		} {
			if *f.dst == nil {
				*f.dst = *f.src
			}
		}
		if cur.Unit == "" {
			cur.Unit = lv.Unit
		}
	}
	for i, l := range lines {
		kinds := scan(l, leaveKinds)
		measures := scan(l, leaveMeasures)
		if cells := leaveCells(lines, i, kinds, measures); cells != nil {
			for _, lv := range cells {
				add(lv)
			}
			continue
		}
		if stacked := leaveStacked(lines, i, measures); stacked != nil {
			for _, lv := range stacked {
				add(lv)
			}
			continue
		}
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

// leaveCells reads a row of "kind measure" labels, each heading a cell whose
// value is printed on the next line. It returns nil when line i is not such
// a row.
func leaveCells(lines []line, i int, kinds []match[payslip.LeaveKind], measures []match[measure]) []payslip.Leave {
	type cell struct {
		kind    match[payslip.LeaveKind]
		measure measure
	}
	var cells []cell
	for _, k := range kinds {
		for _, m := range measures {
			if m.firstWord == k.afterWord {
				cells = append(cells, cell{k, m.value})
			}
		}
	}
	if len(cells) < 2 {
		return nil
	}
	l := lines[i]
	byKind := map[payslip.LeaveKind]*payslip.Leave{}
	var order []payslip.LeaveKind
	for c, cl := range cells {
		x0, x1 := cl.kind.x0-2, math.Inf(1)
		if c+1 < len(cells) {
			x1 = cells[c+1].kind.x0 - 2
		}
		lv, ok := byKind[cl.kind.value]
		if !ok {
			lv = &payslip.Leave{Kind: cl.kind.value, Label: cl.kind.label, Unit: leaveUnit(l)}
			byKind[cl.kind.value] = lv
			order = append(order, cl.kind.value)
		}
		for j := i + 1; j < len(lines) && lines[j].y-l.y <= 3*l.h; j++ {
			if leaveLabelLine(lines[j]) {
				break
			}
			for _, t := range lines[j].toks {
				if c := center(t); c >= x0 && c < x1 {
					if v, ok := quantity(t.Text); ok {
						setMeasure(lv, cl.measure, v)
					}
				}
			}
		}
	}
	out := make([]payslip.Leave, 0, len(order))
	for _, k := range order {
		out = append(out, *byKind[k])
	}
	return out
}

// leaveLabelLine reports whether a line is itself a row of leave labels.
func leaveLabelLine(l line) bool {
	return len(scan(l, leaveKinds)) >= 2 && len(scan(l, leaveMeasures)) >= 2
}

// leaveStacked reads side-by-side leave blocks: line i repeats one measure
// label once per block ("RESIDUO ... RESIDUO ..."), a qualifier on the line
// just below may change its meaning ("A.P." makes it the previous balance),
// the value sits in the block on the same or the next line, and each block's
// kind is the kind label printed at the block's left edge nearby. It returns
// nil when line i is not such a row.
func leaveStacked(lines []line, i int, measures []match[measure]) []payslip.Leave {
	if len(measures) < 2 {
		return nil
	}
	for _, m := range measures[1:] {
		if m.value != measures[0].value {
			return nil
		}
	}
	l := lines[i]
	var below *line
	if i+1 < len(lines) && lines[i+1].y-l.y <= 6 {
		below = &lines[i+1]
	}
	var out []payslip.Leave
	for b, m := range measures {
		x0, x1 := m.x0-3, 0.0
		if b+1 < len(measures) {
			x1 = measures[b+1].x0 - 3
		} else {
			x1 = m.x0 + (m.x0 - measures[b-1].x0) - 3
		}
		kind, label, ok := blockKind(lines, i, m.x0)
		if !ok {
			continue
		}
		lv := payslip.Leave{Kind: kind, Label: label}
		meaning := m.value
		var value *int64
		for _, ln := range []*line{&l, below} {
			if ln == nil {
				continue
			}
			for _, t := range ln.toks {
				if t.X < x0 || center(t) >= x1 {
					continue
				}
				norm := taxonomy.Normalize(t.Text)
				switch {
				case ln == below && meaning == mRemaining && (norm == "a" || norm == "ap" || norm == "a p" ||
					strings.HasPrefix(norm, "prec") || strings.HasPrefix(norm, "a p ")):
					meaning = mPrevious
				case norm == "o" || norm == "ore":
					lv.Unit = "hours"
				case norm == "g" || norm == "gg":
					lv.Unit = "days"
				}
				if v, ok := quantity(t.Text); ok && value == nil {
					value = &v
				}
			}
		}
		if value != nil {
			setMeasure(&lv, meaning, *value)
		}
		out = append(out, lv)
	}
	return out
}

// blockKind finds the kind label aligned with the left edge of a block, within
// a few lines above or below line i.
func blockKind(lines []line, i int, left float64) (payslip.LeaveKind, string, bool) {
	for d := 0; d <= 12; d++ {
		for _, j := range []int{i - d, i + d} {
			if j < 0 || j >= len(lines) || math.Abs(lines[j].y-lines[i].y) > 45 {
				continue
			}
			for _, k := range scan(lines[j], leaveKinds) {
				if k.x0 >= left-12 && k.x0 <= left+8 {
					return k.value, k.label, true
				}
			}
		}
	}
	return "", "", false
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
