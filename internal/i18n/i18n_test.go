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
	if got := T(IT, "change.level", "3", "4"); got != "Il livello è passato da 3 a 4." {
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
