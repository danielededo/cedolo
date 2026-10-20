// Package synth builds synthetic payslips: fictitious people and companies
// with amounts that add up, rendered as PDF or directly as a text layout in
// several styles that imitate the variety of Italian payroll software. They
// are the only payslip documents allowed in tests and examples (see ADR 0003);
// real payslips never enter the repository.
package synth

import (
	"strconv"
	"strings"
	"time"

	"github.com/danielededo/cedolo/internal/pii"
)

// Item is one line of the payslip body. Amount is in cents and always
// positive; Deduction says on which side it is printed. Key is the canonical
// taxonomy key a correct extraction must assign to it.
type Item struct {
	Code        string
	Description string
	Quantity    string // as printed, e.g. "10,00"; empty if not applicable
	Base        string // rate or base as printed; empty if not applicable
	Amount      int64
	Deduction   bool
	Key         string
}

// Leave is a leave balance as printed, in hours. Key is the canonical kind a
// correct extraction must assign to it.
type Leave struct {
	Kind                               string
	Key                                string
	Previous, Accrued, Used, Remaining string
}

// Payslip is the canonical content of a synthetic payslip.
type Payslip struct {
	Employer       string
	Employee       string
	EmployeeNumber string
	// EmployeeCode and EmployerCode are the fiscal codes printed in the
	// header when set: see CodiceFiscale and VATNumber.
	EmployeeCode, EmployerCode string
	Year                       int
	Month                      time.Month
	CCNL                       string
	Level                      string
	Qualification              string
	Items                      []Item
	INPSBase                   int64
	IRPEFBase                  int64
	TFRAccrued                 int64
	Leave                      []Leave
	// Contributions and Tax are withheld without an item of their own:
	// some payslips print the INPS contributions and the IRPEF only in
	// their totals, below the item table.
	Contributions, Tax int64
}

// Totals returns gross pay, total deductions and net pay in cents.
func (p Payslip) Totals() (gross, deductions, net int64) {
	for _, it := range p.Items {
		if it.Deduction {
			deductions += it.Amount
		} else {
			gross += it.Amount
		}
	}
	deductions += p.Contributions + p.Tax
	return gross, deductions, gross - deductions
}

// Example returns the deterministic synthetic payslip used by the classic
// style.
func Example() Payslip { return ExampleFor(StyleClassic) }

// ExampleFor returns a deterministic synthetic payslip whose item wording
// follows the given style. Names and companies are invented; amounts are
// plausible but not computed from real tax rules, except the INPS employee
// contribution, which is 9.19 percent of its base so that cross-checks have
// something real to verify.
func ExampleFor(s Style) Payslip {
	words := descriptions[s]
	p := Payslip{
		Employer:       "ESEMPIO SINTETICO SRL",
		Employee:       "BIANCHI ANNA",
		EmployeeNumber: "000042",
		Year:           2026,
		Month:          time.October,
		CCNL:           "COMMERCIO TERZIARIO",
		Level:          "4",
		Qualification:  "IMPIEGATO",
		Items: []Item{
			{Code: "0001", Description: words[0], Quantity: "168,00", Amount: 200000, Key: "earnings.base_salary"},
			{Code: "0250", Description: words[1], Quantity: "10,00", Base: "14,45", Amount: 14450, Key: "earnings.overtime"},
			{Code: "0300", Description: words[2], Amount: 5000, Key: "earnings.allowance"},
		},
		TFRAccrued: 16256,
		Leave: []Leave{
			{Kind: "FERIE", Key: "holiday", Previous: "90,00", Accrued: "14,00", Used: "8,00", Remaining: "96,00"},
			{Kind: "PERMESSI ROL", Key: "rol", Previous: "36,00", Accrued: "6,00", Used: "0,00", Remaining: "42,00"},
		},
	}
	gross, _, _ := p.Totals()
	inps := (gross*919 + 5000) / 10000 // 9.19 percent, rounded to the cent
	p.INPSBase = gross
	p.IRPEFBase = gross - inps
	p.Items = append(p.Items,
		Item{Code: "9001", Description: words[3], Base: "9,19%", Amount: inps, Deduction: true, Key: "deductions.inps"},
		Item{Code: "9100", Description: words[4], Amount: 29543, Deduction: true, Key: "tax.irpef"},
		Item{Code: "9150", Description: words[5], Amount: 2200, Deduction: true, Key: "tax.surcharge.regional"},
		Item{Code: "9160", Description: words[6], Amount: 800, Deduction: true, Key: "tax.surcharge.municipal"},
	)
	if s == StyleSigned {
		p.Leave[0].Kind, p.Leave[1].Kind, p.Leave[1].Key = "Ferie", "Permessi", "permits"
	}
	if s == StyleGrid {
		// This style prints the CNEL code of the contract before its name.
		p.CCNL = "H011 TERZIARIO SERVIZI"
		p.Leave[1].Kind, p.Leave[1].Key = "PERMESSI", "permits"
		p.Leave = append(p.Leave, Leave{Kind: "EX FEST.", Key: "former_holidays", Previous: "8,00", Accrued: "2,67", Used: "0,00", Remaining: "10,67"})
		// This style rounds the net pay up to whole euros with an earnings
		// item, as several payroll packages do.
		_, _, net := p.Totals()
		if r := (100 - net%100) % 100; r > 0 {
			p.Items = append(p.Items, Item{Code: "9990", Description: "ARROTONDAMENTO MESE CORRENTE", Amount: r, Key: "rounding"})
		}
	}
	if s == StyleSpaced {
		// This style prints the INPS contributions and the IRPEF only as
		// totals, and its leave table names the reduced hours "ROL".
		var items []Item
		for _, it := range p.Items {
			switch it.Key {
			case "deductions.inps":
				p.Contributions = it.Amount
			case "tax.irpef":
				p.Tax = it.Amount
			default:
				items = append(items, it)
			}
		}
		p.Items = items
		p.Leave[1].Kind = "ROL"
	}
	return p
}

// descriptions holds the item wording of each style, in the order of the
// items built by ExampleFor.
var descriptions = map[Style][7]string{
	StyleClassic: {"RETRIBUZIONE ORDINARIA", "STRAORDINARIO 25%", "INDENNITÀ DI FUNZIONE", "CONTRIBUTO IVS",
		"RITENUTA IRPEF", "ADDIZIONALE REGIONALE", "ADDIZIONALE COMUNALE"},
	StyleSigned: {"Retribuzione mensile", "Lavoro straordinario 25%", "Indennità di funzione", "Contributi INPS c/dipendente",
		"Ritenute IRPEF", "Add.le regionale IRPEF", "Add.le comunale IRPEF"},
	StyleGrid: {"PAGA BASE", "STRAORD. FERIALE 25%", "IND. FUNZIONE", "CTR. F.P.L.D. 9,19%",
		"IRPEF NETTA", "ADD. REG. RATA", "ADD. COM. RATA"},
	StyleSpaced: {"LAVORO ORDINARIO", "STRAORDINARIO 25%", "IND. FUNZIONE", "CONTR. IVS",
		"RITENUTA IRPEF", "RATA ADDIZ. REGIONALE", "RATA ADDIZ. COMUNALE"},
}

// CodiceFiscale returns an invented, check-valid codice fiscale for a name
// printed surname first, such as "BIANCHI ANNA": the letters of the
// surname and the given name, then a fixed birth date and place. Only the
// first fifteen characters are in the source, so that no valid code is.
func CodiceFiscale(name string) string {
	words := strings.Fields(strings.ToUpper(name))
	if len(words) < 2 {
		words = append(words, "X")
	}
	code := func(s string, given bool) string {
		var cons, vows string
		for _, r := range s {
			switch {
			case strings.ContainsRune("AEIOU", r):
				vows += string(r)
			case r >= 'A' && r <= 'Z':
				cons += string(r)
			}
		}
		if given && len(cons) >= 4 {
			return cons[:1] + cons[2:4]
		}
		return (cons + vows + "XXX")[:3]
	}
	body := code(words[0], false) + code(strings.Join(words[1:], ""), true) + "80A01H501"
	return body + string(pii.CodiceFiscaleCheck(body))
}

// VATNumber returns an invented, check-valid VAT number from ten digits.
func VATNumber(first10 string) string {
	sum := 0
	for i := 0; i < 10; i++ {
		d := int(first10[i] - '0')
		if i%2 == 1 {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
	}
	return first10 + strconv.Itoa((10-sum%10)%10)
}
