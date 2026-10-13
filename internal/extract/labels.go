package extract

// Labels are normalized phrases (see taxonomy.Normalize), ordered from most to
// least specific within each list: the first phrase that yields a value wins.

var (
	grossLabels = []string{"totale competenze", "totale spettanze", "totale assegni", "totale lordo mensile",
		"totale lordo", "retribuzione lorda", "lordo mensile", "totale compensi", "lordo"}
	deductionLabels = []string{"totale trattenute", "totale ritenute", "totale trattenute e ritenute"}
	netLabels       = []string{"netto del mese", "netto in busta", "netto a pagare", "netto da pagare",
		"netto busta", "netto mese", "netto"}
	inpsBaseLabels = []string{"imponibile inps", "imponibile previdenziale", "imponibile contributivo",
		"imponibile contributi", "imp inps", "imp previdenziale"}
	irpefBaseLabels = []string{"imponibile irpef", "imponibile fiscale", "imp irpef", "imp fiscale"}
	tfrLabels       = []string{"quota tfr maturata", "quota maturata nel mese", "tfr del mese", "quota tfr",
		"tfr mese", "tfr maturato", "quota maturata"}
)

// Column header words of the item table, by column kind.
var columnWords = map[string]colKind{
	"competenze": colEarnings, "competenza": colEarnings, "spettanze": colEarnings,
	"assegni": colEarnings, "compensi": colEarnings,
	"trattenute": colDeductions, "trattenuta": colDeductions, "ritenute": colDeductions, "ritenuta": colDeductions,
	"importo": colAmount, "importi": colAmount,
	"quantita": colQuantity, "ore": colQuantity, "gg": colQuantity, "giorni": colQuantity,
	"qta": colQuantity, "rif": colQuantity, "riferimento": colQuantity, "numero": colQuantity,
	"base": colBase, "dato": colBase, "unitario": colBase, "tariffa": colBase, "aliquota": colBase,
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
	thirteenthWords = []string{"tredicesima", "13ma", "13a mensilita", "gratifica natalizia", "13 mensilita"}
	fourteenthWords = []string{"quattordicesima", "14ma", "14a mensilita", "14 mensilita"}
)
