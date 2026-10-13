# ADR 0001: Use trunk-based development with squash merges

- **Status:** Accepted
- **Date:** 2026-10-13
- **Deciders:** maintainer (danielededo)

## Context

cedolo is maintained by one person at about five hours a week. It ships "the
latest version" (a CLI first, then a static web app) and has no users pinned
to old releases. [docs/BRANCHING-STRATEGY.md](../BRANCHING-STRATEGY.md)
offers two strategies and requires each project to record its choice.

## Decision

We will use trunk-based development (Option A in the branching guide):
short-lived topic branches named `type/short-description` off `main`, merged
through pull requests with squash merge only. The maintainer reviews and merges
every pull request, including those prepared by AI agents. Releases are
annotated SemVer tags on `main`.

## Options considered

### Option 1: Trunk-based / GitHub Flow

- Pros: minimal ceremony for a solo maintainer; small, frequent integrations;
  squash merges keep one Conventional Commit per change on `main`, which
  feeds the changelog directly.
- Cons: no branch to stabilize a release while development continues; `main`
  must always pass CI.

### Option 2: Release branches

- Pros: allows stabilizing and backporting to supported versions.
- Cons: cherry-pick overhead with no current need; nobody depends on old
  versions.

## Consequences

- `main` is always releasable; unfinished features land behind flags or in
  vertical slices.
- Old versions get no backports. If a supported release line is ever needed,
  this decision is superseded by a new ADR adopting release branches.

## References

- [docs/BRANCHING-STRATEGY.md](../BRANCHING-STRATEGY.md)
- [docs/COMMIT-CONVENTION.md](../COMMIT-CONVENTION.md)
