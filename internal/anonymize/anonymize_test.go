package anonymize

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/danielededo/cedolo/internal/extract"
	"github.com/danielededo/cedolo/internal/layout"
	"github.com/danielededo/cedolo/internal/pii"
	"github.com/danielededo/cedolo/internal/synth"
)

func texts(d layout.Document) string {
	var parts []string
	for _, p := range d.Pages {
		for _, t := range p.Tokens {
			parts = append(parts, t.Text)
		}
	}
	return strings.Join(parts, " ")
}

func TestPersonalDataIsMasked(t *testing.T) {
	for _, s := range synth.Styles() {
		p := synth.ExampleFor(s)
		in := synth.Layout(p, s)
		out, st := Layout(in, Options{})
		all := texts(out)
		for _, secret := range []string{"BIANCHI", "ANNA", "ESEMPIO", "SINTETICO", "000042", "2.194,50", "1.667,40", "2026"} {
			if strings.Contains(all, secret) {
				t.Errorf("%s: anonymized layout leaks %q", s, secret)
			}
		}
		if strings.ContainsAny(all, "123456789") {
			t.Errorf("%s: a non-zero digit survived", s)
		}
		if st.Kept == 0 || st.Masked == 0 || st.Kept+st.Masked != in.TokenCount() {
			t.Errorf("%s: stats %+v for %d tokens", s, st, in.TokenCount())
		}
		if in.Pages[0].Tokens[0] == out.Pages[0].Tokens[0] && !keep(in.Pages[0].Tokens[0].Text, vocabulary()) {
			t.Errorf("%s: input modified or first token not masked", s)
		}
	}
}

// TestStructureSurvives checks that the anonymized layout is still useful:
// extraction finds the same items with the same categories, and the same
// leave balances, only with masked values.
func TestStructureSurvives(t *testing.T) {
	for _, s := range synth.Styles() {
		p := synth.ExampleFor(s)
		in := synth.Layout(p, s)
		out, _ := Layout(in, Options{})
		want := extract.Document(in)[0]
		got := extract.Document(out)
		if len(got) != 1 {
			t.Fatalf("%s: got %d payslips", s, len(got))
		}
		if len(got[0].Items) != len(want.Items) {
			t.Fatalf("%s: %d items after anonymizing, %d before", s, len(got[0].Items), len(want.Items))
		}
		for i := range want.Items {
			if got[0].Items[i].Key != want.Items[i].Key || got[0].Items[i].Side != want.Items[i].Side {
				t.Errorf("%s: item %d is %s/%s after anonymizing, %s/%s before (%q)", s, i,
					got[0].Items[i].Key, got[0].Items[i].Side, want.Items[i].Key, want.Items[i].Side, got[0].Items[i].Description)
			}
		}
		if len(got[0].Leave) != len(want.Leave) {
			t.Errorf("%s: %d leave balances after anonymizing, %d before", s, len(got[0].Leave), len(want.Leave))
		}
	}
}

func TestSensitiveWordsAndIdentifiersAreMasked(t *testing.T) {
	body := "RSSMRA80A01H501"
	cf := body + string(pii.CodiceFiscaleCheck(body))
	bban := "X0542811101000000123456"
	iban := "IT" + pii.IBANCheckDigits("IT", bban) + bban
	in := layout.Document{Pages: []layout.Page{{Number: 1, Tokens: []layout.Token{
		{Text: "MALATTIA"}, {Text: "INPS"}, {Text: "Trattenuta"}, {Text: "sindacale"},
		{Text: "Pignoramento"}, {Text: "Via"}, {Text: "Roma"}, {Text: cf}, {Text: iban}, {Text: "Netto"}, {Text: "a"}, {Text: "pagare"},
	}}}}
	out, _ := Layout(in, Options{})
	all := texts(out)
	for _, leak := range []string{"MALATTIA", "sindacale", "Pignoramento", "Via", "Roma"} {
		if strings.Contains(all, leak) {
			t.Errorf("leaked %q in %q", leak, all)
		}
	}
	for _, kept := range []string{"INPS", "Trattenuta", "Netto a pagare"} {
		if !strings.Contains(all, kept) {
			t.Errorf("lost %q in %q", kept, all)
		}
	}
	data, _ := json.Marshal(out)
	if m := pii.Find(string(data)); len(m) > 0 {
		t.Errorf("identifiers survived: %+v", m)
	}
}
