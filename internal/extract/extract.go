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
	"unicode"

	"github.com/danielededo/cedolo/internal/ccnl"
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
		kind        payslip.Kind // set when the period itself names an additional month
		closed      bool         // a page already printed the net pay
	}
	var groups []group
	for _, p := range doc.Pages {
		lines := buildLines(p)
		year, month, kind := findPeriod(lines)
		n := len(groups)
		switch {
		// A new period starts a new payslip; so does a page with a period
		// after a payslip that already printed its net pay, as when two
		// payslips of the same month share a file. Continuation pages repeat
		// the header but leave the net to the last page.
		case n == 0, (month != 0 || kind != "") && (month != groups[n-1].month || year != groups[n-1].year ||
			kind != groups[n-1].kind || groups[n-1].closed):
			groups = append(groups, group{pages: []layout.Page{p}, year: year, month: month, kind: kind})
		default:
			groups[n-1].pages = append(groups[n-1].pages, p)
		}
		if _, ok := amountField(lines, p.Width, netLabels); ok {
			groups[len(groups)-1].closed = true
		}
	}
	out := make([]payslip.Payslip, 0, len(groups))
	for _, g := range groups {
		p := extractPayslip(g.pages, g.year, g.month)
		if g.kind != "" {
			p.Kind = g.kind
		}
		out = append(out, p)
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
	// "Mensilità aggiuntiva" alone: the fourteenth is paid in June or July,
	// the thirteenth in December.
	if p.Kind == payslip.Monthly {
		for _, l := range header {
			if l.has(additionalWords...) {
				p.Kind = payslip.Thirteenth
				if p.Month == 6 || p.Month == 7 {
					p.Kind = payslip.Fourteenth
				}
			}
		}
	}
	p.CCNL = textField(header, pages[0].Width, "ccnl", "contratto")
	if c, ok := ccnl.FindCode(p.CCNL); ok {
		p.CCNLCode = c.Code
	}
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
	p.TFRBase = find(tfrBaseLabels)
	if !p.TFR.Found() {
		// Some layouts print the month's TFR only as a neutral item, such as
		// its transfer to a pension fund.
		for _, it := range p.Items {
			if it.Key == "tfr.accrued" {
				src := it.Source
				p.TFR = payslip.Amount{Cents: p.TFR.Cents + it.Cents, Status: payslip.Extracted, Source: &src}
			}
		}
	}
	p.SocialContributions = find(socialLabels)
	p.Taxes = find(taxLabels)
	p.INPS = find(inpsLabels)
	p.RoundingPrevious = find(roundingPreviousLabels)
	p.RoundingCurrent = find(roundingCurrentLabels)

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

var numericPeriod = regexp.MustCompile(`^(0?[1-9]|1[0-2]|13|14)[/.-]((?:19|20)\d\d)$`)

// findPeriod returns the first "month year" in reading order, or a numeric
// "mm/yyyy" on a line that is, or is just below, a period label. Some
// payroll software numbers the additional months 13 and 14: "13/2025" is
// the thirteenth month, paid in December, and "14/2025" the fourteenth,
// whose month of payment the period does not tell.
func findPeriod(lines []line) (year, month int, kind payslip.Kind) {
	for i, l := range lines {
		for k, w := range l.words {
			m, ok := monthNames[w.text]
			if !ok || k+1 >= len(l.words) {
				continue
			}
			if y, err := strconv.Atoi(l.words[k+1].text); err == nil && y >= 1990 && y <= 2099 {
				return y, m, ""
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
				switch mo {
				case 13:
					return y, 12, payslip.Thirteenth
				case 14:
					return y, 0, payslip.Fourteenth
				}
				return y, mo, ""
			}
		}
	}
	return 0, 0, ""
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
	if i > 0 && l.y-lines[i-1].y <= 1.6*l.h {
		tok := l.toks[o.first]
		for _, w := range lines[i-1].words {
			above := lines[i-1].toks[w.tok]
			if ccnlQualifiers[w.text] && above.X < tok.Right()+5 && above.Right() > tok.X-5 {
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
	for _, label := range labels {
		for i, l := range lines {
			occs := l.findAll(label)
			occs = append(occs, stackedLabel(lines, i, label)...)
			for _, o := range occs {
				for _, t := range amountCandidates(lines, i, o.first, o.last, pageWidth) {
					cents, err := money.Parse(t.Text)
					if err != nil {
						continue
					}
					src := sourceOf(t, l.page)
					return payslip.Amount{Cents: cents, Status: payslip.Extracted, Source: &src}, true
				}
			}
		}
	}
	return payslip.Amount{}, false
}

// stackedLabel finds a label whose first words end line i and whose remaining
// words start the line just below, at the same horizontal position.
func stackedLabel(lines []line, i int, label string) []occurrence {
	pw := strings.Fields(label)
	if len(pw) < 2 || i+1 >= len(lines) || lines[i+1].y-lines[i].y > 6 {
		return nil
	}
	l, next := lines[i], lines[i+1]
	var out []occurrence
	for k := 1; k < len(pw); k++ {
		head, tail := strings.Join(pw[:k], " "), strings.Join(pw[k:], " ")
		for _, h := range l.findAll(head) {
			for _, t := range next.findAll(tail) {
				tok := next.toks[t.first]
				if tok.X >= l.toks[h.first].X-5 && tok.X <= l.toks[h.last].Right()+5 {
					out = append(out, occurrence{first: h.first, last: h.last, afterWord: h.afterWord})
				}
			}
		}
	}
	return out
}

// amountCandidates returns the amount printed right after a label on its
// line, or failing that the amounts in the label's cell on the following
// lines. A value to the right counts only when nothing but currency signs or
// punctuation separates it from the label, so that a neighbouring block (a
// calendar, another column) is not mistaken for the label's value.
func amountCandidates(lines []line, i, first, last int, pageWidth float64) []layout.Token {
	l := lines[i]
	for k := last + 1; k < len(l.toks); k++ {
		t := l.toks[k]
		if money.IsAmount(t.Text) {
			return []layout.Token{t}
		}
		if !isFiller(t.Text) {
			break
		}
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
	// colInfo is a recognized column whose values are not pay, such as
	// statistical data; its values become information items.
	colInfo
	// colUnknown is a header the extractor does not recognize, such as a
	// presence calendar printed beside the table; its values are dropped.
	colUnknown
)

func (k colKind) numeric() bool { return k >= colQuantity && k <= colAmount }

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
				k = colUnknown
			}
			t := l.toks[w.tok]
			// "Voce" as the first column header names the item code.
			if k == colDescription && w.tok == 0 && w.text == "voce" {
				k = colCode
			}
			// Another word of a header token already classified: the first
			// recognized word decides ("VAL.UNIT./NEUTRA" is a base column).
			if n := len(cols); n > 0 && cols[n-1].x0 == t.X && cols[n-1].x1 == t.Right() {
				if cols[n-1].kind == colUnknown && k != colUnknown {
					cols[n-1].kind = k
					kinds[k] = true
				}
				continue
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

// ordinalMonth is "13ma", "14a" and the like: the start of a description
// naming an additional month, not an item code.
var ordinalMonth = regexp.MustCompile(`(?i)^1[34](a|ma|esima)$`)

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
		if center(tok) <= t.numericStart {
			continue
		}
		k := t.nearestColumn(tok)
		switch {
		case (k.numeric() || k == colInfo) && numericToken.MatchString(tok.Text):
			assigned[i] = k
		case k == colDescription:
			// A description word printed far to the right.
		default:
			// Values under unrecognized columns, and anything that is not a
			// number under a value column, are dropped.
			assigned[i] = colUnknown
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
	var infoTok *layout.Token // the value of a row that is not pay
	for i, tok := range l.toks {
		k := assigned[i]
		// Some layouts share one reference column between quantities and
		// rates; a percentage is always a rate.
		if k == colQuantity && strings.HasSuffix(tok.Text, "%") {
			k = colBase
		}
		if (k == colInfo || k == colBase) && !strings.HasSuffix(tok.Text, "%") {
			infoTok = &l.toks[i]
		}
		switch k {
		case colUnknown, colInfo:
			continue
		case colQuantity:
			if qty == "" {
				qty = tok.Text
			}
		case colBase:
			base = tok.Text
		case colEarnings, colDeductions, colAmount:
			amounts = append(amounts, amt{tok, k})
		default:
			if i == 0 && codeToken.MatchString(tok.Text) && strings.ContainsAny(tok.Text, "0123456789") && !ordinalMonth.MatchString(tok.Text) {
				code = tok.Text
				continue
			}
			desc = append(desc, tok.Text)
		}
	}
	description := strings.Join(desc, " ")
	var items []payslip.Item
	// A row with no pay amount but a value in a neutral or statistical
	// column is an information item: it is shown, never summed.
	if len(amounts) == 0 && infoTok != nil && strings.IndexFunc(description, unicode.IsLetter) >= 0 {
		if cents, err := money.Parse(infoTok.Text); err == nil {
			items = append(items, payslip.Item{Code: code, Description: description, Quantity: qty,
				Cents: cents, Side: payslip.Info, Key: taxonomy.Classify(description, payslip.Info),
				Source: sourceOf(*infoTok, page)})
		}
	}
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

// nearestColumn returns the kind of the column a token belongs to. Each
// header column owns the zone from its left edge to the next header's left
// edge, and a token belongs to the zone holding its right edge: values are
// usually right-aligned and may extend past the end of a narrow or centered
// header, but not into the next column. A token ending before every header
// goes to the closest one.
func (t table) nearestColumn(tok layout.Token) colKind {
	right := tok.Right()
	zone := -1
	for i, c := range t.cols {
		if c.x0 <= right-1 {
			zone = i
		}
	}
	if zone >= 0 {
		return t.cols[zone].kind
	}
	best, bestDist := colNone, math.Inf(1)
	for _, c := range t.cols {
		d := math.Min(math.Abs(right-c.x1), math.Abs(center(tok)-(c.x0+c.x1)/2))
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
