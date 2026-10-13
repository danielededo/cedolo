# ADR 0002: License cedolo under AGPL-3.0-or-later

- **Status:** Accepted
- **Date:** 2026-10-13
- **Deciders:** maintainer (danielededo)

## Context

cedolo will be public and open source. The main licensing risk is a closed
fork that hosts cedolo as a service, possibly with improved payroll vendor
templates, without sharing its changes. Templates are the project's most
valuable and most expensive asset, since each one requires real layout
knowledge contributed by users. The project has no plans to be embedded in
commercial products. The repository template only bundles MIT, Apache-2.0 and
a proprietary notice; copyleft is a deliberate choice outside it.

## Decision

We will license the whole repository, including templates and the item
taxonomy, under the GNU Affero General Public License v3.0 or later.
Contributions are accepted under the Developer Certificate of Origin
(Signed-off-by), without a CLA.

## Options considered

### Option 1: AGPL-3.0-or-later

- Pros: the network clause requires anyone offering a modified version over a
  network to publish the source, which covers a hosted fork; protects the
  template library; well known in the open source community.
- Cons: deters companies from embedding or contributing; some contributors
  avoid AGPL projects; relicensing later needs consent from all contributors.

### Option 2: GPL-3.0-or-later

- Pros: strong copyleft for distributed software, slightly less controversial.
- Cons: no network clause, so a modified hosted server mode could stay closed.

### Option 3: MIT or Apache-2.0

- Pros: maximal adoption, simplest for contributors and companies.
- Cons: explicitly allows the closed hosted fork the project wants to prevent.

### Option 4: MPL-2.0

- Pros: file-level copyleft, friendlier to mixed codebases.
- Cons: no network clause; a hosted fork could keep new files closed.

## Consequences

- The AGPL text is kept verbatim in LICENSE; the template's licenses/
  directory and license guide were removed during initialization.
- Source files may carry an SPDX identifier (AGPL-3.0-or-later).
- Dependencies must be AGPL-compatible; commercially licensed PDF SDKs are
  excluded anyway, permissive ones (MIT, BSD, Apache-2.0) are fine.
- If the goal ever becomes embedding cedolo in vendor or HR products, this
  decision must be superseded, which requires agreement from all copyright
  holders. Settling it before the first external contribution keeps that
  option cheap.

## References

- [docs/design/mvp-design.md](../design/mvp-design.md), section 7
- https://www.gnu.org/licenses/agpl-3.0.html
- https://developercertificate.org/
