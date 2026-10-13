// Package extract reads payslip data from a text layout without knowing which
// payroll software produced it. Italian payslips share a legally required
// content and a common structure (header, a table of items with earnings and
// deductions, totals, taxable bases, TFR), so the extractor recognizes labels
// from a dictionary of synonyms, derives table columns from the table header,
// classifies items through the taxonomy, and then cross-checks the arithmetic
// to decide how far each value can be trusted. Vendor-specific tuning can be
// layered on top later; it is not required for a first reading.
package extract

import (
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/danielededo/cedolo/internal/layout"
	"github.com/danielededo/cedolo/internal/money"
	"github.com/danielededo/cedolo/internal/payslip"
	"github.com/danielededo/cedolo/internal/taxonomy"
)

// Document extracts every payslip in a document. A page that names a pay
// period different from the previous page starts a new payslip; pages without
// a period continue the current one.
func Document(doc layout.Document) []payslip.Payslip {
	type group struct {
		pages       []layout.Page
		year, month int
	}
	var groups []group
	for _, p := range doc.Pages {
		year, month := findPeriod(buildLines(p))
		n := len(groups)
		switch {
		case n == 0, month != 0 && (month != groups[n-1].month || year != groups[n-1].year):
			groups = append(groups, group{pages: []layout.Page{p}, year: year, month: month})
		default:
			groups[n-1].pages = append(groups[n-1].pages, p)
		}
	}
	out := make([]payslip.Payslip, 0, len(groups))
	for _, g := range groups {
		out = append(out, extractPayslip(g.pages, g.year, g.month))
	}
	return out
}

func extractPayslip(pages []layout.Page, year, month int) payslip.Payslip {
	p := payslip.Payslip{Year: year, Month: month, Kind: payslip.Monthly}
	var all [][]line
	for _, pg := range pages {
		p.Pages = append(p.Pages, pg.Number)
		all = append(all, buildLines(pg))
	}

	first := all[0]
	headerEnd := len(first)
	for i, pg := range all {
		t, ok := findTable(pg, pages[i].Width)
		if !ok {
			continue
		}
		if i == 0 {
			headerEnd = t.headerLine
		}
		p.Items = append(p.Items, t.rows(pg, pages[i].Number)...)
	}

	header := first[:headerEnd]
	for _, l := range header {
		if l.has(thirteenthWords...) {
			p.Kind = payslip.Thirteenth
		}
		if l.has(fourteenthWords...) {
			p.Kind = payslip.Fourteenth
		}
	}
	p.CCNL = textField(header, pages[0].Width, "ccnl", "contratto")
	p.Level = levelField(header, pages[0].Width)

	// Totals usually close the payslip, so later pages are searched first.
	find := func(labels []string) payslip.Amount {
		for i := len(all) - 1; i >= 0; i-- {
			if a, ok := amountField(all[i], pages[i].Width, labels); ok {
				return a
			}
		}
		return payslip.Amount{Status: payslip.Missing}
	}
	p.Gross = find(grossLabels)
	p.Deductions = find(deductionLabels)
	p.Net = find(netLabels)
	p.INPSBase = find(inpsBaseLabels)
	p.IRPEFBase = find(irpefBaseLabels)
	p.TFR = find(tfrLabels)

	for _, pg := range all {
		for _, lv := range findLeave(pg) {
			if !hasLeave(p.Leave, lv.Kind) {
				p.Leave = append(p.Leave, lv)
			}
		}
	}

	validate(&p)
	checkLeave(&p)
	return p
}

func hasLeave(ls []payslip.Leave, k payslip.LeaveKind) bool {
	for _, l := range ls {
		if l.Kind == k {
			return true
		}
	}
	return false
}

var numericPeriod = regexp.MustCompile(`^(0?[1-9]|1[0-2])[/.-]((?:19|20)\d\d)$`)

// findPeriod returns the first "month year" in reading order, or a numeric
// "mm/yyyy" on a line that is, or is just below, a period label.
func findPeriod(lines []line) (year, month int) {
	for i, l := range lines {
		for k, w := range l.words {
			m, ok := monthNames[w.text]
			if !ok || k+1 >= len(l.words) {
				continue
			}
			if y, err := strconv.Atoi(l.words[k+1].text); err == nil && y >= 1990 && y <= 2099 {
				return y, m
			}
		}
		labelled := l.has(periodLabels...) || (i > 0 && lines[i-1].has(periodLabels...))
		if !labelled {
			continue
		}
		for _, t := range l.toks {
			if m := numericPeriod.FindStringSubmatch(t.Text); m != nil {
				mo, _ := strconv.Atoi(m[1])
				y, _ := strconv.Atoi(m[2])
				return y, mo
			}
		}
	}
	return 0, 0
}

// valueTokens returns the tokens that hold a label's value: the tokens to the
// right of the label on the same line, up to the next label or a wide gap, or
// failing that the tokens in the label's cell on the next line.
func valueTokens(lines []line, i, first, last int, pageWidth float64) []layout.Token {
	l := lines[i]
	var right []layout.Token
	prevRight := l.toks[last].Right()
	for k := last + 1; k < len(l.toks); k++ {
		t := l.toks[k]
		if t.X-prevRight > 40 || isHeaderLabel(t.Text) {
			break
		}
		right = append(right, t)
		prevRight = t.Right()
	}
	if len(right) > 0 {
		return right
	}
	x0, x1 := l.cell(first, last, pageWidth)
	for j := i + 1; j < len(lines) && lines[j].y-l.y <= 3*l.h; j++ {
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
			first, last, ok := l.find(label)
			if !ok {
				continue
			}
			// Skip label words that only qualify the label ("CCNL applicato").
			for last+1 < len(l.toks) && isHeaderLabel(l.toks[last+1].Text) && taxonomy.Normalize(l.toks[last+1].Text) == "applicato" {
				last++
			}
			var parts []string
			for _, t := range valueTokens(lines, i, first, last, pageWidth) {
				if taxonomy.Normalize(t.Text) == "ccnl" {
					continue
				}
				parts = append(parts, t.Text)
			}
			if len(parts) > 0 {
				return strings.Join(parts, " ")
			}
		}
	}
	return ""
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
func amountField(lines []line, pageWidth float64, labels []string) (payslip.Amount, bool) {
	for _, label := range labels {
		for i, l := range lines {
			first, last, ok := l.find(label)
			if !ok {
				continue
			}
			for _, t := range amountCandidates(lines, i, first, last, pageWidth) {
				cents, err := money.Parse(t.Text)
				if err != nil {
					continue
				}
				src := sourceOf(t, l.page)
				return payslip.Amount{Cents: cents, Status: payslip.Extracted, Source: &src}, true
			}
		}
	}
	return payslip.Amount{}, false
}

// amountCandidates returns the amounts to the right of a label on its line,
// nearest first, or the amounts in the label's cell on the following lines.
func amountCandidates(lines []line, i, first, last int, pageWidth float64) []layout.Token {
	l := lines[i]
	var right []layout.Token
	for k := last + 1; k < len(l.toks); k++ {
		if money.IsAmount(l.toks[k].Text) {
			right = append(right, l.toks[k])
		}
	}
	if len(right) > 0 {
		return right[:1]
	}
	x0, x1 := l.cell(first, last, pageWidth)
	for j := i + 1; j < len(lines) && lines[j].y-l.y <= 3*l.h; j++ {
		for _, t := range lines[j].toks {
			if c := center(t); c >= x0 && c < x1 && money.IsAmount(t.Text) {
				return []layout.Token{t}
			}
		}
	}
	return nil
}

func sourceOf(t layout.Token, page int) payslip.Source {
	return payslip.Source{Page: page, X: t.X, Y: t.Y, W: t.W, H: t.H}
}

// Table extraction.

type colKind int

const (
	colNone colKind = iota
	colCode
	colDescription
	colQuantity
	colBase
	colEarnings
	colDeductions
	colAmount // a single signed amount column
)

func (k colKind) numeric() bool { return k >= colQuantity }

type column struct {
	kind   colKind
	x0, x1 float64
}

type table struct {
	headerLine   int
	cols         []column
	numericStart float64
}

// findTable locates the item table by its header line: a line naming both an
// earnings and a deductions column, or a single amount column next to a
// description or quantity column.
func findTable(lines []line, pageWidth float64) (table, bool) {
	for i, l := range lines {
		var cols []column
		kinds := map[colKind]bool{}
		for _, w := range l.words {
			k, ok := columnWords[w.text]
			if !ok {
				continue
			}
			t := l.toks[w.tok]
			// "Voce" as the first column header names the item code.
			if k == colDescription && w.tok == 0 && w.text == "voce" {
				k = colCode
			}
			if n := len(cols); n > 0 && cols[n-1].kind == k {
				cols[n-1].x1 = math.Max(cols[n-1].x1, t.Right())
				continue
			}
			cols = append(cols, column{kind: k, x0: t.X, x1: t.Right()})
			kinds[k] = true
		}
		twoSided := kinds[colEarnings] && kinds[colDeductions]
		signed := kinds[colAmount] && (kinds[colDescription] || kinds[colQuantity])
		if !twoSided && !signed {
			continue
		}
		start := pageWidth
		for _, c := range cols {
			if c.kind.numeric() {
				start = math.Min(start, c.x0)
			}
		}
		return table{headerLine: i, cols: cols, numericStart: start - 25}, true
	}
	return table{}, false
}

var numericToken = regexp.MustCompile(`^[-+]?[0-9][0-9.,]*%?-?$`)
var codeToken = regexp.MustCompile(`^[A-Za-z0-9]{1,6}$`)

// rows reads the item lines below the table header until the totals or the
// end of the table.
func (t table) rows(lines []line, page int) []payslip.Item {
	var items []payslip.Item
	lastY := lines[t.headerLine].y
	allLabels := concat(grossLabels, deductionLabels, netLabels, inpsBaseLabels, irpefBaseLabels, tfrLabels)
	for _, l := range lines[t.headerLine+1:] {
		if l.y-lastY > 40 || l.has(allLabels...) || l.has("totale", "totali") {
			break
		}
		lastY = l.y
		items = append(items, t.row(l, page)...)
	}
	return items
}

func (t table) row(l line, page int) []payslip.Item {
	assigned := make([]colKind, len(l.toks))
	for i, tok := range l.toks {
		if numericToken.MatchString(tok.Text) && center(tok) > t.numericStart {
			assigned[i] = t.nearestNumeric(tok)
		}
	}
	var code string
	var desc []string
	var qty, base string
	type amt struct {
		tok  layout.Token
		kind colKind
	}
	var amounts []amt
	for i, tok := range l.toks {
		k := assigned[i]
		// Some layouts share one reference column between quantities and
		// rates; a percentage is always a rate.
		if k == colQuantity && strings.HasSuffix(tok.Text, "%") {
			k = colBase
		}
		switch k {
		case colQuantity:
			qty = tok.Text
		case colBase:
			base = tok.Text
		case colEarnings, colDeductions, colAmount:
			amounts = append(amounts, amt{tok, k})
		default:
			if i == 0 && codeToken.MatchString(tok.Text) && strings.ContainsAny(tok.Text, "0123456789") {
				code = tok.Text
				continue
			}
			desc = append(desc, tok.Text)
		}
	}
	description := strings.Join(desc, " ")
	var items []payslip.Item
	for _, a := range amounts {
		cents, err := money.Parse(a.tok.Text)
		if err != nil {
			continue
		}
		side := payslip.Earning
		switch {
		case a.kind == colDeductions:
			side = payslip.Deduction
		case a.kind == colAmount && cents < 0:
			side = payslip.Deduction
		case a.kind == colAmount && taxonomy.Classify(description, payslip.Deduction) != taxonomy.OtherDeduction:
			// An unsigned single column: fall back on what the item is.
			side = payslip.Deduction
		}
		if a.kind == colAmount && cents < 0 {
			cents = -cents
		}
		items = append(items, payslip.Item{
			Code:        code,
			Description: description,
			Quantity:    qty,
			Base:        base,
			Cents:       cents,
			Side:        side,
			Key:         taxonomy.Classify(description, side),
			Source:      sourceOf(a.tok, page),
		})
	}
	return items
}

// nearestNumeric assigns a numeric token to the numeric column whose right
// edge or center is closest; amounts are usually right-aligned while headers
// may be centered.
func (t table) nearestNumeric(tok layout.Token) colKind {
	best, bestDist := colNone, math.Inf(1)
	for _, c := range t.cols {
		if !c.kind.numeric() {
			continue
		}
		d := math.Min(math.Abs(tok.Right()-c.x1), math.Abs(center(tok)-(c.x0+c.x1)/2))
		if d < bestDist {
			best, bestDist = c.kind, d
		}
	}
	return best
}

func concat(lists ...[]string) []string {
	var out []string
	for _, l := range lists {
		out = append(out, l...)
	}
	sort.SliceStable(out, func(i, j int) bool { return len(out[i]) > len(out[j]) })
	return out
}
