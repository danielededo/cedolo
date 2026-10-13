---
name: release
description: Cut a release following the project's release checklist. Picks the SemVer version from the Unreleased changelog section, updates CHANGELOG.md, opens the release pull request, then tags and publishes the GitHub Release once merged. Use when asked to release, tag, or publish a version.
---

# Release

Follow the "Releasing" section of docs/BRANCHING-STRATEGY.md; this skill
only sequences the steps and the checks.

1. Read the Unreleased section of CHANGELOG.md. If it is empty, stop and
   say there is nothing to release.
2. Derive the version with SemVer: MAJOR if any entry or commit since the
   last tag is a breaking change, MINOR if there is at least one feature,
   PATCH otherwise. State the proposed version and the reasoning, and
   confirm it with the user before continuing.
3. In CHANGELOG.md, rename Unreleased to "[X.Y.Z] - YYYY-MM-DD" with today's
   date, add a fresh empty Unreleased section above it, and update the
   compare links at the bottom.
4. Bump the version wherever the project stores it (package manifest,
   version file). If no such place exists, say so and skip.
5. Commit as "chore(release): vX.Y.Z" on a topic branch and open a pull
   request against main. Do not tag yet.
6. After the pull request is merged: create an annotated tag vX.Y.Z on the
   merge commit, push the tag, and create the GitHub Release from it with
   the changelog section as the notes. If you lack permission for tags or
   releases, hand the exact commands to the user instead of skipping the
   step silently.
