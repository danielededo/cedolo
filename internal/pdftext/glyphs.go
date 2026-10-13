package pdftext

import (
	"math"
	"strings"
	"unicode"

	"github.com/danielededo/cedolo/internal/layout"
)

// glyph is one character with its box in top-left page coordinates, as
// reported by a PDF library before words are formed.
type glyph struct {
	r           rune
	x0, x1      float64
	top, bottom float64
	size        float64 // font size, used to scale the grouping thresholds
}

// groupWords merges glyphs, in content order, into word tokens. A word ends
// at whitespace, at a change of line, or where the horizontal gap to the
// next glyph is larger than kerning but smaller than a space.
func groupWords(glyphs []glyph) []layout.Token {
	var (
		out  []layout.Token
		cur  strings.Builder
		word glyph // running box of the current word
		last glyph
	)
	flush := func() {
		if cur.Len() == 0 {
			return
		}
		out = append(out, layout.Token{
			Text: cur.String(),
			X:    word.x0,
			Y:    word.top,
			W:    word.x1 - word.x0,
			H:    word.bottom - word.top,
		})
		cur.Reset()
	}
	for _, g := range glyphs {
		if unicode.IsSpace(g.r) {
			flush()
			continue
		}
		if cur.Len() > 0 {
			gap := g.x0 - last.x1
			sameLine := math.Abs(g.bottom-last.bottom) < 0.3*g.size
			if !sameLine || gap > 0.15*g.size || gap < -0.5*g.size {
				flush()
			}
		}
		if cur.Len() == 0 {
			word = g
		} else {
			word.x1 = math.Max(word.x1, g.x1)
			word.top = math.Min(word.top, g.top)
			word.bottom = math.Max(word.bottom, g.bottom)
		}
		cur.WriteRune(g.r)
		last = g
	}
	flush()
	return out
}
