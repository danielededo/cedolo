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

func TestGroupWordsVertical(t *testing.T) {
	// "FERIE" printed upward in a column, next to a horizontal value.
	var glyphs []glyph
	for i, r := range "FERIE" {
		top := 100 - float64(i+1)*6
		glyphs = append(glyphs, glyph{r: r, x0: 20, x1: 27, top: top, bottom: top + 6, size: 8, vertical: true})
	}
	glyphs = append(glyphs, glyph{r: '9', x0: 40, x1: 44, top: 90, bottom: 98, size: 8})
	got := groupWords(glyphs)
	if len(got) != 2 || got[0].Text != "FERIE" || got[1].Text != "9" {
		t.Fatalf("got %+v", got)
	}
	if got[0].H < 25 || got[0].W > 8 {
		t.Errorf("vertical word box = %+v", got[0])
	}
}
