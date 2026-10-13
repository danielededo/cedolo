# ADR 0006: Extract generically first, tune per vendor only where needed

- **Status:** Accepted
- **Date:** 2026-10-13
- **Deciders:** maintainer (danielededo)

## Context

[ADR 0005](0005-deterministic-extraction-with-validation.md) chose
deterministic extraction with arithmetic validation and no machine learning,
built around one template per payroll software with a fallback that only
reads the main totals on unknown layouts. The maintainer asked for cedolo to
work with the most common payslips regardless of the payroll software. No
real payslips are available yet, so a template for a specific vendor cannot
be written or tested today anyway.

Italian payslips share a content required by law (Law 4/1953) and a common
structure: a header with period, contract and level; a table of items with
earnings and deductions columns; totals; taxable bases; TFR. What differs
between vendors is mostly wording, column order and the position of values
relative to their labels.

## Decision

We will extract with a single generic engine that does not know the
vendor. It matches labels from a dictionary of synonyms, finds values to the
right of a label or in the label's cell below it, derives the item table's
columns from its header line (two columns for earnings and deductions, or one
signed amount column), classifies items by description through the
taxonomy, and assigns every total a status from the cross-checks (verified,
extracted, inferred, inconsistent, missing). ADR 0005's constraints remain:
deterministic, no machine learning, no external service, arithmetic as the
source of confidence. Vendor profiles (extra synonyms, column hints, layout
quirks) are added later only for layouts the generic engine gets wrong, and
are data, not code.

## Options considered

### Option 1: Generic engine, vendor profiles as tuning

- Pros: useful on any text payslip from day one; one code path to test and
  maintain; contributors improve coverage by adding synonyms; arithmetic
  checks catch most misreadings, and unrecognized items still count in the
  totals.
- Cons: heuristics can be wrong in ways a template would not; layouts with
  unusual structure (two item tables side by side, labels split over two
  lines) need specific handling.

### Option 2: One template per vendor (ADR 0005)

- Pros: exact on supported layouts.
- Cons: coverage grows one vendor at a time; useless until a template exists
  for the user's software; employers configure the same software
  differently, so even one vendor needs several templates.

## Consequences

- ADR 0005 is superseded by this ADR.
- Test data must cover layout variety. The synthetic generator produces
  several styles (two columns, signed single column, grid totals) and tests
  also use hand-built layouts the generator does not produce. Without real
  payslips there is still a risk that the heuristics fit the styles they
  were written against; the first real payslips are the true test.
- Payroll packages differ in ways the first synthetic styles did not cover,
  and the extractor handles them generically, each case covered by tests
  with invented values: contributions, taxes and roundings printed in
  summary boxes outside the item table; columns that are not pay (neutral
  items, statistical data, a presence calendar beside the table); and
  printed totals whose meaning varies ("totale trattenute" can be the item
  table total or everything withheld), resolved by a net balance check that
  tries the known readings and keeps the one under which the arithmetic
  holds. The PDF text layer joins glyphs of vertical text. The main risk
  recorded above stands: heuristics written against invented layouts can
  miss real ones, so every newly reported layout is worth more than any
  amount of synthetic variety.
- Checks that test a relation which legitimately fails in some situations
  (INPS rate, IRPEF taxable base, TFR ratio) are advisory: a mismatch is
  reported as a note and does not mark values inconsistent.
- Leave balances follow the same approach: measures and kinds are matched
  from synonyms, in tables of either orientation or inline.
- Known gaps today: scanned payslips, item tables printed in two side-by-side
  blocks, item descriptions wrapped over two lines, labels split over two
  lines. Each will be handled when a real layout shows it.
- The extraction output always says how far each value can be trusted, so a
  wrong reading shows up as an inconsistency rather than a silently wrong
  number.

## References

- [docs/design/mvp-design.md](../design/mvp-design.md), sections 2 and 6
- [ADR 0005](0005-deterministic-extraction-with-validation.md)
