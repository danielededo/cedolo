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

Only the text layer exists so far. With Go installed:

```
go run ./cmd/cedolo synth -o synthetic.pdf      # write a payslip with invented data
go run ./cmd/cedolo inspect synthetic.pdf       # check that a PDF has a readable text layer
```

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
│   └── piiscan/              # CI check for codici fiscali and IBANs
├── internal/
│   ├── layout/               # positioned text: the boundary between PDF readers and the core
│   ├── pdftext/              # PDF readers (PDFium in WebAssembly, poppler)
│   ├── money/                # Italian amount parsing, in cents
│   ├── pii/                  # personal identifier detection
│   └── synth/                # synthetic payslips for tests
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

## License

Copyright (C) 2026 danielededo and cedolo contributors.

cedolo is free software: you can redistribute it and/or modify it under the
terms of the GNU Affero General Public License as published by the Free
Software Foundation, either version 3 of the License, or (at your option) any
later version. See [LICENSE](LICENSE) and
[ADR 0002](docs/adr/0002-license-agpl-3.0.md).
