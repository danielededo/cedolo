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
