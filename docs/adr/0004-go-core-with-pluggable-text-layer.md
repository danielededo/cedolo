# ADR 0004: Write the core in Go behind a text layout boundary

- **Status:** Proposed
- **Date:** 2026-10-13 (updated after the phase 0 spike)
- **Deciders:** maintainer (danielededo)

## Context

The maintainer wants to use Go. The same extraction engine must run in a CLI
(v0.1), in a browser (public release, see ADR 0003) and possibly on a
self-hosted server later. Pure Go PDF text extraction libraries are less
mature than pdf.js (JavaScript), PDFium (C++) or poppler (C): the
[phase 0 spike](../design/spike-pdf-text-layer.md) found that the best known
pure Go reader loses glyph positions on CID fonts, the encoding most modern
PDF generators use.

## Decision

We will write the core (vendor detection, templates, normalization,
validation, corrections, insights) as a pure Go package with no file system,
network or PDF dependencies. Its input is a text layout: pages with text tokens
and bounding boxes. Each frontend supplies a text layer that converts a PDF
into that layout. In the CLI the text layer is PDFium compiled to WebAssembly
and run inside the Go binary by a pure Go runtime (go-pdfium with wazero), with
poppler's pdftotext as an optional reference reader. In the browser it is
pdf.js, or PDFium in WebAssembly if phase 2 shows the bundle size is
acceptable, with the core compiled to WebAssembly.

## Options considered

### Option 1: Go core, pluggable text layer

- Pros: one engine everywhere; tests use small synthetic layout fixtures
  instead of PDFs, which also keeps personal data out of the repository; the
  weakest dependency (PDF reading) is replaceable per platform.
- Cons: two toolchains (Go and TypeScript) once the browser app exists; Go
  WebAssembly bundle size and startup time on phones are unproven; the CLI
  binary carries a 5 MB PDFium module and takes about 3 seconds to start it.

### Option 2: Go end to end, including a pure Go PDF parser

- Pros: single language and toolchain, smallest binary.
- Cons: the spike showed github.com/ledongthuc/pdf losing positions on CID
  fonts; fixing it means maintaining a PDF parser fork.

### Option 3: TypeScript core

- Pros: native to the browser, large contributor pool, mature PDF tooling.
- Cons: the maintainer wants Go; a server mode would run on Node; does not
  meet the learning and preference goal.

## Consequences

- The spike passed on synthetic documents from three PDF generators. This ADR
  is accepted once `cedolo inspect -reader both` also passes on real
  payslips from at least one payroll software.
- The public UI is TypeScript with pdf.js, unless v0.2 chooses a local Go
  binary with a server-rendered UI instead (open question in the design doc).
- If Go WebAssembly fails the phase 2 performance criteria, a TypeScript
  core may supersede this ADR; declarative templates limit the duplication.

## References

- [docs/design/mvp-design.md](../design/mvp-design.md), sections 4 and 8
- [docs/design/spike-pdf-text-layer.md](../design/spike-pdf-text-layer.md)
