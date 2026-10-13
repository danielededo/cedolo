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
	// Advisory checks test a relation that usually holds but legitimately
	// fails in some situations (special contribution rates, deductible
	// pension contributions); a mismatch is a note, not an error.
	Advisory bool `json:"advisory,omitempty"`
}

// Kind distinguishes ordinary monthly payslips from additional months.
type Kind string

const (
	Monthly    Kind = "monthly"
	Thirteenth Kind = "13th"
	Fourteenth Kind = "14th"
)

// LeaveKind identifies a leave balance.
type LeaveKind string

const (
	// Holiday is annual paid leave (ferie).
	Holiday LeaveKind = "holiday"
	// ReducedHours is leave from working-time reduction (ROL, PAR).
	ReducedHours LeaveKind = "rol"
	// Permits is paid leave printed simply as "permessi", which some layouts
	// keep separate from ROL.
	Permits LeaveKind = "permits"
	// FormerHolidays is leave replacing abolished public holidays (ex festività).
	FormerHolidays LeaveKind = "former_holidays"
	// HoursBank is overtime banked as time off (banca ore).
	HoursBank LeaveKind = "hours_bank"
	// Flexibility is time off from flexible working hours (flessibilità).
	Flexibility LeaveKind = "flexibility"
)

// Leave is one leave balance as printed. Quantities are in hundredths of the
// unit (900 is 9 hours or 9 days); nil means not printed.
type Leave struct {
	Kind      LeaveKind `json:"kind"`
	Label     string    `json:"label"`
	Unit      string    `json:"unit,omitempty"` // "hours", "days", or empty when not printed
	Previous  *int64    `json:"previous,omitempty"`
	Accrued   *int64    `json:"accrued,omitempty"`
	Used      *int64    `json:"used,omitempty"`
	Remaining *int64    `json:"remaining,omitempty"`
}

// Payslip is the normalized content of one payslip.
type Payslip struct {
	Pages []int  `json:"pages"`
	Year  int    `json:"year,omitempty"`
	Month int    `json:"month,omitempty"`
	Kind  Kind   `json:"kind"`
	CCNL  string `json:"ccnl,omitempty"`
	Level string `json:"level,omitempty"`

	Items []Item `json:"items"`

	// Gross is the gross pay as printed. Some payroll software excludes
	// non-taxable items such as expense reimbursements from it.
	Gross Amount `json:"gross"`
	// Deductions is everything withheld between gross and net: item table
	// deductions plus contributions and taxes, whether the payslip prints
	// them in the item table or in summary boxes.
	Deductions Amount `json:"deductions"`
	Net        Amount `json:"net"`
	INPSBase   Amount `json:"inps_base"`
	IRPEFBase  Amount `json:"irpef_base"`
	TFR        Amount `json:"tfr"`

	// Summary values that some layouts print outside the item table.
	SocialContributions Amount `json:"social_contributions"`
	Taxes               Amount `json:"taxes"`
	INPS                Amount `json:"inps"`
	RoundingPrevious    Amount `json:"rounding_previous"`
	RoundingCurrent     Amount `json:"rounding_current"`

	Leave []Leave `json:"leave,omitempty"`

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
