// Package pii detects personal identifiers that must never appear in the
// repository: Italian tax codes (codice fiscale) and IBANs. Both carry a
// checksum, so detection validates it and reports only well-formed values,
// which keeps false positives low on code and prose.
package pii

import (
	"math/big"
	"regexp"
	"strings"
)

// Kind names the type of identifier found.
type Kind string

const (
	CodiceFiscale Kind = "codice fiscale"
	IBAN          Kind = "IBAN"
)

// Match is one identifier found in a text. Masked shows only the first and
// last two characters, so reports never repeat the value they flag.
type Match struct {
	Kind   Kind
	Offset int
	Masked string
}

// Omocodia lets the National Insurance substitute digits with letters, so the
// numeric positions accept both.
var cfPattern = regexp.MustCompile(`(?i)\b[A-Z]{6}[0-9LMNPQRSTUV]{2}[A-Z][0-9LMNPQRSTUV]{2}[A-Z][0-9LMNPQRSTUV]{3}[A-Z]\b`)

// IBANs are printed either compact or in groups of four.
var ibanPattern = regexp.MustCompile(`(?i)\b[A-Z]{2}[0-9]{2}(?: ?[A-Z0-9]{4}){2,7}(?: ?[A-Z0-9]{1,4})?\b`)

// Find returns every valid codice fiscale and IBAN in text.
func Find(text string) []Match {
	var out []Match
	for _, loc := range cfPattern.FindAllStringIndex(text, -1) {
		v := strings.ToUpper(text[loc[0]:loc[1]])
		if ValidCodiceFiscale(v) {
			out = append(out, Match{Kind: CodiceFiscale, Offset: loc[0], Masked: mask(v)})
		}
	}
	for _, loc := range ibanPattern.FindAllStringIndex(text, -1) {
		v := strings.ToUpper(strings.ReplaceAll(text[loc[0]:loc[1]], " ", ""))
		if ValidIBAN(v) {
			out = append(out, Match{Kind: IBAN, Offset: loc[0], Masked: mask(v)})
		}
	}
	return out
}

func mask(v string) string {
	if len(v) <= 4 {
		return strings.Repeat("*", len(v))
	}
	return v[:2] + strings.Repeat("*", len(v)-4) + v[len(v)-2:]
}

// Odd-position values for the codice fiscale check character, indexed by
// letter (A to Z); digits map to the letter at the same index.
var cfOdd = [26]int{1, 0, 5, 7, 9, 13, 15, 17, 19, 21, 2, 4, 18, 20, 11, 3, 6, 8, 12, 14, 16, 10, 22, 25, 24, 23}

// CodiceFiscaleCheck computes the check character for the first 15
// characters of a codice fiscale.
func CodiceFiscaleCheck(body string) byte {
	sum := 0
	for i := 0; i < 15; i++ {
		c := body[i]
		var idx int
		if c >= '0' && c <= '9' {
			idx = int(c - '0')
		} else {
			idx = int(c - 'A')
		}
		if i%2 == 0 { // 1-based odd positions
			sum += cfOdd[idx]
		} else {
			sum += idx
		}
	}
	return byte('A' + sum%26)
}

// ValidCodiceFiscale reports whether v is a 16-character codice fiscale with
// a correct check character. v must be upper case.
func ValidCodiceFiscale(v string) bool {
	if len(v) != 16 || !cfPattern.MatchString(v) {
		return false
	}
	return CodiceFiscaleCheck(v[:15]) == v[15]
}

// ValidIBAN reports whether v (upper case, no spaces) passes the ISO 13616
// mod-97 check. Italian IBANs must also be 27 characters long.
func ValidIBAN(v string) bool {
	if len(v) < 15 || len(v) > 34 {
		return false
	}
	if strings.HasPrefix(v, "IT") && len(v) != 27 {
		return false
	}
	return ibanMod97(v[4:]+v[:4]) == 1
}

// IBANCheckDigits computes the two check digits for a country code and BBAN.
func IBANCheckDigits(country, bban string) string {
	r := 98 - ibanMod97(bban+country+"00")
	return string([]byte{byte('0' + r/10), byte('0' + r%10)})
}

func ibanMod97(s string) int {
	var b strings.Builder
	for _, c := range s {
		switch {
		case c >= '0' && c <= '9':
			b.WriteRune(c)
		case c >= 'A' && c <= 'Z':
			b.WriteString(big.NewInt(int64(c - 'A' + 10)).String())
		default:
			return -1
		}
	}
	n, ok := new(big.Int).SetString(b.String(), 10)
	if !ok {
		return -1
	}
	return int(new(big.Int).Mod(n, big.NewInt(97)).Int64())
}
