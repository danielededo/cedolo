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

// occurrence is one match of a phrase in a line.
type occurrence struct {
	first, last int // token indexes
	afterWord   int // index of the first word after the phrase
}

// findAll returns every occurrence of phrase in the line that is a complete
// label: the word after it is not one that turns it into a longer label with
// another meaning (see labelContinuations).
func (l line) findAll(phrase string) []occurrence {
	pw := strings.Fields(phrase)
	var out []occurrence
	for i := 0; i+len(pw) <= len(l.words); i++ {
		match := true
		for k := range pw {
			if l.words[i+k].text != pw[k] {
				match = false
				break
			}
		}
		if !match {
			continue
		}
		after := i + len(pw)
		// A word repeating the label's last word starts a new label
		// ("IMPONIBILE IRPEF" next to "IRPEF LORDA").
		if after < len(l.words) && labelContinuations[l.words[after].text] && l.words[after].text != pw[len(pw)-1] &&
			l.continues(l.words[after-1].tok, l.words[after].tok) {
			continue
		}
		out = append(out, occurrence{first: l.words[i].tok, last: l.words[after-1].tok, afterWord: after})
	}
	return out
}

// continues reports whether token b follows token a closely enough to be part
// of the same label: the same token, or the next one after a gap narrower
// than a few characters. Labels printed side by side in a grid of cells are
// further apart.
func (l line) continues(a, b int) bool {
	if a == b {
		return true
	}
	return l.toks[b].X-l.toks[a].Right() < wordGap*l.toks[a].H
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
// to the start of the next token on the same line that is not punctuation
// closing the label, such as "+/-", or to the page edge.
func (l line) cell(first, last int, pageWidth float64) (x0, x1 float64) {
	x0 = l.toks[first].X - cellMargin
	x1 = pageWidth
	for k := last + 1; k < len(l.toks); k++ {
		if !isFiller(l.toks[k].Text) {
			x1 = l.toks[k].X - cellMargin
			break
		}
	}
	return x0, x1
}

func center(t layout.Token) float64 { return t.X + t.W/2 }
