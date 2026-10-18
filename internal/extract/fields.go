package extract

import (
	"math"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"github.com/danielededo/cedolo/internal/layout"
	"github.com/danielededo/cedolo/internal/money"
	"github.com/danielededo/cedolo/internal/payslip"
	"github.com/danielededo/cedolo/internal/taxonomy"
)

// valueTokens returns the tokens that hold a label's value: the tokens to the
// right of the label on the same line, up to the next label or a wide gap, or
// failing that the tokens in the label's cell on the next line.
func valueTokens(lines []line, i, first, last int, pageWidth float64) []layout.Token {
	l := lines[i]
	var right []layout.Token
	prevRight := l.toks[last].Right()
	for k := last + 1; k < len(l.toks); k++ {
		t := l.toks[k]
		if t.X-prevRight > wideGap || isHeaderLabel(t.Text) {
			break
		}
		right = append(right, t)
		prevRight = t.Right()
	}
	if len(right) > 0 {
		return right
	}
	x0, x1 := l.cell(first, last, pageWidth)
	for j := i + 1; j < len(lines) && lines[j].y-l.y <= belowReach*l.h; j++ {
		var below []layout.Token
		for _, t := range lines[j].toks {
			if c := center(t); c >= x0 && c < x1 {
				below = append(below, t)
			}
		}
		if len(below) > 0 {
			return below
		}
	}
	return nil
}

func isHeaderLabel(s string) bool {
	for _, w := range strings.Fields(taxonomy.Normalize(s)) {
		if headerLabelWords[w] {
			return true
		}
	}
	return false
}

// textField returns the text printed for the first of the labels found.
func textField(lines []line, pageWidth float64, labels ...string) string {
	for _, label := range labels {
		for i, l := range lines {
			for _, o := range l.findAll(label) {
				if s := textAt(lines, i, o, pageWidth); s != "" {
					return s
				}
			}
		}
	}
	return ""
}

// Words before "CCNL" that make it qualify another field, as in "ORE CCNL"
// (contractual hours) or "MINIMO CCNL".
var ccnlQualifiers = map[string]bool{
	"ore": true, "orario": true, "minimo": true, "minimi": true, "paga": true, "retribuzione": true,
	"tabellare": true, "gg": true, "giorni": true, "liv": true, "livello": true,
}

// textAt returns the text printed for one occurrence of a label, or ""
// when the label only qualifies another field ("ORE CCNL").
func textAt(lines []line, i int, o occurrence, pageWidth float64) string {
	l := lines[i]
	for k, w := range l.words {
		if w.tok == o.first && k > 0 && ccnlQualifiers[l.words[k-1].text] && l.continues(l.words[k-1].tok, o.first) {
			return ""
		}
	}
	// The qualifier may also sit right above, in a label on two lines
	// ("GG." over "CCNL").
	if i > 0 && l.y-lines[i-1].y <= qualifierReach*l.h {
		tok := l.toks[o.first]
		for _, w := range lines[i-1].words {
			above := lines[i-1].toks[w.tok]
			if ccnlQualifiers[w.text] && above.X < tok.Right()+labelSlack && above.Right() > tok.X-labelSlack {
				return ""
			}
		}
	}
	last := o.last
	// Skip label words that only qualify the label ("CCNL applicato").
	for last+1 < len(l.toks) && taxonomy.Normalize(l.toks[last+1].Text) == "applicato" {
		last++
	}
	var parts []string
	for _, t := range valueTokens(lines, i, o.first, last, pageWidth) {
		if taxonomy.Normalize(t.Text) == "ccnl" {
			continue
		}
		parts = append(parts, t.Text)
	}
	return strings.Join(parts, " ")
}

var levelPattern = regexp.MustCompile(`^([0-9]{1,2}[A-Za-z]?|[A-Za-z][0-9]?|Q)[°^]?$`)

// levelField returns the contract level, such as "4" from "4° LIV.".
func levelField(lines []line, pageWidth float64) string {
	for _, label := range []string{"livello", "liv"} {
		for i, l := range lines {
			first, last, ok := l.find(label)
			if !ok {
				continue
			}
			for _, t := range valueTokens(lines, i, first, last, pageWidth) {
				if m := levelPattern.FindStringSubmatch(strings.TrimSuffix(t.Text, ".")); m != nil {
					return m[1]
				}
			}
		}
	}
	return ""
}

// amountField returns the amount printed for the first label that has one.
// A label may be split over two stacked lines ("ARROTONDAMENTO" above
// "ATTUALE"); its value is then searched from the first line of the label.
func amountField(lines []line, pageWidth float64, labels []string) (payslip.Amount, bool) {
	if rs := amountReadings(lines, pageWidth, labels, false); len(rs) > 0 {
		return rs[0], true
	}
	return payslip.Amount{}, false
}

// amountReadings returns the amounts printed for the labels, most preferred
// first: the preferred amount of each occurrence of each label, in label
// order, and then, if asked, the other amounts each occurrence could be read
// as, the farthest last. The first reading is the one amountField returns.
func amountReadings(lines []line, pageWidth float64, labels []string, others bool) []payslip.Amount {
	var first, rest, far []payslip.Amount
	for _, label := range labels {
		for i, l := range lines {
			occs := l.findAll(label)
			occs = append(occs, stackedLabel(lines, i, label)...)
			for _, o := range occs {
				read := func(t layout.Token) (payslip.Amount, bool) {
					cents, err := money.Parse(t.Text)
					if err != nil {
						return payslip.Amount{}, false
					}
					src := sourceOf(t, l.page)
					return payslip.Amount{Cents: cents, Status: payslip.Extracted, Source: &src}, true
				}
				cands := amountCandidates(lines, i, o.first, o.last, pageWidth)
				for k, t := range cands {
					if a, ok := read(t); ok && k == 0 {
						first = append(first, a)
					} else if ok {
						rest = append(rest, a)
					}
				}
				if t, ok := farBelow(lines, i, o.first, o.last, pageWidth); ok && !slices.Contains(cands, t) {
					if a, ok := read(t); ok {
						far = append(far, a)
					}
				}
			}
		}
	}
	if !others {
		return first
	}
	return append(append(first, rest...), far...)
}

// stackedLabel finds a label whose first words end line i and whose remaining
// words start the line just below, at the same horizontal position.
func stackedLabel(lines []line, i int, label string) []occurrence {
	pw := strings.Fields(label)
	if len(pw) < 2 || i+1 >= len(lines) || lines[i+1].y-lines[i].y > stackedGap {
		return nil
	}
	l, next := lines[i], lines[i+1]
	var out []occurrence
	for k := 1; k < len(pw); k++ {
		head, tail := strings.Join(pw[:k], " "), strings.Join(pw[k:], " ")
		for _, h := range l.findAll(head) {
			for _, t := range next.findAll(tail) {
				tok := next.toks[t.first]
				if tok.X >= l.toks[h.first].X-labelSlack && tok.X <= l.toks[h.last].Right()+labelSlack {
					out = append(out, occurrence{first: h.first, last: h.last, afterWord: h.afterWord})
				}
			}
		}
	}
	return out
}

// amountCandidates returns the amounts a label could be read as, preferred
// first: the amount printed right after the label on its line, or failing
// that the amounts in the label's cell on the following lines. A value to the
// right counts only when nothing but currency signs or punctuation separates
// it from the label, so that a neighbouring block (a calendar, another
// column) is not mistaken for the label's value. The other candidates are the
// readings the preferred one won over.
func amountCandidates(lines []line, i, first, last int, pageWidth float64) []layout.Token {
	l := lines[i]
	label := l.toks[last]
	x0, x1 := l.cell(first, last, pageWidth)
	below := func(fits func(layout.Token) bool) []layout.Token {
		for j := i + 1; j < len(lines) && lines[j].y-l.y <= belowReach*math.Max(l.h, lines[j].h); j++ {
			if yearToDate(lines, j, x0) {
				continue
			}
			for _, t := range lines[j].toks {
				if c := center(t); c >= x0 && c < x1 && fits(t) && money.IsAmount(t.Text) {
					return []layout.Token{t}
				}
			}
		}
		return nil
	}
	under := below(func(t layout.Token) bool { return t.X <= label.Right() })
	cell := below(func(layout.Token) bool { return true })
	var beside []layout.Token
	for k := last + 1; k < len(l.toks); k++ {
		t := l.toks[k]
		if money.IsAmount(t.Text) {
			beside = []layout.Token{t}
			break
		}
		if !isFiller(t.Text) {
			break
		}
	}
	var order [][]layout.Token
	switch {
	case beside != nil && math.Abs((beside[0].Y+beside[0].H)-(label.Y+label.H)) > baselineSlack*label.H:
		// An amount grouped into the label's line but on another baseline
		// may sit in a column beside the label, such as statistics: a
		// number under the label is then its value.
		order = [][]layout.Token{under, beside, cell}
	case beside != nil:
		order = [][]layout.Token{beside, under, cell}
	case last+1 == len(l.toks):
		// A label that ends its line has no neighbour to bound its cell, so
		// a number starting under the label beats one beside it, which may
		// belong to another column.
		order = [][]layout.Token{under, cell}
	default:
		order = [][]layout.Token{cell, under}
	}
	var out []layout.Token
	for _, ts := range order {
		for _, t := range ts {
			if !slices.Contains(out, t) {
				out = append(out, t)
			}
		}
	}
	return out
}

// farBelow returns the first amount in a label's cell beyond the reach of
// amountCandidates but within farReach, with nothing else in the cell
// between: the value of a total printed at the foot of a tall box.
func farBelow(lines []line, i, first, last int, pageWidth float64) (layout.Token, bool) {
	l := lines[i]
	x0, x1 := l.cell(first, last, pageWidth)
	for j := i + 1; j < len(lines); j++ {
		h := math.Max(l.h, lines[j].h)
		if lines[j].y-l.y > farReach*h {
			break
		}
		for _, t := range lines[j].toks {
			if c := center(t); c < x0 || c >= x1 {
				continue
			}
			if lines[j].y-l.y <= belowReach*h || !money.IsAmount(t.Text) || yearToDate(lines, j, x0) {
				return layout.Token{}, false
			}
			return t, true
		}
	}
	return layout.Token{}, false
}

// yearToDate reports whether line j holds the year's running totals rather
// than the month's: a row label such as "PROGRES. ANNUI", left of the label's
// cell, overlaps it vertically.
func yearToDate(lines []line, j int, x0 float64) bool {
	for _, l := range lines {
		if l.y >= lines[j].y+lines[j].h || l.y+l.h <= lines[j].y {
			continue
		}
		for _, t := range l.toks {
			if t.X < x0 && progressiveWords[taxonomy.Normalize(t.Text)] {
				return true
			}
		}
	}
	return false
}

// isFiller reports whether a token carries no information between a label
// and its value: a currency sign or punctuation.
func isFiller(s string) bool {
	if s == "€" || strings.EqualFold(s, "eur") {
		return true
	}
	return strings.IndexFunc(s, func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }) < 0
}

func sourceOf(t layout.Token, page int) payslip.Source {
	return payslip.Source{Page: page, X: t.X, Y: t.Y, W: t.W, H: t.H}
}
