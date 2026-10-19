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
	{Key: "tfr.accrued", LabelEN: "TFR accrued this month", LabelIT: "TFR maturato nel mese",
		Side: payslip.Info, Keywords: []string{"storno t f r", "storno tfr", "tfr a fondo", "tfr a f p c", "tfr a fpc",
			"quota tfr", "tfr maturato", "tfr mese", "tfr del mese", "tfr maturata"}},
	{Key: "pension_fund.employee", LabelEN: "Your pension fund contribution", LabelIT: "Il tuo contributo al fondo pensione",
		Side: payslip.Info, Keywords: []string{"fpc dipendente", "fpc dip", "f p c dipendente", "f p c dip", "fondo pensione dipendente",
			"previdenza complementare dipendente", "fpc lavoratore", "fondo pensione lavoratore"}},
	{Key: "pension_fund.contribution", LabelEN: "Supplementary pension contribution", LabelIT: "Contributo previdenza complementare",
		Side: payslip.Info, Keywords: []string{"fpc", "f p c", "fondo pensione", "previdenza complementare"}},
	{Key: "rounding", LabelEN: "Rounding", LabelIT: "Arrotondamento",
		Keywords: []string{"arrotondamento", "arrotond", "arrot", "arr to"}},
	{Key: "meal_vouchers", LabelEN: "Meal vouchers (taxable part)", LabelIT: "Buoni pasto (parte imponibile)",
		Keywords: []string{"buoni pasto", "buono pasto", "ticket restaurant", "assoggettamento pasti", "pasti"}},
	{Key: "tax.return", LabelEN: "Tax return settlement (730)", LabelIT: "Conguaglio 730",
		Keywords: []string{"730", "mod 730", "modello 730", "assistenza fiscale"}},
	{Key: "earnings.reimbursement", LabelEN: "Expense reimbursement", LabelIT: "Rimborso spese",
		Side: payslip.Earning, Keywords: []string{"nota spese", "rimborso spese", "rimborsi spese", "rimborso", "rimborsi", "rimb spese"}},
	{Key: "earnings.contribution_relief", LabelEN: "Social security contribution relief", LabelIT: "Esonero contributivo",
		Side: payslip.Earning, Keywords: []string{"esonero contributivo", "eson contr", "esonero ctr", "esonero contr", "eson ivs", "esonero ivs", "esonero"}},
	{Key: "tax.surcharge.regional", LabelEN: "Regional income tax surcharge", LabelIT: "Addizionale regionale",
		Side: payslip.Deduction, Keywords: []string{"addizionale regionale", "add le regionale", "add regionale", "add reg", "add le reg", "addiz reg", "addiz regionale", "add region"}},
	{Key: "tax.surcharge.municipal", LabelEN: "Municipal income tax surcharge", LabelIT: "Addizionale comunale",
		Side: payslip.Deduction, Keywords: []string{"addizionale comunale", "add le comunale", "add comunale", "add com", "add comun", "add le com", "addiz com", "addiz comunale", "acconto addizionale comunale"}},
	{Key: "tax.credit", LabelEN: "Tax credit paid in the payslip", LabelIT: "Trattamento integrativo e bonus fiscali",
		Side: payslip.Earning, Keywords: []string{"trattamento integrativo", "tratt integrativo", "ulteriore detrazione", "cuneo fiscale", "bonus irpef", "somma integrativa",
			"dl 66", "bonus 80", "bonus renzi"}},
	{Key: "earnings.family_allowance", LabelEN: "Family allowance (ANF)", LabelIT: "Assegno per il nucleo familiare",
		Side: payslip.Earning, Keywords: []string{"assegno nucleo familiare", "assegno per il nucleo familiare", "assegni familiari",
			"assegno familiare", "anf", "ass nucleo fam", "ass to nucleo"}},
	{Key: "deductions.absence", LabelEN: "Pay withheld for absence", LabelIT: "Trattenuta per assenza",
		Side: payslip.Deduction, Keywords: []string{"assenza", "assenze", "ore non lavorate", "giorni non lavorati"}},
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
		Side: payslip.Earning, Keywords: []string{"tredicesima", "13ma", "13esima", "13a mensilita", "13 mensilita", "13a mens", "13 mens",
			"xiii mensilita", "gratifica natalizia", "rateo 13", "ratei 13"}},
	{Key: "earnings.fourteenth", LabelEN: "Fourteenth month", LabelIT: "Quattordicesima",
		Side: payslip.Earning, Keywords: []string{"quattordicesima", "14ma", "14esima", "14a mensilita", "14 mensilita", "14a mens", "14 mens",
			"xiv mensilita", "rateo 14", "ratei 14"}},
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
		Side: payslip.Earning, Keywords: []string{"retribuzione", "paga base", "minimo", "stipendio", "minimo contrattuale", "paga", "retrib",
			"lavoro ordinario", "lavoro ordin", "ore ordinarie", "ordinaria", "ordinario"}},
}

// Other keys for items that match no entry, by side.
const (
	OtherEarning   = "earnings.other"
	OtherDeduction = "deductions.other"
	OtherInfo      = "info.other"
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
	switch side {
	case payslip.Deduction:
		return OtherDeduction
	case payslip.Info:
		return OtherInfo
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
	case OtherInfo:
		return Entry{Key: key, LabelEN: "Other information", LabelIT: "Altre informazioni", Side: payslip.Info}, true
	}
	return Entry{}, false
}

// Normalize lowercases s, removes accents, and replaces every run of
// characters that are not letters or digits with a single space, so that
// "Add.le Reg." and "ADD LE REG" compare equal. Compatibility characters
// become their plain form first, so "13ª" reads as "13a".
func Normalize(s string) string {
	var b strings.Builder
	space := true
	for _, r := range norm.NFKD.String(strings.ToLower(s)) {
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
