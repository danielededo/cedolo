package synth

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/danielededo/cedolo/internal/money"
)

// A4 page size in points.
const (
	pageWidth  = 595.0
	pageHeight = 842.0
)

// Column positions of the payslip body. Amount columns are right-aligned on
// these edges, as payroll software prints them.
const (
	colCode        = 40.0
	colDescription = 80.0
	colQuantityR   = 330.0
	colBaseR       = 400.0
	colEarningsR   = 480.0
	colDeductionsR = 555.0
)

var italianMonths = [...]string{"", "GENNAIO", "FEBBRAIO", "MARZO", "APRILE", "MAGGIO", "GIUGNO",
	"LUGLIO", "AGOSTO", "SETTEMBRE", "OTTOBRE", "NOVEMBRE", "DICEMBRE"}

// WritePDF renders p as a one-page PDF with a text layer, using the standard
// Helvetica fonts with an explicit width table so that text positions are
// well defined for every reader.
func WritePDF(w io.Writer, p Payslip) error {
	c := &canvas{}
	c.text(colCode, 800, 14, true, "CEDOLINO PAGA")
	c.text(colCode, 780, 9, true, p.Employer)
	c.text(400, 780, 9, false, "PERIODO")
	c.text(450, 780, 9, true, fmt.Sprintf("%s %d", italianMonths[p.Month], p.Year))

	c.text(colCode, 750, 8, false, "DIPENDENTE")
	c.text(colCode, 738, 9, true, p.Employee)
	c.text(250, 750, 8, false, "MATRICOLA")
	c.text(250, 738, 9, false, p.EmployeeNumber)
	c.text(330, 750, 8, false, "QUALIFICA")
	c.text(330, 738, 9, false, p.Qualification)
	c.text(410, 750, 8, false, "LIVELLO")
	c.text(410, 738, 9, false, p.Level)
	c.text(colCode, 722, 8, false, "CCNL")
	c.text(colDescription, 722, 9, false, p.CCNL)

	y := 695.0
	c.text(colCode, y, 8, true, "COD")
	c.text(colDescription, y, 8, true, "DESCRIZIONE")
	c.textRight(colQuantityR, y, 8, true, "QUANTITÀ")
	c.textRight(colBaseR, y, 8, true, "BASE")
	c.textRight(colEarningsR, y, 8, true, "COMPETENZE")
	c.textRight(colDeductionsR, y, 8, true, "TRATTENUTE")
	c.rule(colCode, y-4, colDeductionsR)

	y -= 18
	for _, it := range p.Items {
		c.text(colCode, y, 9, false, it.Code)
		c.text(colDescription, y, 9, false, it.Description)
		if it.Quantity != "" {
			c.textRight(colQuantityR, y, 9, false, it.Quantity)
		}
		if it.Base != "" {
			c.textRight(colBaseR, y, 9, false, it.Base)
		}
		col := colEarningsR
		if it.Deduction {
			col = colDeductionsR
		}
		c.textRight(col, y, 9, false, money.Format(it.Amount))
		y -= 14
	}

	gross, deductions, net := p.Totals()
	y -= 6
	c.rule(colCode, y+10, colDeductionsR)
	c.text(colDescription, y, 9, true, "TOTALE COMPETENZE")
	c.textRight(colEarningsR, y, 9, true, money.Format(gross))
	y -= 14
	c.text(colDescription, y, 9, true, "TOTALE TRATTENUTE")
	c.textRight(colDeductionsR, y, 9, true, money.Format(deductions))
	y -= 20
	c.text(colDescription, y, 11, true, "NETTO DEL MESE")
	c.textRight(colDeductionsR, y, 11, true, money.Format(net))

	y -= 40
	c.text(colCode, y, 8, true, "TFR")
	c.text(colDescription, y, 8, false, "QUOTA MATURATA NEL MESE")
	c.textRight(colEarningsR, y, 8, false, money.Format(p.TFRAccrued))

	y -= 24
	c.text(colCode, y, 8, true, "ORE")
	c.textRight(330, y, 8, true, "MATURATE")
	c.textRight(400, y, 8, true, "GODUTE")
	c.textRight(480, y, 8, true, "RESIDUE")
	for _, l := range p.Leave {
		y -= 12
		c.text(colCode, y, 8, false, l.Kind)
		c.textRight(330, y, 8, false, l.Accrued)
		c.textRight(400, y, 8, false, l.Used)
		c.textRight(480, y, 8, false, l.Remaining)
	}

	c.text(colCode, 40, 7, false, fmt.Sprintf("Documento sintetico generato da cedolo il %s. Dati inventati.",
		time.Date(p.Year, p.Month, 27, 0, 0, 0, 0, time.UTC).Format("02/01/2006")))

	_, err := w.Write(c.document())
	return err
}

type canvas struct {
	content bytes.Buffer
}

func (c *canvas) text(x, y, size float64, bold bool, s string) {
	font := "F1"
	if bold {
		font = "F2"
	}
	fmt.Fprintf(&c.content, "BT /%s %.1f Tf 1 0 0 1 %.2f %.2f Tm (%s) Tj ET\n", font, size, x, y, pdfString(s))
}

func (c *canvas) textRight(right, y, size float64, bold bool, s string) {
	c.text(right-textWidth(s, size), y, size, bold, s)
}

func (c *canvas) rule(x1, y, x2 float64) {
	fmt.Fprintf(&c.content, "0.5 w %.2f %.2f m %.2f %.2f l S\n", x1, y, x2, y)
}

func (c *canvas) document() []byte {
	widths := make([]string, 0, 224)
	for code := 32; code <= 255; code++ {
		widths = append(widths, fmt.Sprint(glyphWidth(byte(code))))
	}
	font := func(name string) string {
		return fmt.Sprintf("<< /Type /Font /Subtype /Type1 /BaseFont /%s /Encoding /WinAnsiEncoding /FirstChar 32 /LastChar 255 /Widths [%s] >>",
			name, strings.Join(widths, " "))
	}
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %g %g] /Resources << /Font << /F1 4 0 R /F2 5 0 R >> >> /Contents 6 0 R >>", pageWidth, pageHeight),
		font("Helvetica"),
		font("Helvetica-Bold"),
		fmt.Sprintf("<< /Length %d >>\nstream\n%sendstream", c.content.Len(), c.content.String()),
	}
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
	return b.Bytes()
}

// pdfString encodes s in WinAnsiEncoding, which matches Latin-1 for the
// accented letters Italian needs, and escapes it for a PDF literal string.
func pdfString(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '(' || r == ')' || r == '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case r >= 32 && r < 127:
			b.WriteRune(r)
		case r >= 0xA0 && r <= 0xFF:
			fmt.Fprintf(&b, "\\%03o", r)
		default:
			b.WriteByte('?')
		}
	}
	return b.String()
}

func textWidth(s string, size float64) float64 {
	w := 0
	for _, r := range s {
		if r > 0xFF {
			r = '?'
		}
		w += glyphWidth(byte(r))
	}
	return float64(w) * size / 1000
}

// helveticaASCII holds Helvetica advance widths for codes 32 to 126.
var helveticaASCII = [...]int{
	278, 278, 355, 556, 556, 889, 667, 191, 333, 333, 389, 584, 278, 333, 278, 278,
	556, 556, 556, 556, 556, 556, 556, 556, 556, 556, 278, 278, 584, 584, 584, 556,
	1015, 667, 667, 722, 722, 667, 611, 778, 722, 278, 500, 667, 556, 833, 722, 778,
	667, 778, 722, 667, 611, 722, 667, 944, 667, 667, 611, 278, 278, 278, 469, 556,
	333, 556, 556, 500, 556, 556, 278, 556, 556, 222, 222, 500, 222, 833, 556, 556,
	556, 556, 333, 500, 278, 556, 500, 722, 500, 500, 500, 334, 260, 334, 584,
}

// glyphWidth returns the advance width of a WinAnsi code in thousandths of
// an em. The same table is written into the PDF for both the regular and the
// bold font: readers and viewers position glyphs from the table, so text
// positions stay exact even though real Helvetica Bold is slightly wider.
func glyphWidth(code byte) int {
	switch {
	case code >= 32 && code <= 126:
		return helveticaASCII[code-32]
	case code >= 0xC0 && code <= 0xDD: // accented capitals
		return 667
	case code == 0xEC || code == 0xED || code == 0xEE || code == 0xEF: // accented i
		return 278
	default:
		return 556
	}
}
