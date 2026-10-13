package synth

import (
	"bytes"
	"strings"
	"testing"
)

func TestExampleTotals(t *testing.T) {
	gross, deductions, net := Example().Totals()
	if gross != 219450 {
		t.Errorf("gross = %d, want 219450", gross)
	}
	if deductions != 52710 {
		t.Errorf("deductions = %d, want 52710", deductions)
	}
	if net != gross-deductions {
		t.Errorf("net = %d, want gross - deductions", net)
	}
}

func TestGridStyleRoundsNet(t *testing.T) {
	_, _, net := ExampleFor(StyleGrid).Totals()
	if net%100 != 0 {
		t.Errorf("grid style net = %d, want whole euros", net)
	}
}

func TestEveryItemHasKey(t *testing.T) {
	for _, s := range Styles() {
		for _, it := range ExampleFor(s).Items {
			if it.Key == "" || it.Description == "" {
				t.Errorf("%s: incomplete item %+v", s, it)
			}
		}
	}
}

func TestWritePDF(t *testing.T) {
	for _, s := range Styles() {
		var b bytes.Buffer
		if err := WritePDF(&b, ExampleFor(s), s); err != nil {
			t.Fatal(err)
		}
		if !bytes.HasPrefix(b.Bytes(), []byte("%PDF-1.4")) || !bytes.HasSuffix(b.Bytes(), []byte("%%EOF\n")) {
			t.Errorf("%s: output is not a complete PDF", s)
		}
	}
}

func TestLayoutSplitsWords(t *testing.T) {
	doc := Layout(Example(), StyleClassic)
	var line string
	for _, l := range doc.Pages[0].Lines(2) {
		if strings.HasPrefix(l.Text(), "0250") {
			line = l.Text()
		}
	}
	if line != "0250 STRAORDINARIO 25% 10,00 14,45 144,50" {
		t.Errorf("overtime line = %q", line)
	}
}

func TestParseStyle(t *testing.T) {
	for _, s := range Styles() {
		got, err := ParseStyle(s.String())
		if err != nil || got != s {
			t.Errorf("ParseStyle(%q) = %v, %v", s, got, err)
		}
	}
	if _, err := ParseStyle("zucchetti"); err == nil {
		t.Error("ParseStyle accepted an unknown style")
	}
}

func TestPDFString(t *testing.T) {
	if got := pdfString(`A (b) \ È`); got != `A \(b\) \\ \310` {
		t.Errorf("pdfString = %q", got)
	}
}
