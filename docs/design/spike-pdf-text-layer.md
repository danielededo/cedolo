# Spike: PDF text layer for the CLI

- Date: 2026-10-13
- Related: [ADR 0004](../adr/0004-go-core-with-pluggable-text-layer.md),
  roadmap phase 0 in [mvp-design.md](mvp-design.md#9-roadmap)

## Question

Can the Go CLI extract text with reliable word positions from payslip PDFs
without depending on external programs? Template extraction reads values by
column, so a reader that gets the characters right but the positions wrong is
not usable.

## Method

No real payslip was available to the spike, and none may enter the
repository, so three synthetic documents with the same invented content were
used, each produced by a different PDF generator to exercise different font
handling:

- cedolo's own generator (`cedolo synth`): standard Helvetica, not embedded,
  WinAnsi encoding, explicit width table.
- Chromium print to PDF: embedded subset CID TrueType fonts with Identity-H
  encoding and ToUnicode maps, which is what most modern PDF engines emit.
- LibreOffice export: embedded subset TrueType and Type 1 fonts with WinAnsi
  and built-in encodings.

Each was read by three readers, and their word boxes were compared with
`cedolo inspect -reader both`, which reports counts and distances only.

## Results

github.com/ledongthuc/pdf, pure Go and BSD licensed, decoded characters
correctly in all files but reports zero glyph width for CID fonts. In the
Chromium file every glyph of a run therefore had the same X coordinate, words
were split or merged arbitrarily ("C E D O L I N O", "T rattenute"), and
column positions were lost. It also emitted stray glyphs from the Type 1 font
with built-in encoding. Fixing this means maintaining a fork of a PDF parser,
so this reader was dropped.

PDFium through github.com/klippa-app/go-pdfium in WebAssembly mode (PDFium is
BSD and Apache licensed, the wrapper MIT, the wazero runtime Apache) read all
three files correctly. Against poppler it matched 100 percent of words on the
cedolo and Chromium files and 97.3 percent on the LibreOffice file (one
ligature and one stray glyph differ), with bottom-left corners within 0.3
points. Horizontal positions agree exactly; vertical extents differ for
non-embedded fonts because PDFium substitutes a font with a larger ascent,
which is why comparisons use the bottom edge. Costs: the binary grows from
about 3 MB to 17 MB, and start-up takes about 3 seconds while the WebAssembly
module compiles. Neither matters for a CLI that reads a handful of files.

Poppler's pdftotext (GPL, run as an external program) read all three files
correctly and is kept as an independent reference for tests and for
`inspect -reader both`, but it requires an installation that non-technical
users will not have.

## Decision

The CLI uses PDFium in WebAssembly as its default text layer and poppler as
an optional reference reader. ADR 0004 is updated accordingly and stays
Proposed until the same check passes on real payslips.

## Open points

- The check must be repeated on real payslips from at least one payroll
  software. The maintainer can run `cedolo inspect -reader both` locally; its
  output contains no names, amounts or file names and can be shared.
- Scanned payslips have no text layer; `inspect` reports them as such. OCR
  remains out of scope until phase 3.
- Start-up time can be cut by enabling wazero's compilation cache in the
  user cache directory, if it ever becomes noticeable.
- PDFium in WebAssembly could also replace pdf.js in the browser, giving the
  CLI and the web app the same text layer. This is worth evaluating in phase 2
  against the bundle size budget.
