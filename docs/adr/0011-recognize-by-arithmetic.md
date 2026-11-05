# ADR 0011: Recognize totals and items by the payslip's arithmetic, with labels as evidence

- **Status:** Proposed
- **Date:** 2026-11-05
- **Deciders:** maintainer (danielededo)

## Context

cedolo finds a payslip's totals and the meaning of its items from labels it
knows: the synonyms of each total (ADR 0006) and the taxonomy of items.
[ADR 0009](0009-resolve-totals-by-arithmetic-agreement.md) lets the arithmetic
choose between the readings near a known label, but every reading still starts
from one. A payroll package that words a label differently leaves its value
unread, or lets another number take its place, and an item it names
differently becomes "other".

The blind test in `internal/extract/blind_test.go` measures this. It prints
synthetic payslips with labels replaced by invented words, in the same places,
and counts what cedolo still reads:

- with no label hidden, 90 of 90 totals and 104 of 104 items;
- with one total's label hidden, the deductions and the net pay are still
  found, because ADR 0009 derives them, but the gross pay is found once in 15
  times, and the taxable bases and the TFR never;
- with every total's label hidden, 2 of 90 totals;
- with every item's label hidden, 0 of 104 items.

Yet a payslip holds more relations than ADR 0009 uses: the item rows (base ×
rate = amount), the employee's INPS contribution as a known share of its base,
the IRPEF base as the INPS base less the contributions, the TFR as the pay over
13.5, the leave balances. Together they pin most numbers to their meaning
whatever the words beside them.

## Decision

We will recognize totals, and then items, by the relations among the numbers on
the page, with the labels as evidence that breaks ties rather than a condition.

- **Candidates.** Every amount on the page is a candidate, with its place: line,
  column, neighbours, and the label beside it, known or not.
- **Relations.** A role (gross pay, net pay, INPS base, contributions, IRPEF
  base, IRPEF, TFR) is given to a number only when it takes part in relations
  that hold, to the cent where they are exact, and when several of them chain
  together. A single sum matched by chance is never enough, as in ADR 0009.
- **Labels as evidence.** A known label next to a number counts for the role
  it names. Between assignments that hold equally, the one the labels agree
  with wins, so that a payslip read right today reads the same.
- **Saying so.** Each value read this way records which relations confirm it.
  When no assignment holds, cedolo says the value is unknown instead of
  guessing.
- **Yearly figures.** The rates and brackets the relations need (INPS shares,
  IRPEF brackets) go in the table of figures set by law, per year, with their
  sources.

It comes in steps, each its own change that must keep every payslip read today
reading the same and must raise the blind test's floor:

1. totals: gross pay, deductions, net pay, beside today's extractor;
2. contributions, taxable bases, taxes and TFR;
3. items: the item rows whose arithmetic tells their meaning;
4. labels as evidence only, where the relations decide.

## Options considered

### Option 1: Keep extending labels and synonyms

- Pros: simple; each fix is local and easy to follow.
- Cons: every unseen wording needs a change before it reads right; what is not
  known is not read at all, or is read wrong.

### Option 2: Recognize by arithmetic, labels as evidence (chosen)

- Pros: unseen wordings read right when the numbers hold together; the
  arithmetic, already the source of confidence (ADR 0005), also finds the
  values; every value can say why it was read.
- Cons: a search, harder to follow than a lookup; coincidences are possible
  and must be guarded against; the relations need yearly figures kept current.

### Option 3: A model trained on payslips

- Rejected, as in ADR 0005, 0006 and 0009: it needs real payslips to learn
  from, which cedolo does not keep, and hides why a value was read.

## Consequences

- The blind test becomes the measure of progress: each step raises its floor,
  and no step may lower it.
- Coincidences are the risk accepted. They are kept rare by requiring chained
  relations, exact sums where the law makes them exact, and the labels'
  agreement between equal assignments.
- The yearly figures table grows, and the upkeep job (cmd/upkeep) asks for its
  check every year.
- The extractor is harder to read than a lookup; each value's recorded
  relations make its reading explainable, to people and in tests.
- Payslips that read right today must read the same at every step; real
  payslips are compared before and after on the maintainer's device only.

## References

- Issue #86
- [ADR 0005](0005-deterministic-extraction-with-validation.md),
  [ADR 0006](0006-generic-extraction-first.md),
  [ADR 0009](0009-resolve-totals-by-arithmetic-agreement.md)
- `internal/extract/blind_test.go`: the blind test and its floor
