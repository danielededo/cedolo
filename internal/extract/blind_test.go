package extract_test

import (
	"fmt"
	"hash/fnv"
	"slices"
	"strings"
	"testing"
	"unicode"

	"github.com/danielededo/cedolo/internal/extract"
	"github.com/danielededo/cedolo/internal/layout"
	"github.com/danielededo/cedolo/internal/payslip"
	"github.com/danielededo/cedolo/internal/synth"
)

// Labels a payroll package might word in a way cedolo has never seen. The
// blind test prints synthetic payslips with some of them replaced by
// invented words of the same length, in the same places, and counts how
// many totals cedolo still reads right: what it gets from the numbers and
// their layout alone, not from knowing the words (issue #86).

// totalLabels are the labels of the totals each style prints, in the order
// gross, deductions, net, INPS base, IRPEF base, TFR.
var totalLabels = map[synth.Style][]string{
	synth.StyleClassic: {"TOTALE COMPETENZE", "TOTALE TRATTENUTE", "NETTO DEL MESE", "IMPONIBILE INPS", "IMPONIBILE IRPEF", "QUOTA MATURATA NEL MESE"},
	synth.StyleSigned:  {"Totale lordo", "Totale ritenute", "Netto in busta", "Imponibile previdenziale", "Imponibile fiscale", "Quota TFR maturata"},
	synth.StyleGrid:    {"TOTALE SPETTANZE", "TOTALE RITENUTE", "NETTO DEL MESE", "IMPONIBILE INPS", "IMPONIBILE IRPEF", "TFR DEL MESE"},
}

// fields names the totals checked, in the order of totalLabels.
var fields = []string{"gross", "deductions", "net", "INPS base", "IRPEF base", "TFR"}

// blind is one way of hiding labels: which phrases of a payslip it
// replaces.
type blind struct {
	name    string
	phrases func(s synth.Style, p synth.Payslip) []string
}

var blinds = []blind{
	{"none", func(synth.Style, synth.Payslip) []string { return nil }},
	{"gross", only(0)}, {"deductions", only(1)}, {"net", only(2)},
	{"INPS base", only(3)}, {"IRPEF base", only(4)}, {"TFR", only(5)},
	{"all totals", func(s synth.Style, _ synth.Payslip) []string { return totalLabels[s] }},
	{"all items", items},
	{"totals and items", func(s synth.Style, p synth.Payslip) []string {
		return append(append([]string(nil), totalLabels[s]...), items(s, p)...)
	}},
}

func only(i int) func(synth.Style, synth.Payslip) []string {
	return func(s synth.Style, _ synth.Payslip) []string { return totalLabels[s][i : i+1] }
}

func items(_ synth.Style, p synth.Payslip) []string {
	var out []string
	for _, it := range p.Items {
		out = append(out, it.Description)
	}
	return out
}

// blindFloor is how many of the totals and of the items, out of all those
// checked, each way of hiding labels must still read right. It records what
// cedolo reads today, so that a change can only raise it: raise it with the
// change that reads more.
var blindFloor = map[string][2]int{
	"none": {90, 104},
	// The totals' rows read by their arithmetic (ADR 0011, first step).
	"gross":      {90, 104},
	"deductions": {90, 104},
	"net":        {90, 104},
	// The taxable bases and the TFR read by their arithmetic (second step).
	"INPS base":  {90, 104},
	"IRPEF base": {90, 104},
	"TFR":        {90, 104},
	"all totals": {90, 104},
	"all items":  {90, 0},
	// With every label hidden, the gross pay, the deductions and the net pay
	// are read by their arithmetic; the taxable bases and the TFR need the
	// INPS contribution, which is an item, and the items are the third step
	// of ADR 0011.
	"totals and items": {45, 0},
}

func TestBlindLabels(t *testing.T) {
	type sample struct {
		style synth.Style
		slip  synth.Payslip
	}
	var samples []sample
	for _, s := range []synth.Style{synth.StyleClassic, synth.StyleSigned, synth.StyleGrid} {
		samples = append(samples, sample{s, synth.ExampleFor(s)})
	}
	for _, p := range synth.Year(2025) {
		samples = append(samples, sample{synth.StyleClassic, p})
	}
	var report strings.Builder
	fmt.Fprintf(&report, "%-18s %s\n", "hidden", "totals read right, items read right, each total over the payslips")
	for _, b := range blinds {
		right, all := 0, 0
		itemsRight, itemsAll := 0, 0
		perField := make([]int, len(fields))
		for _, sm := range samples {
			doc := relabel(synth.Layout(sm.slip, sm.style), b.phrases(sm.style, sm.slip))
			got := extract.Document(doc)
			want := expected(sm.slip)
			for i, f := range fields {
				all++
				if len(got) == 1 && read(got[0], f) == want[i] {
					right++
					perField[i]++
				}
			}
			// The items: each read with its meaning and amount.
			for _, it := range sm.slip.Items {
				itemsAll++
				if len(got) == 1 && slices.ContainsFunc(got[0].Items, func(g payslip.Item) bool { return g.Key == it.Key && g.Cents == it.Amount }) {
					itemsRight++
				}
			}
		}
		fmt.Fprintf(&report, "%-18s %3d/%-3d items %3d/%-3d", b.name, right, all, itemsRight, itemsAll)
		for i, f := range fields {
			fmt.Fprintf(&report, "  %s %d/%d", f, perField[i], len(samples))
		}
		report.WriteString("\n")
		if floor, ok := blindFloor[b.name]; ok && (right < floor[0] || itemsRight < floor[1]) {
			t.Errorf("%s hidden: %d totals and %d items read right, fewer than the %d and %d before", b.name, right, itemsRight, floor[0], floor[1])
		}
	}
	t.Log("\n" + report.String())
}

func expected(p synth.Payslip) []int64 {
	gross, deductions, net := p.Totals()
	return []int64{gross, deductions, net, p.INPSBase, p.IRPEFBase, p.TFRAccrued}
}

// read returns a total cedolo read, or -1 when it did not find it.
func read(p payslip.Payslip, field string) int64 {
	a := map[string]payslip.Amount{
		"gross": p.Gross, "deductions": p.Deductions, "net": p.Net,
		"INPS base": p.INPSBase, "IRPEF base": p.IRPEFBase, "TFR": p.TFR,
	}[field]
	if !a.Found() {
		return -1
	}
	return a.Cents
}

// relabel replaces each phrase, wherever its words follow one another on a
// line, with invented words of the same lengths, keeping their places.
func relabel(doc layout.Document, phrases []string) layout.Document {
	for pi := range doc.Pages {
		page := &doc.Pages[pi]
		for _, phrase := range phrases {
			words := strings.Fields(phrase)
			for _, l := range page.Lines(2) {
				for s := 0; s+len(words) <= len(l.Tokens); s++ {
					match := true
					for k, w := range words {
						if l.Tokens[s+k].Text != w {
							match = false
							break
						}
					}
					if !match {
						continue
					}
					for k := range words {
						old := l.Tokens[s+k]
						for ti, tok := range page.Tokens {
							if tok == old {
								page.Tokens[ti].Text = invent(tok.Text)
							}
						}
					}
				}
			}
		}
	}
	return doc
}

// invent returns a made-up word as long as w, in the same case, from w
// itself so that the test reads the same every time: consonants and
// vowels in turn, which spells no payroll word.
func invent(w string) string {
	h := fnv.New32a()
	h.Write([]byte(w))
	n := h.Sum32()
	const consonants, vowels = "BDFGKLMPRVZ", "AEIOU"
	var b strings.Builder
	for i, r := range []rune(w) {
		if !unicode.IsLetter(r) {
			b.WriteRune(r)
			continue
		}
		var c byte
		if i%2 == 0 {
			c = consonants[(n+uint32(i)*7)%uint32(len(consonants))]
		} else {
			c = vowels[(n+uint32(i)*3)%uint32(len(vowels))]
		}
		if unicode.IsLower(r) {
			c += 'a' - 'A'
		}
		b.WriteByte(c)
	}
	return b.String()
}
