# AI Agents Guidelines

How to set up a project so AI coding agents (Claude Code, Copilot, Cursor, and
similar) work well in it. The goal is one source of truth for agent
instructions, wired into each project from day one.

## The instruction files

| File | Read by | Purpose |
| --- | --- | --- |
| `AGENTS.md` | growing cross-tool standard (many agents) | generic, tool-agnostic project instructions |
| `CLAUDE.md` | Claude Code | Claude-specific entry point |
| `.github/copilot-instructions.md` | GitHub Copilot | Copilot-specific entry point |

### Recommended layout: one source of truth

Write the real content once in `AGENTS.md`, and make tool-specific files thin
pointers to it:

```
# CLAUDE.md
See [AGENTS.md](AGENTS.md) for project instructions.
```

Claude Code also treats an `AGENTS.md` referenced this way as loaded context.
A symlink (`ln -s AGENTS.md CLAUDE.md`) works too, if every environment you use
handles symlinks; the pointer file is the safer default. Duplicate actual
content across these files only as a last resort; it *will* drift.

## What belongs in AGENTS.md

Keep it short (a screenful or two) and factual. Agents read it on every
session; every line costs attention. Include:

- **What the project is**: one paragraph, plus the tech stack.
- **Commands**: how to build, test, lint, and run locally. Exact commands,
  not prose.
- **Conventions**: pointers to [COMMIT-CONVENTION.md](COMMIT-CONVENTION.md)
  and [BRANCHING-STRATEGY.md](BRANCHING-STRATEGY.md) rather than restating them.
- **Layout**: where source, tests, and docs live, if not obvious.
- **Boundaries**: what the agent must not do, such as files it must not edit
  (generated code, vendored deps), secrets locations, "never push to main",
  "never commit .env".
- **Writing style**: the tone and formatting you expect in text the agent
  produces (docs, commits, PR text). Prose over reflexive bullet lists, no
  filler phrases, no emoji or decorative markdown, plain-text commit messages.

Leave out: long style guides (linters enforce those better), aspirational
process, anything already enforced by tooling, and anything you wouldn't tell
a new human contributor in their first hour.

### Skeleton to copy into new projects

```markdown
# <PROJECT_NAME>

[one-paragraph description and stack]

## Commands

- Build: [TO BE FILLED IN]
- Test: [TO BE FILLED IN]
- Lint: [TO BE FILLED IN]

## Conventions

- Commits: Conventional Commits; see docs/COMMIT-CONVENTION.md
- Branching: see docs/BRANCHING-STRATEGY.md
- All code, comments, and docs are written in English.
- Writing style: prose over reflexive lists, no filler phrases, no emoji or
  decorative markdown, plain-text commit messages.
- No tool attribution in commits or pull requests: no AI co-author
  trailers, session links, or "generated with" footers.

## Boundaries

- Never commit secrets; .env files are local-only.
- Do not edit [TO BE FILLED IN: generated/vendored paths].
```

## Init templates and generators

When a tool offers to generate its instruction file (e.g. Claude Code's `/init`
command), let it generate, then **merge the result into `AGENTS.md`** and reduce
the tool file to a pointer again. Generators are good at discovering commands
and layout, but the generated file should not become a second source of truth.

## Skills, plugins, and tool-specific configuration

Instruction files tell an agent what the project is; **skills** tell it how to
carry out a specific procedure. A skill is a directory with a `SKILL.md` file
(the Agent Skills format: YAML frontmatter with `name` and `description`,
instructions in the body) that the agent loads only when the task matches.
Skills belong in the repository when the procedure is project-specific and
already documented for humans; the skill then sequences the steps and points
to the document instead of duplicating it. This template ships three: `new-adr`, `release`, and `init-from-template`.
Keep yours equally thin; a skill that restates a guide will drift from it.

The specification defines what a skill contains, not where it lives, and each
tool scans its own directory. Skills are stored under `.agents/skills/`, the
location the specification recommends to client implementers as the shared
convention, and each one is exposed to Claude Code through a symlink
`.claude/skills/<name>` pointing to `../../.agents/skills/<name>` (Claude Code
follows symlinked skill folders and loads each skill once). A tool with yet
another location gets another symlink to the same source, never a copy. On
Windows checkouts without symlink support, replace each symlink with a pointer
skill: a `SKILL.md` carrying the same `name` and `description` and a one-line
body that says to follow `.agents/skills/<name>/SKILL.md`.

**Plugins** (tool-specific bundles of skills, commands, and hooks distributed
through a marketplace) are personal tooling: they belong in your user-level
configuration or in a dedicated personal repository, never in a project.
Committing them would couple every repository created from this template to
one tool and one plugin version.

What to commit and what to keep local:

| Commit | Keep local (git-ignored) |
| --- | --- |
| `AGENTS.md`, `CLAUDE.md` pointer | `CLAUDE.local.md` and other per-user instruction files |
| `.agents/skills/` for project procedures, with the `.claude/skills/` symlinks | personal skills and plugins |
| `.claude/settings.json` when it holds shared project permissions or hooks | `.claude/settings.local.json` |
| `.mcp.json` for project MCP servers, with secrets read from the environment | any file containing tokens or keys |

## Hygiene rules

- **Instruction files are code**: review changes to them in PRs like anything
  else, keep them in English, and update them when commands or layout change;
  a stale `AGENTS.md` is worse than none.
- **Never put secrets** or private URLs in instruction files; agents may echo
  them into logs, PRs, or third-party services.
- **Personal preferences stay local**: per-user agent config
  (e.g. `CLAUDE.local.md`, editor-level rules) is git-ignored, not committed.
- **Review agent output** like any contribution: agents follow this repo's
  contribution rules (commit convention, PR flow); the human remains
  responsible for what gets merged.
- If the project has directory-specific rules, nested `AGENTS.md` files in
  subdirectories are supported by most tools and beat one giant root file.
