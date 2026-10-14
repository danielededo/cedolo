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

Early design and foundation work; nothing is usable yet. The first release
(v0.1) will be a command-line tool supporting one payroll software layout,
followed by a browser app in English and Italian. See the roadmap in
[docs/design/mvp-design.md](docs/design/mvp-design.md#9-roadmap).

## Features

Planned for the first releases:

- Extraction of header, totals, line items, TFR and leave balances from text
  PDF payslips of supported payroll software.
- Arithmetic cross-checks on every payslip, so each value is marked as verified,
  extracted, inferred, missing, or corrected by you.
- Time series of gross and net pay, deduction breakdown, and comparison between
  periods.
- Plain-language explanations of payslip items, in English and Italian.
- Local storage only, with CSV and JSON export and one-step deletion.

## Privacy

Payslips contain sensitive data, including in some cases health information
and union membership. cedolo processes them on your device, keeps no copy of
the original PDF by default, sends nothing over the network, and contains no
analytics or telemetry. See [ADR 0003](docs/adr/0003-on-device-processing-by-default.md).

Never attach a real payslip or a screenshot of one to an issue or pull request.

## Trying the current CLI

With Go installed:

```
go run ./cmd/cedolo synth -o synthetic.pdf      # write a payslip with invented data
go run ./cmd/cedolo inspect synthetic.pdf       # check that a PDF has a readable text layer
go run ./cmd/cedolo extract synthetic.pdf       # read it: items, totals, cross-checks
go run ./cmd/cedolo compare jan.pdf feb.pdf     # what changed from one month to the next
go run ./cmd/cedolo report -o report.html *.pdf  # an offline HTML report with charts
go run ./cmd/cedolo ccnl -search metalmeccanici  # look up a collective agreement (CCNL)
```

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
│   └── ccnlgen/              # regenerates the CCNL list from the CNEL open data
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
