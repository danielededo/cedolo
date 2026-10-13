package synth

import (
	"bytes"
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

func TestWritePDF(t *testing.T) {
	var b bytes.Buffer
	if err := WritePDF(&b, Example()); err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(b.Bytes(), []byte("%PDF-1.4")) || !bytes.HasSuffix(b.Bytes(), []byte("%%EOF\n")) {
		t.Error("output is not a complete PDF")
	}
}

func TestPDFString(t *testing.T) {
	if got := pdfString(`A (b) \ È`); got != `A \(b\) \\ \310` {
		t.Errorf("pdfString = %q", got)
	}
}
