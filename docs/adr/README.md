# Architecture Decision Records (ADRs)

## What an ADR is

An Architecture Decision Record is a short document that captures **one**
significant decision: the context that forced it, the decision itself, the
options that were considered, and the consequences. The collection of ADRs is
the project's decision log: it answers "why is it built this way?" months
later, when nobody remembers.

"Architecturally significant" means the decision is expensive to reverse or
shapes future work: choice of database or framework, API style, auth approach,
hosting model, a build-vs-buy call. Formatting rules and small implementation
details do **not** need an ADR.

## Why bother on a personal project

Because future-you is a different person. An ADR takes ten minutes to write and
saves the hour of re-deriving (or worse, re-litigating) a decision. It also
makes onboarding trivial if the project ever gains collaborators.

## How to use them

1. Copy [`0000-template.md`](0000-template.md) to `NNNN-short-title.md`, where
   `NNNN` is the next number (e.g. `0001-use-postgres.md`).
2. Fill it in. Keep it short: one page is plenty. Write in English, in full
   sentences; bullet lists are fine for options and consequences.
3. Commit it together with (or just before) the change it justifies
   (`docs: add ADR 0001 on database choice`), and reference it from the PR.
4. Never rewrite an accepted ADR's substance. Decisions get **superseded**, not
   edited: write a new ADR, set the old one's status to
   `Superseded by [ADR-NNNN](NNNN-....md)`, and move on. Fixing typos or adding
   links is fine.

## Statuses

- **Proposed**: written but not yet settled (useful if you want to sleep on it).
- **Accepted**: the decision is in effect.
- **Deprecated**: no longer relevant (e.g. the component was removed).
- **Superseded**: replaced by a newer ADR; link to it.

## Index

| ADR | Title | Status |
| --- | --- | --- |
| [0000](0000-template.md) | Template | n/a |
| [0001](0001-trunk-based-development.md) | Use trunk-based development for this repository | Accepted |

[TO BE FILLED IN: add a row per ADR as they are created]
