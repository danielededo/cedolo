package pdftext

import "testing"

func TestGroupWords(t *testing.T) {
	g := func(r rune, x, top float64) glyph {
		return glyph{r: r, x0: x, x1: x + 5, top: top, bottom: top + 10, size: 10}
	}
	glyphs := []glyph{
		g('N', 0, 0), g('E', 5, 0), g('T', 10, 0), // NET
		g('T', 15.5, 0), g('O', 20.5, 0), // kerning gap of 0.5: same word
		{r: ' '},
		g('1', 100, 0), g(',', 105, 0), g('0', 110, 0), g('0', 115, 0),
		g('A', 0, 20), // next line
		g('B', 9, 20), // gap of 4: new word
	}
	got := groupWords(glyphs)
	want := []string{"NETTO", "1,00", "A", "B"}
	if len(got) != len(want) {
		t.Fatalf("got %d tokens: %+v", len(got), got)
	}
	for i, w := range want {
		if got[i].Text != w {
			t.Errorf("token %d = %q, want %q", i, got[i].Text, w)
		}
	}
	if got[0].X != 0 || got[0].W != 25.5 || got[0].H != 10 {
		t.Errorf("NETTO box = %+v", got[0])
	}
}
