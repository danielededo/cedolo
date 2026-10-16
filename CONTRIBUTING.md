# Contributing to cedolo

Thanks for taking the time to contribute! This project is maintained by a solo developer (or a small team), so the process is intentionally lightweight. The goal of this document is to make contributions smooth, not to add bureaucracy.

## Personal data comes first

cedolo works on payslips, which contain sensitive personal data. Never commit,
attach, or paste a real payslip, a screenshot of one, or values copied from
one, in code, test fixtures, issues, pull requests, or discussions. Test data
in this repository is synthetic. CI rejects fixtures that contain a valid
codice fiscale or IBAN, but that check is a safety net, not a permission.

## Reporting a payslip layout

When cedolo misreads a payslip, the most useful report is an anonymized
copy, never the payslip itself:

1. Run `cedolo anonymize -o anonymized.pdf payslip.pdf` on your own machine.
   It writes a new PDF rebuilt from the payslip's text; nothing else of the
   original file is copied.
2. Open `anonymized.pdf` and read all of it. Only payroll labels and amounts
   should be readable. Anything else (a name, a street, a company, a date, a
   code) should appear as `Xxxx` or `0000`. If something personal survived,
   do not share the file and report that instead, since it is a bug.
3. The amounts are kept so that errors in the totals can be reproduced.
   Together they can identify you to someone who knows your pay, such as a
   payroll office or a colleague. If that matters, add `-amounts mask`:
   most layout problems can still be found.
4. Attach the file to an issue, with a description in words of what cedolo
   got wrong (for example "the net pay is read from the wrong cell").

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
5. **Test your changes.** Run `go vet ./...` and `go test ./...` before opening the PR; CI runs the same checks plus a formatting check.
6. **Update documentation** if your change affects behavior, configuration, or public APIs. Add an entry to the `Unreleased` section of `CHANGELOG.md` when the change is user-visible.
7. **Open a pull request** against `main` and fill in the PR template.

## Sign-off

Contributions are accepted under the [Developer Certificate of Origin](https://developercertificate.org/).
Sign off every commit with `git commit -s`, which adds a `Signed-off-by:` line
certifying that you have the right to submit the change under the project's
license (AGPL-3.0-or-later).

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
