// Package anonymize turns the text layout of a real payslip into one that can
// be shared to report a layout cedolo reads badly. It works by allowlist: a
// token is kept only when every word in it is known payroll vocabulary and it
// contains no digit; every other token is masked character by character
// (letters become x or X, digits 0), keeping its position and shape. Names,
// companies, addresses, identifiers and all amounts are therefore masked
// without having to recognize them. Words that would reveal special
// categories of personal data (health, union membership, family situation,
// debts) are masked even though they are known vocabulary.
package anonymize

import (
	"strings"
	"unicode"

	"github.com/danielededo/cedolo/internal/extract"
	"github.com/danielededo/cedolo/internal/layout"
	"github.com/danielededo/cedolo/internal/taxonomy"
)

// Stats counts kept and masked tokens.
type Stats struct {
	Kept, Masked int
}

// Layout returns an anonymized copy of doc. The input is not modified.
func Layout(doc layout.Document) (layout.Document, Stats) {
	vocab := vocabulary()
	var st Stats
	out := layout.Document{Pages: make([]layout.Page, len(doc.Pages))}
	for i, p := range doc.Pages {
		np := p
		np.Tokens = make([]layout.Token, len(p.Tokens))
		for j, t := range p.Tokens {
			if keep(t.Text, vocab) {
				st.Kept++
			} else {
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
