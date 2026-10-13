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

- There is no code yet; the Go module, build, test and lint commands arrive
  with phase 0 of the roadmap and must be added here in the same pull request.
- Project procedures are available as skills in `.agents/skills/` (exposed to
  Claude Code through symlinks in `.claude/skills/`): `new-adr` and `release`.

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
- `LICENSE` (AGPL-3.0) and the body of `CODE_OF_CONDUCT.md` are canonical
  third-party texts: never reword them.
- When adding, renaming, or removing top-level files or directories, update
  the layout section in `README.md` in the same pull request.
