# Design: taxable bases and TFR by the payslip's arithmetic

- Date: 2026-11-07
- Related: [ADR 0011](../adr/0011-recognize-by-arithmetic.md) (second step),
  [ADR 0009](../adr/0009-resolve-totals-by-arithmetic-agreement.md), issue #86,
  `internal/extract/arith.go` (first step)

## Goal

Read the INPS taxable base, the IRPEF taxable base and the month's TFR when
no known label gives them, from the relations among the numbers on the page.
Payslips read right today must read the same, and the blind test's floor must
rise.

Out of scope: the contributions and taxes summary boxes, IRPEF from the
yearly brackets, items (third step), labels as evidence only (fourth step).

## What real layouts show

Two real payroll layouts were measured on the maintainer's device (no values
leave it; only whether each relation holds):

| Relation | Layout 1 (9 payslips) | Layout 2 (12 payslips) |
|---|---|---|
| INPS contribution = INPS base × a known rate | 9 of 9 | 12 of 12 |
| TFR = base / 13.5 | 7 of 9 | the TFR is not printed |
| IRPEF base = INPS base − contributions | 0 of 9 | 0 of 12 |

The INPS rate is a reliable anchor. The IRPEF base relation holds on the
synthetic payslips but not on these real ones, where pension fund
contributions and other deductible items lower the IRPEF base. It is
therefore used only when it holds exactly, and never loosened (decision A).

Layout 1 also misses its IRPEF base for a label reason: it prints
"IMPONIBILE IRPEF LORDA", and the word between the label and the value stops
the reading. This change adds that label.

## Relations

With `c` the employee's INPS contribution, `b` a candidate INPS base, `t` a
candidate TFR and `i` a candidate IRPEF base:

- **R1, rate:** `c` is `b` times one of `inpsRates` (9.19%, 9.49%, 5.84%),
  within one cent, as the `inps_rate` check accepts.
- **R2, TFR:** `t` is `b / 13.5`, as is or net of the 0.5% contribution,
  within two cents, as the `tfr_ratio` check accepts.
- **R3, IRPEF base:** `i` is exactly `b − c`.

`c` is taken as `validate` takes it: the INPS items (`deductions.inps`), or
failing them the INPS box, or failing both the total of social
contributions. Without `c` nothing is read.

## Candidates

Every amount printed on the payslip's pages, with its place, except:

- the amounts of the item table's rows;
- the amounts that already are the source of a value read (totals, bases,
  roundings and the other boxes);
- amounts that are zero or negative.

Leave rows and year-to-date rows are not excluded. Leave values carry no
place to exclude them by, and a count of hours matching a relation to the
cent is unlikely; when one does, the ambiguity rule leaves the value unknown
rather than wrong. Year-to-date rows repeat the month's values in January,
which the same rule handles.

## Assignment

Only values that no label gave are filled; a value found by its label is
never replaced.

1. **INPS base.** A candidate `b` that satisfies R1 and at least one of R2
   or R3, with the TFR or IRPEF base read by its label or found among the
   candidates. R1 alone is never enough.
2. **TFR.** A candidate `t` that satisfies R2 with an INPS base confirmed by
   R1, read by its label or in point 1. When a TFR base is read by its
   label, R2 is taken on it instead, as the `tfr_ratio` check does, and the
   TFR is then no second link for the INPS base in point 1.
3. **IRPEF base.** A candidate `i` that satisfies R3 with an INPS base
   confirmed by R1.

**Ambiguity.** When candidates of different amounts satisfy the same
relation, the value stays unknown. When they share the amount, the first in
reading order is taken.

**Saying so.** A `bases_by_arithmetic` check, OK, records each value read and
the relations that confirm it. `validate` then runs `inps_rate`, `tfr_ratio`
and `irpef_base` on them as on any other value.

## Placement

`basesByArithmetic(p *payslip.Payslip, all [][]line)` in
`internal/extract/bases.go`, called in `extractPayslip` after
`totalsByArithmetic` and the label search of the boxes, before `resolve`.

The INPS rates stay in `inpsRates`: they do not change from year to year, so
moving them to the yearly figures table would widen this step without need.

## Tests

- **Unit tests** (`bases_test.go`):
  - each value read, then verified by `validate`;
  - R1 alone rejected;
  - different amounts satisfying the same relation rejected;
  - equal amounts accepted;
  - labels never replaced;
  - no contribution, nothing read.
- **Label:** a test that "IMPONIBILE IRPEF LORDA" followed by its value is
  read as the IRPEF base.
- **Blind test:** the floor raised to what the change reads. Expected:
  - INPS base, IRPEF base and TFR at 15 of 15 when their label alone is
    hidden;
  - "all totals" above 45 of 90.

  "Totals and items" likely stays at 45: without item labels the INPS
  contribution is not known, which is the third step.
- **No regressions on synthetic payslips:** the JSON of every synthetic
  payslip, with labels, identical before and after.
- **Real payslips,** before pushing, on the maintainer's device with
  `regress.sh`:
  - layout 2 identical;
  - layout 1 differing only by the IRPEF base now read (its `irpef_base`
    advisory check then fails, as the arithmetic says).
