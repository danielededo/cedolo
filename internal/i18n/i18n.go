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
		"period.13th.year": "Thirteenth month %d",
		"period.14th.year": "Fourteenth month %d",
		"period.unknown":   "unknown period",
		"change.level":     "Your level is now %[2]s, before %[1]s.",
		"change.ccnl":      "The collective agreement is now %[2]s, before %[1]s.",
		"change.net":       "You receive %[2]s instead of %[1]s (%[3]s%%).",
		"change.gross":     "Gross pay is %[2]s instead of %[1]s (%[3]s%%).",
		"change.item_new":  "A new item: %s, %s €.",
		"change.item_gone": "%s is no longer there.",
		"change.tax_rate":  "Taxes take %[2]s%% of gross pay, before %[1]s%%.",
		"change.inps_rate": "INPS contributions are %[2]s%% of pay, before %[1]s%%.",
		"change.leave":     "%[1]s: %[3]s left, before %[2]s (%[4]s).",
		"change.gap":       "A payslip is missing between %s and %s.",
		"change.unchecked": "Some numbers of this payslip do not add up, so these changes may be reading errors.",

		"why.surcharge_start":  "Regional and municipal income tax surcharges for the previous year are withheld in instalments, usually from January (regional) and March (municipal advance).",
		"why.surcharge_end":    "Surcharge instalments usually end in November.",
		"why.thirteenth":       "The thirteenth month is usually paid in December.",
		"why.fourteenth":       "The fourteenth month, where the contract provides it, is usually paid in June or July.",
		"why.year_end":         "In December and after the tax return, the year-end adjustment (conguaglio) can change taxes a lot.",
		"why.overtime":         "Overtime varies from month to month.",
		"why.new_contract":     "A change of level or contract usually comes with a change of base pay.",
		"why.inps_threshold":   "An additional 1%% INPS contribution applies to pay above a yearly threshold.",
		"why.leave_used":       "Leave taken this month reduces the balance.",
		"why.pension_fund":     "Your pension fund contribution changed: it is deducted from the income IRPEF is computed on, so a larger one lowers the taxes.",
		"why.different_employ": "The collective agreement changed: these payslips may come from different employers.",

		"report.title":                "Payslip report",
		"report.range":                "%d payslips, from %s to %s",
		"report.range.one":            "1 payslip, %s",
		"report.privacy":              "Made on your device with no network connection. It holds your payslip data: keep it private.",
		"report.disclaimer":           "cedolo explains what your payslips say; it is not tax, legal or labour advice and it can be wrong. For decisions, ask your payroll office, a CAF, a patronato or union office, or a consulente del lavoro.",
		"report.net":                  "Net pay",
		"report.gross":                "Gross pay",
		"report.deductions":           "Deductions",
		"report.vs_previous":          "%s vs previous month",
		"report.summary.share":        "Of every 100 € of gross pay you receive %d €; the rest goes to contributions, taxes and other deductions.",
		"report.summary.same":         "You receive the same as in %s.",
		"report.summary.more":         "Compared with %s you receive %s more.",
		"report.summary.less":         "Compared with %s you receive %s less.",
		"report.summary.more_because": "Compared with %s you receive %s more, mostly because of “%s” (%s).",
		"report.summary.less_because": "Compared with %s you receive %s less, mostly because of “%s” (%s).",
		"report.trend":                "Where your gross pay went, month by month",
		"report.trend.tap":            "Tap a column of a chart to see its amounts and open that month.",
		"report.chart.open":           "Open %s",
		"report.variable":             "Overtime, bonuses and allowances, month by month",
		"report.variable.series":      "Overtime, bonuses and allowances",
		"report.years":                "Year by year",
		"report.year.months":          "Payslips",
		"report.year.count_extra":     "%d + %d",
		"report.year.mean":            "Average monthly net",
		"report.year.share":           "Net of every 100 € of gross",
		"report.year.partial":         "* Fewer than twelve monthly payslips: the year is incomplete.",
		"report.year.note":            "Additional months included; the average net counts the monthly payslips only.",
		"report.tfr":                  "TFR accrued in your payslips",
		"report.tfr.series":           "TFR accrued",
		"report.fund":                 "Your pension fund",
		"report.fund.span":            "paid in over %d payslips",
		"report.fund.own":             "Your contribution",
		"report.fund.employer":        "Employer's contribution",
		"report.fund.tfr":             "TFR",
		"report.fund.total":           "To the fund",
		"report.fund.of.own":          "%s from you",
		"report.fund.of.employer":     "%s from the employer",
		"report.fund.of.tfr":          "%s of TFR",
		"report.fund.month":           "To your pension fund this month: %s (%s).",
		"report.fund.note":            "What the payslips you added show as paid in. It is not the fund's balance, which adds the returns and the months missing here: you find that in your fund's reserved area.",
		"report.year.fund":            "Paid into the pension fund",
		"report.year.deductible":      "Pension fund deductible used",
		"report.year.deductible.note": "Your contribution and the employer's take up to 5.164,57 € a year off the taxable income, 5.300 € from 2026; the TFR is not counted. Here only the payslips added count, not payments made straight to the fund.",
		"report.year.deductible.over": "Over the limit in some year: tell your fund the part above it by 31 December of the year after, so that it is not taxed again when paid out.",
		"report.tfr.span":             "accrued over %d payslips",
		"report.tfr.note":             "Without the yearly revaluation or the months missing here. If your TFR goes to a pension fund, the fund holds it.",
		"report.group.time":           "Over time",
		"report.tabs":                 "Sections of the report",
		"report.tab.month":            "The month",
		"report.leave.before":         "before %s",
		"report.leave.accrued":        "accrued %s",
		"report.leave.used":           "used %s",
		"report.prev":                 "Previous month",
		"report.next":                 "Next month",
		"report.group.reference":      "Glossary",
		"report.table.show":           "Show as a table",
		"report.leave.over":           "Holidays and leave over time",
		"report.contrib":              "Contributions",
		"report.taxes":                "Taxes",
		"report.rounding":             "Rounding",
		"report.reread":               "To make the totals add up, cedolo read some amounts from a less usual place on the payslip, or from boxes of their own (%s): compare them with the payslip.",
		"report.col.period":           "Period",
		"report.col.status":           "Check",
		"report.verified":             "adds up",
		"report.unverified":           "partly checked",
		"report.inconsistent":         "does not add up",
		"report.changes":              "What changed",
		"report.all_changes":          "Show all",
		"report.leave":                "Holidays and leave",
		"report.of_gross":             "out of %s gross pay",
		"report.pick":                 "Choose a payslip",
		"report.short.13th":           "13th",
		"report.short.14th":           "14th",
		"report.where":                "Where your gross pay goes",
		"report.part.net":             "Net pay",
		"report.part.net.text":        "What is paid into your account.",
		"report.part.contrib":         "Contributions",
		"report.part.contrib.text":    "Paid to INPS for your pension and other benefits, such as unemployment and sick pay.",
		"report.part.taxes":           "Taxes",
		"report.part.taxes.text":      "IRPEF and the regional and municipal surcharges, paid to the State for you.",
		"report.part.other":           "Other deductions",
		"report.part.other.text":      "Everything else withheld, such as a pension fund, union dues or meal vouchers.",
		"report.earnings":             "What makes up your gross pay",
		"report.adjustment":           "This month's taxes are the year-end or end-of-employment adjustment: the year's IRPEF is worked out again on the whole income, and the difference from what was paid in the months before is taken now. When employment ends, the regional and municipal surtaxes, usually withheld in instalments the year after, are taken all at once too. That is why they are higher than usual.",
		"report.aside":                "Set aside for you",
		"report.info":                 "Other information",
		"report.info.text":            "Printed with the items, they do not change this month's pay and are not set aside for you: amounts paid by the employer, or values for statistics.",
		"report.aside.text":           "Not paid this month: it builds up for you as severance pay (TFR) or in a pension fund.",
		"report.generated":            "Generated by cedolo",
		"unit.hours":                  "h",
		"unit.days":                   "d",
		"web.title":                   "cedolo: understand your payslip",
		"web.heading":                 "Understand your payslip",
		"web.lead":                    "Add the PDFs of your payslips: cedolo checks the totals and explains every item in plain words.",
		"web.feature.private":         "Stays on your device",
		"web.feature.check":           "Checks the totals",
		"web.feature.explain":         "Explains every item",
		"web.github":                  "cedolo on GitHub",
		"web.drop":                    "Drop your payslip PDFs here, or",
		"web.choose":                  "Choose files",
		"web.drop.hint":               "Text PDFs only, not scans. Several months give a better picture.",
		"web.loading":                 "Starting cedolo…",
		"web.reading":                 "Reading %d of %d…",
		"web.file.ok":                 "%d payslip(s) read",
		"web.file.none":               "no payslip recognized",
		"web.file.notext":             "no text in this PDF: it is probably a scan, which is not supported yet",
		"web.file.error":              "could not be opened: %s",
		"web.file.too_big":            "too large: payslips are well under %d MB",
		"web.file.too_long":           "too many pages: payslips have at most %d",
		"web.file.repeated":           "%d payslip(s) read, %d already read in another file and ignored",
		"web.file.locked":             "protected by a password: skipped",
		"web.file.damaged":            "not a PDF, or a damaged one",
		"web.password.title":          "%s is protected by a password",
		"web.password.hint":           "Payslips sent by email often use your codice fiscale as password. It is used only on this device, to open the file.",
		"web.password.open":           "Open",
		"web.password.skip":           "Skip this file",
		"web.password.wrong":          "Wrong password, try again.",
		"web.too_many":                "Too many files at once: only the first %d were read.",
		"web.files.attention":         "Some files need a look:",
		"web.files.pick":              "Choose the file read badly:",
		"web.tabs":                    "Sections of the report",
		"web.download":                "Download the report",
		"web.add":                     "Add payslips",
		"web.added.one":               "Added: %s.",
		"web.added.many":              "Added %d payslips: %s.",
		"web.added.none":              "No new payslip in the files picked.",
		"web.none":                    "No payslip was recognized in these files. If they are text PDFs, the layout may be one cedolo does not read yet.",
		"web.failed":                  "cedolo could not start in this browser: %s",
		"web.source":                  "GitHub",
		"web.anon.button":             "Anonymized copy",
		"web.anon.title":              "Anonymized copy of %s",
		"web.anon.lead":               "A new PDF with only the payslip's text: names, companies, addresses, codes and dates are masked. It is made on this device.",
		"web.anon.report":             "Report the problem on GitHub",
		"web.anon.ask":                "Something read badly? Make an anonymized copy to report it.",
		"web.anon.amounts":            "Keep the amounts, which are needed to find errors in the totals",
		"web.anon.warning":            "Taken together, the amounts can identify you to someone who knows your pay, such as a payroll office or a colleague.",
		"web.anon.review":             "This is all the text the copy contains. Read it before sharing it: if you see anything personal, do not share the copy, and report that instead.",
		"web.anon.download":           "Download the copy",
		"web.anon.close":              "Close",
		"web.anon.file":               "anonymized-payslip.pdf",
		"web.anon.leak":               "cedolo could not anonymize this file safely, so it made no copy. Please report this, without attaching the file.",
		"web.language":                "Language",
		"ccnl.section":                "Your contract",
		"ccnl.printed":                "Printed on the payslip: %s",
		"ccnl.code":                   "CNEL code",
		"ccnl.sectors":                "Sector",
		"ccnl.period":                 "Latest agreement",
		"ccnl.from_until":             "from %s to %s",
		"ccnl.from":                   "from %s",
		"ccnl.employers":              "Signed by employers",
		"ccnl.unions":                 "Signed by unions",
		"ccnl.expired":                "The latest agreement expired on %s. A contract usually keeps applying until it is renewed: ask your union or payroll office whether a renewal is under way.",
		"ccnl.no_code":                "The payslip does not print the CNEL code of the contract. Contract names are not unique, so cedolo does not guess it: look it up with \"cedolo ccnl -search\" or ask your payroll office.",
		"ccnl.source":                 "Contract data: %s, list of %s.",
		"ccnl.not_found":              "No contract with code %s in the list of %s.",
		"ccnl.no_results":             "No contract matches.",
		"ccnl.suggestions":            "Contracts with similar names; names are not unique, so check the code with your payroll office:",
	},
	IT: {
		"period":           "%s %d",
		"period.13th":      "%s %d, tredicesima",
		"period.14th":      "%s %d, quattordicesima",
		"period.13th.year": "Tredicesima %d",
		"period.14th.year": "Quattordicesima %d",
		"period.unknown":   "periodo sconosciuto",
		"change.level":     "Il tuo livello ora è %[2]s, prima era %[1]s.",
		"change.ccnl":      "Il contratto collettivo ora è %[2]s, prima era %[1]s.",
		"change.net":       "Ricevi %[2]s invece di %[1]s (%[3]s%%).",
		"change.gross":     "Il lordo è %[2]s invece di %[1]s (%[3]s%%).",
		"change.item_new":  "C'è una voce nuova: %s, %s €.",
		"change.item_gone": "Non c'è più la voce %s.",
		"change.tax_rate":  "Le imposte pesano il %[2]s%% del lordo, prima il %[1]s%%.",
		"change.inps_rate": "I contributi INPS sono il %[2]s%% della retribuzione, prima il %[1]s%%.",
		"change.leave":     "%[1]s: ne restano %[3]s, prima %[2]s (%[4]s).",
		"change.gap":       "Manca il cedolino tra %s e %s.",
		"change.unchecked": "Alcuni numeri di questo cedolino non tornano: queste variazioni potrebbero essere errori di lettura.",

		"why.surcharge_start":  "Le addizionali regionale e comunale dell'anno precedente si trattengono a rate, di solito da gennaio (regionale) e da marzo (acconto comunale).",
		"why.surcharge_end":    "Le rate delle addizionali di solito terminano a novembre.",
		"why.thirteenth":       "La tredicesima di solito si paga a dicembre.",
		"why.fourteenth":       "La quattordicesima, dove il contratto la prevede, di solito si paga a giugno o luglio.",
		"why.year_end":         "A dicembre e dopo il 730 il conguaglio può cambiare molto le imposte.",
		"why.overtime":         "Gli straordinari variano da un mese all'altro.",
		"why.new_contract":     "Un cambio di livello o di contratto di solito cambia anche la paga base.",
		"why.inps_threshold":   "Sopra una soglia annuale si applica un contributo INPS aggiuntivo dell'1%%.",
		"why.leave_used":       "Le ferie o i permessi goduti nel mese riducono il residuo.",
		"why.pension_fund":     "È cambiato il tuo contributo al fondo pensione: si sottrae dal reddito su cui si calcola l'IRPEF, quindi se cresce le imposte scendono.",
		"why.different_employ": "Il contratto collettivo è cambiato: questi cedolini potrebbero essere di datori di lavoro diversi.",

		"report.title":                "Report cedolini",
		"report.range":                "%d cedolini, da %s a %s",
		"report.range.one":            "1 cedolino, %s",
		"report.privacy":              "Creato sul tuo dispositivo, senza connessioni a internet. Contiene i dati dei tuoi cedolini: tienilo riservato.",
		"report.disclaimer":           "cedolo spiega cosa dicono i tuoi cedolini; non è una consulenza fiscale, legale o del lavoro e può sbagliare. Per decidere, rivolgiti all'ufficio paghe, a un CAF, a un patronato o sindacato, o a un consulente del lavoro.",
		"report.net":                  "Netto",
		"report.gross":                "Lordo",
		"report.deductions":           "Trattenute",
		"report.vs_previous":          "%s rispetto al mese precedente",
		"report.summary.share":        "Di ogni 100 € di lordo ne ricevi %d; il resto va in contributi, tasse e altre trattenute.",
		"report.summary.same":         "Ricevi quanto %s.",
		"report.summary.more":         "Rispetto %s ricevi %s in più.",
		"report.summary.less":         "Rispetto %s ricevi %s in meno.",
		"report.summary.more_because": "Rispetto %s ricevi %s in più, soprattutto per la voce «%s» (%s).",
		"report.summary.less_because": "Rispetto %s ricevi %s in meno, soprattutto per la voce «%s» (%s).",
		"report.trend":                "Dove è andato il lordo, mese per mese",
		"report.trend.tap":            "Tocca una colonna di un grafico per vederne gli importi e aprire quel mese.",
		"report.chart.open":           "Apri %s",
		"report.variable":             "Straordinari, premi e indennità, mese per mese",
		"report.variable.series":      "Straordinari, premi e indennità",
		"report.years":                "Anno per anno",
		"report.year.months":          "Cedolini",
		"report.year.count_extra":     "%d + %d",
		"report.year.mean":            "Netto medio mensile",
		"report.year.share":           "Netto su 100 € di lordo",
		"report.year.partial":         "* Meno di dodici cedolini mensili: l'anno è incompleto.",
		"report.year.note":            "Mensilità aggiuntive comprese; il netto medio conta solo i cedolini mensili.",
		"report.tfr":                  "Il TFR maturato nei tuoi cedolini",
		"report.tfr.series":           "TFR maturato",
		"report.fund":                 "Il tuo fondo pensione",
		"report.fund.span":            "versati in %d cedolini",
		"report.fund.own":             "Il tuo contributo",
		"report.fund.employer":        "Contributo dell'azienda",
		"report.fund.tfr":             "TFR",
		"report.fund.total":           "Al fondo",
		"report.fund.of.own":          "%s da te",
		"report.fund.of.employer":     "%s dall'azienda",
		"report.fund.of.tfr":          "%s di TFR",
		"report.fund.month":           "Al tuo fondo pensione questo mese: %s (%s).",
		"report.fund.note":            "Quanto risulta versato nei cedolini caricati. Non è il saldo del fondo, che comprende i rendimenti e i mesi che mancano qui: lo trovi nell'area riservata del tuo fondo.",
		"report.year.fund":            "Versato al fondo pensione",
		"report.year.deductible":      "Deducibile del fondo usato",
		"report.year.deductible.note": "Il tuo contributo e quello dell'azienda si deducono dal reddito fino a 5.164,57 € l'anno, 5.300 € dal 2026; il TFR non conta. Qui ci sono solo i cedolini caricati, non i versamenti fatti direttamente al fondo.",
		"report.year.deductible.over": "Oltre il limite in qualche anno: comunica al fondo la parte in eccesso entro il 31 dicembre dell'anno dopo, così non viene tassata di nuovo quando la ritiri.",
		"report.tfr.span":             "maturati in %d cedolini",
		"report.tfr.note":             "Senza la rivalutazione annuale né i mesi che mancano qui. Se il TFR va a un fondo pensione, lo trovi nel fondo.",
		"report.group.time":           "Nel tempo",
		"report.tabs":                 "Sezioni del report",
		"report.tab.month":            "Il mese",
		"report.leave.before":         "prima %s",
		"report.leave.accrued":        "maturato %s",
		"report.leave.used":           "goduto %s",
		"report.prev":                 "Mese precedente",
		"report.next":                 "Mese successivo",
		"report.group.reference":      "Glossario",
		"report.table.show":           "Mostra come tabella",
		"report.leave.over":           "Ferie e permessi nel tempo",
		"report.contrib":              "Contributi",
		"report.taxes":                "Imposte",
		"report.rounding":             "Arrotondamento",
		"report.reread":               "Per far tornare i conti, cedolo ha letto alcuni importi da una posizione meno abituale del cedolino o da riquadri separati (%s): confrontali con il cedolino.",
		"report.col.period":           "Periodo",
		"report.col.status":           "Verifica",
		"report.verified":             "i conti tornano",
		"report.unverified":           "verificato in parte",
		"report.inconsistent":         "i conti non tornano",
		"report.changes":              "Cosa è cambiato",
		"report.all_changes":          "Mostra tutte",
		"report.leave":                "Ferie e permessi",
		"report.of_gross":             "su %s di lordo",
		"report.pick":                 "Scegli un cedolino",
		"report.short.13th":           "13ª",
		"report.short.14th":           "14ª",
		"report.where":                "Dove va il tuo lordo",
		"report.part.net":             "Netto",
		"report.part.net.text":        "Quello che ricevi sul conto.",
		"report.part.contrib":         "Contributi",
		"report.part.contrib.text":    "Versati all'INPS per la tua pensione e altre tutele, come disoccupazione e malattia.",
		"report.part.taxes":           "Imposte",
		"report.part.taxes.text":      "IRPEF e addizionali regionale e comunale, versate allo Stato al posto tuo.",
		"report.part.other":           "Altre trattenute",
		"report.part.other.text":      "Tutto il resto trattenuto, come fondo pensione, quota sindacale o buoni pasto.",
		"report.earnings":             "Cosa compone il lordo",
		"report.adjustment":           "Le imposte di questo mese sono il conguaglio di fine anno o di fine rapporto: l'IRPEF dell'anno viene ricalcolata su tutto il reddito e ora si paga la differenza rispetto a quanto versato nei mesi precedenti. A fine rapporto si pagano anche, tutte insieme, le addizionali regionale e comunale che di solito si trattengono a rate l'anno dopo. Per questo sono più alte del solito.",
		"report.aside":                "Messo da parte per te",
		"report.info":                 "Altre informazioni",
		"report.info.text":            "Stampate insieme alle voci, non cambiano la paga del mese e non sono messe da parte per te: importi a carico dell'azienda o dati statistici.",
		"report.aside.text":           "Non ti viene pagato ora: si accumula per te come TFR o nel fondo pensione.",
		"report.generated":            "Generato da cedolo",
		"unit.hours":                  "ore",
		"unit.days":                   "gg",
		"web.title":                   "cedolo: comprendi il tuo cedolino",
		"web.heading":                 "Comprendi il tuo cedolino",
		"web.lead":                    "Aggiungi i PDF dei tuoi cedolini: cedolo controlla i totali e spiega ogni voce in parole semplici.",
		"web.feature.private":         "Resta sul tuo dispositivo",
		"web.feature.check":           "Controlla i totali",
		"web.feature.explain":         "Spiega ogni voce",
		"web.github":                  "cedolo su GitHub",
		"web.drop":                    "Trascina qui i PDF dei cedolini, oppure",
		"web.choose":                  "Scegli i file",
		"web.drop.hint":               "Solo PDF con testo, non scansioni. Più mesi danno un quadro migliore.",
		"web.loading":                 "Avvio di cedolo…",
		"web.reading":                 "Lettura %d di %d…",
		"web.file.ok":                 "cedolini letti: %d",
		"web.file.none":               "nessun cedolino riconosciuto",
		"web.file.notext":             "questo PDF non contiene testo: probabilmente è una scansione, non ancora supportata",
		"web.file.error":              "impossibile aprirlo: %s",
		"web.file.too_big":            "troppo grande: un cedolino pesa molto meno di %d MB",
		"web.file.too_long":           "troppe pagine: un cedolino ne ha al massimo %d",
		"web.file.repeated":           "cedolini letti: %d, di cui %d già letti in un altro file e ignorati",
		"web.file.locked":             "protetto da password: saltato",
		"web.file.damaged":            "non è un PDF, oppure è danneggiato",
		"web.password.title":          "%s è protetto da password",
		"web.password.hint":           "I cedolini inviati per email usano spesso il codice fiscale come password. Serve solo ad aprire il file, su questo dispositivo.",
		"web.password.open":           "Apri",
		"web.password.skip":           "Salta questo file",
		"web.password.wrong":          "Password errata, riprova.",
		"web.too_many":                "Troppi file insieme: sono stati letti solo i primi %d.",
		"web.files.attention":         "Alcuni file vanno controllati:",
		"web.files.pick":              "Scegli il file letto male:",
		"web.tabs":                    "Sezioni del report",
		"web.download":                "Scarica il report",
		"web.add":                     "Aggiungi cedolini",
		"web.added.one":               "Aggiunto: %s.",
		"web.added.many":              "Aggiunti %d cedolini: %s.",
		"web.added.none":              "Nei file scelti non c'è nessun cedolino nuovo.",
		"web.none":                    "In questi file non è stato riconosciuto nessun cedolino. Se sono PDF con testo, potrebbe essere un layout che cedolo non legge ancora.",
		"web.failed":                  "cedolo non è riuscito ad avviarsi in questo browser: %s",
		"web.source":                  "GitHub",
		"web.anon.button":             "Copia anonima",
		"web.anon.title":              "Copia anonima di %s",
		"web.anon.lead":               "Un PDF nuovo con il solo testo del cedolino: nomi, aziende, indirizzi, codici e date sono mascherati. Viene creato su questo dispositivo.",
		"web.anon.report":             "Segnala il problema su GitHub",
		"web.anon.ask":                "Qualcosa è letto male? Crea una copia anonima da segnalare.",
		"web.anon.amounts":            "Tieni gli importi, che servono a trovare gli errori nei totali",
		"web.anon.warning":            "Tutti insieme, gli importi possono farti riconoscere da chi conosce il tuo stipendio, come l'ufficio paghe o un collega.",
		"web.anon.review":             "Questo è tutto il testo che la copia contiene. Leggilo prima di condividerla: se vedi qualcosa di personale, non condividerla e segnalalo invece.",
		"web.anon.download":           "Scarica la copia",
		"web.anon.close":              "Chiudi",
		"web.anon.file":               "cedolino-anonimo.pdf",
		"web.anon.leak":               "cedolo non è riuscito ad anonimizzare questo file in modo sicuro, quindi non ha creato la copia. Segnalalo, senza allegare il file.",
		"web.language":                "Lingua",
		"ccnl.section":                "Il tuo contratto",
		"ccnl.printed":                "Stampato sul cedolino: %s",
		"ccnl.code":                   "Codice CNEL",
		"ccnl.sectors":                "Settore",
		"ccnl.period":                 "Ultimo accordo",
		"ccnl.from_until":             "dal %s al %s",
		"ccnl.from":                   "dal %s",
		"ccnl.employers":              "Firmato dalle parti datoriali",
		"ccnl.unions":                 "Firmato dai sindacati",
		"ccnl.expired":                "L'ultimo accordo è scaduto il %s. Di solito il contratto continua ad applicarsi finché non viene rinnovato: chiedi al sindacato o all'ufficio paghe se è in corso un rinnovo.",
		"ccnl.no_code":                "Il cedolino non stampa il codice CNEL del contratto. I nomi dei contratti non sono univoci, quindi cedolo non lo indovina: cercalo con \"cedolo ccnl -search\" o chiedi all'ufficio paghe.",
		"ccnl.source":                 "Dati sui contratti: %s, elenco del %s.",
		"ccnl.not_found":              "Nessun contratto con codice %s nell'elenco del %s.",
		"ccnl.no_results":             "Nessun contratto corrisponde.",
		"ccnl.suggestions":            "Contratti con nomi simili; i nomi non sono univoci, quindi verifica il codice con l'ufficio paghe:",
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
