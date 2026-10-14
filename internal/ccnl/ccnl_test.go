package ccnl

import (
	"regexp"
	"strings"
	"testing"
)

func TestTable(t *testing.T) {
	if Len() < 500 {
		t.Fatalf("only %d contracts", Len())
	}
	if !regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`).MatchString(AsOf()) {
		t.Errorf("AsOf = %q", AsOf())
	}
	date := regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})?$`)
	for _, c := range contracts {
		if !codeToken.MatchString(c.Code) || len(c.Code) != 4 || c.Title == "" {
			t.Errorf("bad row %+v", c)
		}
		if !date.MatchString(c.From) || !date.MatchString(c.Until) {
			t.Errorf("%s: bad dates %q %q", c.Code, c.From, c.Until)
		}
	}
}

func TestLookup(t *testing.T) {
	c, ok := Lookup(" h011 ")
	if !ok || !strings.Contains(strings.ToLower(c.Title), "terziario") {
		t.Fatalf("H011 = %+v, %v", c, ok)
	}
	if _, ok := Lookup("ZZZZ"); ok {
		t.Error("unknown code found")
	}
}

func TestFindCode(t *testing.T) {
	for text, want := range map[string]string{
		"H011 - COMMERCIO TERZIARIO":    "H011",
		"Metalmeccanica industria C011": "C011",
		"COMMERCIO":                     "",
		"CCNL":                          "",
		"Liv. 4 matr. 1234":             "",
	} {
		c, ok := FindCode(text)
		if ok != (want != "") || c.Code != want {
			t.Errorf("FindCode(%q) = %q, %v; want %q", text, c.Code, ok, want)
		}
	}
}

func TestSearch(t *testing.T) {
	got := Search("metalmeccanici installazione impianti federmeccanica", 3)
	if len(got) == 0 {
		t.Fatal("no results")
	}
	found := false
	for _, c := range got {
		found = found || c.Code == "C011"
	}
	if !found {
		t.Errorf("C011 not among %v", got)
	}
	if Search("ccnl per i dipendenti", 5) != nil {
		t.Error("stopwords alone matched")
	}
}
