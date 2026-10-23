# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

<!--
How to use this file:

- Add every user-visible change under "Unreleased", in the appropriate category:
  Added, Changed, Deprecated, Removed, Fixed, Security.
- When releasing, rename "Unreleased" to the new version and date
  (e.g. "[1.2.0] - 2026-08-23"), then start a fresh "Unreleased" section on top.
- Keep entries short, in plain language, written for users, not commit messages.
- Link versions to the corresponding GitHub compare/tag URLs at the bottom.
-->

## [Unreleased]

### Added

- A person with payslips of several years can see one year at a time, or
  all: a choice of years under their name narrows the whole report, its
  months, charts and totals, and keeps the tab shown. A month is still
  compared with the one before it in the year before.

### Changed

- The month opens with one line less: "Net pay for October 2026" over the
  amount, and whether the numbers add up beside the gross pay below it.
- "Where your gross pay goes" reads as a waterfall: from the gross pay,
  what is added to it, such as an expense refund or a contribution relief,
  then each deduction, down to the net pay, each with a span that starts
  where the one before it ended. The gross pay's items open from its row,
  and what is added is no longer counted in it.
- On a wide screen the month takes two columns, what it is beside where the
  gross pay went, and the glossary two; the charts over time keep a size at
  which their text reads well, and the web app widens with the report.

## [0.9.0] - 2026-10-22

### Added

- The report knows whose payslips they are: it reads the worker's codice
  fiscale and name and the employer's fiscal code or VAT number and name
  (ADR 0010). The name is at the top of the report. Payslips of two or more
  people make a report for each, chosen at the top; a month is compared only
  with the previous one of the same person and employer, and when a person
  has several employers each month names its employer.
- The year by year table says how much of the pension fund deductible
  each year used: your contribution and the employer's, without the TFR,
  against the limit of 5.164,57 € a year, 5.300 € from 2026. A year over
  the limit comes with what to do about it.
- What goes into your supplementary pension fund: each month says how much
  it paid in, from you, from the employer and as TFR; "Over time" adds it
  up with a chart of the three parts, and the year by year table has a row
  for it. It is what the payslips show as paid in, not the fund's balance.
- In the web app, payslips can be added to a report already shown, with
  "Add payslips" or by dropping them on the page; a payslip already read is
  pointed out and not counted twice.
- The report sums the payslips of each year: gross, contributions, taxes,
  net, the average monthly net and how much of every 100 euros of gross was
  paid.
- A chart follows overtime, bonuses and allowances month by month, on a
  scale of their own, so that the part of the pay that changes shows.
- A chart follows the TFR accrued in the payslips, added up month by month.

### Changed

- The note confirming the payslips added goes after five seconds: it
  dissolves, then the room it took closes smoothly; not while the pointer
  or the focus is on it.
- In the web app the page no longer lists the files: it says only how many
  need a look, if any. The files are in a sheet of their own, opened from
  that line or from "Files read" under the report: what needs a look
  first, the files already read as one line, then the payslips read by
  month, the latest first, with the anonymized copy of each. The back
  button closes it.
- The report has four tabs, each answering one question. "The month" puts
  what changed right under the net pay; the gross pay's items open from a
  row above where it goes; what is set aside, holidays and leave, and other
  information are rows that open. "Over time" keeps the charts and the year
  by year totals. "Accrued", new, gathers what builds up for you: holidays
  and leave, the pension fund with its yearly deductible, and the TFR.
  The "net of every 100 € of gross" row is gone, and the notes are shorter.
- Swiping between tabs follows the finger, goes on a flick or past a
  quarter of the screen and springs back otherwise; it works over charts
  and over the months row once it can scroll no further.
- Charts on a phone: a touch shows the amounts of a column under the
  chart, where the finger does not cover them, with a button to open that
  month, instead of leaving the chart at once; the other columns fade. A
  finger dragged sideways moves along the months instead of changing tab.
- The pension fund chart starts at the first payslip that paid into the
  fund, without empty columns for the months before.
- The report is split into three tabs: the month, "Over time" with the
  year by year totals and the charts, and "Glossary" with the payslip words
  and the contract, open rather than collapsed. A link under the month leads to the payslips over time,
  and a column of a chart leads back to its month. A single payslip keeps
  one page; printing shows every tab.
- Holiday and leave balances are shown with the month they belong to,
  instead of only the latest ones at the bottom of the page.
- What makes up the gross pay and the amounts set aside have headings of
  their own, apart from where the gross pay goes.
- The month by month table opens under the chart it shows, as "Show as a
  table".
- On a phone the charts are drawn for its width, with text that stays
  readable, instead of a wide chart scaled down.
- The TFR shows its total in large type over a small line.
- A month no longer repeats in its list of changes what its summary already
  says: the change of the net pay, whose reasons now follow the summary, the
  item that moved it most, and the leave balances shown in its table. The
  TFR is no longer a row of the year by year table, as its own section adds
  it up.

- The web app's report keeps two buttons, "Download the report" and "Add
  payslips"; "Start over" is gone, as reloading the page starts over.

- In the web app the report's tabs sit in the bar at the top, which stays
  in view while scrolling; the tab picked stays picked when payslips are
  added. A sideways swipe moves to the next or previous tab, and arrows
  beside the month step to the month before or after.
- The list of files shows only when a file needs a look; "Something read
  badly?" sits under the report and opens it to pick the file to copy.

- After "Add payslips" the web app says what was added ("Added: March
  2026."), or that the files picked held no new payslip.
- Fewer small texts: the hints and notes that repeated what a heading or a
  tab already says are gone, and inside the web app the report no longer
  repeats its title. Holiday and leave balances are a short list, the
  balance first and how it moved under it, instead of a five-column table.

### Fixed

- "Set aside for you" includes your own pension fund contribution withheld
  from the pay, so that it matches what goes to the fund, and says that
  part is also among the other deductions; "Other deductions" says when it
  is mostly the pension fund. A separate line about the fund is left only
  when part of the TFR stays with the employer.
- The month's summary can name the taxes or the contributions as what
  moved the net pay, as in a year-end adjustment, and a month that
  explains its adjustment no longer repeats the general note about it.
- "Set aside for you" lists only the TFR and the pension fund. Other values
  printed with the items, such as a contribution relief of the employer's
  or a tax deduction, are listed apart as "Other information", without a
  total.
- A month whose taxes are the year-end or end-of-employment adjustment
  (conguaglio) now says so, and why they are higher than usual; when the
  payslip prints its parts, the taxes list the IRPEF still due and the
  regional and municipal surtaxes withheld at once. The taxes are no
  longer pointed out as read from an unusual place.
- Your own pension fund contribution, which some payroll software prints
  inside the contributions box beside INPS, is no longer counted as a
  contribution to INPS: it is among the other deductions, a change of it
  explains a change of the net, and a lower tax share is explained by it.
- Percentages in the changes use the decimal comma in Italian: "-30,3%".

## [0.8.0] - 2026-10-18

### Changed

- When a number next to a total is not its value, cedolo now tries the
  other places the total could be read from and keeps the one with which
  the payslip adds up. The report says when this happens, so the amount
  can be checked against the payslip.

### Fixed

- Payslips that print the contributions total at the foot of a tall box,
  the taxes as "IRPEF + IMP. SOST.", or the family allowance outside the
  item table now add up.
- A final payslip whose gross already leaves out the pay withheld for an
  absence, and whose taxes are the separate taxation of TFR plus the
  end-of-employment IRPEF adjustment, now adds up.
- The 80 euro bonus ("Credito art.1 DL 66/2014") is recognized as a tax
  credit.

## [0.7.0] - 2026-10-17

### Added

- Holidays and leave show how each balance moved over the months, one
  small chart per kind with its latest value, above the table.

### Changed

- The chart of pay over time shows where the gross pay of each payslip
  went, as stacked columns in the colors of the month's bar: net pay,
  contributions, taxes and other deductions. Thirteenth and fourteenth
  months have their own column, the axis reads in euros ("3.000 €"), each
  column tells its amounts under the pointer, and tapping one opens that
  month.

### Fixed

- Tax return settlements paid or withheld in the payslip ("IRPEF A
  CREDITO -MOD.730-", "ADD.REG.CREDITO -MOD.730-", assistenza fiscale) are a
  kind of their own, "Conguaglio 730", with its explanation. A refund is
  left out of the gross pay, as payroll software prints it, so such a
  payslip no longer shows that its numbers do not add up.

## [0.6.0] - 2026-10-17

### Added

- Each month of the report opens with one sentence under the net pay: how
  much more or less you receive than the month before and the item that
  moved it most, such as "Rispetto a settembre 2026 ricevi 45,40 € in più,
  soprattutto per la voce «Straordinario» (+50,00 €)". For the first month,
  or a thirteenth or fourteenth month, it says how much of every 100 € of
  gross pay you receive.

### Fixed

- A file that is not a payslip, such as a tax return summary with a date
  that reads like a pay period, is listed as "no payslip recognized"
  instead of showing empty months in the report.
- The row of months in the report no longer jumps when you pick a month:
  it moved whenever the report changed height. A month at the edge of the
  row now slides into view.
- The report opens on the latest payslip even when it is a thirteenth or
  fourteenth month, highlighted in the month row. Before, it opened on the
  last ordinary month.
- The anonymized copy reads back like the payslip in more layouts. Column
  headings that mix known and unknown abbreviations, such as
  "ORE/GG./NUM.", keep their known words, so the copy still has its
  quantity and rate columns and their items. Text is placed so that labels
  in two sizes stay on one line, so totals labeled that way are still found.

## [0.5.0] - 2026-10-17

### Added

- A fourth synthetic layout style, `cedolo synth -style spaced`, that
  prints every character on its own with its totals under small labels, a
  statistics column beside them and a row of year-to-date values. The
  tests read it on every change, with PDFium and in the web app, so
  payslips laid out this way keep being read correctly.

### Fixed

- The anonymized copy of a payslip printed one character at a time keeps
  its labels and amounts. Before, every character was masked on its own and
  the copy could not be read back.
- Payslips that print each character on its own, with extra spacing, are
  read: before, every word and amount came out in pieces, so no total was
  found and the month was unknown. This works both in the web app and in the
  terminal.
- Totals printed under small labels are read from their own cell, not from a
  statistics column printed just beside the labels. Abbreviated labels such
  as "TOT. TRATTENUTE", "TOT. TRAT. IRPEF" and "TOT. CONTR. SOC." are
  recognized, and so are the regional and municipal surcharges written
  "ADDIZ." and the contribution relief written "ESON. IVS".
- The IRPEF taxable base is no longer read from a row of year-to-date
  values, such as one labeled "PROGRES. ANNUI" or "PROGR.", which some
  payslips print under the same column headings as the month's values.
- Thirteenth and fourteenth month payslips are recognized in more of the
  ways payroll software prints them: "13ª MENSILITÀ", "XIII MENSILITA'",
  "13^ MENS.", "MENSILITÀ AGGIUNTIVA" (the fourteenth when paid in June or
  July), and a period numbered 13 or 14 such as "13/2025". Their items are
  classified as thirteenth or fourteenth month instead of other earnings,
  and "14ma" at the start of a description is no longer taken for an item
  code. A header naming the additional month with its year only, such as
  "13MA MENS. 2025", gives the period too. A fourteenth month known only
  by its year is shown as "Quattordicesima 2025" instead of an unknown
  period.

## [0.4.0] - 2026-10-17

### Changed

- The web app's Italian heading reads "Comprendi il tuo cedolino".
- Smoother steps in the web app: a progress bar while the files are read,
  the start page fading out and the results rising in, a placeholder in the
  report's place until it is ready instead of an empty frame, and the same
  in reverse on "Start over". People who ask their system for reduced
  motion get no animation.

### Fixed

- After a new version is published, the web app no longer mixes files
  cached from the previous one, which could leave the report hidden behind
  its placeholder. Each build now serves its files from its own folder.
- With many months, the report's row of months opens scrolled to the month
  it shows, the latest, instead of starting from January with the marked
  month out of sight.

## [0.3.0] - 2026-10-17

### Changed

- The report is simpler and interactive. It shows one payslip at a time,
  the latest first, with a picker for the others: the net pay in large
  type, a bar of where the gross pay goes (net, contributions, taxes,
  other deductions), and every printed item with its explanation one tap
  away. Pay over time appears from two months on; the month table, leave,
  the contract and the payslip's words are closed until opened.
- "What changed" speaks plainly ("You receive 1.668,00 € instead of
  1.667,40 €") and shows the two most telling changes; the others are
  behind "Show all".
- "Anonymized copy" is offered next to files where no payslip was read;
  for the others it appears after "Something read badly?".
- The web app's start page is shorter, and once the payslips are read it
  makes room for the report: the introduction is hidden until "Start over".
- The web app has a more polished look: a logo, a link to the project on
  GitHub in the header, the three promises as small labelled icons, a
  clearer upload area with a "Choose files" button, icons on the main
  buttons and a tidier footer, in light and dark mode.

### Added

- An issue form, "A payslip is read badly", guides a report with the
  anonymized copy; the web app links to it from the copy.
- `cedolo anonymize -o anonymized.pdf` writes a copy of a payslip to share
  when cedolo reads it badly. It is a new PDF rebuilt from the masked text,
  with nothing else from the original file. Amounts are now kept by default
  so errors in the totals can be reproduced; `-amounts mask` masks them as
  before. Dates and other numbers stay masked. Before writing, cedolo
  checks that no masked word, codice fiscale or IBAN survived.
- The web app makes the same anonymized copy: "Anonymized copy" next to a
  file shows all the text the copy contains, with or without amounts, and
  downloads it under a fixed name.

### Fixed

- In the web app the disclaimer appeared twice at the end of the page: the
  report now leaves its own footer to the page, and a large gap below it is
  gone. A downloaded report keeps its footer.

## [0.2.0] - 2026-10-16

### Added

- A glossary: the report explains every kind of item found in your
  payslips (IRPEF, addizionali, TFR, ratei and so on) and the words around
  them, such as imponibile and conguaglio, in Italian or English.
  `cedolo glossary` prints it in the terminal, or only the entries matching
  some words.

- Payslips protected by a password open in the CLI, which asks for it in
  the terminal without showing it (or reads `CEDOLO_PDF_PASSWORD`), and in
  the web app, which asks in the page. A password that worked is tried first
  on the next protected file, since one employer usually uses one; it is
  kept in memory only.
- `cedolo synth -password` and `-rotate` write protected and turned
  synthetic payslips, for tests.

- A web app: drop the PDFs of your payslips in the page and get the report,
  in Italian or English. It runs entirely in the browser (pdf.js reads the
  PDFs, the cedolo core runs as WebAssembly): files are never uploaded and
  nothing is saved. It is a set of static files, built with
  `go run ./cmd/webbuild`, attached to each release and publishable with
  GitHub Pages.

### Security

- The web app parses PDFs in the page instead of a Web Worker, so its
  Content-Security-Policy also covers pdf.js: a worker loaded from a file
  does not inherit a policy set by a `<meta>` tag. The policy now also
  forbids workers, fonts, frames, plugins and media, and DNS prefetching is
  off. pdf.js hands no fonts to the browser and ignores forms and system
  fonts. Files over 20 MB or 50 pages are refused, and an internal error no
  longer stops the core. See `docs/design/web-security.md`.
- GitHub Actions are pinned to commit hashes, checkouts keep no credentials
  where none are needed, and npm packages are installed without running
  their install scripts.

### Fixed

- The web app opens payslips in Safari (iPhone, iPad, Mac): pdf.js reads
  text with a feature Safari lacks, now filled in, and the browser tests run
  in WebKit too. An empty white frame no longer shows when no payslip is
  read.
- In the web app the report no longer grows without end, which left a
  large empty space below it on phones: it now takes the height of its
  content.
- Two complete payslips of the same month in one file are read as two, not
  merged into one that does not add up.
- A file that cannot be read no longer stops the others: the CLI and the web
  app report it in plain words (protected, damaged, too large) and carry on.
- The same payslip given twice is counted once, with a note.
- The web app reads pages stored turned for display with their own
  coordinates, and uses the pdf.js build for older browsers, where protected
  PDFs otherwise failed to open; a browser that cannot run it gets a plain
  message instead of an empty page.
- The synthetic "grid" payslip printed the contract name past the edge of
  the page.

## [0.1.0] - 2026-10-14

### Added

- Each GitHub release has ready-to-run binaries for Linux, macOS and
  Windows (amd64 and arm64), built without external dependencies.
- `cedolo inspect` reports whether PDF files have a readable text layer,
  printing counts only so the output contains no personal data.
- `cedolo layout` prints the positioned text of a PDF as JSON, with `-redact`
  to mask letters and digits.
- `cedolo synth` writes a synthetic payslip PDF with invented data, in three
  layout styles (`-style classic|signed|grid`).
- `cedolo extract` reads payslips from any payroll software with a text PDF:
  period, contract level, items classified into a canonical taxonomy, totals,
  taxable bases and TFR, each marked as verified, extracted, inferred,
  inconsistent or missing according to arithmetic cross-checks.
- `cedolo extract` also reads leave balances (ferie, permessi and ROL, ex
  festività, banca ore) printed as rows, columns or inline, and checks that
  previous plus accrued minus used equals the remaining balance.
- Extraction now handles contributions, taxes and roundings printed in
  summary boxes outside the item table, ignores columns that are not pay
  (neutral items, statistical data, presence calendars), reads labels split
  over two lines, and decides from the payslip's arithmetic what its printed
  totals mean. Leave balances are also read from rows of "kind measure"
  cells and from side-by-side blocks with vertical titles. Checks that can
  legitimately fail (INPS rate, IRPEF base, TFR ratio) are shown as notes.
- Rows in neutral or statistical columns are kept as information items,
  never summed: for example the TFR transferred to a pension fund or the
  pension contributions. The month's TFR is read from them when no TFR label
  is printed, together with the pay the TFR is computed on, and checked as
  that pay divided by 13.5, with or without the 0.5 percent contribution.
- `cedolo compare` reads payslips of several months and lists what changed
  from one month to the next (level or contract, net and gross beyond 10
  percent, items that appear or disappear, tax and INPS rates, leave
  balances, missing months), each with the most likely explanation, such as
  surcharge instalments, the thirteenth month or the year-end tax
  adjustment. Texts are in Italian or English (`-lang it|en`).
- `cedolo report` writes one self-contained HTML page, readable offline in
  any browser and only by its owner: the latest month at a glance, gross and
  net over time, what is withheld each month (contributions, taxes, other),
  a month-by-month table with the outcome of the checks, what changed, leave
  balances and a glossary of the items found, in Italian or English, with a
  light and a dark theme.
- The list of national collective agreements in force, from the CNEL open
  data (Italian Open Data License v2.0), is built into cedolo. When a
  payslip prints the contract's CNEL code, `extract` and the report show its
  official title, sector, latest agreement and signatories, and the report
  says when the agreement has expired. `cedolo ccnl CODE` shows a contract;
  `cedolo ccnl -search WORDS` suggests contracts by name, warning that names
  are not unique. `cmd/ccnlgen` regenerates the list from a new export.
- `cedolo anonymize` prints a payslip's layout with everything but known
  payroll labels masked (names, companies, identifiers, every digit, and
  words revealing health, union membership, family or debts), so a layout
  that cedolo reads badly can be reported without sharing personal data.

### Fixed

- Header labels where "CCNL" qualifies another field, such as "ORE CCNL"
  (contractual hours) or "GG." printed above "CCNL", are no longer read as
  the contract.
- Item table values are assigned to columns by zone, from each header to the
  next, so right-aligned amounts that extend past a narrow header are no
  longer taken for the next column.
- The PDFium text layer joins glyphs of vertical text into words, as poppler
  does, instead of reporting them as scattered letters.

[Unreleased]: https://github.com/danielededo/cedolo/compare/v0.9.0...HEAD
[0.9.0]: https://github.com/danielededo/cedolo/compare/v0.8.0...v0.9.0
[0.8.0]: https://github.com/danielededo/cedolo/compare/v0.7.0...v0.8.0
[0.7.0]: https://github.com/danielededo/cedolo/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/danielededo/cedolo/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/danielededo/cedolo/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/danielededo/cedolo/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/danielededo/cedolo/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/danielededo/cedolo/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/danielededo/cedolo/releases/tag/v0.1.0
