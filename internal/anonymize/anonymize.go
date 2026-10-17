// Package anonymize turns the text layout of a real payslip into one that can
// be shared to report a layout cedolo reads badly (ADR 0008). It works by
// allowlist: a token is kept only when every word in it is known payroll
// vocabulary and it contains no digit. In a token without digits, the words
// that are not known are masked and the known ones kept; every other token is
// masked whole. Masking goes character by character (letters become x or X,
// digits 0), keeping position and shape. Names, companies, addresses and identifiers are therefore masked
// without having to recognize them. Words that would reveal special
// categories of personal data (health, union membership, family situation,
// debts) are masked even though they are known vocabulary. Amounts are masked
// too unless Options.KeepAmounts is set.
package anonymize

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/danielededo/cedolo/internal/extract"
	"github.com/danielededo/cedolo/internal/layout"
	"github.com/danielededo/cedolo/internal/pii"
	"github.com/danielededo/cedolo/internal/taxonomy"
)

// Stats counts kept and masked tokens.
type Stats struct {
	Kept, Masked int
}

// Options choose what survives besides payroll vocabulary.
type Options struct {
	// KeepAmounts keeps amounts and quantities written with decimals
	// (1.234,56, 168,00, 9,19%), years and month/year periods, so the
	// arithmetic cross-checks can be reproduced. Dates, codes and other
	// numbers stay masked.
	KeepAmounts bool
}

// Layout returns an anonymized copy of doc. The input is not modified.
func Layout(doc layout.Document, o Options) (layout.Document, Stats) {
	// Words printed a character at a time are rejoined first: single
	// letters match no vocabulary, so every label would be masked.
	doc = layout.JoinSpaced(doc)
	vocab := vocabulary()
	var st Stats
	out := layout.Document{Pages: make([]layout.Page, len(doc.Pages))}
	for i, p := range doc.Pages {
		np := p
		np.Tokens = make([]layout.Token, len(p.Tokens))
		for j, t := range p.Tokens {
			switch {
			case keep(t.Text, vocab) || (o.KeepAmounts && number(t.Text)):
				st.Kept++
			case strings.IndexFunc(t.Text, unicode.IsDigit) < 0:
				// Known words of a token keep their place ("ORE/GG./NUM."
				// becomes "ORE/GG./XXX."), so a column header still names
				// its column.
				t.Text = maskWords(t.Text, vocab)
				st.Masked++
			default:
				t.Text = mask(t.Text)
				st.Masked++
			}
			np.Tokens[j] = t
		}
		out.Pages[i] = np
	}
	return out, st
}

func keep(text string, vocab map[string]bool) bool {
	if strings.IndexFunc(text, unicode.IsDigit) >= 0 {
		return false
	}
	words := strings.Fields(taxonomy.Normalize(text))
	if len(words) == 0 {
		return true // punctuation only
	}
	for _, w := range words {
		if !vocab[w] || sensitive[w] {
			return false
		}
	}
	return true
}

var (
	decimal = regexp.MustCompile(`^[-+]?(?:\d{1,3}(?:\.\d{3})+|\d+),\d{1,6}[-+]?%?$`)
	year    = regexp.MustCompile(`^(?:19|20)\d\d$`)
	period  = regexp.MustCompile(`^(?:0?[1-9]|1[0-2])[/.-](?:19|20)\d\d$`)
)

// number tells whether a token is an amount, a quantity, a year or a
// month/year period: values a payslip's arithmetic needs and that identify
// no one by themselves. Full dates, which can be a birth or hiring date, are
// not numbers here.
func number(s string) bool {
	return decimal.MatchString(s) || year.MatchString(s) || period.MatchString(s)
}

// Leaks returns the masked words of the original, four letters or longer,
// that still appear in the anonymized document, and the personal identifiers
// it contains. Anything it returns means the output must not be shared.
func Leaks(original, anonymized layout.Document) []string {
	original = layout.JoinSpaced(original) // as Layout reads it
	out := map[string]bool{}
	for _, p := range anonymized.Pages {
		for _, t := range p.Tokens {
			for _, w := range words(t.Text) {
				out[w] = true
			}
		}
	}
	// Payroll vocabulary is public: a masked "INPS-2025" leaks nothing when
	// "INPS" is kept elsewhere.
	vocab := vocabulary()
	var leaks []string
	seen := map[string]bool{}
	for i, p := range original.Pages {
		for j, t := range p.Tokens {
			if i >= len(anonymized.Pages) || j >= len(anonymized.Pages[i].Tokens) || anonymized.Pages[i].Tokens[j].Text == t.Text {
				continue // kept on purpose
			}
			for _, w := range words(t.Text) {
				if len([]rune(w)) >= 4 && out[w] && !seen[w] && (!vocab[taxonomy.Normalize(w)] || sensitive[taxonomy.Normalize(w)]) {
					seen[w] = true
					leaks = append(leaks, w)
				}
			}
		}
	}
	var all strings.Builder
	for _, p := range anonymized.Pages {
		for _, t := range p.Tokens {
			all.WriteString(t.Text + " ")
		}
	}
	for _, m := range pii.Find(all.String()) {
		leaks = append(leaks, string(m.Kind))
	}
	return leaks
}

// words splits s into lowercase runs of letters.
func words(s string) []string {
	return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return !unicode.IsLetter(r) })
}

// maskWords masks the runs of letters of s that are not known payroll
// vocabulary, or that reveal special categories of data, and keeps the rest.
func maskWords(s string, vocab map[string]bool) string {
	var b strings.Builder
	r := []rune(s)
	for i := 0; i < len(r); {
		if !unicode.IsLetter(r[i]) {
			b.WriteRune(r[i])
			i++
			continue
		}
		j := i
		for j < len(r) && unicode.IsLetter(r[j]) {
			j++
		}
		w := string(r[i:j])
		if n := taxonomy.Normalize(w); vocab[n] && !sensitive[n] {
			b.WriteString(w)
		} else {
			b.WriteString(mask(w))
		}
		i = j
	}
	return b.String()
}

func mask(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case unicode.IsDigit(r):
			return '0'
		case unicode.IsUpper(r):
			return 'X'
		case unicode.IsLetter(r):
			return 'x'
		}
		return r
	}, s)
}

func vocabulary() map[string]bool {
	v := map[string]bool{}
	for _, w := range extract.Vocabulary() {
		v[w] = true
	}
	for _, e := range taxonomy.Entries {
		for _, kw := range e.Keywords {
			for _, w := range strings.Fields(kw) {
				v[w] = true
			}
		}
	}
	for _, w := range strings.Fields(common) {
		v[w] = true
	}
	return v
}

// common holds payroll words that carry structure but no personal
// information, and Italian function words.
const common = `
totale totali netto netta lordo lorda mese mensile anno annuo annuale progressivi progressivo
imponibile imponibili contributo contributi contributiva contributivo previdenziale previdenziali
assistenziale ritenuta ritenute trattenuta trattenute competenze competenza retribuzione retributivi
retribuito retribuita elementi elemento voce voci codice descrizione quantita importo importi base
dato ore giorni gg ordinarie ordinario ordinaria ordinari lavorate lavorati lavorabili festivi
festivita detrazioni detrazione lavoro dipendente dipendenti carico conguaglio fiscale fiscali
irpef inps inail tfr fondo fondi addizionale addizionali regionale comunale acconto saldo rata rate
credito debito arretrati arretrato cedolino busta paga prospetto periodo data assunzione cessazione
qualifica livello mansione matricola posizione azienda ditta sede filiale cognome nome ccnl
contratto scatti anzianita aliquota aliquote scaglione applicata imposta maturato maturata mensilita
straordinario straordinari feriale festivo notturno funzione indennita superminimo minimo
contingenza edr una tantum premio produzione risultato welfare buoni pasto ticket trasferta
rimborso rimborsi spese km chilometrico arrotondamento corrente precedente successivo calcolo
mensile giornaliera giornaliero orario oraria unitario totale complessivo complessiva quota quote
lavoratore datore impiegato impiegata operaio operaia quadro dirigente apprendista tempo pieno
parziale determinato indeterminato part full time ctr c dip az prec corr tot imp rit comp tratt
liv cod descr qta dt riposi conto giustificativi legenda calendario presenze statistici dati neutre
neutra figurative elaborazione lavorate tot res mat god
a ad al alla alle ai agli all che con da dal dalla dei del della delle dello di e ed i il in la le
lo nel nella negli per su sul sulla tra fra un una uno o
`

// sensitive words reveal health, union membership, family situation, debts
// or similar special categories of data, and are masked even when known.
var sensitive = map[string]bool{
	"malattia": true, "infortunio": true, "carenza": true, "ricovero": true, "degenza": true,
	"sindacale": true, "sindacali": true, "sindacato": true, "sciopero": true, "associativa": true,
	"maternita": true, "paternita": true, "congedo": true, "parentale": true, "allattamento": true,
	"gravidanza": true, "nucleo": true, "anf": true, "assegno": true, "familiare": true, "familiari": true,
	"figli": true, "coniuge": true, "lutto": true, "matrimonio": true, "matrimoniale": true,
	"pignoramento": true, "pignoramenti": true, "cessione": true, "quinto": true, "prestito": true,
	"delegazione": true, "finanziamento": true, "alimenti": true, "aspettativa": true,
	"invalidita": true, "disabile": true, "disabilita": true, "handicap": true, "donazione": true, "sangue": true,
}
