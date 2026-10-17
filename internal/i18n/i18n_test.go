package i18n

import (
	"strings"
	"testing"

	"github.com/danielededo/cedolo/internal/taxonomy"
)

func TestCataloguesMatch(t *testing.T) {
	for _, k := range Keys(EN) {
		if _, ok := catalog[IT][k]; !ok {
			t.Errorf("Italian catalogue lacks %q", k)
		}
	}
	for _, k := range Keys(IT) {
		if _, ok := catalog[EN][k]; !ok {
			t.Errorf("English catalogue lacks %q", k)
		}
	}
	for _, l := range []Lang{EN, IT} {
		for k, v := range catalog[l] {
			if strings.Count(v, "%") != strings.Count(catalog[EN][k], "%") {
				t.Errorf("%s %q has different placeholders than English", l, k)
			}
		}
	}
}

func TestT(t *testing.T) {
	if got := T(IT, "change.level", "3", "4"); got != "Il tuo livello ora è 4, prima era 3." {
		t.Errorf("T = %q", got)
	}
	if got := T(EN, "no.such.key"); got != "no.such.key" {
		t.Errorf("missing key = %q", got)
	}
	if Month(IT, 10) != "ottobre" || Month(EN, 10) != "October" || Month(EN, 0) != "?" {
		t.Error("Month")
	}
	if ItemLabel(IT, "tax.irpef") != "IRPEF" || ItemLabel(EN, "earnings.overtime") != "Overtime" {
		t.Error("ItemLabel")
	}
	if l, err := Parse("en"); err != nil || l != EN {
		t.Error("Parse")
	}
	if _, err := Parse("fr"); err == nil {
		t.Error("Parse accepted fr")
	}
}

func TestGlossaryCoversTaxonomy(t *testing.T) {
	keys := []string{taxonomy.OtherEarning, taxonomy.OtherDeduction, taxonomy.OtherInfo}
	for _, e := range taxonomy.Entries {
		keys = append(keys, e.Key)
	}
	for _, l := range []Lang{EN, IT} {
		for _, k := range keys {
			if ItemText(l, k) == "" {
				t.Errorf("%s glossary lacks item %q", l, k)
			}
		}
		for _, term := range Terms {
			if T(l, "term."+term) == "term."+term || T(l, "term."+term+".text") == "term."+term+".text" {
				t.Errorf("%s glossary lacks term %q", l, term)
			}
		}
	}
	if ItemText(IT, "no.such.key") != "" {
		t.Error("ItemText of an unknown key")
	}
	if !strings.Contains(ItemText(EN, "deductions.inps"), "9-10% ") {
		t.Errorf("percent signs are not rendered: %q", T(EN, "item.deductions.inps"))
	}
}

// TestChangeTexts formats every change text as insights does, so a wrong
// verb or argument index shows up as Go's %! marker.
func TestChangeTexts(t *testing.T) {
	args := map[string][]any{
		"change.level":     {"3", "4"},
		"change.ccnl":      {"A", "B"},
		"change.net":       {"1.000,00 €", "1.100,00 €", 10.0},
		"change.gross":     {"1.000,00 €", "1.100,00 €", 10.0},
		"change.item_new":  {"IRPEF", "10,00"},
		"change.item_gone": {"IRPEF"},
		"change.tax_rate":  {12.5, 13.5},
		"change.inps_rate": {9.19, 9.49},
		"change.leave":     {"Ferie", "10,00", "8,00", "-2,00"},
		"change.gap":       {"gennaio 2026", "marzo 2026"},
		"change.unchecked": nil,
	}
	for _, l := range []Lang{EN, IT} {
		for k, a := range args {
			if got := T(l, k, a...); strings.Contains(got, "%!") || got == k {
				t.Errorf("%s %s = %q", l, k, got)
			}
		}
	}
	if got := T(EN, "change.net", "a", "b", 2.5); got != "You receive b instead of a (+2.5%)." {
		t.Errorf("change.net = %q", got)
	}
}
