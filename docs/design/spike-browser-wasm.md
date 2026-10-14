# Spike: cedolo in the browser with WebAssembly

- Date: 2026-10-14
- Related: [ADR 0003](../adr/0003-on-device-processing-by-default.md),
  [ADR 0004](../adr/0004-go-core-with-pluggable-text-layer.md),
  [mvp-design.md](mvp-design.md) section 4 and the v0.2 web app in the roadmap

## Question

Can the cedolo engine run inside a web page, on the user's device, with a
download size and speed acceptable on a mid-range phone? The CLI reads PDFs
with PDFium compiled to WebAssembly and run by wazero, a WebAssembly runtime
written in Go. Compiling the CLI itself to WebAssembly would run PDFium in an
interpreter inside WebAssembly, so the text layer is the open point.

## Method

Two builds of the engine for `GOOS=js GOARCH=wasm`, Go 1.26, `-trimpath -ldflags "-s -w"`:

- **A. Core only.** Extraction, checks, comparison, report and the CCNL list,
  with a JavaScript entry point that takes text layouts (the
  `internal/layout` JSON) and returns the payslips and the HTML report
  (`cmd/cedolo-wasm`). The page reads PDFs with pdf.js 4.10 (Apache-2.0),
  splitting each text run into words and sharing its width by character.
- **B. Core with PDFium.** The CLI's PDFium reader compiled as is, run under
  Node.

Synthetic payslips in the three generator styles were read by both text
layers. The comparison records whether totals, statuses, item and leave
counts are identical, and the share of words with the same text and position
within 2 points; it never prints document content. Timings come from Node 22
and from headless Chromium, with a 4x CPU slowdown standing in for a phone.

## Results

Download size (gzip):

| Build | WebAssembly | pdf.js | Total |
| --- | --- | --- | --- |
| A. core + pdf.js | 1.9 MB (7.5 MB raw) | 0.5 MB (1.7 MB raw) | 2.4 MB |
| B. core + PDFium | 5.0 MB (15.5 MB raw) | none | 5.0 MB |

Speed:

| | Start | Per payslip |
| --- | --- | --- |
| A, Chromium | 0.5 s | about 0.2 s (pdf.js, plus 0.03 s for the core) |
| A, Chromium with 4x slower CPU | 0.5 s | about 0.25 s; 12 payslips and report in 3.2 s |
| B, Node | 3.8 s | 1.3 s (native CLI: 4.3 s once, then 0.015 s) |

Accuracy of A against the CLI (PDFium): on every synthetic style the
payslips, totals and statuses (all verified), items and leave balances were
identical. Only 57 to 75 percent of the words have the same position within 2
points, because pdf.js gives the width of a run and not of each glyph; the
extractor's column zones absorb the difference.

## Conclusion

Option A is viable: a page of about 2.4 MB that reads a year of payslips in a
few seconds on a slow device, with no server. Option B works but is twice the
download and about 90 times slower per payslip, because PDFium runs in an
interpreter inside WebAssembly.

The browser therefore uses pdf.js for the text layer and the Go core as
WebAssembly, as ADR 0004 anticipated, while the CLI keeps PDFium. Two text
layers must stay in agreement: the comparison used here becomes a test on the
synthetic styles, and a check any contributor can run locally on a real layout
without sharing it.

## Open points

- Word positions from pdf.js: measuring glyph widths from the font would make
  positions match PDFium closely; worth doing if a layout depends on them.
- Size: the core could shrink (TinyGo, or loading the CCNL list on demand),
  but 1.9 MB is already acceptable.
- Vertical text is approximated from the run's direction and needs a test on
  a synthetic layout that uses it.
