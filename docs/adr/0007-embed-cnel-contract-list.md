# ADR 0007: Embed the CNEL list of national collective agreements

- **Status:** Accepted
- **Date:** 2026-10-14
- **Deciders:** maintainer (danielededo)

## Context

A payslip names the collective agreement (CCNL) it applies, but the name is
free text chosen by the payroll software. The CNEL assigns every national
agreement filed in its archive a unique four-character code, which
employers report to INPS every month; some payslips print it. Knowing the
contract lets cedolo show its official title, sector, latest agreement and
signatories, say when the agreement has expired, and later compare pay with
the contract's minimums.

The CNEL publishes the agreements in force as open data under the Italian
Open Data License v2.0, which allows redistribution with attribution. The
export is a JSON file of about 2,300 agreements and 1,160 codes. cedolo
processes payslips on the user's device (ADR 0003) and must work offline.

Names do not identify a contract: hundreds of agreements share nearly the
same title ("imprese edili e affini" appears under dozens of codes), and the
export carries no measure of how many workers each one covers.

## Decision

We will embed a table generated from the CNEL export in the binary
(`internal/ccnl/ccnl.tsv`, about 240 KB), regenerated with `cmd/ccnlgen` from
a downloaded export and reviewed as an ordinary diff. A contract is
identified only from a CNEL code printed on the payslip; from a name, cedolo
offers suggestions (`cedolo ccnl -search`) and says that names are not
unique, but never picks one.

## Options considered

### Option 1: embed a generated table (chosen)

- Pros: works offline, no request reveals which contract a user has, the
  data is versioned and reviewable, lookups are instant.
- Cons: the list ages between releases; the binary grows by about 240 KB.

### Option 2: query the CNEL API at run time

- Pros: always current.
- Cons: a network request per user leaks the contract to a third party and
  contradicts ADR 0003; fails offline and when the API changes.

### Option 3: match contracts by name

- Pros: works on payslips that do not print the code.
- Cons: wrong far too often given near-identical titles; a wrong contract
  would make later checks against minimums misleading.

## Consequences

- The list must be refreshed before releases; its date is shown wherever it
  is used, so a stale list is visible.
- Attribution to the CNEL and the license name travel with the table and
  appear in the CLI output and in the report.
- Payslips without the code get no contract details. Reading the code from
  other fields (such as Uniemens summaries) can extend coverage later.
- Comparing pay with contract minimums needs the minimum tables, which the
  export does not contain; that is separate work.

## References

- CNEL open data: https://www.cnel.it/archivio-contratti/contratti-open-data
- D.L. 76/2020, art. 16-quater (unique code of the agreements); INPS circular
  170/2021 (the code in Uniemens)
