# scaffold

A meta/template repository with a sensible baseline for personal projects: licensing, governance, documentation, and contribution workflow. It is deliberately **not** language-specific: it contains the parts that every project needs regardless of stack.

## What this repo is

This repository is a starting point for new personal projects, both public and private. It collects the boilerplate that is easy to get wrong or to postpone forever:

- License texts and a guide for choosing between them
- Community and governance files (`CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, `SECURITY.md`)
- A changelog skeleton following [Keep a Changelog](https://keepachangelog.com/) and [Semantic Versioning](https://semver.org/)
- Documentation on branching strategies, commit conventions, and Architecture Decision Records (ADRs)
- GitHub-specific configuration (issue and PR templates, `CODEOWNERS`, Dependabot)
- Cross-language editor, line-ending, and ignore rules (`.editorconfig`, `.gitattributes`, `.gitignore`)
- A checklist for the GitHub settings that files can't carry ([docs/REPO-SETTINGS.md](docs/REPO-SETTINGS.md))

[TO BE FILLED IN: anything else this template grows to include]

## How to use it as a template

1. Click **Use this template** on GitHub (or clone and re-init: `git clone`, delete `.git`, `git init`).
   If you work with an AI coding agent, the `init-from-template` skill in `.agents/skills/` walks through steps 2 to 6.
2. Search for the placeholders and replace them:
   - `<PROJECT_NAME>`: the name of your new project
   - `<OWNER_HANDLE>`: your GitHub username
   - `<SECURITY_CONTACT_EMAIL>`: the address for vulnerability reports
   - `<CONTACT_EMAIL>`: the code of conduct enforcement contact
   - `<YEAR>` / `<COPYRIGHT_HOLDER>`: in the chosen license file
   - `[TO BE FILLED IN]`: free-form sections that need project-specific content
3. Pick a license: read [docs/LICENSE-GUIDE.md](docs/LICENSE-GUIDE.md), copy the right file from [licenses/](licenses/) to the repository root as `LICENSE`, and delete the `licenses/` directory.
4. Rewrite this `README.md` for the actual project, starting from
   [docs/README-TEMPLATE.md](docs/README-TEMPLATE.md).
5. Configure the repository settings following [docs/REPO-SETTINGS.md](docs/REPO-SETTINGS.md).
6. Delete what you don't need (see the notes below on optional files).
7. Reset `CHANGELOG.md` to a fresh `Unreleased` section.

### Optional files

- **`CODE_OF_CONDUCT.md`**: included because it is useful the moment a public project receives external contributors, but it is optional: feel free to remove it for private or strictly personal repositories.
- **`.github/CODEOWNERS`**: optional for single-maintainer projects; see the comments inside the file.
- **`docs/adr/`**: keep it only if you intend to record architecture decisions.
- **`.agents/skills/`** (and the `.claude/skills/` symlinks): `init-from-template` runs once and deletes itself; keep `new-adr` and `release` only if an agent will run those procedures for you.

## What it does NOT include (and why)

- **Language-specific tooling** (linters, formatters, build files, package manifests): every stack has its own conventions and generators; duplicating them here would only go stale.
- **CI/CD workflows**: pipelines depend heavily on the language, hosting, and deployment target. Add them per project.
- **A `LICENSE` file at the root**: choosing a license is a per-project decision. The `licenses/` directory provides the candidates and `docs/LICENSE-GUIDE.md` explains how to choose.
- **Issue tracker automation, bots, org policies**: this template targets personal projects maintained by one person or a small team, not an organization.

## Structure

```
/
├── README.md                       # this file; replace with the project's own README
├── AGENTS.md                       # AI agent instructions (single source of truth)
├── CLAUDE.md                       # pointer to AGENTS.md
├── CONTRIBUTING.md                 # contribution workflow
├── CODE_OF_CONDUCT.md              # Contributor Covenant (optional for private repos)
├── SECURITY.md                     # how to report vulnerabilities
├── CHANGELOG.md                    # Keep a Changelog + SemVer skeleton
├── .editorconfig                   # cross-language editor rules
├── .gitattributes                  # line-ending normalization and binary handling
├── .gitignore                      # OS, IDE, and secrets ignores (not language-specific)
├── licenses/                       # license candidates; pick one, move to ./LICENSE
│   ├── MIT.txt
│   ├── APACHE-2.0.txt
│   └── PROPRIETARY.txt
├── docs/
│   ├── LICENSE-GUIDE.md            # how to choose a license
│   ├── README-TEMPLATE.md          # README skeleton for new projects
│   ├── REPO-SETTINGS.md            # GitHub settings checklist per repository
│   ├── BRANCHING-STRATEGY.md       # trunk-based/GitHub Flow vs release-based
│   ├── COMMIT-CONVENTION.md        # Conventional Commits, with examples
│   ├── AI-AGENTS.md                # guidelines for CLAUDE.md / AGENTS.md and agent setup
│   └── adr/
│       ├── README.md               # what ADRs are and how to use them
│       ├── 0000-template.md        # ADR template
│       └── 0001-trunk-based-development.md  # this repo's own branching decision
├── .agents/
│   └── skills/                     # project skills (Agent Skills format), see docs/AI-AGENTS.md
│       ├── new-adr/SKILL.md
│       ├── release/SKILL.md
│       └── init-from-template/SKILL.md
├── .claude/
│   └── skills/                     # symlinks to .agents/skills/*, so Claude Code finds them
└── .github/
    ├── CODEOWNERS                  # optional for single-maintainer projects
    ├── PULL_REQUEST_TEMPLATE.md
    ├── ISSUE_TEMPLATE/
    │   ├── bug_report.md
    │   ├── feature_request.md
    │   └── config.yml
    └── dependabot.yml              # base config, adapt per ecosystem
```
