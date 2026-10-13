# Branching Strategy

Two strategies, pick one per project and write the choice down (in the README or
an ADR). Both assume `main` is protected in spirit: it should always build and
pass tests, and changes land via pull request, even on solo projects, where the
PR is mostly a self-review checkpoint and a place for CI to run.

## Option A: Trunk-based / GitHub Flow

One long-lived branch (`main`) plus short-lived topic branches.

### How it works

1. Branch off `main`: `feat/...`, `fix/...`, `docs/...`, `chore/...`.
2. Keep the branch small and short-lived: hours or days, not weeks.
3. Open a PR, let CI run, self-review (or get a review), then merge into `main`.
4. `main` is always releasable. Releases are just tags on `main`
   (e.g. `v1.4.0`), created whenever you decide to ship.
5. Delete the topic branch after merging.

If a feature is too big to land in days, hide the unfinished part behind a
feature flag or land it in vertical slices; don't let the branch age.

### Pros

- Minimal ceremony and minimal merge pain: small diffs, frequent integration.
- The history of `main` reflects the actual order of development.
- Perfect fit for continuous deployment: every merge can ship.
- Easy to explain: there is essentially one rule ("branch, PR, merge").

### Cons

- No place to stabilize a release while development continues; `main` must stay
  green at all times, which demands good tests and discipline.
- Supporting old versions (backporting fixes to v1 while v2 is in progress) is
  awkward without additional branches.
- Half-finished features need flags or slicing, which is its own overhead.

### When to use it

Default choice for personal projects: apps, websites, tools, and services that
ship "the latest version" and don't maintain old releases. If in doubt, start
here; you can adopt release branches later when you actually need them.

## Option B: Release-based (release branches)

Trunk plus long-lived `release/x.y` branches that stabilize and maintain
specific versions.

### How it works

1. Day-to-day development happens exactly like Option A: topic branches merged
   into `main` via PR.
2. When preparing a release, cut `release/1.4` from `main`. Only fixes and
   release chores land there, never new features.
3. Tag the release (`v1.4.0`) on the release branch. Later patches on that
   branch become `v1.4.1`, `v1.4.2`, ...
4. Fixes needed in both places are merged to `main` first, then cherry-picked
   to the release branch (or the other way around; pick one direction and be
   consistent).
5. Delete or freeze the release branch when that version goes out of support.

### Pros

- You can stabilize/harden a release while `main` moves on with new features.
- Supporting multiple versions in parallel is natural: each has its branch.
- Clear place to answer "exactly what code is in v1.4.2?"; audits and
  reproducible builds get easier.

### Cons

- Real maintenance overhead: cherry-picks, double-merges, and the risk of a fix
  landing in one branch but not the other.
- More process than a solo project usually needs; dormant release branches rot.
- Slower feedback: code can sit in a release branch state instead of shipping.

### When to use it

Libraries and tools with users pinned to major versions, anything with a
support commitment ("v1 gets security fixes for a year"), on-prem or packaged
software, or projects where releases go through a validation phase.

## Releasing (applies to both strategies)

Cutting a release is the same checklist in either model; only *where* you tag
differs (on `main` for Option A, on the release branch for Option B):

1. **Pick the version** with SemVer, derived from what sits in `Unreleased`:
   MAJOR for any breaking change, MINOR for any `feat`, PATCH for fixes only.
2. **Update `CHANGELOG.md`**: rename `Unreleased` to `[X.Y.Z] - YYYY-MM-DD`,
   start a fresh empty `Unreleased` on top, and update the compare links at the
   bottom.
3. **Land it like any change**: a PR with a single
   `chore(release): vX.Y.Z` commit.
4. **Tag the merge commit**: `git tag -a vX.Y.Z -m "vX.Y.Z"` and
   `git push origin vX.Y.Z`. Annotated tags, always; they carry author and date.
5. **Create the GitHub Release** from the tag and paste the changelog section
   into the notes, so users get notified and the notes are browsable.
6. Optional automation: [release-please](https://github.com/googleapis/release-please)
   can do steps 1–5 from the Conventional Commits history once the project has CI.

## Rules that apply in both

- Branch names: `type/short-kebab-description` (`feat/user-auth`,
  `fix/null-pointer-on-login`), matching the types in
  [COMMIT-CONVENTION.md](COMMIT-CONVENTION.md).
- Never commit directly to `main`; never force-push shared branches.
- Rebase or merge your topic branch on latest `main` before opening the PR, so
  the PR diff is honest.
- Tags are the source of truth for versions and follow SemVer (`vMAJOR.MINOR.PATCH`).
- Update `CHANGELOG.md` as part of the change, not as an afterthought at
  release time.
