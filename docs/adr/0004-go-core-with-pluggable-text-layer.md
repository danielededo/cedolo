# ADR 0004: Write the core in Go behind a text layout boundary

- **Status:** Proposed
- **Date:** 2026-10-13
- **Deciders:** maintainer (danielededo)

## Context

The maintainer wants to use Go. The same extraction engine must run in a CLI
(v0.1), in a browser (public release, see ADR 0003) and possibly on a
self-hosted server later. Go's PDF text extraction libraries are less mature
than pdf.js (JavaScript) or poppler (C), while the browser has pdf.js
available natively.

## Decision

We will write the core (vendor detection, templates, normalization,
validation, corrections, insights) as a pure Go package with no file system,
network or PDF dependencies. Its input is a text layout: pages with text tokens
and bounding boxes. Each frontend supplies a text layer that converts a PDF
into that layout: a Go PDF reader (or poppler pdftotext) in the CLI, pdf.js in
the browser, with the core compiled to WebAssembly.

## Options considered

### Option 1: Go core, pluggable text layer

- Pros: one engine everywhere; tests use small synthetic layout fixtures
  instead of PDFs, which also keeps personal data out of the repository; the
  weakest dependency (PDF reading) is replaceable per platform.
- Cons: two toolchains (Go and TypeScript) once the browser app exists; Go
  WebAssembly bundle size and startup time on phones are unproven.

### Option 2: Go end to end, including PDF parsing

- Pros: single language and toolchain.
- Cons: ties correctness to a less mature PDF library and makes the browser
  harder; no benefit from pdf.js.

### Option 3: TypeScript core

- Pros: native to the browser, large contributor pool, mature PDF tooling.
- Cons: the maintainer wants Go; a server mode would run on Node; does not
  meet the learning and preference goal.

## Consequences

- The first phase 0 task is a spike on the maintainer's payslips to confirm a
  Go reader recovers text positions; if not, the CLI uses pdftotext.
- The public UI is TypeScript with pdf.js, unless v0.2 chooses a local Go
  binary with a server-rendered UI instead (open question in the design doc).
- If Go WebAssembly fails the phase 2 performance criteria, a TypeScript
  core may supersede this ADR; declarative templates limit the duplication.

## References

- [docs/design/mvp-design.md](../design/mvp-design.md), sections 4 and 8
