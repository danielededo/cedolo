package extract

import "strings"

// Labels are normalized phrases (see taxonomy.Normalize), ordered from most to
// least specific within each list: the first phrase that yields a value wins.

var (
	grossLabels = []string{"totale competenze", "totale spettanze", "totale assegni", "totale lordo mensile",
		"totale lordo", "retribuzione lorda", "lordo mensile", "totale compensi", "lordo"}
	deductionLabels = []string{"totale trattenute", "totale ritenute", "totale trattenute e ritenute"}
	netLabels       = []string{"netto del mese", "netto in busta", "netto a pagare", "netto da pagare",
		"netto busta", "netto mese", "netto"}
	inpsBaseLabels = []string{"imponibile inps", "imponibile previdenziale", "imponibile contributivo",
		"imponibile contributi sociali", "imponibile contributi", "imponibile f p l d", "imp le f p l d",
		"impon contr soc", "imp contr soc", "imp le inps", "imp inps", "imp previdenziale"}
	irpefBaseLabels = []string{"imponibile irpef", "imponibile fiscale", "imp irpef", "imp fiscale"}
	socialLabels    = []string{"totale contributi sociali", "totale tratt sociali", "totale trattenute sociali",
		"totale contributi previdenziali", "totale contributi"}
	taxLabels = []string{"totale trattenute irpef", "totale tratt fiscale", "totale trattenute fiscali",
		"totale ritenute fiscali", "totale ritenute irpef", "totale irpef"}
	inpsLabels = []string{"ctr f p l d", "contributo f p l d", "contributi f p l d", "contributo ivs",
		"contributi ivs", "ctr ivs"}
	roundingPreviousLabels = []string{"arrotondamento mese precedente", "arrotondamento precedente",
		"arrotond precedente", "arrotond preced", "arr precedente", "arr prec"}
	roundingCurrentLabels = []string{"arrotondamento attuale", "arrotond attuale", "arrotondamento corrente",
		"arrotond corrente", "arr attuale"}
	tfrBaseLabels = []string{"retribuzione utile per il t f r", "retribuzione utile tfr", "retribuzione utile t f r",
		"retribuzione per t f r", "retribuzione per tfr", "imponibile tfr", "imponibile t f r", "imp tfr", "retrib utile tfr"}
	tfrLabels = []string{"quota tfr maturata", "quota maturata nel mese", "tfr del mese", "quota tfr",
		"tfr mese", "tfr maturato", "quota maturata"}
)

// labelContinuations are words that, right after a label, show that the
// label is part of a longer one with a different meaning: "totale trattenute"
// followed by "irpef", "imponibile irpef" followed by "annuo".
var labelContinuations = map[string]bool{
	"irpef": true, "fiscale": true, "fiscali": true, "sociali": true, "sociale": true, "previdenziali": true,
	"corpo": true, "t": true, "ts": true, "inail": true, "inps": true, "annuo": true, "annua": true,
	"annue": true, "annui": true, "progressivi": true, "progr": true, "ap": true, "anno": true, "tassaz": true,
}

// Column header words of the item table, by column kind.
var columnWords = map[string]colKind{
	"competenze": colEarnings, "competenza": colEarnings, "spettanze": colEarnings,
	"assegni": colEarnings, "compensi": colEarnings,
	"trattenute": colDeductions, "trattenuta": colDeductions, "ritenute": colDeductions, "ritenuta": colDeductions,
	"importo": colAmount, "importi": colAmount,
	"quantita": colQuantity, "ore": colQuantity, "gg": colQuantity, "giorni": colQuantity,
	"qta": colQuantity, "rif": colQuantity, "riferimento": colQuantity, "numero": colQuantity,
	"base": colBase, "dato": colBase, "unitario": colBase, "tariffa": colBase, "aliquota": colBase,
	"unit": colBase, "neutra": colBase, "neutre": colBase,
	"statistici": colInfo, "figurative": colInfo, "figurativi": colInfo, "neutri": colInfo,
	"descrizione": colDescription, "voce": colDescription, "voci": colDescription,
	"cod": colCode, "codice": colCode,
}

// Words that start a label in the header area; a value read to the right of
// a label stops at the first of these.
var headerLabelWords = map[string]bool{
	"ccnl": true, "applicato": true, "contratto": true, "livello": true, "liv": true, "qualifica": true,
	"matricola": true, "periodo": true, "azienda": true, "ditta": true, "dipendente": true, "cognome": true,
	"nome": true, "mansione": true, "data": true, "assunzione": true, "codice": true, "fiscale": true,
	"posizione": true, "inps": true, "inail": true, "anzianita": true, "scatti": true,
}

var monthNames = map[string]int{
	"gennaio": 1, "febbraio": 2, "marzo": 3, "aprile": 4, "maggio": 5, "giugno": 6,
	"luglio": 7, "agosto": 8, "settembre": 9, "ottobre": 10, "novembre": 11, "dicembre": 12,
}

// Words near a numeric period such as "10/2026" that confirm it is the pay
// period and not some other date.
var periodLabels = []string{"periodo", "mese", "competenza", "retribuzione", "paga"}

var (
	thirteenthWords = []string{"tredicesima", "13ma", "13esima", "13a mensilita", "13 mensilita", "13a mens",
		"13 mens", "xiii mensilita", "gratifica natalizia"}
	fourteenthWords = []string{"quattordicesima", "14ma", "14esima", "14a mensilita", "14 mensilita", "14a mens",
		"14 mens", "xiv mensilita"}
	// additionalWords name an additional month without saying which one;
	// the month of payment tells (June or July for the fourteenth).
	additionalWords = []string{"mensilita aggiuntiva", "mens aggiuntiva", "mensilita supplementare"}
)

// Vocabulary returns every word the extractor recognizes in labels, column
// headers, periods and leave tables, normalized. The anonymizer keeps these
// words and masks everything else.
func Vocabulary() []string {
	var phrases []string
	for _, list := range [][]string{grossLabels, deductionLabels, netLabels, inpsBaseLabels, irpefBaseLabels,
		tfrLabels, tfrBaseLabels, socialLabels, taxLabels, inpsLabels, roundingPreviousLabels, roundingCurrentLabels,
		periodLabels, thirteenthWords, fourteenthWords, additionalWords} {
		phrases = append(phrases, list...)
	}
	for w := range columnWords {
		phrases = append(phrases, w)
	}
	for w := range headerLabelWords {
		phrases = append(phrases, w)
	}
	for w := range monthNames {
		phrases = append(phrases, w)
	}
	for _, p := range leaveKinds {
		phrases = append(phrases, p.words)
	}
	for _, p := range leaveMeasures {
		phrases = append(phrases, p.words)
	}
	var words []string
	for _, p := range phrases {
		words = append(words, strings.Fields(p)...)
	}
	return words
}
