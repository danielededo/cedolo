package synth

import (
	"strings"

	"github.com/danielededo/cedolo/internal/layout"
)

// Vertical extent of a Helvetica glyph box as a fraction of the font size,
// matching what poppler reports for the PDF version of the same page.
const (
	ascent  = 0.718
	descent = 0.207
)

// Layout renders p in the given style directly as a text layout, without
// going through PDF. Tokens have the same text and boxes a PDF reader
// recovers from WritePDF's output, so extraction can be tested quickly and
// independently of the text layer.
func Layout(p Payslip, s Style) layout.Document {
	page := layout.Page{Number: 1, Width: pageWidth, Height: pageHeight}
	for _, t := range render(p, s).texts {
		x := t.x
		space := textWidth(" ", t.size)
		for i, word := range strings.Split(t.s, " ") {
			if i > 0 {
				x += space
			}
			w := textWidth(word, t.size)
			if word != "" {
				page.Tokens = append(page.Tokens, layout.Token{
					Text: word,
					X:    x,
					Y:    pageHeight - t.y - ascent*t.size,
					W:    w,
					H:    (ascent + descent) * t.size,
				})
			}
			x += w
		}
	}
	return layout.Document{Pages: []layout.Page{page}}
}
