package i18n

import "maps"

// The glossary explains payslip items (one text per taxonomy key, under
// "item.<key>") and the terms printed around them ("term.<name>" and
// "term.<name>.text"). The texts are general: they say what an item usually
// is, never what a specific contract or year provides, and they avoid rates
// and limits that change every year.

// Terms lists the glossary terms in reading order.
var Terms = []string{"gross", "deductions", "net", "inps_base", "irpef_base", "tax_deductions",
	"year_end", "accruals", "leave", "level"}

// ItemText returns the explanation of a taxonomy key, or "" when there is none.
func ItemText(l Lang, key string) string {
	if _, ok := catalog[l]["item."+key]; !ok {
		return ""
	}
	return T(l, "item."+key)
}

func init() {
	for l, texts := range glossary {
		maps.Copy(catalog[l], texts)
	}
}

var glossary = map[Lang]map[string]string{
	EN: {
		"glossary.intro":    "General explanations: what each item usually is. Your collective agreement, your company's agreements and the rules of each year decide the details.",
		"glossary.terms":    "Payslip words",
		"glossary.no_match": "No item or term matches.",

		"item.tfr.accrued":                  "TFR (trattamento di fine rapporto) is severance pay that builds up every month, a little less than one thirteenth of your yearly pay, and is paid when the employment ends. It is not paid now: the employer keeps it (in larger companies it goes to an INPS fund), or sends it to your supplementary pension fund if you chose one. The amount kept is revalued every year.",
		"item.pension_fund.contribution":    "Money paid into your supplementary pension fund that does not come out of this month's net pay, such as the employer's contribution or your TFR sent to the fund. It is shown for information.",
		"item.rounding":                     "Many payslips round the net pay, for example to the next whole euro. The amount added this month is taken back in the next payslip, so roundings cancel out over time.",
		"item.meal_vouchers":                "Meal vouchers are free of tax and contributions up to a daily limit set by law, higher for electronic vouchers. The part above the limit is taxable: it appears as an earning to be taxed and is then taken back, because you already received it as vouchers.",
		"item.earnings.reimbursement":       "Repayment of expenses you paid for work, such as travel. Documented expenses are usually neither taxed nor subject to contributions, so they add to net pay in full.",
		"item.earnings.contribution_relief": "A reduction of the social security contributions you pay, granted by law to some workers. It shows up as an earning or as lower INPS contributions. From 2025 the general cut for lower incomes became a tax measure instead (see the tax credits item).",
		"item.tax.surcharge.regional":       "A regional tax on top of IRPEF, at a rate set by your region of residence. The amount due for a year is calculated at the year-end adjustment and withheld in monthly instalments during the following year, so it reflects last year's income, not this month's pay.",
		"item.tax.surcharge.municipal":      "A municipal tax on top of IRPEF, at a rate set by your municipality of residence. Part is withheld during the year as an advance (acconto), the rest as a balance (saldo) in instalments during the following year.",
		"item.tax.credit":                   "Amounts that reduce your income tax and are paid in the payslip when the tax due is lower than them, such as the trattamento integrativo and, from 2025, the measures that replaced the contribution cut. They depend on your yearly income: they are paid month by month on an estimate and recalculated at the year-end adjustment, which can take part of them back.",
		"item.tax.irpef":                    "The national income tax. Your employer withholds it each month and pays it to the State for you. It is calculated on taxable income with rates that grow by income bracket, minus the tax deductions for employment and dependants. At the year-end adjustment it is recalculated over the whole year.",
		"item.deductions.pension_fund":      "Your own contribution to a supplementary pension fund, such as the one of your collective agreement. It is subtracted from taxable income up to a yearly limit, so it also lowers your IRPEF.",
		"item.deductions.health_fund":       "Your contribution to the supplementary health fund provided by your collective agreement, which refunds medical expenses. The employer usually pays most of it; the employee part is not taxed up to a limit.",
		"item.deductions.union":             "Union membership dues, withheld only if you joined a union and authorized the employer to collect them.",
		"item.deductions.inps":              "The social security contributions you pay to INPS for your pension and other benefits: a percentage of the contribution base, around 9-10%% for most private-sector employees, plus 1%% on pay above a yearly threshold. Your employer pays a larger share on top, which is not taken from your pay. Contributions are subtracted from taxable income.",
		"item.earnings.thirteenth":          "An extra month's pay provided by every collective agreement, usually paid in December. If you joined or left during the year you get the share for the months worked.",
		"item.earnings.fourteenth":          "An extra month's pay provided only by some collective agreements, such as commerce and tourism, usually paid in June or July.",
		"item.earnings.overtime":            "Hours worked beyond your normal schedule, paid with a premium set by the collective agreement, which also sets limits and rules.",
		"item.earnings.seniority":           "Periodic increases for years of service with the same employer, at the intervals and up to the number set by the collective agreement.",
		"item.earnings.superminimum":        "Pay above the minimum of your collective agreement, agreed with you or by the company. If it is \"absorbable\" (assorbibile) it can be reduced when the contractual minimum rises.",
		"item.earnings.contingency":         "Fixed amounts left from the old cost-of-living allowance, frozen in the early 1990s, such as the indennità di contingenza and the EDR (elemento distinto della retribuzione). Many contracts still list them as part of pay.",
		"item.earnings.bonus":               "Bonuses and one-off payments. Productivity bonuses (premi di risultato) under a company or local agreement can be taxed at a reduced rate instead of IRPEF, or converted into welfare services.",
		"item.earnings.allowance":           "Extra pay for particular conditions of work, such as shifts, night work, travel (trasferta), on-call duty or a role, set by the collective agreement or the company.",
		"item.earnings.leave":               "Pay for holidays (ferie), paid leave hours (ROL and ex festività) and public holidays. They are usually paid like normal work; a payslip may list them separately, or pay a public holiday that falls on a Sunday on top.",
		"item.earnings.sick_pay":            "Pay during sick leave or injury. For many employees INPS pays an allowance from the fourth day, advanced by the employer in the payslip, and the collective agreement often has the employer top it up. INAIL pays for injuries at work.",
		"item.earnings.base_salary":         "The ordinary pay for your level under the collective agreement (paga base, minimo contrattuale), for a full-time month; part-time pay is proportional. Workers paid by the hour see hours times the hourly rate.",
		"item.earnings.other":               "Earnings cedolo could not match to a known kind of item. Check the description printed on the payslip.",
		"item.deductions.other":             "Deductions cedolo could not match to a known kind of item, such as loans repaid from pay (cessione del quinto) or advances. Check the description printed on the payslip.",
		"item.info.other":                   "Information printed with the items that does not change this month's pay, such as amounts paid by the employer or statistical values.",

		"term.gross":               "Gross pay (totale competenze)",
		"term.gross.text":          "The sum of all earnings of the month before anything is withheld.",
		"term.deductions":          "Deductions (totale trattenute)",
		"term.deductions.text":     "Everything withheld from gross pay: social security contributions, taxes and other items such as union dues or a pension fund.",
		"term.net":                 "Net pay (netto in busta)",
		"term.net.text":            "What is actually paid to you: gross pay minus deductions, plus or minus roundings.",
		"term.inps_base":           "Contribution base (imponibile previdenziale)",
		"term.inps_base.text":      "The part of pay on which social security contributions are calculated. Items such as documented reimbursements are left out.",
		"term.irpef_base":          "Taxable income (imponibile fiscale)",
		"term.irpef_base.text":     "The part of pay on which IRPEF is calculated: roughly the gross pay minus your social security contributions and other deductible items, such as pension fund contributions.",
		"term.tax_deductions":      "Tax deductions (detrazioni)",
		"term.tax_deductions.text": "Amounts subtracted from IRPEF, not from income: for employment and for dependants. They shrink as income grows and are recalculated over the year at the year-end adjustment.",
		"term.year_end":            "Year-end adjustment (conguaglio)",
		"term.year_end.text":       "In December, or when the employment ends, the employer recalculates the taxes of the whole year and withholds or refunds the difference. After your tax return (730), refunds or payments also come through the payslip, usually from July.",
		"term.accruals":            "Accruals (ratei)",
		"term.accruals.text":       "The share of the thirteenth and fourteenth month, holidays and leave that you earn for each month worked.",
		"term.leave":               "Holidays and leave (ferie, ROL, ex festività)",
		"term.leave.text":          "Balances of paid time off: what was left from the previous year, plus what accrued this year, minus what you used. ROL are paid leave hours provided by the collective agreement; ex festività are hours that replace public holidays abolished in 1977.",
		"term.level":               "Level (livello)",
		"term.level.text":          "Your grade under the collective agreement. It depends on your tasks and sets your minimum pay.",
	},
	IT: {
		"glossary.intro":    "Spiegazioni generali: cosa è di solito ogni voce. I dettagli dipendono dal tuo contratto collettivo, dagli accordi aziendali e dalle regole di ogni anno.",
		"glossary.terms":    "Le parole del cedolino",
		"glossary.no_match": "Nessuna voce o parola corrisponde.",

		"item.tfr.accrued":                  "Il TFR (trattamento di fine rapporto) è una somma che matura ogni mese, poco meno di un tredicesimo della retribuzione dell'anno, e viene pagata quando il rapporto di lavoro finisce. Non viene pagato ora: lo accantona l'azienda (in quelle più grandi va a un fondo dell'INPS) oppure va al tuo fondo pensione, se l'hai scelto. La parte accantonata viene rivalutata ogni anno.",
		"item.pension_fund.contribution":    "Somme versate al tuo fondo pensione che non escono dal netto di questo mese, come il contributo dell'azienda o il TFR destinato al fondo. Sono indicate per informazione.",
		"item.rounding":                     "Molti cedolini arrotondano il netto, per esempio all'euro successivo. Quanto aggiunto questo mese viene recuperato nel cedolino successivo, quindi gli arrotondamenti si compensano.",
		"item.meal_vouchers":                "I buoni pasto sono esenti da tasse e contributi fino a un limite giornaliero fissato dalla legge, più alto per quelli elettronici. La parte oltre il limite è imponibile: compare come competenza da tassare e poi viene trattenuta, perché l'hai già ricevuta in buoni.",
		"item.earnings.reimbursement":       "Restituzione di spese sostenute per lavoro, come le trasferte. Le spese documentate di solito non sono tassate né soggette a contributi, quindi si aggiungono al netto per intero.",
		"item.earnings.contribution_relief": "Una riduzione dei contributi previdenziali a tuo carico, prevista dalla legge per alcuni lavoratori. Compare come competenza o come minori contributi INPS. Dal 2025 il taglio generale per i redditi più bassi è diventato una misura fiscale (vedi la voce sui bonus fiscali).",
		"item.tax.surcharge.regional":       "Un'imposta regionale che si aggiunge all'IRPEF, con aliquota decisa dalla regione di residenza. Quella dovuta per un anno viene calcolata al conguaglio e trattenuta a rate mensili durante l'anno successivo, quindi riflette il reddito dell'anno prima, non la busta di questo mese.",
		"item.tax.surcharge.municipal":      "Un'imposta comunale che si aggiunge all'IRPEF, con aliquota decisa dal comune di residenza. Una parte viene trattenuta durante l'anno come acconto, il resto come saldo a rate durante l'anno successivo.",
		"item.tax.credit":                   "Somme che riducono le tasse e vengono pagate in busta quando le tasse dovute sono più basse, come il trattamento integrativo e, dal 2025, le misure che hanno sostituito il taglio dei contributi. Dipendono dal reddito dell'anno: si pagano mese per mese su una stima e vengono ricalcolate al conguaglio, che può recuperarne una parte.",
		"item.tax.irpef":                    "L'imposta nazionale sul reddito. L'azienda la trattiene ogni mese e la versa allo Stato al posto tuo. Si calcola sull'imponibile fiscale con aliquote che crescono per scaglioni di reddito, meno le detrazioni per lavoro dipendente e per i familiari a carico. Al conguaglio viene ricalcolata su tutto l'anno.",
		"item.deductions.pension_fund":      "Il tuo contributo a un fondo pensione complementare, come quello del tuo contratto collettivo. Si sottrae dall'imponibile fiscale fino a un limite annuo, quindi riduce anche l'IRPEF.",
		"item.deductions.health_fund":       "Il tuo contributo al fondo sanitario integrativo previsto dal contratto collettivo, che rimborsa spese mediche. Di solito l'azienda ne paga la parte maggiore; la quota a tuo carico non è tassata fino a un limite.",
		"item.deductions.union":             "La quota di iscrizione al sindacato, trattenuta solo se ti sei iscritto e hai autorizzato l'azienda a trattenerla.",
		"item.deductions.inps":              "I contributi previdenziali che versi all'INPS per la pensione e altre prestazioni: una percentuale dell'imponibile previdenziale, circa il 9-10%% per la maggior parte dei dipendenti privati, più l'1%% sulla retribuzione oltre una soglia annua. L'azienda ne paga in aggiunta una parte più grande, che non esce dalla tua busta. I contributi si sottraggono dall'imponibile fiscale.",
		"item.earnings.thirteenth":          "Una mensilità in più prevista da tutti i contratti collettivi, pagata di solito a dicembre. Se sei entrato o uscito durante l'anno ricevi la parte dei mesi lavorati.",
		"item.earnings.fourteenth":          "Una mensilità in più prevista solo da alcuni contratti collettivi, come commercio e turismo, pagata di solito a giugno o luglio.",
		"item.earnings.overtime":            "Ore lavorate oltre l'orario normale, pagate con una maggiorazione fissata dal contratto collettivo, che ne stabilisce anche limiti e regole.",
		"item.earnings.seniority":           "Aumenti periodici per gli anni di servizio presso la stessa azienda, con la cadenza e fino al numero stabiliti dal contratto collettivo.",
		"item.earnings.superminimum":        "Retribuzione oltre il minimo del contratto collettivo, concordata con te o decisa dall'azienda. Se è \"assorbibile\" può ridursi quando il minimo contrattuale aumenta.",
		"item.earnings.contingency":         "Importi fissi rimasti dalla vecchia indennità di carovita, congelata all'inizio degli anni '90, come l'indennità di contingenza e l'EDR (elemento distinto della retribuzione). Molti contratti li indicano ancora come parte della retribuzione.",
		"item.earnings.bonus":               "Premi e pagamenti una tantum. I premi di risultato previsti da un accordo aziendale o territoriale possono essere tassati con un'imposta ridotta al posto dell'IRPEF, oppure convertiti in servizi di welfare.",
		"item.earnings.allowance":           "Compensi in più per particolari condizioni di lavoro, come turni, lavoro notturno, trasferte, reperibilità o un ruolo, previsti dal contratto collettivo o dall'azienda.",
		"item.earnings.leave":               "Retribuzione di ferie, permessi retribuiti (ROL ed ex festività) e festività. Di solito sono pagati come il lavoro normale; il cedolino può indicarli a parte, o pagare in più una festività che cade di domenica.",
		"item.earnings.sick_pay":            "Retribuzione durante malattia o infortunio. Per molti dipendenti l'INPS paga un'indennità dal quarto giorno, anticipata dall'azienda in busta, e il contratto collettivo spesso prevede un'integrazione a carico dell'azienda. Per gli infortuni sul lavoro paga l'INAIL.",
		"item.earnings.base_salary":         "La retribuzione ordinaria del tuo livello secondo il contratto collettivo (paga base, minimo contrattuale), per un mese a tempo pieno; con il part-time è proporzionale. Chi è pagato a ore vede le ore per la paga oraria.",
		"item.earnings.other":               "Competenze che cedolo non ha ricondotto a un tipo di voce noto. Controlla la descrizione stampata sul cedolino.",
		"item.deductions.other":             "Trattenute che cedolo non ha ricondotto a un tipo di voce noto, come prestiti rimborsati con la busta paga (cessione del quinto) o anticipi. Controlla la descrizione stampata sul cedolino.",
		"item.info.other":                   "Informazioni stampate insieme alle voci che non cambiano la retribuzione del mese, come importi a carico dell'azienda o valori statistici.",

		"term.gross":               "Lordo (totale competenze)",
		"term.gross.text":          "La somma di tutte le competenze del mese prima di qualsiasi trattenuta.",
		"term.deductions":          "Trattenute (totale trattenute)",
		"term.deductions.text":     "Tutto ciò che viene tolto dal lordo: contributi previdenziali, tasse e altre voci come la quota sindacale o il fondo pensione.",
		"term.net":                 "Netto (netto in busta)",
		"term.net.text":            "Quello che ti viene pagato davvero: il lordo meno le trattenute, più o meno gli arrotondamenti.",
		"term.inps_base":           "Imponibile previdenziale",
		"term.inps_base.text":      "La parte della retribuzione su cui si calcolano i contributi previdenziali. Ne restano fuori voci come i rimborsi spese documentati.",
		"term.irpef_base":          "Imponibile fiscale",
		"term.irpef_base.text":     "La parte della retribuzione su cui si calcola l'IRPEF: più o meno il lordo meno i tuoi contributi previdenziali e altre voci deducibili, come i contributi al fondo pensione.",
		"term.tax_deductions":      "Detrazioni",
		"term.tax_deductions.text": "Importi che si sottraggono dall'IRPEF, non dal reddito: per lavoro dipendente e per i familiari a carico. Diminuiscono al crescere del reddito e vengono ricalcolate sull'anno al conguaglio.",
		"term.year_end":            "Conguaglio",
		"term.year_end.text":       "A dicembre, o quando il rapporto di lavoro finisce, l'azienda ricalcola le tasse di tutto l'anno e trattiene o rimborsa la differenza. Dopo la dichiarazione dei redditi (730) anche rimborsi o pagamenti passano dalla busta paga, di solito da luglio.",
		"term.accruals":            "Ratei",
		"term.accruals.text":       "La parte di tredicesima e quattordicesima, ferie e permessi che maturi per ogni mese lavorato.",
		"term.leave":               "Ferie, ROL ed ex festività",
		"term.leave.text":          "I saldi dei permessi retribuiti: quanto restava dall'anno precedente, più quanto maturato quest'anno, meno quanto hai goduto. I ROL sono ore di permesso previste dal contratto collettivo; le ex festività sono ore che sostituiscono le festività abolite nel 1977.",
		"term.level":               "Livello",
		"term.level.text":          "Il tuo inquadramento nel contratto collettivo. Dipende dalle mansioni e determina la retribuzione minima.",
	},
}
