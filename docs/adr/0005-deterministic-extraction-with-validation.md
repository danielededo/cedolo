# ADR 0005: Extract with templates and arithmetic validation, without ML

- **Status:** Superseded by [ADR-0006](0006-generic-extraction-first.md)
- **Date:** 2026-10-13
- **Deciders:** maintainer (danielededo)

## Context

Processing is on-device and no external LLM may see payslip content (ADR
0003). The product's credibility depends on numbers being exact to the cent; a
plausible but wrong number is the worst failure. Most payslips are text PDFs
produced by a handful of payroll software vendors, and each payslip contains
internal redundancy (totals, rates, balances) that can be checked.

## Decision

We will extract data with per-vendor templates over the PDF text layer, a
label-dictionary fallback that only reads the main totals on unknown layouts,
and arithmetic cross-checks that assign each field a status (verified,
extracted, inferred, missing, corrected). Uncertain fields are confirmed by
the user. The first template is written in Go; a declarative template format
is extracted when the second vendor is added. Scans and OCR are out of scope
until phase 3.

## Options considered

### Option 1: Templates plus validation (hybrid without ML)

- Pros: exact and explainable on supported layouts; every value traceable to
  a page region; free; runs anywhere.
- Cons: coverage grows one vendor at a time; templates need maintenance when
  layouts change.

### Option 2: OCR with heuristics

- Pros: handles scans and photos.
- Cons: lower and variable accuracy, especially for table structure and
  signs; heavy in the browser.

### Option 3: Local ML or small local LLM

- Pros: could generalize across layouts.
- Cons: heavy for browsers and average laptops; unknown accuracy without a
  labelled data set; can hallucinate numbers.

## Consequences

- An anonymizer producing redacted layouts is needed so the community can
  contribute templates; it becomes the main driver of coverage.
- Unknown layouts get partial support only, shown honestly to the user.
- If early users mostly have scanned payslips, OCR must move up and this
  decision should be revisited.

## References

- [docs/design/mvp-design.md](../design/mvp-design.md), sections 2 and 6
