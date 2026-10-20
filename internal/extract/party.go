package extract

import (
	"slices"
	"strings"
	"unicode"

	"github.com/danielededo/cedolo/internal/layout"
	"github.com/danielededo/cedolo/internal/payslip"
	"github.com/danielededo/cedolo/internal/pii"
	"github.com/danielededo/cedolo/internal/taxonomy"
)

// The person and the employer of a payslip (ADR 0010). The person is told by
// the codice fiscale printed on it, which has a check character; the name is
// the run of words that its first six letters encode, wherever it is
// printed. The employer is told by an eleven-digit fiscal code or VAT number
// on a line that names it as one, and named by the line with a company form.

// person finds the worker's codice fiscale and name.
func person(all [][]line) payslip.Party {
	for _, pg := range all {
		for _, l := range pg {
			for _, t := range l.toks {
				if v := strings.ToUpper(strings.Trim(t.Text, ".,:;()[]")); pii.ValidCodiceFiscale(v) {
					return payslip.Party{ID: v, Name: nameOf(all, v)}
				}
			}
		}
	}
	return payslip.Party{}
}

// nameOf returns the name whose surname and given name give the first six
// letters of the codice fiscale, read from two to five words in a row on a
// line, in either order. It is shown as given name then surname, each word
// capitalized.
func nameOf(all [][]line, cf string) string {
	for _, pg := range all {
		for _, l := range pg {
			for _, run := range wordRuns(l) {
				for n := 2; n <= min(5, len(run)); n++ {
					for s := 0; s+n <= len(run); s++ {
						if name, ok := matchName(run[s:s+n], cf); ok {
							return name
						}
					}
				}
			}
		}
	}
	return ""
}

// wordRuns splits a line into runs of tokens made of letters only, such as
// "DE DOMINICIS DANIELE" between a code and a date.
func wordRuns(l line) [][]string {
	var out [][]string
	var run []string
	for _, t := range l.toks {
		if isWord(t.Text) {
			run = append(run, t.Text)
			continue
		}
		if len(run) > 0 {
			out = append(out, run)
		}
		run = nil
	}
	if len(run) > 0 {
		out = append(out, run)
	}
	return out
}

func isWord(s string) bool {
	letters := 0
	for _, r := range s {
		switch {
		case unicode.IsLetter(r):
			letters++
		case r == '\'' || r == '’':
		default:
			return false
		}
	}
	return letters > 0
}

func matchName(words []string, cf string) (string, bool) {
	for i := 1; i < len(words); i++ {
		// Surname first, as payslips print it, or the given name first.
		for _, order := range [2][2][]string{{words[:i], words[i:]}, {words[i:], words[:i]}} {
			surname, given := order[0], order[1]
			if surnameCode(surname) == cf[:3] && givenCode(given) == cf[3:6] {
				return capitalize(given) + " " + capitalize(surname), true
			}
		}
	}
	return "", false
}

// letters returns the consonants and the vowels of words, upper case and
// without accents.
func letters(words []string) (cons, vows string) {
	for _, r := range strings.ToUpper(taxonomy.Normalize(strings.Join(words, ""))) {
		switch {
		case strings.ContainsRune("AEIOU", r):
			vows += string(r)
		case r >= 'A' && r <= 'Z':
			cons += string(r)
		}
	}
	return cons, vows
}

func surnameCode(words []string) string {
	cons, vows := letters(words)
	return (cons + vows + "XXX")[:3]
}

// givenCode skips the second consonant when there are four or more.
func givenCode(words []string) string {
	cons, vows := letters(words)
	if len(cons) >= 4 {
		return cons[:1] + cons[2:4]
	}
	return (cons + vows + "XXX")[:3]
}

func capitalize(words []string) string {
	out := make([]string, len(words))
	for i, w := range words {
		rs := []rune(strings.ToLower(w))
		up := true
		for k, r := range rs {
			if up && unicode.IsLetter(r) {
				rs[k] = unicode.ToUpper(r)
			}
			up = r == '\'' || r == '’'
		}
		out[i] = string(rs)
	}
	return strings.Join(out, " ")
}

// employerWords mark a line that names the employer's fiscal code or VAT
// number.
var employerWords = []string{"codice fiscale", "cod fiscale", "c f", "cf", "partita iva", "p iva", "piva", "iva"}

// employerLabels head the employer's name, on its line or the line above.
var employerLabels = []string{"ditta", "azienda", "datore di lavoro", "ragione sociale", "societa"}

// companyForms are the legal forms that end a company's name.
var companyForms = []string{"s r l", "srl", "s p a", "spa", "s a s", "sas", "s n c", "snc", "s r l s", "srls",
	"s c a r l", "scarl", "s c r l", "scrl", "soc coop", "societa cooperativa", "cooperativa", "onlus"}

// employer finds the employer's fiscal code or VAT number and its name in
// the header of the payslip, above the item table: below it are the bank,
// the payroll software and other companies. A code on a line that says what
// it is wins over one alone.
func employer(header []line) payslip.Party {
	var out payslip.Party
	var loose string
	for i, l := range header {
		for _, t := range l.toks {
			v := strings.Trim(t.Text, ".,:;()[]")
			if !validVAT(v) {
				continue
			}
			if out.ID == "" && l.has(employerWords...) {
				out.ID = v
			}
			if loose == "" {
				loose = v
			}
		}
		if out.Name == "" {
			out.Name = companyName(l)
		}
		if out.Name == "" && i > 0 && isLabel(header[i-1]) {
			out.Name = strings.Join(texts(l.toks), " ")
		}
	}
	if out.ID == "" {
		out.ID = loose
	}
	return out
}

// isLabel reports whether a line holds only the label of the employer.
func isLabel(l line) bool {
	var words []string
	for _, w := range l.words {
		words = append(words, w.text)
	}
	return slices.Contains(employerLabels, strings.Join(words, " "))
}

func texts(toks []layout.Token) []string {
	out := make([]string, len(toks))
	for i, t := range toks {
		out[i] = t.Text
	}
	return out
}

// validVAT reports whether v is an Italian VAT number or company fiscal
// code: eleven digits whose last is the check digit.
func validVAT(v string) bool {
	if len(v) != 11 || strings.Trim(v, "0123456789") != "" || v == "00000000000" {
		return false
	}
	sum := 0
	for i := 0; i < 10; i++ {
		d := int(v[i] - '0')
		if i%2 == 1 {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
	}
	return (10-sum%10)%10 == int(v[10]-'0')
}

// companyName returns the words of the line up to a company form, from the
// start of the phrase that holds it: "GO Reply S.r.l." of "GO Reply S.r.l.
// con Unico Socio", "Esempio Srl" of "Ditta: Esempio Srl".
func companyName(l line) string {
	for _, form := range companyForms {
		_, last, ok := l.find(form)
		if !ok {
			continue
		}
		first := last
		for first > 0 && l.toks[first].X-l.toks[first-1].Right() < 3*l.h && !strings.HasSuffix(l.toks[first-1].Text, ":") {
			first--
		}
		// A label printed before the name is not part of it.
		for first < last && slices.Contains(employerLabels, taxonomy.Normalize(l.toks[first].Text)) {
			first++
		}
		words := make([]string, 0, last-first+1)
		for _, t := range l.toks[first : last+1] {
			words = append(words, t.Text)
		}
		return strings.Join(words, " ")
	}
	return ""
}
