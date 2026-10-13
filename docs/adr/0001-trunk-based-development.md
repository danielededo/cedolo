# ADR 0001: Use trunk-based development for this repository

- **Status:** Accepted
- **Date:** 2026-08-23
- **Deciders:** repository maintainer

## Context

This repository is a documentation/template baseline maintained by a single
person. It has no runtime, no versioned artifacts to support, and no users
pinned to old versions: consumers always take the latest state via GitHub's
"Use this template". [docs/BRANCHING-STRATEGY.md](../BRANCHING-STRATEGY.md)
offers two strategies and requires each project to pick one explicitly; this
repository must eat its own dog food and record the choice.

## Decision

We will use **trunk-based development** (Option A in the branching guide):
short-lived topic branches off `main`, merged via pull request with **squash
merge only**. Releases, if ever needed, are tags on `main`.

## Options considered

### Option 1: Trunk-based / GitHub Flow

- Pros: minimal ceremony; single always-current branch matches how template
  consumers use the repo; squash merges keep a readable, conventional history.
- Cons: no stabilization branch; acceptable, since there is nothing to
  stabilize in a docs-only repo.

### Option 2: Release-based

- Pros: would allow maintaining versioned template lines (e.g. a frozen "v1"
  baseline).
- Cons: real cherry-pick/double-merge overhead with no current need; dormant
  release branches would rot.

## Consequences

- `main` is always the recommended state of the template; anything merged is
  immediately "released".
- No backporting is possible: projects created from older states of the
  template must diff against current `main` manually if they want updates.
- If versioned template releases ever become necessary, this decision must be
  superseded by a new ADR adopting Option 2.

## References

- [docs/BRANCHING-STRATEGY.md](../BRANCHING-STRATEGY.md)
- [docs/COMMIT-CONVENTION.md](../COMMIT-CONVENTION.md)
