// Package taxonomy is the canonical catalogue of payslip items. Payroll
// software and employers name the same item in many ways and give it
// arbitrary codes; the taxonomy maps descriptions to stable English keys with
// labels in English and Italian, independently of the vendor.
package taxonomy

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"

	"github.com/danielededo/cedolo/internal/payslip"
)

// Entry is one canonical item.
type Entry struct {
	Key     string
	LabelEN string
	LabelIT string
	// Side is the side the item normally appears on; empty when it can be
	// either (rounding, adjustments).
	Side payslip.Side
	// Keywords are normalized phrases (see Normalize) that identify the item
	// in a description. A description matches when it contains any of them
	// as whole words.
	Keywords []string
}

// Entries is ordered from most to least specific: the first entry whose
// keywords match a description wins, so "addizionale regionale" is tested
// before anything that would match "irpef" alone.
var Entries = []Entry{
	{Key: "rounding", LabelEN: "Rounding", LabelIT: "Arrotondamento",
		Keywords: []string{"arrotondamento", "arrot", "arr to"}},
	{Key: "tax.surcharge.regional", LabelEN: "Regional income tax surcharge", LabelIT: "Addizionale regionale",
		Side: payslip.Deduction, Keywords: []string{"addizionale regionale", "add le regionale", "add reg", "add le reg", "addiz reg"}},
	{Key: "tax.surcharge.municipal", LabelEN: "Municipal income tax surcharge", LabelIT: "Addizionale comunale",
		Side: payslip.Deduction, Keywords: []string{"addizionale comunale", "add le comunale", "add com", "add le com", "addiz com", "acconto addizionale comunale"}},
	{Key: "tax.credit", LabelEN: "Tax credit paid in the payslip", LabelIT: "Trattamento integrativo e bonus fiscali",
		Side: payslip.Earning, Keywords: []string{"trattamento integrativo", "tratt integrativo", "ulteriore detrazione", "cuneo fiscale", "bonus irpef", "somma integrativa"}},
	{Key: "tax.irpef", LabelEN: "Income tax (IRPEF)", LabelIT: "IRPEF",
		Side: payslip.Deduction, Keywords: []string{"irpef", "ritenute fiscali", "ritenuta fiscale", "ritenuta d acconto"}},
	{Key: "deductions.pension_fund", LabelEN: "Supplementary pension fund", LabelIT: "Fondo pensione complementare",
		Side: payslip.Deduction, Keywords: []string{"fondo pensione", "previdenza complementare", "fon te", "fonte", "cometa", "previndai", "fondapi", "byblos", "previambiente"}},
	{Key: "deductions.health_fund", LabelEN: "Health fund", LabelIT: "Fondo sanitario integrativo",
		Side: payslip.Deduction, Keywords: []string{"fondo sanitario", "assistenza sanitaria", "cassa sanitaria", "fondo est", "metasalute", "fasi", "quas", "fasdac"}},
	{Key: "deductions.union", LabelEN: "Union dues", LabelIT: "Trattenuta sindacale",
		Side: payslip.Deduction, Keywords: []string{"sindacale", "sindacato", "quota associativa"}},
	{Key: "deductions.inps", LabelEN: "Social security contributions (INPS)", LabelIT: "Contributi INPS",
		Side: payslip.Deduction, Keywords: []string{"inps", "ivs", "f p l d", "fpld", "contributo previdenziale", "contributi previdenziali", "contrib previdenziali", "cigs", "c i g s", "naspi", "fis"}},
	{Key: "earnings.thirteenth", LabelEN: "Thirteenth month", LabelIT: "Tredicesima",
		Side: payslip.Earning, Keywords: []string{"tredicesima", "13ma", "13a mensilita", "gratifica natalizia", "rateo 13"}},
	{Key: "earnings.fourteenth", LabelEN: "Fourteenth month", LabelIT: "Quattordicesima",
		Side: payslip.Earning, Keywords: []string{"quattordicesima", "14ma", "14a mensilita", "rateo 14"}},
	{Key: "earnings.overtime", LabelEN: "Overtime", LabelIT: "Straordinario",
		Side: payslip.Earning, Keywords: []string{"straordinario", "straordinari", "straord", "lavoro supplementare", "supplementare"}},
	{Key: "earnings.seniority", LabelEN: "Seniority increments", LabelIT: "Scatti di anzianità",
		Side: payslip.Earning, Keywords: []string{"scatti", "scatto", "anzianita", "aumenti periodici"}},
	{Key: "earnings.superminimum", LabelEN: "Individual supplement (superminimo)", LabelIT: "Superminimo",
		Side: payslip.Earning, Keywords: []string{"superminimo", "super minimo", "ad personam", "assegno ad personam"}},
	{Key: "earnings.contingency", LabelEN: "Contingency allowance", LabelIT: "Contingenza",
		Side: payslip.Earning, Keywords: []string{"contingenza", "edr", "elemento distinto"}},
	{Key: "earnings.bonus", LabelEN: "Bonus", LabelIT: "Premio",
		Side: payslip.Earning, Keywords: []string{"premio", "una tantum", "bonus", "incentivo", "welfare"}},
	{Key: "earnings.allowance", LabelEN: "Allowance", LabelIT: "Indennità",
		Side: payslip.Earning, Keywords: []string{"indennita", "ind", "maggiorazione", "trasferta", "reperibilita", "turno"}},
	{Key: "earnings.leave", LabelEN: "Paid leave and holidays", LabelIT: "Ferie, permessi e festività",
		Side: payslip.Earning, Keywords: []string{"ferie", "permessi", "permesso", "festivita", "rol", "ex festivita"}},
	{Key: "earnings.sick_pay", LabelEN: "Sick pay", LabelIT: "Malattia",
		Side: payslip.Earning, Keywords: []string{"malattia", "infortunio", "carenza"}},
	{Key: "earnings.base_salary", LabelEN: "Base salary", LabelIT: "Retribuzione base",
		Side: payslip.Earning, Keywords: []string{"retribuzione", "paga base", "minimo", "stipendio", "minimo contrattuale", "paga", "retrib"}},
}

// Other keys for items that match no entry, by side.
const (
	OtherEarning   = "earnings.other"
	OtherDeduction = "deductions.other"
)

// Classify returns the canonical key for an item description printed on the
// given side. An entry whose usual side differs is still accepted when it has
// no side or when the item is printed as its opposite (for example a negative
// adjustment of a deduction); otherwise the item falls back to "other".
func Classify(description string, side payslip.Side) string {
	words := " " + Normalize(description) + " "
	for _, e := range Entries {
		for _, kw := range e.Keywords {
			if strings.Contains(words, " "+kw+" ") {
				if e.Side == "" || e.Side == side {
					return e.Key
				}
			}
		}
	}
	if side == payslip.Deduction {
		return OtherDeduction
	}
	return OtherEarning
}

// Lookup returns the entry for a key.
func Lookup(key string) (Entry, bool) {
	for _, e := range Entries {
		if e.Key == key {
			return e, true
		}
	}
	switch key {
	case OtherEarning:
		return Entry{Key: key, LabelEN: "Other earnings", LabelIT: "Altre competenze", Side: payslip.Earning}, true
	case OtherDeduction:
		return Entry{Key: key, LabelEN: "Other deductions", LabelIT: "Altre trattenute", Side: payslip.Deduction}, true
	}
	return Entry{}, false
}

// Normalize lowercases s, removes accents, and replaces every run of
// characters that are not letters or digits with a single space, so that
// "Add.le Reg." and "ADD LE REG" compare equal.
func Normalize(s string) string {
	var b strings.Builder
	space := true
	for _, r := range norm.NFD.String(strings.ToLower(s)) {
		switch {
		case unicode.Is(unicode.Mn, r):
			// Combining accent: dropped.
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			space = false
		default:
			if !space {
				b.WriteByte(' ')
				space = true
			}
		}
	}
	return strings.TrimSpace(b.String())
}
