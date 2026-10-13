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
- `cedolo anonymize` prints a payslip's layout with everything but known
  payroll labels masked (names, companies, identifiers, every digit, and
  words revealing health, union membership, family or debts), so a layout
  that cedolo reads badly can be reported without sharing personal data.

### Fixed

- Item table values are assigned to columns by zone, from each header to the
  next, so right-aligned amounts that extend past a narrow header are no
  longer taken for the next column.
- The PDFium text layer joins glyphs of vertical text into words, as poppler
  does, instead of reporting them as scattered letters.

[Unreleased]: https://github.com/danielededo/cedolo/commits/main
<!-- After the first release, switch to:
[Unreleased]: https://github.com/danielededo/cedolo/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/danielededo/cedolo/releases/tag/v0.1.0
-->
