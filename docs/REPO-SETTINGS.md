# Repository Settings Checklist

Files can't carry GitHub *settings*, so every repository created from this
template needs a few clicks of manual configuration. This checklist keeps that
step from being forgotten. It targets personal projects (solo maintainer or
small team); items are ordered by importance, and the optional ones say so.

## One-time: on the template repository itself

- [ ] **Settings > General > Template repository: enable.** Without this flag
      the **"Use this template"** button the README relies on does not exist.

## For every new repository created from the template

### General (Settings > General)

- [ ] **Default branch** is `main` (GitHub default; verify if the repo was
      imported).
- [ ] **Pull requests**: allow **squash merging only** (disable merge commits
      and rebase merging). One squash commit per PR keeps history readable and,
      done from the web UI, the squash commit is signed by GitHub (Verified).
      - Set "Default commit message" to **"Pull request title and description"**
        so the squash message follows the commit convention.
- [ ] **Automatically delete head branches**: enable. Merged topic branches
      have no further purpose.

### Branch protection (Settings > Branches or Rulesets)

Even solo, a minimal ruleset on `main` prevents accidents:

- [ ] Require a pull request before merging (approvals can stay at 0 for a
      solo maintainer; the point is blocking direct pushes, not self-review
      theater).
- [ ] Block force pushes and deletions.
- [ ] Require status checks to pass: `test` and `web` (the jobs of the CI
      workflow).
- [ ] Optional: **require signed commits**, only if you have GPG/SSH signing
      configured everywhere you commit from (including remote/CI environments),
      otherwise it will block your own workflow.

### Security (Settings > Advanced Security / Code security)

- [ ] **Dependabot alerts** and **security updates**: enable.
- [ ] **Private vulnerability reporting**: enable; `SECURITY.md` points
      reporters to it as the preferred channel.
- [ ] **Secret scanning** and **push protection**: enable (free for public
      repos).

### Features (Settings > General > Features)

- [ ] Disable what the project doesn't use (Wiki, Projects, Discussions) to
      reduce surface and confusion. Enable **Discussions** only if you want
      Q&A separated from issues (then link it in
      `.github/ISSUE_TEMPLATE/config.yml`).

### Before making the repository public

- [ ] Check that nothing personal is in the history (commits, pull request
      descriptions and their edit history, force-pushed commits linked from
      pull requests): what becomes public includes all of it.
- [ ] Actions (Settings > Actions > General): workflow permissions
      **read-only by default**, and do not allow Actions to create or approve
      pull requests; workflows that need more ask for it themselves.
- [ ] Private vulnerability reporting and secret scanning with push
      protection enabled (see Security above).

### GitHub Pages (the web app)

- [ ] Settings > Pages > Source: **GitHub Actions**; then run the "Web app"
      workflow from the Actions tab.
- [ ] **Enforce HTTPS** enabled (see `docs/design/web-security.md`).

### Optional / public projects

- [ ] **Social preview image** (Settings > General) for nicer link sharing.
- [ ] **Topics and description** on the repo home; it helps discovery.
- [ ] **Sponsor button** via `.github/FUNDING.yml`, if you want one.
- [ ] Actions permissions (Settings > Actions): restrict to actions you trust,
      and set workflow permissions to **read-only by default**.
