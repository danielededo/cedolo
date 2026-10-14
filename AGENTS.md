# cedolo

cedolo helps Italian employees understand their payslips (cedolini). It reads
payslip PDFs, extracts and normalizes their data, checks it for internal
consistency, and shows it over time with plain-language explanations. All
processing happens on the user's device. The core is written in Go; see
[docs/design/mvp-design.md](docs/design/mvp-design.md) for the design and
[docs/adr/](docs/adr/) for the decisions behind it.

This file is the single source of truth for agent instructions, as prescribed
by [docs/AI-AGENTS.md](docs/AI-AGENTS.md); `CLAUDE.md` is a pointer to it.

## Commands

- Build: `go build ./cmd/cedolo`
- Test: `go test ./...` (the poppler comparison tests are skipped when
  `pdftotext` is not installed; CI installs it)
- Lint: `gofmt -l .` must print nothing; `go vet ./...`;
  `go run honnef.co/go/tools/cmd/staticcheck@2026.1 ./...`
- Personal data scan: `go run ./cmd/piiscan .`
- Go version: as declared in `go.mod`; the toolchain is downloaded
  automatically if the local one is older.
- Project procedures are available as skills in `.agents/skills/` (exposed to
  Claude Code through symlinks in `.claude/skills/`): `new-adr` and `release`.

## Layout

- `cmd/cedolo`: the CLI. `cmd/piiscan`: the CI scanner for personal identifiers.
  `cmd/ccnlgen`: regenerates `internal/ccnl/ccnl.tsv` from a CNEL open data
  export (ADR 0007); never edit the table by hand.
- `internal/ccnl`: the embedded list of national collective agreements by
  CNEL code. A contract is identified only from a printed code, never from
  its name.
- `internal/layout`: the positioned-text model, the boundary between text
  layers and the core (ADR 0004).
- `internal/pdftext`: PDF readers producing a layout (PDFium in WebAssembly,
  poppler).
- `internal/money`: Italian amount parsing and formatting, always in cents.
- `internal/pii`: detection of codici fiscali and IBANs.
- `internal/synth`: synthetic payslips in several layout styles, the only
  payslip documents tests use.
- `internal/payslip`: the normalized payslip model with value statuses.
- `internal/taxonomy`: canonical item keys, bilingual labels, and the
  description keywords that classify items independently of the vendor.
- `internal/anonymize`: allowlist anonymizer for sharing real layouts.
- `internal/insights`: month-to-month comparison with explanations.
- `internal/report`: the self-contained HTML report; it must never load
  anything from the network (a Content-Security-Policy forbids it and a test
  checks the page).
- `internal/i18n`: user-facing texts in Italian and English; every new text
  goes into both catalogues.
- `internal/extract`: the generic extractor and the arithmetic cross-checks
  (ADR 0006). New layout traits get a test in `foreign_test.go`.

## Conventions

- Commits: Conventional Commits; see [docs/COMMIT-CONVENTION.md](docs/COMMIT-CONVENTION.md).
- Branching: trunk-based with squash merges; see
  [docs/BRANCHING-STRATEGY.md](docs/BRANCHING-STRATEGY.md) and
  [ADR 0001](docs/adr/0001-trunk-based-development.md). The maintainer merges
  every pull request; agents open pull requests but never merge them.
- Commit identity is the maintainer's GitHub account with the GitHub noreply
  address (`danielededo <37807398+danielededo@users.noreply.github.com>`),
  never a personal email address.
- All repository content (docs, comments, commit messages, file names) is
  written in English. The user interface will support English and Italian.
- Significant technical decisions get an ADR (`new-adr` skill).

## Writing style

These rules apply to any text produced in this repository, by humans and
agents alike: documentation, commit messages, pull request and issue text,
code comments.

- Prefer prose. Use bullet or numbered lists only when the content is
  genuinely enumerable or sequential (options, checklists, step-by-step
  procedures), not as a default layout for explanations.
- Start with the substance. No filler or throat-clearing phrases.
- No generic disclaimers, hedging boilerplate, or pseudo-human tone. State
  facts and uncertainty plainly.
- No emoji and no arrow glyphs anywhere. No decorative formatting; in running
  prose avoid bold for emphasis and prefer commas, colons, or separate
  sentences over em dashes.
- Commit messages are plain text: no markdown syntax at all.
- Commit messages and pull request descriptions carry no tool attribution:
  no AI co-author trailers, no session links, no "generated with" footers.

## Boundaries

- Never commit directly to `main`; every change goes through a pull request.
- Never commit, paste, or upload a real payslip, a screenshot of one, or values
  taken from one: not in code, fixtures, issues, pull requests, or logs. Test
  data is synthetic. Public sample payslips found online are used only
  outside the repository.
- No code path may send document content, extracted values, or file names
  over the network, and logs never contain them (ADR 0003).
- No external AI or LLM service is used for extraction (ADR 0003, ADR 0005).
- Money is always `int64` cents, never floating point.
- Test documents are generated at test time from `internal/synth`; never add
  PDF or image fixtures (they are git-ignored on purpose).
- `LICENSE` (AGPL-3.0) and the body of `CODE_OF_CONDUCT.md` are canonical
  third-party texts: never reword them.
- When adding, renaming, or removing top-level files or directories, update
  the layout section in `README.md` in the same pull request.
