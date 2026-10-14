package extract

import (
	"encoding/binary"
	"math"
	"testing"
	"time"

	"github.com/danielededo/cedolo/internal/layout"
	"github.com/danielededo/cedolo/internal/synth"
)

// FuzzDocument feeds the extractor layouts built from random bytes, seeded
// with real synthetic pages, and requires that it never panics or loops:
// any PDF can reach it.
func FuzzDocument(f *testing.F) {
	for _, s := range synth.Styles() {
		f.Add(encode(synth.Layout(synth.ExampleFor(s), s)))
	}
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, data []byte) {
		done := make(chan struct{})
		go func() {
			defer close(done)
			Document(decode(data))
		}()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Fatalf("Document took over 10s on %d tokens", len(data)/5)
		}
	})
}

// Vocabulary the decoder draws words from, so that random input reaches the
// label and amount paths, not only unknown words.
var fuzzWords = []string{"NETTO", "TOTALE", "COMPETENZE", "TRATTENUTE", "IRPEF", "INPS", "IMPONIBILE",
	"FERIE", "PERMESSI", "ROL", "RESIDUO", "MATURATO", "GODUTO", "VOCE", "DESCRIZIONE", "ORE", "CCNL",
	"LIVELLO", "OTTOBRE", "2026", "10/2026", "1.234,56", "-12,00", "0,00", "100,00%", "9,19%", "T.F.R.",
	"ARROTONDAMENTO", "PRECEDENTE", "ATTUALE", "MESE", "DEL", "A", "PAGARE", "€", "H011", "13A", "MENSILITA"}

// encode turns a layout into bytes the decoder reads back, so seeds start
// from realistic pages.
func encode(d layout.Document) []byte {
	var b []byte
	for _, p := range d.Pages {
		for _, t := range p.Tokens {
			w := 0
			for i, fw := range fuzzWords {
				if fw == t.Text {
					w = i
				}
			}
			b = append(b, byte(w), byte(t.X/3), byte(t.Y/4), byte(t.W/2), byte(t.H))
		}
	}
	return b
}

// decode reads tokens of five bytes: word, x, y, width, height; a 0xFF
// word starts a new page.
func decode(data []byte) layout.Document {
	page := layout.Page{Number: 1, Width: 595, Height: 842}
	var doc layout.Document
	for len(data) >= 5 {
		c := data[:5]
		data = data[5:]
		if c[0] == 0xFF {
			doc.Pages = append(doc.Pages, page)
			page = layout.Page{Number: page.Number + 1, Width: 595, Height: 842}
			continue
		}
		x := float64(c[1]) * 3
		if c[1] == 0xFE {
			x = math.Float64frombits(uint64(binary.LittleEndian.Uint32(c[1:5]))) // odd values too
		}
		page.Tokens = append(page.Tokens, layout.Token{
			Text: fuzzWords[int(c[0])%len(fuzzWords)],
			X:    x, Y: float64(c[2]) * 4, W: float64(c[3]) / 2, H: float64(c[4] % 20),
		})
	}
	return layout.Document{Pages: append(doc.Pages, page)}
}
