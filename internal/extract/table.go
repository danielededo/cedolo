package extract

import (
	"math"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/danielededo/cedolo/internal/layout"
	"github.com/danielededo/cedolo/internal/money"
	"github.com/danielededo/cedolo/internal/payslip"
	"github.com/danielededo/cedolo/internal/taxonomy"
)

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
		return table{headerLine: i, cols: cols, numericStart: start - numericMargin}, true
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
		if l.y-lastY > wideGap || l.has(allLabels...) || l.has("totale", "totali") {
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
