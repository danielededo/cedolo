// Package ccnl is the list of national collective agreements (CCNL) in
// force, with the unique code the CNEL assigns to each and that employers
// report to INPS. The list is embedded: lookups never use the network.
package ccnl

import (
	_ "embed"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/danielededo/cedolo/internal/taxonomy"
)

//go:embed ccnl.tsv
var table string

// Contract is one national collective agreement in force.
type Contract struct {
	Code      string   `json:"code"`
	Title     string   `json:"title"`
	Sectors   []string `json:"sectors,omitempty"`
	From      string   `json:"from,omitempty"`  // start of the latest agreement, YYYY-MM-DD
	Until     string   `json:"until,omitempty"` // its contractual expiry, YYYY-MM-DD
	Employers []string `json:"employers,omitempty"`
	Unions    []string `json:"unions,omitempty"`
}

// Source names where the list comes from, as its license asks.
const Source = "CNEL, Archivio nazionale dei contratti e degli accordi collettivi di lavoro (open data, Italian Open Data License v2.0)"

var (
	once      sync.Once
	contracts []Contract
	byCode    map[string]int
	asOf      string
)

var datePattern = regexp.MustCompile(`export of (\d{4}-\d{2}-\d{2})`)

func load() {
	byCode = map[string]int{}
	for _, row := range strings.Split(table, "\n") {
		if strings.HasPrefix(row, "#") {
			if m := datePattern.FindStringSubmatch(row); m != nil {
				asOf = m[1]
			}
			continue
		}
		f := strings.Split(row, "\t")
		if len(f) != 7 {
			continue
		}
		byCode[f[0]] = len(contracts)
		contracts = append(contracts, Contract{
			Code: f[0], Title: f[1], Sectors: split(f[2]), From: f[3], Until: f[4],
			Employers: split(f[5]), Unions: split(f[6]),
		})
	}
}

func split(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "|")
}

// AsOf returns the date of the CNEL export the list was built from.
func AsOf() string {
	once.Do(load)
	return asOf
}

// Len returns the number of contracts in the list.
func Len() int {
	once.Do(load)
	return len(contracts)
}

// Lookup returns the contract with the given CNEL code.
func Lookup(code string) (Contract, bool) {
	once.Do(load)
	i, ok := byCode[strings.ToUpper(strings.TrimSpace(code))]
	if !ok {
		return Contract{}, false
	}
	return contracts[i], true
}

var codeToken = regexp.MustCompile(`\b[A-Za-z][0-9A-Za-z]{3}\b`)

// FindCode returns the contract whose CNEL code appears in a text printed
// on a payslip, such as "H011 - Commercio terziario". Only four-character
// words with a digit that are codes in the list are taken.
func FindCode(text string) (Contract, bool) {
	for _, w := range codeToken.FindAllString(text, -1) {
		if !strings.ContainsAny(w, "0123456789") {
			continue
		}
		if c, ok := Lookup(w); ok {
			return c, true
		}
	}
	return Contract{}, false
}

// Words too common in contract titles to tell them apart.
var stopwords = map[string]bool{
	"ccnl": true, "per": true, "i": true, "il": true, "la": true, "le": true, "gli": true, "lo": true,
	"di": true, "da": true, "dei": true, "del": true, "della": true, "delle": true, "dello": true, "degli": true,
	"dal": true, "dalla": true, "dalle": true, "dai": true, "e": true, "ed": true, "a": true, "al": true,
	"alle": true, "ai": true, "in": true, "nel": true, "nelle": true, "nei": true, "con": true,
	"dipendenti": true, "lavoratori": true, "addetti": true, "personale": true, "aziende": true,
	"imprese": true, "settore": true, "contratto": true, "collettivo": true, "nazionale": true, "lavoro": true,
}

func words(s string) []string {
	var out []string
	for _, w := range strings.Fields(taxonomy.Normalize(s)) {
		if !stopwords[w] {
			out = append(out, w)
		}
	}
	return out
}

// Search returns up to n contracts whose title, sector or parties best
// match the query, best first. Names are not unique: many contracts share
// almost the same title, so a match is a suggestion, never an
// identification.
func Search(query string, n int) []Contract {
	once.Do(load)
	q := words(query)
	if len(q) == 0 {
		return nil
	}
	type hit struct {
		i     int
		score int
	}
	var hits []hit
	for i, c := range contracts {
		title := words(c.Title)
		parties := words(strings.Join(slices.Concat(c.Employers, c.Unions), " "))
		sectors := words(strings.Join(c.Sectors, " "))
		score := 0
		for _, w := range q {
			switch {
			case hasPrefix(title, w):
				score += 3
			case hasPrefix(parties, w):
				score += 2
			case hasPrefix(sectors, w):
				score++
			}
		}
		if score > 0 {
			hits = append(hits, hit{i, score})
		}
	}
	slices.SortStableFunc(hits, func(a, b hit) int {
		if a.score != b.score {
			return b.score - a.score
		}
		return strings.Compare(contracts[a.i].Code, contracts[b.i].Code)
	})
	var out []Contract
	for _, h := range hits[:min(n, len(hits))] {
		out = append(out, contracts[h.i])
	}
	return out
}

// hasPrefix reports whether a word in ws starts with w less its final
// vowel, so that "metalmeccanici" and "metalmecc" find "metalmeccaniche".
func hasPrefix(ws []string, w string) bool {
	if len(w) > 3 {
		w = strings.TrimRight(w, "aeiou")
	}
	for _, x := range ws {
		if strings.HasPrefix(x, w) {
			return true
		}
	}
	return false
}
