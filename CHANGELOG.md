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

- Each month of the report opens with one sentence under the net pay: how
  much more or less you receive than the month before and the item that
  moved it most, such as "Rispetto a settembre 2026 ricevi 45,40 € in più,
  soprattutto per la voce «Straordinario» (+50,00 €)". For the first month,
  or a thirteenth or fourteenth month, it says how much of every 100 € of
  gross pay you receive.

### Fixed

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

[Unreleased]: https://github.com/danielededo/cedolo/compare/v0.5.0...HEAD
[0.5.0]: https://github.com/danielededo/cedolo/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/danielededo/cedolo/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/danielededo/cedolo/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/danielededo/cedolo/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/danielededo/cedolo/releases/tag/v0.1.0
