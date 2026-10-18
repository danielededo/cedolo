# ADR 0009: Resolve totals by the reading that adds up

- **Status:** Accepted
- **Date:** 2026-10-17
- **Deciders:** maintainer (danielededo)

## Context

[ADR 0006](0006-generic-extraction-first.md) reads each total (gross pay,
deductions, net pay, contributions, taxes, roundings, taxable bases, TFR) from
the first label in a list of synonyms that yields a value, beside the label or
in its cell below. The cross-checks run afterwards and only grade what was
read: verified, or inconsistent when the arithmetic fails.

When a layout puts another number where the extractor looks first, the
reading is wrong and stays wrong. The cross-check flags it but tries nothing
else. Each such layout has so far been fixed with a geometric rule of its own:
a statistics column beside the labels, a value under a label that ends its
line, a row of year-to-date values. The rules work, but they accumulate and
interact, and every new layout risks needing one more. Meanwhile the right
value was usually among the numbers the extractor could have read: the
arithmetic would have picked it.

## Decision

We will resolve the totals of a payslip by choosing, among the candidate
readings of each total, the combination under which the cross-checks pass,
and by falling back to today's first reading when none does.

- **Candidates.** For each total, every occurrence of each of its labels
  yields its readings in today's order of preference: beside the label on its
  baseline, in its cell below, beside it on another baseline. Readings that
  are never preferred follow: a value at the foot of a tall box below the
  label, and, for taxes, the sum of boxes of their own (separate taxation of
  TFR, IRPEF adjustment). A total that today reads as missing can only be
  found through them. At most a few per total, so that the search stays
  small.
- **Choice.** Combinations are tried from the closest to the preferred
  readings, and the first under which every cross-check passes (net balance,
  deductions sum, earnings sum) wins. One that only passes more checks than
  the preferred readings is not taken: a single sum matched by chance is
  likelier than all of them. A payslip that today reads right and verifies keeps exactly the
  same values, because its first readings already pass.
- **Exactness.** A combination counts only when its arithmetic holds to the
  cent, as today. A reading chosen over the first because it reconciles is
  verified, and the check says which reading it is and why.
- **Scope.** Totals only. The item table, the period, the contract and leave
  keep their own logic.

The geometric rules stay as preferences between candidates, not as the only
way to the right value.

## Options considered

### Option 1: Keep adding geometric rules per layout

- Pros: each rule is small and local; nothing changes for layouts that work.
- Cons: rules accumulate and interact; each new layout needs a code change
  before it reads right; a wrong reading is reported but never corrected.

### Option 2: Choose the reading that adds up (chosen)

- Pros: unseen layouts with an extra number near a label read right without
  a new rule; the arithmetic, already the source of confidence (ADR 0005), also
  steers the reading; existing rules become preferences.
- Cons: a wrong combination could add up by coincidence; the extractor is
  harder to follow than a first match; a search instead of a single pass.

### Option 3: Per-vendor templates or a learned model

- Rejected again for the reasons in ADR 0005 and ADR 0006: templates cover
  one layout at a time, and a model needs real payslips to learn from and
  hides why a value was read.

## Consequences

- Coincidences are the risk being accepted. They stay rare because
  candidates come only from the cells of known labels, never from arbitrary
  numbers on the page, and the sums must match to the cent; preference
  breaks ties towards today's reading.
- The report and the JSON tell when a total was read from a less preferred
  place to make the arithmetic hold, so a person can check it.
- Every existing test must pass unchanged, and the real layouts compared
  before and after must keep their values, except where a value that was
  inconsistent becomes verified.
- Later, rules written for single layouts can be reviewed: some may become
  unnecessary once their layouts reconcile on their own. Removing one needs a
  test showing the layout still reads right.

## References

- [ADR 0005](0005-deterministic-extraction-with-validation.md),
  [ADR 0006](0006-generic-extraction-first.md)
- Pull requests that added layout rules for totals: #41
