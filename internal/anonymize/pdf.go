package anonymize

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/danielededo/cedolo/internal/layout"
)

// Vertical extent of a glyph box as a fraction of the font size, as text
// layers report it for the standard fonts; see internal/synth.
const (
	ascent  = 0.718
	descent = 0.207
	// Every Courier glyph is 600 thousandths of an em wide, so a token is
	// scaled to its original width without a width table per glyph.
	glyph = 0.6
)

// WritePDF writes doc as a new PDF that holds nothing but its text: each
// token drawn in Courier, a standard font that is not embedded, at its
// position and scaled to its width. There are no images, metadata or
// compression, so the bytes can be checked for what they contain. The
// document should already be anonymized; WritePDF does not mask anything.
func WritePDF(w io.Writer, doc layout.Document) error {
	widths := strings.TrimSpace(strings.Repeat("600 ", 224))
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"", // page tree, below
		"<< /Type /Font /Subtype /Type1 /BaseFont /Courier /Encoding /WinAnsiEncoding /FirstChar 32 /LastChar 255 /Widths [" + widths + "] >>",
	}
	var kids []string
	for _, p := range doc.Pages {
		var content bytes.Buffer
		for _, t := range p.Tokens {
			draw(&content, p.Height, t)
		}
		n := len(objects) + 1
		kids = append(kids, fmt.Sprintf("%d 0 R", n))
		objects = append(objects,
			fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %.2f %.2f] /Resources << /Font << /F1 3 0 R >> >> /Contents %d 0 R >>", p.Width, p.Height, n+1),
			fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", content.Len(), content.Bytes()))
	}
	objects[1] = fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", strings.Join(kids, " "), len(kids))

	var b bytes.Buffer
	b.WriteString("%PDF-1.4\n%\xe2\xe3\xcf\xd3\n")
	offsets := make([]int, len(objects))
	for i, obj := range objects {
		offsets[i] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for _, off := range offsets {
		fmt.Fprintf(&b, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	_, err := w.Write(b.Bytes())
	return err
}

// draw writes one token. A box much taller than wide with several
// characters is text printed upward, as some payslips print block titles.
func draw(c *bytes.Buffer, pageHeight float64, t layout.Token) {
	n := utf8.RuneCountInString(t.Text)
	if n == 0 || t.W <= 0 || t.H <= 0 {
		return
	}
	if n >= 2 && t.H > 1.5*t.W {
		size := t.W / (ascent + descent)
		fmt.Fprintf(c, "BT /F1 %.2f Tf %.2f Tz 0 1 -1 0 %.2f %.2f Tm (%s) Tj ET\n",
			size, 100*t.H/(float64(n)*glyph*size), t.X+ascent*size, pageHeight-t.Y-t.H, pdfString(t.Text))
		return
	}
	size := t.H / (ascent + descent)
	fmt.Fprintf(c, "BT /F1 %.2f Tf %.2f Tz 1 0 0 1 %.2f %.2f Tm (%s) Tj ET\n",
		size, 100*t.W/(float64(n)*glyph*size), t.X, pageHeight-t.Y-ascent*size, pdfString(t.Text))
}

// pdfString encodes s in WinAnsiEncoding and escapes it for a PDF literal
// string. Characters outside the encoding become "?".
func pdfString(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '(' || r == ')' || r == '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case r >= 32 && r < 127:
			b.WriteRune(r)
		case r == '€':
			b.WriteString(`\200`)
		case r >= 0xA0 && r <= 0xFF:
			fmt.Fprintf(&b, "\\%03o", r)
		default:
			b.WriteByte('?')
		}
	}
	return b.String()
}
