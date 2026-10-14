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

		"report.title":        "Payslip report",
		"report.range":        "%d payslips, from %s to %s",
		"report.privacy":      "This file contains your payslip data. It was created on your computer, makes no network connections, and is not sent anywhere: keep it private.",
		"report.disclaimer":   "cedolo explains what your payslips say; it is not tax, legal or labour advice and it can be wrong. For decisions, ask your payroll office, a CAF, a patronato or union office, or a consulente del lavoro.",
		"report.latest":       "Latest month: %s",
		"report.net":          "Net pay",
		"report.gross":        "Gross pay",
		"report.deductions":   "Deductions",
		"report.vs_previous":  "%s vs previous month",
		"report.trend":        "Gross and net pay over time",
		"report.trend.note":   "Ordinary monthly payslips only; additional months are listed in the table.",
		"report.composition":  "What is withheld each month",
		"report.contrib":      "Contributions",
		"report.taxes":        "Taxes",
		"report.other":        "Other deductions",
		"report.months":       "Month by month",
		"report.col.period":   "Period",
		"report.col.status":   "Check",
		"report.verified":     "adds up",
		"report.unverified":   "partly checked",
		"report.inconsistent": "does not add up",
		"report.changes":      "What changed",
		"report.no_changes":   "No notable changes.",
		"report.leave":        "Leave balances in the latest payslip",
		"report.col.kind":     "Kind",
		"report.col.previous": "Previous year",
		"report.col.accrued":  "Accrued",
		"report.col.used":     "Used",
		"report.col.remain":   "Remaining",
		"report.items":        "Items in your payslips",
		"report.col.item":     "Item",
		"report.col.label_it": "Italian name",
		"report.col.months":   "Months present",
		"report.generated":    "Generated by cedolo",
		"report.amounts":      "Amounts in euros.",
		"report.table":        "Show the numbers",
		"unit.hours":          "h",
		"unit.days":           "d",
		"ccnl.section":        "Your contract",
		"ccnl.printed":        "Printed on the payslip: %s",
		"ccnl.code":           "CNEL code",
		"ccnl.sectors":        "Sector",
		"ccnl.period":         "Latest agreement",
		"ccnl.from_until":     "from %s to %s",
		"ccnl.from":           "from %s",
		"ccnl.employers":      "Signed by employers",
		"ccnl.unions":         "Signed by unions",
		"ccnl.expired":        "The latest agreement expired on %s. A contract usually keeps applying until it is renewed: ask your union or payroll office whether a renewal is under way.",
		"ccnl.no_code":        "The payslip does not print the CNEL code of the contract. Contract names are not unique, so cedolo does not guess it: look it up with \"cedolo ccnl -search\" or ask your payroll office.",
		"ccnl.source":         "Contract data: %s, list of %s.",
		"ccnl.not_found":      "No contract with code %s in the list of %s.",
		"ccnl.no_results":     "No contract matches.",
		"ccnl.suggestions":    "Contracts with similar names; names are not unique, so check the code with your payroll office:",
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

		"report.title":        "Report cedolini",
		"report.range":        "%d cedolini, da %s a %s",
		"report.privacy":      "Questo file contiene i dati dei tuoi cedolini. È stato creato sul tuo computer, non si collega a internet e non viene inviato da nessuna parte: tienilo riservato.",
		"report.disclaimer":   "cedolo spiega cosa dicono i tuoi cedolini; non è una consulenza fiscale, legale o del lavoro e può sbagliare. Per decidere, rivolgiti all'ufficio paghe, a un CAF, a un patronato o sindacato, o a un consulente del lavoro.",
		"report.latest":       "Ultimo mese: %s",
		"report.net":          "Netto",
		"report.gross":        "Lordo",
		"report.deductions":   "Trattenute",
		"report.vs_previous":  "%s rispetto al mese precedente",
		"report.trend":        "Lordo e netto nel tempo",
		"report.trend.note":   "Solo i cedolini mensili ordinari; le mensilità aggiuntive sono nella tabella.",
		"report.composition":  "Cosa viene trattenuto ogni mese",
		"report.contrib":      "Contributi",
		"report.taxes":        "Imposte",
		"report.other":        "Altre trattenute",
		"report.months":       "Mese per mese",
		"report.col.period":   "Periodo",
		"report.col.status":   "Verifica",
		"report.verified":     "i conti tornano",
		"report.unverified":   "verificato in parte",
		"report.inconsistent": "i conti non tornano",
		"report.changes":      "Cosa è cambiato",
		"report.no_changes":   "Nessuna variazione rilevante.",
		"report.leave":        "Ferie e permessi nell'ultimo cedolino",
		"report.col.kind":     "Tipo",
		"report.col.previous": "Anno precedente",
		"report.col.accrued":  "Maturato",
		"report.col.used":     "Goduto",
		"report.col.remain":   "Residuo",
		"report.items":        "Voci presenti nei tuoi cedolini",
		"report.col.item":     "Voce",
		"report.col.label_it": "Nome in italiano",
		"report.col.months":   "Mesi presenti",
		"report.generated":    "Generato da cedolo",
		"report.amounts":      "Importi in euro.",
		"report.table":        "Mostra i numeri",
		"unit.hours":          "ore",
		"unit.days":           "gg",
		"ccnl.section":        "Il tuo contratto",
		"ccnl.printed":        "Stampato sul cedolino: %s",
		"ccnl.code":           "Codice CNEL",
		"ccnl.sectors":        "Settore",
		"ccnl.period":         "Ultimo accordo",
		"ccnl.from_until":     "dal %s al %s",
		"ccnl.from":           "dal %s",
		"ccnl.employers":      "Firmato dalle parti datoriali",
		"ccnl.unions":         "Firmato dai sindacati",
		"ccnl.expired":        "L'ultimo accordo è scaduto il %s. Di solito il contratto continua ad applicarsi finché non viene rinnovato: chiedi al sindacato o all'ufficio paghe se è in corso un rinnovo.",
		"ccnl.no_code":        "Il cedolino non stampa il codice CNEL del contratto. I nomi dei contratti non sono univoci, quindi cedolo non lo indovina: cercalo con \"cedolo ccnl -search\" o chiedi all'ufficio paghe.",
		"ccnl.source":         "Dati sui contratti: %s, elenco del %s.",
		"ccnl.not_found":      "Nessun contratto con codice %s nell'elenco del %s.",
		"ccnl.no_results":     "Nessun contratto corrisponde.",
		"ccnl.suggestions":    "Contratti con nomi simili; i nomi non sono univoci, quindi verifica il codice con l'ufficio paghe:",
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
