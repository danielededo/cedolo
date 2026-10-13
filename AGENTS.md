# scaffold

Meta/template repository: the language-agnostic baseline (licensing, governance,
documentation, contribution workflow) that new personal projects start from.
Everything here is documentation or configuration; there is no code to build,
test, or run.

This file is the single source of truth for agent instructions, as prescribed
by [docs/AI-AGENTS.md](docs/AI-AGENTS.md); `CLAUDE.md` is a pointer to it.

## Commands

- There is no build, test, or lint pipeline. Validation is manual: check that
  Markdown renders correctly and that YAML files (`.github/*.yml`) stay parseable.
- Project procedures are available as skills in `.agents/skills/` (exposed to
  Claude Code through symlinks in `.claude/skills/`): `new-adr`, `release`,
  and `init-from-template` (the last one is for repositories created from
  this template, not for this repository itself).

## Conventions

- Commits: Conventional Commits; see [docs/COMMIT-CONVENTION.md](docs/COMMIT-CONVENTION.md).
- Branching: trunk-based; see [docs/BRANCHING-STRATEGY.md](docs/BRANCHING-STRATEGY.md)
  and [ADR 0001](docs/adr/0001-trunk-based-development.md).
- All content (docs, comments, commit messages, file names) is written in
  English, no exceptions.
- Per-project customization points use the placeholders `<PROJECT_NAME>`,
  `<OWNER_HANDLE>`, `<SECURITY_CONTACT_EMAIL>`, `<CONTACT_EMAIL>`, `<YEAR>`,
  `<COPYRIGHT_HOLDER>`, and `[TO BE FILLED IN]`; reuse them, don't invent new ones.

## Writing style

These rules apply to any text produced in this repository, by humans and
agents alike: documentation, commit messages, pull request and issue text,
code comments.

- Prefer prose. Use bullet or numbered lists only when the content is
  genuinely enumerable or sequential (options, checklists, step-by-step
  procedures), not as a default layout for explanations.
- Start with the substance. No filler or throat-clearing phrases
  ("as you can see", "it's worth noting", "let me explain").
- No generic disclaimers, hedging boilerplate, or pseudo-human tone (fake
  enthusiasm, apologies, self-narration). State facts and uncertainty plainly.
- No emoji and no arrow glyphs anywhere. No decorative formatting: headings
  and tables belong to documents that need structure, not to short texts;
  in running prose avoid bold for emphasis. In new prose prefer commas,
  colons, or separate sentences over em dashes.
- Commit messages are plain text: no markdown syntax at all.
- Commit messages and pull request descriptions carry no tool attribution:
  no AI co-author trailers, no session links, no "generated with" footers.
  Authorship belongs to the account that commits; the tool used to write
  the change is not part of the history.
- Numbered sections and closing summaries only where the document type calls
  for them (a guide with a procedure), never as reflex.

## Boundaries

- Never commit directly to `main`; every change goes through a pull request.
- `licenses/MIT.txt`, `licenses/APACHE-2.0.txt`, and the body of
  `CODE_OF_CONDUCT.md` are canonical third-party texts: never reword them
  (placeholders and the top note in the code of conduct are the only local parts).
- Keep this repo language-agnostic: no language-specific tooling, manifests, or
  CI; that boundary is documented in the README and is deliberate.
- When adding, renaming, or removing files, update the structure tree in
  `README.md` in the same pull request.
