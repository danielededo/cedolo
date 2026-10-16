// Package i18n holds the user-facing texts of cedolo in English and Italian.
// Repository content is in English; texts shown to users come from here so
// that every command and the report can switch language.
package i18n

import (
	"fmt"
	"time"

	"github.com/danielededo/cedolo/internal/taxonomy"
)

// Lang is a supported language.
type Lang string

const (
	EN Lang = "en"
	IT Lang = "it"
)

// Parse returns the language for a code, defaulting to Italian, the language
// of the payslips cedolo reads.
func Parse(code string) (Lang, error) {
	switch code {
	case "", "it":
		return IT, nil
	case "en":
		return EN, nil
	}
	return "", fmt.Errorf("unknown language %q (available: it, en)", code)
}

// T formats the text with the given key. A missing key is returned as is, so
// a gap in a catalogue shows up instead of failing.
func T(l Lang, key string, args ...any) string {
	tpl, ok := catalog[l][key]
	if !ok {
		tpl, ok = catalog[EN][key]
	}
	if !ok {
		return key
	}
	return fmt.Sprintf(tpl, args...)
}

var monthsIT = [...]string{"", "gennaio", "febbraio", "marzo", "aprile", "maggio", "giugno", "luglio",
	"agosto", "settembre", "ottobre", "novembre", "dicembre"}

// Month returns the name of a month.
func Month(l Lang, m int) string {
	if m < 1 || m > 12 {
		return "?"
	}
	if l == IT {
		return monthsIT[m]
	}
	return time.Month(m).String()
}

// Template returns the text of a key with its placeholders (%s, %d) left in,
// for callers that substitute them themselves, such as the web app.
func Template(l Lang, key string) string {
	if tpl, ok := catalog[l][key]; ok {
		return tpl
	}
	if tpl, ok := catalog[EN][key]; ok {
		return tpl
	}
	return key
}

// Date renders an ISO date (YYYY-MM-DD) as 31/03/2027 in Italian and
// 2027-03-31 in English; anything else is returned unchanged.
func Date(l Lang, iso string) string {
	t, err := time.Parse(time.DateOnly, iso)
	if err != nil || l != IT {
		return iso
	}
	return t.Format("02/01/2006")
}

// ItemLabel returns the label of a taxonomy key.
func ItemLabel(l Lang, key string) string {
	e, ok := taxonomy.Lookup(key)
	if !ok {
		return key
	}
	if l == IT {
		return e.LabelIT
	}
	return e.LabelEN
}

var catalog = map[Lang]map[string]string{
	EN: {
		"period":           "%s %d",
		"period.13th":      "%s %d, thirteenth month",
		"period.14th":      "%s %d, fourteenth month",
		"period.unknown":   "unknown period",
		"change.level":     "Contract level changed from %s to %s.",
		"change.ccnl":      "Collective agreement changed from %s to %s.",
		"change.net":       "Net pay went from %s to %s (%+.1f%%).",
		"change.gross":     "Gross pay went from %s to %s (%+.1f%%).",
		"change.item_new":  "New item: %s (%s).",
		"change.item_gone": "Item no longer present: %s.",
		"change.tax_rate":  "Taxes went from %.1f%% to %.1f%% of gross pay.",
		"change.inps_rate": "The INPS contribution rate went from %.2f%% to %.2f%%.",
		"change.leave":     "%s remaining: from %s to %s (%s).",
		"change.gap":       "No payslip between %s and %s.",
		"change.unchecked": "Some values of this payslip do not add up; changes involving them may be reading errors.",

		"why.surcharge_start":  "Regional and municipal income tax surcharges for the previous year are withheld in instalments, usually from January (regional) and March (municipal advance).",
		"why.surcharge_end":    "Surcharge instalments usually end in November.",
		"why.thirteenth":       "The thirteenth month is usually paid in December.",
		"why.fourteenth":       "The fourteenth month, where the contract provides it, is usually paid in June or July.",
		"why.year_end":         "In December, and in February or March after the annual tax return, the year-end tax adjustment (conguaglio) can change taxes considerably.",
		"why.overtime":         "Overtime varies from month to month.",
		"why.new_contract":     "A change of level or contract usually comes with a change of base pay.",
		"why.inps_threshold":   "An additional 1%% INPS contribution applies to pay above a yearly threshold.",
		"why.leave_used":       "Leave taken this month reduces the balance.",
		"why.different_employ": "The collective agreement changed: these payslips may come from different employers.",

		"report.title":             "Payslip report",
		"report.range":             "%d payslips, from %s to %s",
		"report.range.one":         "1 payslip, %s",
		"report.privacy":           "Made on your device with no network connection. It holds your payslip data: keep it private.",
		"report.disclaimer":        "cedolo explains what your payslips say; it is not tax, legal or labour advice and it can be wrong. For decisions, ask your payroll office, a CAF, a patronato or union office, or a consulente del lavoro.",
		"report.net":               "Net pay",
		"report.gross":             "Gross pay",
		"report.deductions":        "Deductions",
		"report.vs_previous":       "%s vs previous month",
		"report.trend":             "Gross and net pay over time",
		"report.contrib":           "Contributions",
		"report.taxes":             "Taxes",
		"report.months":            "Month by month",
		"report.col.period":        "Period",
		"report.col.status":        "Check",
		"report.verified":          "adds up",
		"report.unverified":        "partly checked",
		"report.inconsistent":      "does not add up",
		"report.changes":           "What changed",
		"report.leave":             "Holidays and leave",
		"report.col.kind":          "Kind",
		"report.col.previous":      "Previous year",
		"report.col.accrued":       "Accrued",
		"report.col.used":          "Used",
		"report.col.remain":        "Remaining",
		"report.of_gross":          "out of %s gross pay",
		"report.pick":              "Choose a payslip",
		"report.short.13th":        "13th",
		"report.short.14th":        "14th",
		"report.where":             "Where your gross pay goes",
		"report.tap":               "Tap a line to see what it is.",
		"report.part.net":          "Net pay",
		"report.part.net.text":     "What is paid into your account.",
		"report.part.contrib":      "Contributions",
		"report.part.contrib.text": "Paid to INPS for your pension and other benefits, such as unemployment and sick pay.",
		"report.part.taxes":        "Taxes",
		"report.part.taxes.text":   "IRPEF and the regional and municipal surcharges, paid to the State for you.",
		"report.part.other":        "Other deductions",
		"report.part.other.text":   "Everything else withheld, such as a pension fund, union dues or meal vouchers.",
		"report.earnings":          "What makes up your gross pay",
		"report.earnings.text":     "Every amount you earned this month, before anything is withheld.",
		"report.aside":             "Set aside for you",
		"report.aside.text":        "Not paid this month: it builds up for you as severance pay (TFR) or in a pension fund.",
		"report.generated":         "Generated by cedolo",
		"unit.hours":               "h",
		"unit.days":                "d",
		"web.title":                "cedolo: understand your payslip",
		"web.heading":              "Understand your payslip",
		"web.lead":                 "Add the PDFs of your payslips: cedolo checks the totals, explains every item and shows what changes from month to month.",
		"web.private.device":       "Everything happens on this device.",
		"web.private.network":      "Your payslips are never uploaded or sent anywhere.",
		"web.private.storage":      "Nothing is saved: closing the page forgets everything.",
		"web.drop":                 "Drop your payslip PDFs here",
		"web.choose":               "or choose files",
		"web.drop.hint":            "Several months at once give the best picture. Only PDFs with text are supported, not scans or photos.",
		"web.loading":              "Starting cedolo…",
		"web.reading":              "Reading %d of %d…",
		"web.file.ok":              "%d payslip(s) read",
		"web.file.none":            "no payslip recognized",
		"web.file.notext":          "no text in this PDF: it is probably a scan, which is not supported yet",
		"web.file.error":           "could not be opened: %s",
		"web.file.too_big":         "too large: payslips are well under %d MB",
		"web.file.too_long":        "too many pages: payslips have at most %d",
		"web.file.repeated":        "%d payslip(s) read, %d already read in another file and ignored",
		"web.file.locked":          "protected by a password: skipped",
		"web.file.damaged":         "not a PDF, or a damaged one",
		"web.password.title":       "%s is protected by a password",
		"web.password.hint":        "Payslips sent by email often use your codice fiscale as password. It is used only on this device, to open the file.",
		"web.password.open":        "Open",
		"web.password.skip":        "Skip this file",
		"web.password.wrong":       "Wrong password, try again.",
		"web.too_many":             "Too many files at once: only the first %d were read.",
		"web.files":                "Files",
		"web.download":             "Download the report",
		"web.restart":              "Start over",
		"web.none":                 "No payslip was recognized in these files. If they are text PDFs, the layout may be one cedolo does not read yet.",
		"web.failed":               "cedolo could not start in this browser: %s",
		"web.source":               "Source code",
		"web.anon.button":          "Anonymized copy",
		"web.anon.title":           "Anonymized copy of %s",
		"web.anon.lead":            "If cedolo reads this payslip badly, attach this copy to an issue on GitHub. It is a new PDF made of the payslip's text only: names, companies, addresses, tax codes and IBANs, dates and other codes are masked. It is made on this device.",
		"web.anon.amounts":         "Keep the amounts, which are needed to find errors in the totals",
		"web.anon.warning":         "Taken together, the amounts can identify you to someone who knows your pay, such as a payroll office or a colleague.",
		"web.anon.review":          "This is all the text the copy contains. Read it before sharing it: if you see anything personal, do not share the copy, and report that instead.",
		"web.anon.download":        "Download the copy",
		"web.anon.close":           "Close",
		"web.anon.file":            "anonymized-payslip.pdf",
		"web.anon.leak":            "cedolo could not anonymize this file safely, so it made no copy. Please report this, without attaching the file.",
		"web.language":             "Language",
		"ccnl.section":             "Your contract",
		"ccnl.printed":             "Printed on the payslip: %s",
		"ccnl.code":                "CNEL code",
		"ccnl.sectors":             "Sector",
		"ccnl.period":              "Latest agreement",
		"ccnl.from_until":          "from %s to %s",
		"ccnl.from":                "from %s",
		"ccnl.employers":           "Signed by employers",
		"ccnl.unions":              "Signed by unions",
		"ccnl.expired":             "The latest agreement expired on %s. A contract usually keeps applying until it is renewed: ask your union or payroll office whether a renewal is under way.",
		"ccnl.no_code":             "The payslip does not print the CNEL code of the contract. Contract names are not unique, so cedolo does not guess it: look it up with \"cedolo ccnl -search\" or ask your payroll office.",
		"ccnl.source":              "Contract data: %s, list of %s.",
		"ccnl.not_found":           "No contract with code %s in the list of %s.",
		"ccnl.no_results":          "No contract matches.",
		"ccnl.suggestions":         "Contracts with similar names; names are not unique, so check the code with your payroll office:",
	},
	IT: {
		"period":           "%s %d",
		"period.13th":      "%s %d, tredicesima",
		"period.14th":      "%s %d, quattordicesima",
		"period.unknown":   "periodo sconosciuto",
		"change.level":     "Il livello è passato da %s a %s.",
		"change.ccnl":      "Il contratto collettivo è passato da %s a %s.",
		"change.net":       "Il netto è passato da %s a %s (%+.1f%%).",
		"change.gross":     "Il lordo è passato da %s a %s (%+.1f%%).",
		"change.item_new":  "Nuova voce: %s (%s).",
		"change.item_gone": "Voce non più presente: %s.",
		"change.tax_rate":  "Le imposte sono passate dal %.1f%% al %.1f%% del lordo.",
		"change.inps_rate": "L'aliquota dei contributi INPS è passata dal %.2f%% al %.2f%%.",
		"change.leave":     "%s residue: da %s a %s (%s).",
		"change.gap":       "Nessun cedolino tra %s e %s.",
		"change.unchecked": "Alcuni valori di questo cedolino non tornano: le variazioni che li riguardano potrebbero essere errori di lettura.",

		"why.surcharge_start":  "Le addizionali regionale e comunale dell'anno precedente si trattengono a rate, di solito da gennaio (regionale) e da marzo (acconto comunale).",
		"why.surcharge_end":    "Le rate delle addizionali di solito terminano a novembre.",
		"why.thirteenth":       "La tredicesima di solito si paga a dicembre.",
		"why.fourteenth":       "La quattordicesima, dove il contratto la prevede, di solito si paga a giugno o luglio.",
		"why.year_end":         "A dicembre, e a febbraio o marzo dopo il 730, il conguaglio fiscale può cambiare molto le imposte.",
		"why.overtime":         "Gli straordinari variano da un mese all'altro.",
		"why.new_contract":     "Un cambio di livello o di contratto di solito cambia anche la paga base.",
		"why.inps_threshold":   "Sopra una soglia annuale si applica un contributo INPS aggiuntivo dell'1%%.",
		"why.leave_used":       "Le ferie o i permessi goduti nel mese riducono il residuo.",
		"why.different_employ": "Il contratto collettivo è cambiato: questi cedolini potrebbero essere di datori di lavoro diversi.",

		"report.title":             "Report cedolini",
		"report.range":             "%d cedolini, da %s a %s",
		"report.range.one":         "1 cedolino, %s",
		"report.privacy":           "Creato sul tuo dispositivo, senza connessioni a internet. Contiene i dati dei tuoi cedolini: tienilo riservato.",
		"report.disclaimer":        "cedolo spiega cosa dicono i tuoi cedolini; non è una consulenza fiscale, legale o del lavoro e può sbagliare. Per decidere, rivolgiti all'ufficio paghe, a un CAF, a un patronato o sindacato, o a un consulente del lavoro.",
		"report.net":               "Netto",
		"report.gross":             "Lordo",
		"report.deductions":        "Trattenute",
		"report.vs_previous":       "%s rispetto al mese precedente",
		"report.trend":             "Lordo e netto nel tempo",
		"report.contrib":           "Contributi",
		"report.taxes":             "Imposte",
		"report.months":            "Mese per mese",
		"report.col.period":        "Periodo",
		"report.col.status":        "Verifica",
		"report.verified":          "i conti tornano",
		"report.unverified":        "verificato in parte",
		"report.inconsistent":      "i conti non tornano",
		"report.changes":           "Cosa è cambiato",
		"report.leave":             "Ferie e permessi",
		"report.col.kind":          "Tipo",
		"report.col.previous":      "Anno precedente",
		"report.col.accrued":       "Maturato",
		"report.col.used":          "Goduto",
		"report.col.remain":        "Residuo",
		"report.of_gross":          "su %s di lordo",
		"report.pick":              "Scegli un cedolino",
		"report.short.13th":        "13ª",
		"report.short.14th":        "14ª",
		"report.where":             "Dove va il tuo lordo",
		"report.tap":               "Tocca una riga per sapere cos'è.",
		"report.part.net":          "Netto",
		"report.part.net.text":     "Quello che ricevi sul conto.",
		"report.part.contrib":      "Contributi",
		"report.part.contrib.text": "Versati all'INPS per la tua pensione e altre tutele, come disoccupazione e malattia.",
		"report.part.taxes":        "Imposte",
		"report.part.taxes.text":   "IRPEF e addizionali regionale e comunale, versate allo Stato al posto tuo.",
		"report.part.other":        "Altre trattenute",
		"report.part.other.text":   "Tutto il resto trattenuto, come fondo pensione, quota sindacale o buoni pasto.",
		"report.earnings":          "Cosa compone il lordo",
		"report.earnings.text":     "Tutto quello che hai guadagnato nel mese, prima delle trattenute.",
		"report.aside":             "Messo da parte per te",
		"report.aside.text":        "Non ti viene pagato ora: si accumula per te come TFR o nel fondo pensione.",
		"report.generated":         "Generato da cedolo",
		"unit.hours":               "ore",
		"unit.days":                "gg",
		"web.title":                "cedolo: capisci il tuo cedolino",
		"web.heading":              "Capisci il tuo cedolino",
		"web.lead":                 "Aggiungi i PDF dei tuoi cedolini: cedolo controlla i totali, spiega ogni voce e mostra cosa cambia da un mese all'altro.",
		"web.private.device":       "Tutto avviene su questo dispositivo.",
		"web.private.network":      "I tuoi cedolini non vengono mai caricati né inviati da nessuna parte.",
		"web.private.storage":      "Non viene salvato nulla: chiudendo la pagina si dimentica tutto.",
		"web.drop":                 "Trascina qui i PDF dei cedolini",
		"web.choose":               "oppure scegli i file",
		"web.drop.hint":            "Più mesi insieme danno il quadro migliore. Sono supportati solo PDF con testo, non scansioni o foto.",
		"web.loading":              "Avvio di cedolo…",
		"web.reading":              "Lettura %d di %d…",
		"web.file.ok":              "cedolini letti: %d",
		"web.file.none":            "nessun cedolino riconosciuto",
		"web.file.notext":          "questo PDF non contiene testo: probabilmente è una scansione, non ancora supportata",
		"web.file.error":           "impossibile aprirlo: %s",
		"web.file.too_big":         "troppo grande: un cedolino pesa molto meno di %d MB",
		"web.file.too_long":        "troppe pagine: un cedolino ne ha al massimo %d",
		"web.file.repeated":        "cedolini letti: %d, di cui %d già letti in un altro file e ignorati",
		"web.file.locked":          "protetto da password: saltato",
		"web.file.damaged":         "non è un PDF, oppure è danneggiato",
		"web.password.title":       "%s è protetto da password",
		"web.password.hint":        "I cedolini inviati per email usano spesso il codice fiscale come password. Serve solo ad aprire il file, su questo dispositivo.",
		"web.password.open":        "Apri",
		"web.password.skip":        "Salta questo file",
		"web.password.wrong":       "Password errata, riprova.",
		"web.too_many":             "Troppi file insieme: sono stati letti solo i primi %d.",
		"web.files":                "File",
		"web.download":             "Scarica il report",
		"web.restart":              "Ricomincia",
		"web.none":                 "In questi file non è stato riconosciuto nessun cedolino. Se sono PDF con testo, potrebbe essere un layout che cedolo non legge ancora.",
		"web.failed":               "cedolo non è riuscito ad avviarsi in questo browser: %s",
		"web.source":               "Codice sorgente",
		"web.anon.button":          "Copia anonima",
		"web.anon.title":           "Copia anonima di %s",
		"web.anon.lead":            "Se cedolo legge male questo cedolino, allega questa copia a una segnalazione su GitHub. È un PDF nuovo fatto solo del testo del cedolino: nomi, aziende, indirizzi, codici fiscali e IBAN, date e altri codici sono mascherati. Viene creata su questo dispositivo.",
		"web.anon.amounts":         "Tieni gli importi, che servono a trovare gli errori nei totali",
		"web.anon.warning":         "Tutti insieme, gli importi possono farti riconoscere da chi conosce il tuo stipendio, come l'ufficio paghe o un collega.",
		"web.anon.review":          "Questo è tutto il testo che la copia contiene. Leggilo prima di condividerla: se vedi qualcosa di personale, non condividerla e segnalalo invece.",
		"web.anon.download":        "Scarica la copia",
		"web.anon.close":           "Chiudi",
		"web.anon.file":            "cedolino-anonimo.pdf",
		"web.anon.leak":            "cedolo non è riuscito ad anonimizzare questo file in modo sicuro, quindi non ha creato la copia. Segnalalo, senza allegare il file.",
		"web.language":             "Lingua",
		"ccnl.section":             "Il tuo contratto",
		"ccnl.printed":             "Stampato sul cedolino: %s",
		"ccnl.code":                "Codice CNEL",
		"ccnl.sectors":             "Settore",
		"ccnl.period":              "Ultimo accordo",
		"ccnl.from_until":          "dal %s al %s",
		"ccnl.from":                "dal %s",
		"ccnl.employers":           "Firmato dalle parti datoriali",
		"ccnl.unions":              "Firmato dai sindacati",
		"ccnl.expired":             "L'ultimo accordo è scaduto il %s. Di solito il contratto continua ad applicarsi finché non viene rinnovato: chiedi al sindacato o all'ufficio paghe se è in corso un rinnovo.",
		"ccnl.no_code":             "Il cedolino non stampa il codice CNEL del contratto. I nomi dei contratti non sono univoci, quindi cedolo non lo indovina: cercalo con \"cedolo ccnl -search\" o chiedi all'ufficio paghe.",
		"ccnl.source":              "Dati sui contratti: %s, elenco del %s.",
		"ccnl.not_found":           "Nessun contratto con codice %s nell'elenco del %s.",
		"ccnl.no_results":          "Nessun contratto corrisponde.",
		"ccnl.suggestions":         "Contratti con nomi simili; i nomi non sono univoci, quindi verifica il codice con l'ufficio paghe:",
	},
}

// Keys returns the keys of a catalogue, for tests.
func Keys(l Lang) []string {
	var out []string
	for k := range catalog[l] {
		out = append(out, k)
	}
	return out
}
