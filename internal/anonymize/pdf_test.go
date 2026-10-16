package anonymize

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danielededo/cedolo/internal/extract"
	"github.com/danielededo/cedolo/internal/layout"
	"github.com/danielededo/cedolo/internal/pdftext"
	"github.com/danielededo/cedolo/internal/synth"
)

// TestRebuiltPDFKeepsTheArithmetic anonymizes every synthetic style keeping
// amounts, writes the rebuilt PDF, reads it back with each text layer and
// checks that extraction gives the same payslip as the original, while the
// file holds none of the personal data and nothing but text.
func TestRebuiltPDFKeepsTheArithmetic(t *testing.T) {
	for _, s := range synth.Styles() {
		in := synth.Layout(synth.ExampleFor(s), s)
		anon, _ := Layout(in, Options{KeepAmounts: true})
		if leaks := Leaks(in, anon); len(leaks) > 0 {
			t.Fatalf("%s: leaks %v", s, leaks)
		}
		var pdf bytes.Buffer
		if err := WritePDF(&pdf, anon); err != nil {
			t.Fatal(err)
		}
		for _, secret := range []string{"BIANCHI", "ANNA", "ESEMPIO", "SINTETICO", "000042", "/Info", "/Image", "/Filter", "/FontFile", "/Metadata"} {
			if bytes.Contains(pdf.Bytes(), []byte(secret)) {
				t.Errorf("%s: rebuilt PDF contains %q", s, secret)
			}
		}
		path := filepath.Join(t.TempDir(), "anon.pdf")
		if err := os.WriteFile(path, pdf.Bytes(), 0o600); err != nil {
			t.Fatal(err)
		}
		want := extract.Document(in)[0]
		for _, name := range pdftext.Names() {
			r, _ := pdftext.ByName(name)
			doc, err := r.ReadFile(path)
			if err != nil {
				if name == "poppler" {
					continue // pdftotext not installed; CI has it
				}
				t.Fatalf("%s/%s: %v", s, name, err)
			}
			if got, want := tokenTexts(doc), tokenTexts(anon); got != want {
				t.Errorf("%s/%s: read back\n%s\nwant\n%s", s, name, got, want)
			}
			slips := extract.Document(doc)
			if len(slips) != 1 {
				t.Fatalf("%s/%s: %d payslips", s, name, len(slips))
			}
			got := slips[0]
			for _, c := range []struct {
				name      string
				got, want int64
				verified  bool
			}{
				{"gross", got.Gross.Cents, want.Gross.Cents, got.Gross.Status == want.Gross.Status},
				{"deductions", got.Deductions.Cents, want.Deductions.Cents, got.Deductions.Status == want.Deductions.Status},
				{"net", got.Net.Cents, want.Net.Cents, got.Net.Status == want.Net.Status},
			} {
				if c.got != c.want || !c.verified {
					t.Errorf("%s/%s: %s %d, want %d (same status: %v)", s, name, c.name, c.got, c.want, c.verified)
				}
			}
			if got.Year != want.Year || got.Month != want.Month {
				t.Errorf("%s/%s: period %d/%d, want %d/%d", s, name, got.Month, got.Year, want.Month, want.Year)
			}
			if len(got.Items) != len(want.Items) {
				t.Fatalf("%s/%s: %d items, want %d", s, name, len(got.Items), len(want.Items))
			}
			for i := range want.Items {
				if got.Items[i].Key != want.Items[i].Key || got.Items[i].Cents != want.Items[i].Cents {
					t.Errorf("%s/%s: item %d %s %d, want %s %d", s, name, i, got.Items[i].Key, got.Items[i].Cents, want.Items[i].Key, want.Items[i].Cents)
				}
			}
		}
	}
}

func tokenTexts(d layout.Document) string {
	var lines []string
	for _, p := range d.Pages {
		for _, l := range p.Lines(2) {
			lines = append(lines, l.Text())
		}
	}
	return strings.Join(lines, "\n")
}

func TestKeepAmounts(t *testing.T) {
	for text, kept := range map[string]bool{
		"1.234,56": true, "168,00": true, "9,19%": true, "-12,50": true, "12,50-": true, "0,00": true,
		"2026": true, "10/2026": true, "1/2026": true,
		"01/10/2026": false, "000042": false, "20121": false, "3331234567": false, "B2": false, "1.234": false, "13/2026": false,
	} {
		in := layout.Document{Pages: []layout.Page{{Tokens: []layout.Token{{Text: text}}}}}
		out, _ := Layout(in, Options{KeepAmounts: true})
		if (out.Pages[0].Tokens[0].Text == text) != kept {
			t.Errorf("%q kept = %v, want %v", text, !kept, kept)
		}
		out, _ = Layout(in, Options{})
		if out.Pages[0].Tokens[0].Text == text && strings.ContainsAny(text, "123456789") {
			t.Errorf("%q kept without KeepAmounts", text)
		}
	}
}

func TestLeaks(t *testing.T) {
	in := layout.Document{Pages: []layout.Page{{Tokens: []layout.Token{{Text: "ROSSI"}, {Text: "INPS-2025"}, {Text: "INPS"}}}}}
	anon, _ := Layout(in, Options{})
	if leaks := Leaks(in, anon); len(leaks) != 0 {
		t.Errorf("false leaks %v", leaks)
	}
	anon.Pages[0].Tokens[0].Text = "Rossi" // as if a mask had failed
	if leaks := Leaks(in, anon); len(leaks) != 1 || leaks[0] != "rossi" {
		t.Errorf("leaks = %v", leaks)
	}
}
