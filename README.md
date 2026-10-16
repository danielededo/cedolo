# cedolo

cedolo helps employees in Italy understand their payslips (cedolini, buste
paga). It reads the PDF payslips you already have, turns them into clean data,
checks that the numbers add up, and shows how your gross pay, net pay,
deductions, TFR and leave change over time, with plain-language explanations
of every item. Everything runs on your own device: your payslips are never
uploaded anywhere.

cedolo explains what a payslip says. It is not tax, legal or labour advice and
it can be wrong; for decisions, ask your payroll office, a CAF, a patronato or
union office, or a consulente del lavoro.

## Status

v0.1 is a command-line tool for text PDF payslips, independent of the payroll
software that produced them. A web app with the same engine, in English and
Italian, is in progress; see the roadmap in
[docs/design/mvp-design.md](docs/design/mvp-design.md#9-roadmap).

## Features

- Reads header, totals, line items, TFR and leave balances from text PDF
  payslips, without templates per payroll software.
- Cross-checks every payslip with its own arithmetic, so each value is marked
  as verified, extracted, inferred, inconsistent or missing.
- Compares months and explains what changed, such as surcharge instalments,
  the thirteenth month or the year-end tax adjustment.
- Writes an offline HTML report with charts over time, in Italian or English.
- Explains in plain words every kind of item it finds and the terms around
  them, such as imponibile and conguaglio, in the report and with
  `cedolo glossary`.
- Identifies the collective agreement (CCNL) from its CNEL code and shows its
  sector, latest agreement and signatories.

## Privacy

Payslips contain sensitive data, including in some cases health information
and union membership. cedolo processes them on your device, keeps no copy of
the original PDF by default, sends nothing over the network, and contains no
analytics or telemetry. See [ADR 0003](docs/adr/0003-on-device-processing-by-default.md).

Never attach a real payslip or a screenshot of one to an issue or pull request.

## Install

Download the archive for your system from the
[latest release](https://github.com/danielededo/cedolo/releases/latest),
unpack it and run `cedolo` from a terminal. Nothing else is needed. Check the
download against `SHA256SUMS` if you like.

The binaries are not signed. On macOS, allow it once with
`xattr -d com.apple.quarantine cedolo`; on Windows, SmartScreen may ask you to
confirm with "More info", then "Run anyway".

With Go installed you can also run it from source: `go run ./cmd/cedolo`.

## Web app

The same engine runs in the browser: drop your payslip PDFs in the page and
read the report. Everything happens on your device; the files are never
uploaded and nothing is saved. Each release has `cedolo-web_<version>.zip`,
static files you can serve with any web server, for example
`python3 -m http.server --bind 127.0.0.1` in the unpacked folder, then open
http://127.0.0.1:8000 (without `--bind` the server is reachable from your
whole network). Browsers do not run it from a file opened directly.

To build it from source, with Go and Node.js installed:

```
(cd web && npm ci --omit=optional --ignore-scripts)
go run ./cmd/webbuild -o dist/web
(cd web && npm test)        # the web app must read payslips as the CLI does
(cd web && npx playwright install chromium && npm run test:e2e)   # in a real browser
```

## Usage

```
cedolo synth -o synthetic.pdf        # write a payslip with invented data
cedolo inspect synthetic.pdf         # check that a PDF has a readable text layer
cedolo extract synthetic.pdf         # read it: items, totals, cross-checks
cedolo compare jan.pdf feb.pdf       # what changed from one month to the next
cedolo report -o report.html *.pdf   # an offline HTML report with charts
cedolo ccnl -search metalmeccanici   # look up a collective agreement (CCNL)
```

Run `cedolo help` for every command and option. Payslips protected by a
password (often the codice fiscale) are opened after cedolo asks for it in
the terminal; scripts can set `CEDOLO_PDF_PASSWORD` instead. Files that
cannot be read are reported and skipped.

`cedolo extract` does not need to know which payroll software produced the
payslip. Every value is marked as verified (confirmed by the payslip's own
arithmetic), extracted, inferred, inconsistent or missing, so a misreading
shows up instead of hiding behind a plausible number. It works on text PDFs
only; scanned payslips are not supported yet.

If cedolo reads your payslip badly, `cedolo anonymize payslip.pdf > layout.json`
produces a version you can share: only known payroll labels are kept, while
names, companies, identifiers and every digit are masked, together with words
that would reveal health, union membership, family situation or debts. Read
it before sharing it; see [CONTRIBUTING.md](CONTRIBUTING.md#reporting-a-payslip-layout).

`cedolo inspect` prints counts only, never names, amounts or file names, so
its output is safe to share when reporting that a payslip format is not read
correctly. `cedolo layout -redact FILE` prints the positioned text with every
letter and digit masked.

## Development

```
go build ./cmd/cedolo
go test ./...
go vet ./...
go run ./cmd/piiscan .
```

Tests that compare against poppler run only when `pdftotext` is installed. See
[CONTRIBUTING.md](CONTRIBUTING.md) for the contribution workflow and
[docs/COMMIT-CONVENTION.md](docs/COMMIT-CONVENTION.md) for commit messages.

## Layout

```
/
├── README.md                 # this file
├── LICENSE                   # GNU AGPL-3.0
├── AGENTS.md                 # instructions for AI coding agents (CLAUDE.md points here)
├── CONTRIBUTING.md           # contribution workflow, sign-off, personal data rules
├── CODE_OF_CONDUCT.md        # Contributor Covenant
├── SECURITY.md               # private vulnerability reporting
├── CHANGELOG.md              # Keep a Changelog, SemVer
├── go.mod, go.sum            # Go module
├── cmd/
│   ├── cedolo/               # the command-line tool
│   ├── piiscan/              # CI check for codici fiscali and IBANs
│   ├── ccnlgen/              # regenerates the CCNL list from the CNEL open data
│   ├── cedolo-wasm/          # the core compiled to WebAssembly for the web app
│   └── webbuild/             # assembles the web app as static files
├── web/                      # the web app: page, pdf.js text layer, tests
├── internal/
│   ├── layout/               # positioned text: the boundary between PDF readers and the core
│   ├── pdftext/              # PDF readers (PDFium in WebAssembly, poppler)
│   ├── money/                # Italian amount parsing, in cents
│   ├── pii/                  # personal identifier detection
│   ├── payslip/              # normalized payslip model
│   ├── taxonomy/             # canonical item catalogue, bilingual labels
│   ├── extract/              # generic extraction and cross-checks
│   ├── anonymize/            # allowlist anonymizer for sharing layouts
│   ├── insights/             # month-to-month comparison
│   ├── report/               # self-contained HTML report with charts
│   ├── ccnl/                 # national collective agreements by CNEL code (embedded data)
│   ├── i18n/                 # user-facing texts in Italian and English
│   └── synth/                # synthetic payslips for tests, in several styles
├── docs/
│   ├── design/               # product and architecture design, spike reports
│   ├── adr/                  # architecture decision records
│   ├── BRANCHING-STRATEGY.md
│   ├── COMMIT-CONVENTION.md
│   ├── REPO-SETTINGS.md      # GitHub settings checklist
│   └── AI-AGENTS.md          # how agent instructions and skills are organized
├── .agents/skills/           # agent procedures: new-adr, release
├── .claude/skills/           # symlinks to .agents/skills
└── .github/                  # CI workflow, issue and pull request templates, Dependabot
```

## Data sources

cedolo embeds the list of national collective agreements in force published
by the CNEL (Consiglio nazionale dell'economia e del lavoro), Archivio
nazionale dei contratti e degli accordi collettivi di lavoro, as open data
under the [Italian Open Data License v2.0](https://www.dati.gov.it/iodl/2.0/).
The list is in `internal/ccnl/ccnl.tsv`, with the date of the export it was
built from; see [ADR 0007](docs/adr/0007-embed-cnel-contract-list.md).

## License

Copyright (C) 2026 danielededo and cedolo contributors.

cedolo is free software: you can redistribute it and/or modify it under the
terms of the GNU Affero General Public License as published by the Free
Software Foundation, either version 3 of the License, or (at your option) any
later version. See [LICENSE](LICENSE) and
[ADR 0002](docs/adr/0002-license-agpl-3.0.md).
