# Commit Convention

This project follows [Conventional Commits 1.0.0](https://www.conventionalcommits.org/en/v1.0.0/).
Structured messages make the history searchable, enable automatic changelog
generation and semantic-release tooling, and force each commit to have a single,
nameable purpose.

## Format

```
<type>[optional scope][!]: <description>

[optional body]

[optional footer(s)]
```

- **description**: imperative mood ("add", not "added"/"adds"), lowercase, no
  trailing period, ideally ≤ 72 characters total for the first line.
- **body**: the *why*, wrapped at ~72 columns. Optional for trivial changes.
- **footers**: `BREAKING CHANGE: ...`, `Refs: #123`, `Closes: #45`, etc.

## Types

| Type | Use for | SemVer effect |
| --- | --- | --- |
| `feat` | a new user-facing capability | MINOR |
| `fix` | a bug fix | PATCH |
| `docs` | documentation only | none |
| `style` | formatting, whitespace, no behavior change | none |
| `refactor` | code change that neither fixes nor adds behavior | none |
| `perf` | performance improvement | PATCH |
| `test` | adding or fixing tests | none |
| `build` | build system, dependencies, packaging | none |
| `ci` | CI configuration and scripts | none |
| `chore` | maintenance that fits nowhere else (tooling, repo config) | none |
| `revert` | reverting a previous commit | mirrors the reverted commit |

A `!` after type/scope, **or** a `BREAKING CHANGE:` footer, marks a breaking
change and implies a MAJOR bump, regardless of type.

## Scope

Optional, in parentheses, naming the affected area: a module, directory, or
component (`feat(parser): ...`, `fix(auth): ...`). Keep a small, consistent
vocabulary per project; skip the scope when the change is global.

## Examples

Minimal:

```
feat: add dark mode toggle
```

With scope and body:

```
fix(auth): refresh expired tokens before retrying request

The retry wrapper re-sent the original request with the stale token,
so a second 401 was guaranteed. Refresh first, then retry once.

Closes: #42
```

Breaking change:

```
feat(api)!: rename user_id to userId in all responses

BREAKING CHANGE: consumers parsing snake_case fields must migrate to
camelCase. No compatibility shim is provided.
```

Revert:

```
revert: feat(api)!: rename user_id to userId in all responses

This reverts commit 1a2b3c4. The mobile client cannot migrate before
the next store release.
```

Chore and docs:

```
chore: bump minimum supported runtime to LTS
docs: document local development setup in README
```

## Practical rules

- **One logical change per commit.** If the message wants an "and", split it.
- Commits on a PR branch don't need to be perfect if the PR is squash-merged,
  but the **squash commit message must follow this convention**, because that's
  what lands in history.
- Don't bypass the convention with `wip` commits on shared branches; `wip` is
  fine locally, squash before pushing.
- When a commit relates to an issue, reference it in a footer (`Refs: #123`),
  not in the description line.

## Tooling (optional)

Solo projects don't need enforcement, but if you want it:
[commitlint](https://commitlint.js.org) with `config-conventional` validates
messages, and [release-please](https://github.com/googleapis/release-please) or
semantic-release can turn this convention into automated versioning and
changelogs. [TO BE FILLED IN: per-project tooling choice]
