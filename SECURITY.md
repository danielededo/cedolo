# Security Policy

This project is maintained by a solo developer (or a small team). Security reports
are taken seriously, but please keep in mind that response times are best-effort,
not backed by a dedicated security team.

## Reporting a vulnerability

**Please do not open a public issue for security problems.**

Instead, use GitHub private vulnerability reporting: go to the **Security**
tab of the repository and click **Report a vulnerability**.

Never attach real payslips, screenshots of payslips, or any personal data to a
report. If a reproduction needs a document, describe the layout or use the
synthetic fixtures in the repository.

A useful report includes:

- A description of the vulnerability and its impact
- Steps to reproduce, or a proof of concept if you have one
- The affected version, commit, or branch
- Any suggestions you have for a fix (optional, but appreciated)

## What to expect

- **Acknowledgement** of your report within a few days (usually faster).
- An honest assessment: whether the issue is confirmed, its severity, and a rough
  timeline for a fix. Simple issues are usually fixed quickly; complex ones may
  take longer; you'll be kept in the loop.
- **Credit** in the release notes or changelog when the fix ships, unless you
  prefer to stay anonymous.

Please give a reasonable window to ship a fix before disclosing the issue
publicly. Coordinated disclosure protects users of the project.

## Supported versions

Unless stated otherwise below, only the **latest release** (and the `main`
branch) receive security fixes.

| Version | Supported |
| ------- | --------- |
| latest  | yes       |
| older   | no        |

## Scope notes

- cedolo processes payslips on the user's device. Any code path that sends
  document content, extracted values, or file names over the network is a
  security issue and in scope.
- The web app's protections and their known limits are described in
  [docs/design/web-security.md](docs/design/web-security.md). A way around
  them (data reaching another origin, script running from a PDF, data kept
  after the page is closed) is in scope.

- Vulnerabilities in third-party dependencies should be reported upstream first;
  a report here is still welcome if this project needs to update or mitigate.
- Findings that require physical access to the maintainer's machine, social
  engineering, or denial of service by sheer volume are generally out of scope.
