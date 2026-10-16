# ADR 0008: Share payslips as rebuilt, allowlist-anonymized PDFs

- **Status:** Accepted
- **Date:** 2026-10-16
- **Deciders:** maintainer (danielededo)

## Context

cedolo reads payslips from many payroll programs, and the only way to fix a
layout it reads badly is to see that layout. Users cannot attach the
payslip: it holds their name, codice fiscale, IBAN, address, employer, and
sometimes health or union data. `cedolo anonymize` already prints the
positioned text with every token masked unless it is known payroll
vocabulary, and every digit masked. That output is a JSON file, which
non-technical users cannot produce from the web app or review. And without
amounts, bugs in the arithmetic cross-checks cannot be reproduced.

Editing the original PDF is not safe. Text stays in the file under a black
box; incremental saves keep the old objects; metadata, embedded font subsets,
images, annotations, attachments and alternate text all leak. Glyph widths
and positions also let an attacker guess covered words. A pseudonym derived
from the real value, such as a hash of the codice fiscale, can be reversed by
trying every possible value.

## Decision

We will share payslips as new PDFs rebuilt from the anonymized text layout,
never as modified originals.

- **Allowlist.** A token is kept only when every word is known payroll
  vocabulary and none reveals special categories of data. Everything else is
  masked character by character: letters become x or X, digits 0, and length
  and position are kept. Nothing is derived from the masked value.
- **Amounts kept by default.** Tokens that are amounts or quantities with
  decimals (`1.234,56`, `168,00`, `9,19`), years, and month/year periods
  are kept, so the cross-checks can be reproduced. Other digits are
  masked: dates, codes, identifiers, postal codes and integers. An option
  masks amounts too.
- **Rebuilt file.** The PDF holds only text drawn in a standard,
  non-embedded font at each token's position, scaled to its width. It has
  no images, metadata or structure from the original, and no compression,
  so what it contains can be checked byte by byte.
- **Checks before writing.** The output must contain no valid codice
  fiscale or IBAN. No masked word of four letters or more may appear in
  the output. If either check
  fails, nothing is written.
- **Review.** Users see the full anonymized text before saving, with a
  warning that amounts are kept.

## Options considered

### Option 1: redact the original PDF

- Pros: the file is identical apart from the covered parts, so every reading
  bug reproduces.
- Cons: every leak listed in the context has to be found and closed in
  arbitrary PDFs; one missed object exposes everything.

### Option 2: detect personal data and replace it with fake values

- Pros: realistic-looking output.
- Cons: detection misses what it does not expect, such as a name in a note
  or an address on two lines. A blocklist fails open.

### Option 3: rebuild from an allowlist-masked layout (chosen)

- Pros: it fails closed, because an unknown word is masked. The output holds
  only what cedolo wrote. It works the same in the CLI and the web app,
  since both have the layout.
- Cons: bugs in reading the PDF itself (fonts, encodings, odd text
  operators) do not reproduce from a rebuilt file. Box widths still show
  roughly how long each masked word was. Amounts, kept by default, can
  identify someone to a person who already knows their pay.

## Consequences

- Users can share a payslip that cedolo reads badly without technical
  steps, and maintainers get a file that runs through the normal pipeline.
- Amounts kept by default are a deliberate trade-off. The warning and the
  preview say so, and the option to mask them stays one click away.
- New payroll vocabulary must be added to the allowlist before it shows up
  in anonymized files; until then it is masked, which costs only
  readability.
- The JSON output of `cedolo anonymize` remains for layout bugs and follows
  the same rules.

## References

- [ADR 0003](0003-on-device-processing-by-default.md): processing on the
  device.
- [ADR 0004](0004-go-core-with-pluggable-text-layer.md): the text layout
  boundary.
