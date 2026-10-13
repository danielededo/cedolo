// Package payslip defines the normalized data extracted from one payslip:
// header, line items mapped to the canonical taxonomy, totals, and the
// results of the cross-checks that establish how far each value can be
// trusted. Money is always in integer cents.
package payslip

// Side says whether an item adds to or subtracts from pay.
type Side string

const (
	Earning   Side = "earning"
	Deduction Side = "deduction"
)

// Status says how a value was obtained and how far it can be trusted.
type Status string

const (
	// Verified values were read from the document and confirmed by at least
	// one cross-check.
	Verified Status = "verified"
	// Extracted values were read from the document but no cross-check could
	// confirm them.
	Extracted Status = "extracted"
	// Inferred values were not found in the document and were computed from
	// other values.
	Inferred Status = "inferred"
	// Inconsistent values were read from the document but contradict other
	// values in a cross-check.
	Inconsistent Status = "inconsistent"
	// Missing values were neither found nor computable.
	Missing Status = "missing"
)

// Source locates a value on the page it was read from, in points with the
// origin at the top-left corner.
type Source struct {
	Page int     `json:"page"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	W    float64 `json:"w"`
	H    float64 `json:"h"`
}

// Amount is a money value with its status and, when read from the document,
// its source.
type Amount struct {
	Cents  int64   `json:"cents"`
	Status Status  `json:"status"`
	Source *Source `json:"source,omitempty"`
}

// Found reports whether the amount has a value, read or inferred.
func (a Amount) Found() bool { return a.Status != Missing && a.Status != "" }

// Item is one line of the payslip body.
type Item struct {
	Code        string `json:"code,omitempty"`
	Description string `json:"description"`
	Quantity    string `json:"quantity,omitempty"`
	Base        string `json:"base,omitempty"`
	Cents       int64  `json:"cents"`
	Side        Side   `json:"side"`
	// Key is the canonical taxonomy key, or "earnings.other" /
	// "deductions.other" when the description is not recognized.
	Key    string `json:"key"`
	Source Source `json:"source"`
}

// Check is the outcome of one cross-check.
type Check struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}

// Kind distinguishes ordinary monthly payslips from additional months.
type Kind string

const (
	Monthly    Kind = "monthly"
	Thirteenth Kind = "13th"
	Fourteenth Kind = "14th"
)

// Payslip is the normalized content of one payslip.
type Payslip struct {
	Pages []int  `json:"pages"`
	Year  int    `json:"year,omitempty"`
	Month int    `json:"month,omitempty"`
	Kind  Kind   `json:"kind"`
	CCNL  string `json:"ccnl,omitempty"`
	Level string `json:"level,omitempty"`

	Items []Item `json:"items"`

	Gross      Amount `json:"gross"`
	Deductions Amount `json:"deductions"`
	Net        Amount `json:"net"`
	INPSBase   Amount `json:"inps_base"`
	IRPEFBase  Amount `json:"irpef_base"`
	TFR        Amount `json:"tfr"`

	Checks []Check `json:"checks"`
}

// Sum returns the total of the items on one side.
func (p Payslip) Sum(side Side) int64 {
	var total int64
	for _, it := range p.Items {
		if it.Side == side {
			total += it.Cents
		}
	}
	return total
}

// SumKey returns the total of the items with the given canonical key.
func (p Payslip) SumKey(key string) (total int64, n int) {
	for _, it := range p.Items {
		if it.Key == key {
			total += it.Cents
			n++
		}
	}
	return total, n
}
