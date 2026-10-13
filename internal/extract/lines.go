package extract

import (
	"math"
	"sort"
	"strings"

	"github.com/danielededo/cedolo/internal/layout"
	"github.com/danielededo/cedolo/internal/taxonomy"
)

// line is a row of tokens with their normalized words, so that labels can be
// matched regardless of case, accents and punctuation ("Add.le" and "ADD LE"
// are the same two words).
type line struct {
	page  int
	y     float64 // top of the line
	h     float64 // tallest token height
	toks  []layout.Token
	words []word
}

type word struct {
	text string
	tok  int // index into toks
}

// lineTolerance is the largest difference, in points, between the tops of two
// tokens on the same printed line.
const lineTolerance = 2.5

func buildLines(p layout.Page) []line {
	var out []line
	for _, l := range p.Lines(lineTolerance) {
		ln := line{page: p.Number, y: l.Y, toks: l.Tokens}
		for i, t := range l.Tokens {
			ln.h = math.Max(ln.h, t.H)
			for _, w := range strings.Fields(taxonomy.Normalize(t.Text)) {
				ln.words = append(ln.words, word{text: w, tok: i})
			}
		}
		out = append(out, ln)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].y < out[j].y })
	return out
}

// find returns the indexes of the first and last tokens of the first
// occurrence of phrase (normalized words separated by spaces) in the line.
func (l line) find(phrase string) (first, last int, ok bool) {
	pw := strings.Fields(phrase)
	for i := 0; i+len(pw) <= len(l.words); i++ {
		match := true
		for k := range pw {
			if l.words[i+k].text != pw[k] {
				match = false
				break
			}
		}
		if match {
			return l.words[i].tok, l.words[i+len(pw)-1].tok, true
		}
	}
	return 0, 0, false
}

// has reports whether the line contains any of the phrases.
func (l line) has(phrases ...string) bool {
	for _, p := range phrases {
		if _, _, ok := l.find(p); ok {
			return true
		}
	}
	return false
}

// cell returns the horizontal extent of the cell a label occupies when labels
// are laid out in a row with their values below: from the label's left edge
// to the start of the next token on the same line, or to the page edge.
func (l line) cell(first, last int, pageWidth float64) (x0, x1 float64) {
	x0 = l.toks[first].X - 2
	x1 = pageWidth
	if last+1 < len(l.toks) {
		x1 = l.toks[last+1].X - 2
	}
	return x0, x1
}

func center(t layout.Token) float64 { return t.X + t.W/2 }
