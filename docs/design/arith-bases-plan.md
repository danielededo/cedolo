# Taxable bases and TFR by arithmetic: implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** read the INPS base, the IRPEF base and the TFR from the page's arithmetic when no label gives them (ADR 0011, step 2), and read the "IMPONIBILE IRPEF LORDA" label.

**Architecture:** a new pass `basesByArithmetic` in `internal/extract/bases.go`, the twin of `totalsByArithmetic`. It collects the page's amounts as candidates and fills a missing base only through chained relations. The relations reuse the checks `validate` already runs, moved into small helpers so both use the same tolerances.

**Tech Stack:** Go 1.26 (toolchain go1.26.10), standard library only.

**Spec:** `docs/design/arith-bases.md`

## Global Constraints

- Money is `int64` cents, never floating point.
- No new dependencies.
- A value found by its label is never replaced.
- R1: `|c − roundDiv(b×rate, 10000)| ≤ 1` for a rate in `inpsRates` (919, 949, 584).
- R2: `|t − roundDiv(b×2, 27)| ≤ 2` or `|t − roundDiv(b×373, 5400)| ≤ 2`.
- R3: `i == b − c`, exactly.
- The check's name is `bases_by_arithmetic`, `OK: true`, not advisory.
- No real payslip, value or file name in code, tests, commits or the PR (AGENTS.md).
- Every synthetic payslip with its labels must extract byte-identical JSON before and after.
- Commits follow Conventional Commits, without AI attribution. No push until Task 4 passes.

## Review Focus

1. **January:** the year-to-date row repeats the month's base, so both candidates have the same amount. The first in reading order must be taken, not "unknown" (Task 3: `TestBasesSameAmountTwice`).
2. **A labelled INPS base that fails R1** (special contracts): no TFR or IRPEF base may be read from it (Task 3: `TestBasesNeedConfirmedBase`).
3. **Signed amounts** such as `201,67-` and zero amounts are not candidates (Task 3: `TestBasesSkipNegativeAndZero`).
4. **A multi-page payslip:** candidates come from every page, and item rows of later pages stay excluded (Task 3: `TestBasesAcrossPages`).
5. **Two different amounts satisfying the same relation:** the value stays unknown (Task 3: `TestBasesAmbiguous`).

---

### Task 1: Share the relations between validate and the new pass

No behaviour change.

**Files:**
- Modify: `internal/extract/validate.go`, the INPS contribution switch and the rate and TFR comparisons.
- Test: `internal/extract/relations_test.go` (new).

**Interfaces:**
- Produces:
  - `func inpsContribution(p *payslip.Payslip) (int64, bool)`: the `deductions.inps` items, else `p.INPS`, else `p.SocialContributions`, as `validate` picks it today.
  - `func rateHolds(c, b int64) (rate int64, ok bool)`: R1, with the first matching rate in `inpsRates` order.
  - `func tfrHolds(t, b int64) (net bool, ok bool)`: R2; `net` reports the 0.5%-net form.

- [ ] **Step 1:** write `TestRelations` in `relations_test.go`:
  - `rateHolds(20167, 219450)` gives `(919, true)`;
  - `rateHolds(20000, 219450)` is not ok;
  - `tfrHolds(16256, 219450)` gives `(false, true)`;
  - `tfrHolds(15158, 219450)` gives `(true, true)`: 162,56 less 0.5% of the base, `roundDiv(219450*373, 5400)`;
  - `tfrHolds(16000, 219450)` is not ok;
  - `inpsContribution` takes the items first, then the `INPS` box, then `SocialContributions`, and is not ok when none is found.
- [ ] **Step 2:** `go test ./internal/extract -run TestRelations` fails: undefined functions.
- [ ] **Step 3:** implement the three helpers in `validate.go` and make `validate` call them. The `inps_rate` and `tfr_ratio` details must keep their current wording and values.
- [ ] **Step 4:** `go test ./internal/extract/...` passes, the existing tests included.
- [ ] **Step 5:** commit `refactor(extract): share the INPS rate and TFR relations`.

### Task 2: Read "IMPONIBILE IRPEF LORDA"

**Files:**
- Modify: `internal/extract/labels.go:18` (`irpefBaseLabels`).
- Test: `internal/extract/foreign_test.go`, adding `TestIRPEFBaseLorda`.

- [ ] **Step 1:** write `TestIRPEFBaseLorda`. Build a layout with `page(...)`, using only invented values:
  - a table header, two earnings and an INPS item;
  - totals with known labels;
  - a line `40:IMPONIBILE IRPEF LORDA 1.992,83` plus `200:TRATTENUTA IRPEF LORDA 400,00` on the same line.

  Assert that `Document(...)[0].IRPEFBase.Cents == 199283` and its status is not `missing`.
- [ ] **Step 2:** run it and see it fail. If it already passes, the real-layout failure has another cause: stop and report it.
- [ ] **Step 3:** add `"imponibile irpef lorda"` first in `irpefBaseLabels`. If the test still fails, look at `valueTokens` and `amountReadings` (`fields.go`) for why the word after the label stops the reading, and make the smallest fix there.
- [ ] **Step 4:** `go test ./internal/extract/...` passes.
- [ ] **Step 5:** commit `fix(extract): read the IRPEF base labelled "imponibile irpef lorda"`.

### Task 3: `basesByArithmetic`

**Files:**
- Create: `internal/extract/bases.go`.
- Modify: `internal/extract/extract.go`, calling it right after `totalsByArithmetic(&p)`.
- Test: `internal/extract/bases_test.go`.

**Interfaces:**
- Consumes: `inpsContribution`, `rateHolds` and `tfrHolds` (Task 1); `number{cents, source}` (`arith.go`); `line`, `sourceOf` and `money.IsAmount`/`money.Parse`.
- Produces:
  - `func basesByArithmetic(p *payslip.Payslip, all [][]line)`;
  - `func pageAmounts(p *payslip.Payslip, all [][]line) []number`: the candidates in reading order (page, then line, then x). It leaves out:
    - tokens on the line of an item's source (same page, `|Y − item.Source.Y| < 1`);
    - tokens that are the source of an amount already read (same page, X and Y within 0.5);
    - amounts ≤ 0.

  The amounts already read are Gross, Deductions, Net, INPSBase, IRPEFBase, TFR, TFRBase, SocialContributions, Taxes, INPS and the two roundings.

The algorithm, as the spec's Assignment section, in this order:
1. `c, ok := inpsContribution(p)`. Without it, return.
2. If `INPSBase` is missing: the candidates `b` with `rateHolds(c, b)` and a second link. The second link is a labelled TFR or a candidate with `tfrHolds(t, b)`, or a labelled IRPEF base or a candidate equal to `b − c`.
   - Distinct amounts among those `b`: exactly one, then set `INPSBase` from its first candidate; more than one, then leave it.
3. With `INPSBase` found and `rateHolds(c, INPSBase)`:
   - fill a missing `TFR` from the candidates with `tfrHolds`;
   - fill a missing `IRPEFBase` from the candidates equal to `INPSBase − c`;
   - each by the same rule: one distinct amount, first candidate.
4. Values set have `Status: payslip.Extracted` and the candidate's `Source`.
5. When at least one is set, append one `bases_by_arithmetic` check. Its detail names each value and its relations, e.g. "INPS base 2.194,50 read without a known label: INPS 201,67 is 9,19% of it, and the TFR 162,56 is it divided by 13.5".

The tests build `[][]line` with `buildLines(page(...))` and a `payslip.Payslip` holding the label readings. Each first fails because `basesByArithmetic` is undefined.

- [ ] **Step 1:** write the tests, using the synthetic example's values (INPS item 201,67, base 2.194,50, IRPEF base 1.992,83, TFR 162,56):
  - `TestBasesAllHidden`: none of the three is labelled. All three are read, `validate` then marks them `verified`, and there is one `bases_by_arithmetic` check.
  - `TestBasesINPSHidden`: the TFR and IRPEF base are labelled and the INPS base is read.
  - `TestBasesTFRHidden` and `TestBasesIRPEFHidden`: each read from a labelled INPS base.
  - `TestBasesRateAlone`: the page holds 2.194,50 but no TFR or IRPEF base that agrees, so nothing is read and there is no check.
  - `TestBasesAmbiguous`: two different TFR candidates (162,56 and 162,57) both within 2 cents, so the TFR stays missing.
  - `TestBasesSameAmountTwice`: 1.992,83 printed twice, so it is read with the first source.
  - `TestBasesNeedConfirmedBase`: a labelled INPS base of 2.000,00 that fails R1, so no TFR or IRPEF base is read.
  - `TestBasesKeepLabels`: labelled values are unchanged, even when a candidate fits better.
  - `TestBasesNoContribution`: no INPS item or box, so nothing is read.
  - `TestBasesSkipNegativeAndZero`: `pageAmounts` leaves out `201,67-` and `0,00`.
  - `TestBasesAcrossPages`: the base on page 2. An item row on page 2 with an amount equal to `b − c` is not taken as the IRPEF base.
  - `TestBasesSkipsItemRows`: an item's amount equal to `b − c` is not a candidate.
- [ ] **Step 2:** `go test ./internal/extract -run TestBases` fails: undefined.
- [ ] **Step 3:** implement `bases.go` and the call in `extract.go`.
- [ ] **Step 4:** `go test ./internal/extract/...` passes. `TestBlindLabels` passes too, since floors are minimums.
- [ ] **Step 5:** commit `feat(extract): read the taxable bases and TFR by their arithmetic`.

### Task 4: Raise the floor, document, verify

**Files:**
- Modify: `internal/extract/blind_test.go` (`blindFloor`), `CHANGELOG.md` (Unreleased → Added and Fixed).
- Local only, never committed: a snapshot test kept outside the repository.

- [ ] **Step 1:** `go test ./internal/extract -run BlindLabels -v` and read the new figures. Set `blindFloor` to them, with a comment as in step 1's change. Expected:
  - "INPS base", "IRPEF base" and "TFR" each at 90;
  - "all totals" above 45.

  If a row reads less than expected, find out why before setting it.
- [ ] **Step 2:** add the CHANGELOG entries:
  - **Added:** the bases and TFR read when their labels are unknown, when the INPS rate and a second relation confirm them (ADR 0011, second step).
  - **Fixed:** the IRPEF base labelled "imponibile irpef lorda".
- [ ] **Step 3:** run the synthetic snapshot. Write `zz_synth_snapshot_test.go` (package `extract_test`, run only when `CEDOLO_SNAPSHOT` names an output file). It writes the JSON of `extract.Document(synth.Layout(p, s))` for:
  - every `synth.Styles()` style with `ExampleFor`;
  - every month of `synth.Year(y)` for 2019 to 2026, in every style.

  Copy it into `internal/extract` of a `main` worktree and of this branch, run both, then `cmp`. Expected: identical. Delete the copies.
- [ ] **Step 4:** compare `main` with this branch on the maintainer's real payslips, on their device only. Print verdicts and the names of fields and checks, never values, and record nothing about them here. Expected:
  - one layout identical;
  - the other different only by `irpef_base` now found, with its advisory check failing.
- [ ] **Step 5:** run the checks: `gofmt -l .` prints nothing; `go vet ./...`; `GOOS=js GOARCH=wasm go vet ./cmd/cedolo-wasm`; `go run honnef.co/go/tools/cmd/staticcheck@2026.1 ./...`; `go test ./...`; `go run ./cmd/piiscan .`. All pass.
- [ ] **Step 6:** commit `test(extract): raise the blind floor for the bases and TFR` with the CHANGELOG. Report the results to the user. Push and open the PR only after their go-ahead.
