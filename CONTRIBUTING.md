# Contributing to <PROJECT_NAME>

Thanks for taking the time to contribute! This project is maintained by a solo developer (or a small team), so the process is intentionally lightweight. The goal of this document is to make contributions smooth, not to add bureaucracy.

## Ways to contribute

- **Report a bug**: open an issue using the bug report template.
- **Suggest a feature**: open an issue using the feature request template. For anything non-trivial, please discuss it in an issue *before* writing code, so we don't waste your time on something that won't be merged.
- **Improve documentation**: typo fixes and clarifications are always welcome; feel free to open a PR directly.
- **Submit code**: see the workflow below.

## Development workflow

1. **Fork** the repository (or create a branch, if you have write access).
2. **Create a branch** from `main` with a descriptive name:
   - `feat/short-description` for features
   - `fix/short-description` for bug fixes
   - `docs/short-description` for documentation
3. **Make your changes.** Keep the scope of each PR small and focused: one logical change per PR is much easier to review than a grab-bag.
4. **Follow the commit convention.** This project uses [Conventional Commits](https://www.conventionalcommits.org/); see [docs/COMMIT-CONVENTION.md](docs/COMMIT-CONVENTION.md) for the format and examples.
5. **Test your changes.** Run the project's test suite and linters before opening the PR. [TO BE FILLED IN: project-specific commands, e.g. `make test`]
6. **Update documentation** if your change affects behavior, configuration, or public APIs. Add an entry to the `Unreleased` section of `CHANGELOG.md` when the change is user-visible.
7. **Open a pull request** against `main` and fill in the PR template.

## Pull request expectations

- CI (if configured) must pass before review.
- The maintainer reviews PRs on a best-effort basis: this is a personal project, so a few days of latency is normal. A friendly ping after a week is fine.
- Expect review feedback; it's about the code, never about you. You may be asked to squash or reword commits so the history stays readable.
- PRs that go silent for a long time may be closed; you're welcome to reopen when you have time again.

## Reporting bugs well

A good bug report includes:

- What you did, what you expected, and what actually happened
- Steps to reproduce, as minimal as you can make them
- Version/commit of the project, OS, and any relevant environment details
- Logs or error messages, as text rather than screenshots when possible

## Questions?

If something in this guide is unclear, open an issue; that's also a contribution.

## Code of Conduct

By participating in this project you agree to abide by the [Code of Conduct](CODE_OF_CONDUCT.md).
