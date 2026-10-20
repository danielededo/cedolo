# ADR 0010: Identify the person and the employer by their fiscal codes

- **Status:** Proposed
- **Date:** 2026-10-20
- **Deciders:** maintainer (danielededo)

## Context

cedolo has so far extracted only amounts, periods and the contract: never who
a payslip belongs to. The report assumes that all the payslips it is given are
of one person with one employer. They are often not:

- a household reads the payslips of two people at once;
- a person who changes job has, for a month or more, payslips from two
  employers, sometimes for the same month.

The report then mixes them. Two payslips of the same month look the same in
the month picker. "What changed" compares a payslip with one of another
employer or person, and the charts and yearly totals add them together.

Every payslip prints the worker's codice fiscale and name, and the employer's
codice fiscale or VAT number and name. The codice fiscale has a fixed format
and a check character, and cedolo already detects it, with its check, in
`internal/pii`. Names have no fixed place or order, and sit near the
employer's.

## Decision

We will identify the person of each payslip by the codice fiscale it prints,
and the employer by its codice fiscale or VAT number. Names are read only to
be shown, never to group.

- The report is built per person. With more than one person, a selector at the
  top switches the whole report from one to the other. A payslip with no
  readable codice fiscale joins the only person there is, or else a group of
  its own.
- Within a person, every payslip counts in the year's totals and charts, but
  the month is compared only with the previous payslip of the same employer.
  The month picker names the employer when a month has more than one.
- The person's name, or else the codice fiscale masked to its first three and
  last three characters, is shown at the top of the report, in the web app
  and in the downloaded report alike. The employer's name is shown only where
  it tells two employers apart.

Everything stays on the device, as [ADR 0003](0003-on-device-processing-by-default.md)
requires. The anonymizer of [ADR 0008](0008-rebuild-anonymized-payslips.md)
keeps masking names and codes.

## Options considered

### Option 1: Group by codice fiscale, show the name

- Pros: the grouping is exact and checkable; the same person is recognized
  whatever the order or spelling of the name; the existing detector is reused.
- Cons: cedolo now handles identifiers; the downloaded report names its owner.

### Option 2: Group by the name

- Pros: no identifier is read.
- Cons: names have no fixed place, order or spelling, and are confused with
  the employer's. Two people would merge, or one person would split.

### Option 3: Group by contract and level, as a proxy

- Pros: no personal data.
- Cons: two people with the same contract merge. A promotion splits one
  person in two.

### Option 4: Do nothing

- Pros: no change.
- Cons: mixed payslips keep giving wrong comparisons and totals.

## Consequences

- The downloaded report holds the person's name. It already holds every
  amount, and it keeps saying that it must be kept private.
- The payslip data gains the person's and employer's identifiers and names. The
  CLI's `extract` output shows them; it already warns that it holds payslip
  data. Tests use invented, check-valid codes only, and piiscan keeps real ones
  out of the repository.
- Comparisons, charts and yearly totals are worked out per person, and
  comparisons per employer. Wherever the report goes over payslips in turn, it
  has to say which payslips that means.
- Reading the name can fail. The masked code then stands in for it, and
  grouping does not depend on it.

## References

- [ADR 0003](0003-on-device-processing-by-default.md), [ADR 0008](0008-rebuild-anonymized-payslips.md)
- `internal/pii`: codice fiscale and IBAN detection with check characters
