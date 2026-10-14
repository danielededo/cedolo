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

[Unreleased]: https://github.com/danielededo/cedolo/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/danielededo/cedolo/releases/tag/v0.1.0
