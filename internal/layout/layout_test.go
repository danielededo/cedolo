package layout

import "testing"

func TestLinesGroupsAndSorts(t *testing.T) {
	p := Page{Tokens: []Token{
		{Text: "NETTO", X: 50, Y: 700.4},
		{Text: "1.234,56", X: 400, Y: 700},
		{Text: "LORDO", X: 50, Y: 100},
		{Text: "A", X: 300, Y: 699.8},
	}}
	lines := p.Lines(1)
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2", len(lines))
	}
	if got := lines[0].Text(); got != "LORDO" {
		t.Errorf("first line = %q, want LORDO", got)
	}
	if got := lines[1].Text(); got != "NETTO A 1.234,56" {
		t.Errorf("second line = %q", got)
	}
}

func TestRedactKeepsShapeAndPosition(t *testing.T) {
	d := Document{Pages: []Page{{Number: 1, Tokens: []Token{
		{Text: "Rossi", X: 10, Y: 20, W: 30, H: 9},
		{Text: "1.234,56-", X: 100, Y: 20, W: 40, H: 9},
		{Text: "Città", X: 1, Y: 2, W: 3, H: 4},
	}}}}
	r := d.Redact()
	want := []string{"Xxxxx", "0.000,00-", "Xxxxx"}
	for i, tok := range r.Pages[0].Tokens {
		if tok.Text != want[i] {
			t.Errorf("token %d = %q, want %q", i, tok.Text, want[i])
		}
		orig := d.Pages[0].Tokens[i]
		if tok.X != orig.X || tok.Y != orig.Y || tok.W != orig.W || tok.H != orig.H {
			t.Errorf("token %d moved", i)
		}
	}
	if d.Pages[0].Tokens[0].Text != "Rossi" {
		t.Error("Redact modified its input")
	}
}

// spacedWord lays out a word one character per token, as a letter-spaced
// PDF prints it, and returns the x after it.
func spacedWord(toks []Token, x, y, h, gap float64, word string) ([]Token, float64) {
	for _, r := range word {
		toks = append(toks, Token{Text: string(r), X: x, Y: y, W: 4, H: h})
		x += 4 + gap
	}
	return toks, x
}

func TestJoinSpaced(t *testing.T) {
	var toks []Token
	x := 20.0
	for _, w := range []string{"TOTALE", "LORDO", "NETTO", "BUSTA"} {
		// Letters 0.5 apart, words 3 apart: under and over a fifth of the
		// height.
		toks, x = spacedWord(toks, x, 100, 10, 0.5, w)
		x += 3 - 0.5
	}
	// An amount whose pieces some readers report with wider gaps, sign
	// included.
	for _, s := range []string{"-", "1", ".", "234", ",", "56"} {
		w := 4 * float64(len(s))
		toks = append(toks, Token{Text: s, X: x, Y: 100, W: w, H: 10})
		x += w + 4
	}
	got := JoinSpaced(Document{Pages: []Page{{Tokens: toks}}}).Pages[0].Tokens
	var words []string
	for _, tok := range got {
		words = append(words, tok.Text)
	}
	want := []string{"TOTALE", "LORDO", "NETTO", "BUSTA", "-1.234,56"}
	if len(words) != len(want) {
		t.Fatalf("words = %q, want %q", words, want)
	}
	for i := range want {
		if words[i] != want[i] {
			t.Errorf("words = %q, want %q", words, want)
			break
		}
	}
	if w := got[0]; w.X != 20 || w.W != 6*4+5*0.5 {
		t.Errorf("TOTALE at %v width %v", w.X, w.W)
	}
}

func TestJoinSpacedLeavesOtherPages(t *testing.T) {
	// Cells that touch on a page of whole words stay apart.
	var toks []Token
	for i := 0; i < 30; i++ {
		toks = append(toks, Token{Text: "12,00", X: float64(i) * 20, Y: 100, W: 20, H: 10})
	}
	got := JoinSpaced(Document{Pages: []Page{{Tokens: toks}}}).Pages[0].Tokens
	if len(got) != len(toks) {
		t.Errorf("got %d tokens, want %d", len(got), len(toks))
	}
}
