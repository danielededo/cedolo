# ADR 0003: Process payslips on the user's device by default

- **Status:** Accepted
- **Date:** 2026-10-13
- **Deciders:** maintainer (danielededo)

## Context

Payslips contain income, contract and tax data, and often special category
data under Article 9 GDPR: union membership (union dues), health (sick leave,
Law 104 leave), pregnancy and parental leave. Target users are non-technical
and cannot evaluate a server's security. The project has one maintainer, no
budget for hosting or compliance work, and a requirement that self-hosting be
possible later without a rewrite.

## Decision

We will process documents entirely on the user's device: in a CLI for v0.1
and in the browser for the public release. No document, extracted value or
telemetry is sent anywhere. No external AI service is used, now or by
default later. A self-hosted server mode may come in a later phase, as an
explicit option with its own ADR.

## Options considered

### Option 1: On-device only

- Pros: the maintainer processes no personal data, so the project needs no
  legal basis, DPIA or breach procedure for payslip content; no hosting cost;
  the privacy claim is verifiable (no network requests).
- Cons: no sync across devices; PDF and OCR tooling in the browser is
  constrained; support is harder because the maintainer never sees failures.

### Option 2: Containerized backend from day one

- Pros: simpler tooling on the server, one runtime, easy debugging.
- Cons: whoever runs it holds Article 9 data and becomes a controller with
  full GDPR obligations; hosting cost; users must trust the operator.

### Option 3: Hybrid, local by default with optional cloud LLM

- Pros: better coverage on unknown layouts.
- Cons: sends documents to a third party; cost; contradicts the core promise.

## Consequences

- Extraction must be deterministic and run locally (see ADR 0005).
- The core must be independent of the runtime so the same code runs in the
  CLI, the browser and a future server (see ADR 0004).
- The public app must enforce a Content Security Policy that forbids outbound
  connections, and tests must verify that no request is made after load.
- Bug reports cannot include documents; an anonymizer for redacted layouts is
  required before the repository is public.

## References

- [docs/design/mvp-design.md](../design/mvp-design.md), sections 5 and 8
