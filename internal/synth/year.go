package synth

import (
	"fmt"
	"time"
)

// Year returns the twelve monthly payslips of one invented worker for a
// year, as the web app's example shows them: overtime that changes from
// month to month, a move up a level with a raise in July, IRPEF that
// follows the pay, leave accrued every month and taken in spring, summer
// and December, and the TFR set aside. Names, companies, codes and
// amounts are all invented; the INPS contribution is 9.19 percent of its
// base, as in Example, and the IRPEF is a plausible figure, not one
// computed by the law.
func Year(year int) []Payslip {
	words := descriptions[StyleClassic]
	overtime := [12]int64{0, 6, 10, 4, 8, 12, 6, 0, 10, 14, 8, 20}
	holidayUsed := [12]int{0, 0, 0, 16, 0, 0, 0, 80, 0, 0, 0, 24}
	rolUsed := [12]int{0, 0, 8, 0, 0, 0, 0, 0, 0, 0, 12, 0}
	holiday, rol := 60, 20
	var out []Payslip
	for m := range 12 {
		base, level := int64(200000), "4"
		if m >= 6 {
			base, level = 208000, "3"
		}
		p := Payslip{
			Employer:       "ESEMPIO SINTETICO SRL",
			EmployerCode:   VATNumber("0123456789"),
			Employee:       "BIANCHI ANNA",
			EmployeeCode:   CodiceFiscale("BIANCHI ANNA"),
			EmployeeNumber: "000042",
			Year:           year,
			Month:          time.Month(m + 1),
			CCNL:           "COMMERCIO TERZIARIO",
			Level:          level,
			Qualification:  "IMPIEGATO",
			Items:          []Item{{Code: "0001", Description: words[0], Quantity: "168,00", Amount: base, Key: "earnings.base_salary"}},
		}
		if h := overtime[m]; h > 0 {
			p.Items = append(p.Items, Item{Code: "0250", Description: words[1], Quantity: fmt.Sprintf("%d,00", h), Base: "14,45", Amount: h * 1445, Key: "earnings.overtime"})
		}
		p.Items = append(p.Items, Item{Code: "0300", Description: words[2], Amount: 5000, Key: "earnings.allowance"})
		gross, _, _ := p.Totals()
		inps := (gross*919 + 5000) / 10000 // 9.19 percent, rounded to the cent
		p.INPSBase, p.IRPEFBase = gross, gross-inps
		p.Items = append(p.Items,
			Item{Code: "9001", Description: words[3], Base: "9,19%", Amount: inps, Deduction: true, Key: "deductions.inps"},
			Item{Code: "9100", Description: words[4], Amount: irpef(p.IRPEFBase), Deduction: true, Key: "tax.irpef"},
			Item{Code: "9150", Description: words[5], Amount: 2200, Deduction: true, Key: "tax.surcharge.regional"},
			Item{Code: "9160", Description: words[6], Amount: 800, Deduction: true, Key: "tax.surcharge.municipal"},
		)
		p.TFRAccrued = (gross*100 + 675) / 1350 // the yearly pay over 13.5, a month at a time
		p.Leave = []Leave{
			leave("FERIE", "holiday", &holiday, 14, holidayUsed[m]),
			leave("PERMESSI ROL", "rol", &rol, 6, rolUsed[m]),
		}
		out = append(out, p)
	}
	return out
}

// irpef is a plausible monthly IRPEF for a taxable base: the base of a
// year at 23 percent up to 28.000 € and 35 percent above, less a fixed
// deduction, a month at a time.
func irpef(base int64) int64 {
	year := base * 12
	tax := min(year, 2800000) * 23 / 100
	if year > 2800000 {
		tax += (year - 2800000) * 35 / 100
	}
	return max(tax/12-15000, 0)
}

// leave moves a balance of hours by a month: accrued, then taken.
func leave(kind, key string, balance *int, accrued, used int) Leave {
	l := Leave{Kind: kind, Key: key, Previous: hours(*balance), Accrued: hours(accrued), Used: hours(used)}
	*balance += accrued - used
	l.Remaining = hours(*balance)
	return l
}

func hours(h int) string { return fmt.Sprintf("%d,00", h) }
