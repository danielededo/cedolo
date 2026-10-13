// Package money parses and formats amounts as printed on Italian payslips.
// Amounts are always held as integer cents; floating point is never used for
// money.
package money

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

// ErrInvalid is returned for text that is not a well-formed amount.
var ErrInvalid = errors.New("money: invalid amount")

// amountPattern accepts an optional leading or trailing minus, digits with
// optional dot thousands separators, and an optional comma with one to two
// decimals. Payslips print negative amounts both as "-12,30" and "12,30-".
var amountPattern = regexp.MustCompile(`^(-)?(\d{1,3}(?:\.\d{3})+|\d+)(?:,(\d{1,2}))?(-)?$`)

// Parse converts an Italian-formatted amount such as "1.234,56", "-12,3" or
// "12,30-" into cents.
func Parse(s string) (int64, error) {
	s = strings.TrimSpace(s)
	m := amountPattern.FindStringSubmatch(s)
	if m == nil || (m[1] != "" && m[4] != "") {
		return 0, ErrInvalid
	}
	units, err := strconv.ParseInt(strings.ReplaceAll(m[2], ".", ""), 10, 64)
	if err != nil {
		return 0, ErrInvalid
	}
	dec := m[3]
	if len(dec) == 1 {
		dec += "0"
	}
	var cents int64
	if dec != "" {
		cents, _ = strconv.ParseInt(dec, 10, 64)
	}
	v := units*100 + cents
	if m[1] != "" || m[4] != "" {
		v = -v
	}
	return v, nil
}

// IsAmount reports whether s looks like an amount with decimals, which is how
// payslips print money. Plain integers are excluded because they are usually
// codes, hours or days.
func IsAmount(s string) bool {
	m := amountPattern.FindStringSubmatch(strings.TrimSpace(s))
	return m != nil && m[3] != "" && !(m[1] != "" && m[4] != "")
}

// Format renders cents in Italian notation with thousands separators and two
// decimals, with a leading minus for negative values.
func Format(cents int64) string {
	neg := cents < 0
	if neg {
		cents = -cents
	}
	units := strconv.FormatInt(cents/100, 10)
	var b strings.Builder
	if neg {
		b.WriteByte('-')
	}
	for i, r := range units {
		if i > 0 && (len(units)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(r)
	}
	b.WriteByte(',')
	frac := cents % 100
	if frac < 10 {
		b.WriteByte('0')
	}
	b.WriteString(strconv.FormatInt(frac, 10))
	return b.String()
}
