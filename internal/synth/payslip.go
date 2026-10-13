// Package synth builds synthetic payslips: fictitious people and companies
// with amounts that add up, rendered as PDF. They are the only payslip
// documents allowed in tests and examples (see ADR 0003); real payslips never
// enter the repository.
package synth

import "time"

// Item is one line of the payslip body. Amount is in cents and always
// positive; Deduction says which column it is printed in.
type Item struct {
	Code        string
	Description string
	Quantity    string // as printed, e.g. "10,00"; empty if not applicable
	Base        string // rate or base as printed; empty if not applicable
	Amount      int64
	Deduction   bool
}

// Leave is a leave balance as printed, in hours.
type Leave struct {
	Kind                     string
	Accrued, Used, Remaining string
}

// Payslip is the canonical content of a synthetic payslip.
type Payslip struct {
	Employer       string
	Employee       string
	EmployeeNumber string
	Year           int
	Month          time.Month
	CCNL           string
	Level          string
	Qualification  string
	Items          []Item
	TFRAccrued     int64
	Leave          []Leave
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
	return gross, deductions, gross - deductions
}

// Example returns a deterministic synthetic payslip. Names and companies are
// invented; amounts are plausible but not computed from real tax rules,
// except the INPS employee contribution, which is 9.19 percent of gross so
// that cross-checks have something real to verify.
func Example() Payslip {
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
			{Code: "0001", Description: "RETRIBUZIONE ORDINARIA", Quantity: "168,00", Amount: 200000},
			{Code: "0250", Description: "STRAORDINARIO 25%", Quantity: "10,00", Base: "14,45", Amount: 14450},
			{Code: "0300", Description: "INDENNITÀ DI FUNZIONE", Amount: 5000},
		},
		TFRAccrued: 16256,
		Leave: []Leave{
			{Kind: "FERIE", Accrued: "14,00", Used: "8,00", Remaining: "96,00"},
			{Kind: "PERMESSI ROL", Accrued: "6,00", Used: "0,00", Remaining: "42,00"},
		},
	}
	gross, _, _ := p.Totals()
	inps := (gross*919 + 5000) / 10000 // 9.19 percent, rounded to the cent
	p.Items = append(p.Items,
		Item{Code: "9001", Description: "CONTRIBUTO IVS", Base: "9,19%", Amount: inps, Deduction: true},
		Item{Code: "9100", Description: "RITENUTA IRPEF", Amount: 29543, Deduction: true},
		Item{Code: "9150", Description: "ADDIZIONALE REGIONALE", Amount: 2200, Deduction: true},
		Item{Code: "9160", Description: "ADDIZIONALE COMUNALE", Amount: 800, Deduction: true},
	)
	return p
}
