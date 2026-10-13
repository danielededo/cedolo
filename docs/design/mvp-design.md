# cedolo: product and architecture design for the MVP

- Status: draft for discussion
- Date: 2026-10-13
- Deciders: maintainer (danielededo)

This document records the design discussion for cedolo, an open source tool that
helps Italian employees understand their payslips (cedolini, buste paga). It is
the input for the ADRs in [docs/adr/](../adr/); where a decision is significant
enough to need one, the section links to it.

## 0. Constraints this design is built on

The maintainer answered a short set of priority questions before any design
work. These answers drive every recommendation below, so they are stated first;
if any of them changes, the affected sections must be revisited.

| Question | Answer | Main consequence |
| --- | --- | --- |
| Goal and time | Ship something useful, about 5 hours a week, solo | Scope must be very small; every component must earn its maintenance cost |
| Default processing location | On-device only | No backend in the MVP; the payslip never leaves the user's machine |
| External LLM | Never; local processing only | Extraction is deterministic (templates plus validation); no AI spend |
| Sample payslips | The maintainer's own, public sample payslips found online, synthetic data | One payroll software layout first; synthetic fixtures are a first-class need |
| First audience | The maintainer first, then the public | v0.1 is a CLI; the public UI comes in v0.2 |
| Language | Go is wanted | Go for the core and CLI; the browser UI question is handled in section 4 |

At 5 hours a week, a realistic pace is about 20 hours a month. The roadmap in
section 9 is sized for that: roughly two to three months to v0.1 and three to
four more to a public v0.2. Anything that does not fit that budget is cut, not
squeezed in.

## 1. Vision and MVP

### Problem

Italian payslips are dense, inconsistent across payroll software vendors, and
full of terms (imponibile, conguaglio, addizionali, ratei, ROL) that most
employees never learn. The questions people actually have are simple: how much
do I really earn, where does the difference between gross and net go, did
something change this month and why, how much leave and TFR have I accumulated.
Answering them today means reading twelve PDFs side by side.

cedolo's core job is to turn a pile of payslips into a reliable time series and
explain it in plain language. Everything else is secondary.

### What cedolo is not

It is not a payroll calculator, a tax advisor, or a tool that says "your
employer is wrong". It shows what the payslip says, checks it for internal
consistency, highlights changes, and explains terms. Judging correctness
against law or a collective agreement (CCNL) is a different, much riskier
product and is out of scope for the MVP.

### What not to build in the first versions

OCR of scanned payslips and photos, recomputing IRPEF or INPS from scratch,
comparison with CCNL minimum tables, any server, accounts or sync, mobile apps,
any external AI service, multi-user support, and analytics or telemetry. Each
of these is either a maintenance sink (CCNL tables, tax rules that change every
year), a privacy liability (server, telemetry), or a large accuracy problem
(OCR) that would dominate a 5-hour week.

### Prioritized features

Must, for v0.1 (maintainer only, CLI):

- Import text-based PDFs produced by one payroll software (the maintainer's).
- Extract the header: period (month, year), payslip kind (monthly, tredicesima,
  quattordicesima, adjustment), employer, CCNL, level, qualifica.
- Extract totals: gross, INPS employee contributions, IRPEF, regional and
  municipal surcharges (addizionali), net in payslip, net paid.
- Extract line items with code, description, quantity, base, rate and amount,
  and map them to a canonical taxonomy.
- Extract TFR accrued in the month and its destination, and leave balances
  (ferie, permessi, ROL: accrued, used, remaining).
- Validate each payslip with arithmetic cross-checks (section 6).
- Store results locally as plain files; export CSV and JSON.
- Produce a static HTML report: gross and net over time, deduction breakdown,
  month-over-month comparison.

Should, for v0.1 or v0.2:

- A second and third vendor template, built from public sample payslips.
- An anonymizer command that turns a real payslip into a shareable, redacted
  layout for contributing templates (section 5).
- A plain-language glossary for every canonical item, in English and Italian.
- Change detection: level or CCNL change, a recurring item that disappeared,
  an unusual jump in net pay, a change in effective tax rate, with an
  explanation of seasonal effects (December conguaglio, surcharges, 13th month).
- Correction workflow: the user confirms or fixes uncertain fields.

Could, later:

- Supplementary pension fund details (employee and employer shares).
- Seniority (anzianità) when printed or derivable from hire date.
- Optional encryption of the local data folder.
- Reconciliation with the annual Certificazione Unica (CU).
- Experimental OCR for scans, running locally.

### First realistic release

v0.1 is a Go command-line tool that reads the maintainer's own payslips from
one vendor and produces a correct, validated data set plus an HTML report. It
is deliberately unimpressive to outsiders and deliberately correct. v0.2 is the
first public release: an offline-capable web app running entirely in the
browser, with two or three vendor templates and EN/IT interface.

Critique of the initial idea: the original feature list covers almost every
field in a payslip plus CCNL comparison and anomaly detection. For a solo
maintainer at 5 hours a week, attempting all of it guarantees a half-working
tool. Extraction correctness for a single vendor is the foundation; the
dashboard is worthless if the numbers are wrong.

## 2. Data extraction

### Options

Deterministic templates per known layout. The PDF text layer is read with
positions, and a template for a specific vendor locates anchors (labels such as
"TOTALE COMPETENZE") and reads values from known regions or columns.
Accuracy on supported layouts is very high, often exact to the cent, and
every value can be traced to a position on the page. The cost is that each
vendor, and sometimes each vendor version or employer configuration, needs a
template, and layouts drift over time. Runs anywhere, costs nothing.

OCR with heuristics. Needed only for scans and photos. Numbers are usually read
well, but table structure, column alignment and minus signs are not.
Accuracy is materially lower and highly variable with scan quality. Local OCR
(Tesseract, also available in the browser) is free but slow and heavy.

Local machine learning models. Document layout models or small local language
models with structured output can generalize across layouts. In practice they
are too heavy to run in a browser or on an average laptop with good latency,
their accuracy on Italian payslips is unknown without a labelled data set we do
not have, and they can produce plausible but wrong numbers. A wrong number that
looks right is the worst failure for this product.

Cloud LLM with structured output. Best coverage with least effort, but excluded
by the constraints: it sends the document to a third party and costs money.

Hybrid. Templates for known layouts, a generic fallback that finds the handful
of totals every Italian payslip prints with recognizable labels, arithmetic
validation on everything, and explicit user confirmation for what remains
uncertain.

### Recommendation

Hybrid, without ML: templates first, a label-dictionary fallback for key
totals on unknown layouts, arithmetic validation as the source of confidence,
and user confirmation for gaps. See
[ADR 0005](../adr/0005-deterministic-extraction-with-validation.md).

The key idea is that confidence does not come from a model score but from the
payslip's own arithmetic. A payslip contains enough redundancy (sum of earnings
equals gross, gross minus deductions equals net, INPS equals taxable base times
rate) that an extraction which balances is almost certainly right, and one that
does not balance is flagged, whatever method produced it.

On unknown layouts the fallback only promises the main totals (gross, net,
INPS, IRPEF, period). The user sees an honest "partially supported" state and
an invitation to contribute a template, rather than a full but unreliable
breakdown.

What would make this wrong: if most target users have scanned or photographed
payslips rather than downloaded PDFs. Payroll portals almost always provide
text PDFs, but this must be checked with early users before v0.2. If scans
dominate, OCR moves up to a must, and the effort estimate roughly doubles.

### Template design: avoid a premature DSL

Long term, templates should be declarative data (YAML or JSON) so contributors
can add a vendor without writing Go: vendor fingerprint, anchors, regions,
column definitions, and a code-to-canonical-item mapping table. But designing
that format before having two real layouts would produce the wrong
abstraction. The first template is written in Go against a small internal API;
the declarative format is extracted when the second vendor is implemented, and
validated on the third.

### Handling ambiguity

Every extracted field has a status: verified (passes a cross-check), extracted
(found by a template, no check available), inferred (fallback heuristic),
missing, or corrected by the user. Ambiguous cases are surfaced as questions
with the source region of the page shown next to the value, never silently
resolved.

Known traps the extractor must handle explicitly: Italian number format
(1.234,56), trailing minus signs and separate earnings/deductions columns,
quantities in hours versus days, multiple payslips in one PDF (yearly bundles),
13th month printed either as a separate payslip or inside December, year-end
tax adjustment (conguaglio), and arrears (arretrati) referring to other periods.

## 3. Data model

### Principles

Money is stored as integer cents (int64), never as floating point. Rates are
stored as decimal strings or basis points. Every extracted value keeps its
provenance: document, page, bounding box, raw text, and the extractor and
template versions that produced it. Extracted data is immutable; user
corrections are a separate layer.

### Entities

- Document: an imported file. Content hash (SHA-256), page count, import date,
  detected vendor. The original file is not retained by default; only the hash
  is, so re-importing the same file is detected.
- ExtractionRun: one run of the engine on a document. Engine version, template
  id and version, timestamp, and the raw extracted values with provenance and
  status. Never modified after creation.
- Payslip: one payroll period within a document (a document can hold several).
  Period year and month, kind (monthly, 13th, 14th, adjustment), reference to
  its ExtractionRun.
- Employer: name and, if printed, VAT number. Deduplicated across payslips.
- EmploymentSnapshot: per payslip, the contract data as printed: CCNL, level,
  qualifica, part-time percentage, hire date, seniority if printed. Kept as a
  snapshot per period, so changes over time are visible by diffing snapshots.
- LineItem: one row of the payslip body. Raw vendor code, raw description,
  quantity, unit, base, rate, amount, and the canonical item key it maps to
  (or "unmapped").
- Totals: the printed totals (gross, INPS taxable base, IRPEF taxable base,
  total deductions, net in payslip, rounding, net paid).
- TaxAndContributions: INPS employee share, IRPEF gross and net of credits,
  regional and municipal surcharges, with installment information if printed.
- TFR: monthly accrual, revaluation if printed, destination (company, INPS
  treasury fund, pension fund), and fund name.
- PensionFund: employee share, employer share, TFR share, fund name.
- LeaveBalance: per type (ferie, permessi, ROL, ex festività), unit, accrued,
  used, remaining, as printed.
- Correction: a user edit to one field of one payslip: field path, old value,
  new value, timestamp, optional note.

There is deliberately no User entity in local mode: the local data folder
belongs to one person. The worker's name, employee number and codice fiscale
are not stored by default (section 5). If the self-hosted mode in phase 4 ever
supports multiple users, a User entity appears there, outside the core model.

### Normalization across vendors

The core of normalization is a canonical item taxonomy with stable English keys
and bilingual labels and explanations, for example
earnings.base_salary, earnings.contingency (contingenza),
earnings.seniority_increments (scatti di anzianità), earnings.overtime,
earnings.allowance.*, earnings.bonus, deductions.inps.employee,
tax.irpef, tax.surcharge.regional, tax.surcharge.municipal,
tfr.accrued, pension_fund.employee, union_dues, and so on.

Each template carries a mapping table from vendor codes to canonical keys.
Codes are not reliable across employers even within the same vendor, because
payroll offices configure their own, so the mapping falls back to description
matching, then to "unmapped". Unmapped items are still shown and summed; they
just lack an explanation. The taxonomy is versioned and only grows: keys are
never renamed, only deprecated with an alias.

### Versioning of extracted data

The effective value of a field is the latest user correction if present,
otherwise the value from the latest ExtractionRun. When a template or the
engine improves, documents can be re-extracted, producing a new run. A
correction is re-applied to the new run only if the raw extracted value it
corrected is unchanged; otherwise the user is asked again. This keeps upgrades
safe without losing user work.

## 4. Technology stack

### Language and core

Go for the core and the CLI, as wanted. It fits well: a single static binary,
excellent standard library for the needed pieces (encoding, html/template,
testing), easy cross-compilation, and the same core can later be compiled to
WebAssembly for the browser or run as a server. See
[ADR 0004](../adr/0004-go-core-with-pluggable-text-layer.md).

The core is organized so it never touches the file system, the network or a PDF
parser directly. Its input is a "text layout": pages with text tokens and their
bounding boxes. Its output is the data model of section 3. This single
boundary is what allows one engine to run in a CLI, in a browser and on a
server without rewrites, and it allows tests to use small synthetic layout
fixtures instead of PDFs.

The weak point of Go is PDF text extraction. Pure-Go libraries exist but are
less mature than pdf.js or poppler, and the best-known complete Go PDF SDK is
commercially licensed. The plan handles this by isolating PDF reading behind
the text layout interface:

- In the CLI, PDFium compiled to WebAssembly and run inside the Go binary,
  chosen after the [phase 0 spike](spike-pdf-text-layer.md) showed the best
  known pure Go reader losing positions on common font encodings; poppler's
  pdftotext stays available as a reference.
- In the browser, pdf.js (Apache-2.0) extracts text and positions and hands a
  text layout to the Go core compiled to WebAssembly.

The phase 0 spike tested this on synthetic documents from three PDF
generators; the same check still has to run on real payslips, which the
maintainer can do locally with `cedolo inspect`, whose output contains no
personal data.

What would make Go wrong: if the public product must be a browser app and the
Go WebAssembly bundle turns out too large or slow on mid-range phones (a few MB
compressed is expected, which is acceptable on desktop and borderline on
mobile), or if contributors mostly know TypeScript. In that case the fallback
is a TypeScript core for the browser, which would duplicate the engine; this
is the main reason to keep templates declarative so that at least they are
shared.

### Storage

CLI: a local data folder with one JSON file per document (runs, payslips,
corrections), plus a small index. A person has about fifteen payslips a year,
so a database is unnecessary; plain files are inspectable, diffable, trivially
exportable and easy to delete. SQLite can be introduced later if a real need
appears.

Browser: IndexedDB holding the same JSON structures, with export and import
of the whole data set as one file so users are never locked into one browser.

Original PDFs are not stored by default in either mode.

### Presentation

v0.1: the CLI prints summaries and writes a self-contained static HTML report
(Go html/template plus one small embedded chart library). No JavaScript build
pipeline is needed for v0.1.

v0.2: two credible options, to be decided after v0.1.

- Static web app: TypeScript UI with a minimal framework (Svelte or Preact),
  pdf.js, Go core as WebAssembly, served as static files from any host or
  opened offline. Zero install, strongest and verifiable privacy claim
  (the page can enforce a Content Security Policy that forbids network
  connections). Cost: a second toolchain.
- Local app: the same Go binary runs `cedolo serve` on 127.0.0.1 with a server-
  rendered UI (html/template plus htmx). Single language, simplest stack.
  Cost: non-technical users must download and run an unsigned binary, which
  macOS and Windows actively discourage; code signing costs money every year.

Recommendation: static web app for the public release, because install
friction is fatal for non-technical users. The Go core boundary keeps both
options open, and the local app remains the natural base for self-hosting.

Charts: a small, dependency-light library such as Chart.js or uPlot.
Internationalization: message catalogs in JSON for en and it, with the
glossary as structured data keyed by canonical item; Italian payroll terms
(TFR, IRPEF, ROL) stay untranslated in both languages, with an explanation.

### Analytics

None in the app, ever by default. The product promise is that nothing leaves
the device; a telemetry exception would undermine it and invite scrutiny. If
the project website needs usage numbers, use a cookieless, self-hostable
page-view counter on the marketing pages only, never on the app.

### CI and tooling

GitHub Actions running go vet, staticcheck or golangci-lint, gofmt and go test;
later a WebAssembly build and a headless browser test. A PII scanner over
test fixtures runs in CI (section 5).

## 5. Privacy, security and GDPR

### Payslips contain special category data

Beyond income, a payslip can reveal union membership (union dues deductions),
health (sick leave, injury, Law 104 leave for disability or caregiving),
pregnancy and parental leave, family situation (family allowances, tax credits
for dependants), and legal or financial trouble (wage garnishments, salary-
backed loans). Union membership and health data are special categories under
Article 9 GDPR. Any design that moves payslips to a server, including a
self-hosted instance used by more than one person, has to treat them as such.

### Legal basis and roles

In on-device mode the project and its maintainer process no payslip data at
all: the software runs on the user's device under the user's control, and a
person processing their own data for personal purposes is outside the GDPR
(household exemption). This is the strongest possible position and the main
reason the on-device decision is right. See
[ADR 0003](../adr/0003-on-device-processing-by-default.md).

The website that serves the static app still processes IP addresses in server
logs; the hosting provider's terms and a short privacy notice cover this. If a
server mode is ever offered, the operator of each instance is the controller,
needs a legal basis (explicit consent for Article 9 data), and should perform a
data protection impact assessment. The project would document this for
self-hosters; it would not take on that role itself.

### Minimization

Do not store the worker's codice fiscale, name, employee number, address or
IBAN by default. The codice fiscale is useful for one thing: detecting that
an imported payslip belongs to a different person. That check can be done with
a salted hash kept locally, never displaying or exporting the value. The
original PDF is discarded after extraction unless the user opts in to keep it.

### Retention and deletion

Data stays as long as the user keeps it, because the user is the only holder.
The CLI and the app provide one obvious action to delete everything, which
removes the data folder or clears IndexedDB, and individual deletion per
document. Exports are explicit user actions.

### Encryption at rest

The default relies on operating system disk encryption, which is what most
users already have. An optional passphrase-encrypted data folder (age format
in the CLI, Web Crypto AES-GCM with a key derived from a passphrase in the
browser) is a could. Encryption with a forgotten passphrase means permanent
data loss, so it must stay opt-in with a clear warning.

### Logs

Logs never contain values, names, descriptions or file names, only event types,
counts, durations, and template identifiers. A debug mode that prints values
must be explicitly enabled and print a warning.

### The repository is a leak vector

This is the risk most likely to actually happen. Users will attach real
payslips or screenshots to issues to report a wrong extraction. Mitigations:

- Issue forms with a mandatory checkbox stating that no real payslip,
  screenshot or personal data is attached, and a pointer to the anonymizer.
- A documented maintainer procedure: delete the comment, which does not remove
  edit history, and ask GitHub support to purge it if it contained personal data.
- A .gitignore that excludes PDF and image files everywhere except an explicit
  synthetic fixtures folder, and a pre-commit hook doing the same check.
- A CI PII scanner over fixtures: valid codici fiscali (with checksum), IBANs
  (with checksum), and Italian phone numbers and email addresses fail the build.
- The maintainer's own payslips and their ground truth live in a git-ignored
  local folder; the corresponding tests run only when that folder exists.
- Screenshots in documentation use synthetic data only.

### Contributing templates without personal data

The anonymizer command turns a real payslip into a redacted text layout using
an allowlist, not a denylist: it keeps only tokens that match a dictionary of
known payroll labels and vendor codes, and replaces every other token with a
type-preserving placeholder (digits become random digits in the same format,
words become a fixed marker). Positions are preserved, so the redacted layout
is enough to write and test a template. Amounts are then regenerated
synthetically so that the totals still balance. The contributor reviews the
redacted output before submitting it, and the CI PII scanner checks it again.
A denylist approach (removing what looks like a name or a codice fiscale)
always misses something, such as a name in a free-text note.

Public sample payslips found online are useful for development but must not be
committed: they are copyrighted documents and may contain real data. They are
used locally, and only redacted layouts derived from them enter the repository.

## 6. Accuracy and trust

### Test data set

Three layers. First, synthetic text layout fixtures in the repository, built
from canonical data and a layout description, cheap to create in volume and
safe to publish, covering every template branch and edge case (13th month,
conguaglio, arrears, negative items, multi-payslip documents). Second,
redacted layouts from real documents, produced by the anonymizer, which
preserve real layout quirks. Third, private ground truth: the maintainer's own
payslips with hand-verified expected values, kept outside the repository and
run locally before each release.

### Metrics

Per canonical field: exact match rate (money must match to the cent; anything
else is an error), and precision and recall for line item mapping. Per
document: the share of payslips where every must field is correct, and the
share passing all arithmetic cross-checks. Per template: the same numbers, so a
template regression is attributable. These are reported by a test command, and
CI fails on any regression on fixtures.

### Cross-checks that produce confidence

- Sum of earnings items equals printed gross; sum of deductions equals printed
  total deductions; gross minus deductions plus or minus rounding equals net.
- INPS employee contribution approximately equals the INPS taxable base times
  the expected rate (9.19 percent in most cases, with known exceptions such as
  apprentices and the additional 1 percent above the annual threshold).
- Monthly TFR accrual is consistent with the period's eligible pay divided by
  13.5, minus the 0.5 percent contribution where shown.
- Leave balances: previous remaining plus accrued minus used equals current
  remaining, across consecutive months.
- Period continuity: no duplicate or missing months without explanation.

### Showing confidence

Each value in the UI carries its status (verified, extracted, inferred,
missing, corrected) with a small visual marker, and clicking a value shows the
region of the original page it came from while the PDF is still in memory.
Aggregates are labelled with how many of their inputs are verified. A failed
cross-check is shown as "these numbers do not add up, please check", never
hidden. Percent confidence scores are avoided: they look precise and are not.

### Errors

Parsing failures produce a clear, actionable message and a pointer to the
anonymizer. A partially extracted payslip is still imported, with gaps marked,
rather than rejected.

## 7. Legal and product aspects

A plain disclaimer, shown at first use and in the footer, not buried: cedolo
explains what a payslip says, it does not provide tax, legal or labour advice,
and it can be wrong; for decisions, consult the employer's payroll office, a
CAF, a patronato or union office, or a consulente del lavoro. In Italy payroll
administration is a regulated activity (Law 12/1979); explaining a document is
not, but the product must not drift into preparing or certifying payroll.

The main interpretation risk is change detection phrased as accusation. A
"possible anomaly" in March is usually the municipal surcharge advance starting,
not an error. Alerts are worded as "changes worth a look", each with the most
likely benign explanation and a confidence level, and seasonal effects are
explained before anything is flagged.

Recomputing IRPEF is excluded for the same reason: tax credits, the 2025
redesign of the cuneo fiscale reduction, regional rates and year-end
adjustments make a correct recomputation a full-time job, and an incorrect one
tells users they were underpaid when they were not.

Liability: the AGPL disclaims warranty and liability to the extent permitted
by law; consumer protection law may still apply to a service the maintainer
hosts. The on-device design, the absence of accounts and payments, and a
clearly non-commercial project reduce exposure further. This is not legal
advice; a short review by a lawyer before the public launch is worth the cost.

The name "cedolo" should be checked against existing trademarks and domains
before the repository goes public. Vendor names (Zucchetti, TeamSystem and
others) may be used descriptively to name templates, without logos and
without implying endorsement.

### License

Recommendation: AGPL-3.0-or-later for the whole repository, including
templates. See [ADR 0002](../adr/0002-license-agpl-3.0.md). It addresses
the stated risk directly: anyone who offers a modified cedolo to users over a
network must publish their changes, including template improvements, which are
the project's most valuable asset. GPL would not cover a hosted server mode;
MIT and Apache-2.0 allow a closed hosted fork. MPL-2.0 is a lighter middle
ground but has no network clause. Contributions under the Developer Certificate
of Origin (sign-off), not a CLA, which keeps the barrier low.

What would make it wrong: if the goal shifts to having payroll vendors, HR
platforms or banks embed cedolo in their products. AGPL deters that. Since that
is not a goal, the risk is accepted. Relicensing later requires consent from
all contributors, so the choice should be settled before the first external
contribution. The repository template does not bundle the AGPL text; the
canonical text must be taken verbatim from the GNU project.

## 8. Architecture

```
  +---------------------------------------------------------------+
  |  Frontends (replaceable)                                       |
  |  v0.1 CLI          v0.2 web app (browser)     later: server   |
  +---------------+-----------------+-------------------+---------+
                  |                 |                   |
  +---------------+-----------------+-------------------+---------+
  |  Text layer: PDF to text layout (pages, tokens, bounding box) |
  |  CLI: pure-Go reader or pdftotext   Browser: pdf.js           |
  +-------------------------------+-------------------------------+
                                  |
  +-------------------------------+-------------------------------+
  |  Go core (pure, no I/O)                                        |
  |  detect vendor | apply template or fallback | normalize items |
  |  validate (cross-checks) | merge corrections | insights        |
  |  templates and taxonomy embedded as data                       |
  +-------------------------------+-------------------------------+
                                  |
  +-------------------------------+-------------------------------+
  |  Storage adapter                                               |
  |  CLI: JSON files in a local folder   Browser: IndexedDB       |
  |  later server: database per instance                          |
  +---------------------------------------------------------------+
```

Flow of one document, in local mode:

1. The user selects one or more PDF files. The file is read into memory on the
   device; nothing is uploaded.
2. The text layer extracts pages and positioned tokens. If there is no text
   layer (a scan), the user is told that scans are not supported yet.
3. The core computes the content hash and checks for duplicates, splits the
   document into payslips, detects the vendor and applies the matching
   template, or the fallback for unknown layouts.
4. Line items are mapped to canonical keys; header, totals, TFR and leave
   fields are filled; every value gets provenance and a status.
5. Cross-checks run and set statuses to verified or flag inconsistencies.
6. The user reviews uncertain fields, with the source region highlighted, and
   confirms or corrects them. Corrections are stored separately.
7. The ExtractionRun, payslips and corrections are saved by the storage
   adapter; the PDF is discarded unless the user chose to keep it.
8. The insights module computes time series, deduction breakdowns, period
   comparisons and change notices from the effective values, and the
   dashboard or HTML report renders them.

What changes in self-hosted mode: step 1 becomes an upload over HTTPS to an
instance the user or their organization runs; the text layer runs on the
server (pdftotext or a Go reader); storage becomes a database with
per-user separation and encryption at rest; authentication, rate limiting,
backups and a privacy notice become required. The core, templates and taxonomy
are identical. Because a self-hosted instance holds Article 9 data for several
people, it is a deliberate later phase, not a configuration flag.

## 9. Roadmap

Effort estimates assume about 5 hours a week.

Phase 0, foundation (2 to 3 weeks). Initialize the repository from the
template, accept or amend ADRs 0002 to 0005, set up the Go module, CI, the
.gitignore guard and the PII scanner, and run the PDF spike. Exit criteria: CI
green on main; ADRs accepted; the spike recovers text with positions from all
of the maintainer's payslips, or a fallback reader is chosen.

Phase 1, v0.1 "for me" (8 to 10 weeks). CLI import, one vendor template,
canonical taxonomy for that vendor's items, cross-checks, JSON storage,
CSV and JSON export, static HTML report. Exit criteria: on the maintainer's
private set (at least 12 consecutive payslips including a 13th month and a
December conguaglio), 100 percent of must fields correct to the cent against
hand-verified ground truth; all cross-checks pass on at least 95 percent of
payslips and every failure is explained; synthetic fixtures cover every
template branch; report generation under 2 seconds for 24 payslips.

Phase 2, v0.2 public preview (12 to 16 weeks). Browser app with the Go core in
WebAssembly, EN/IT, correction workflow, glossary, anonymizer and contribution
guide, two more vendor templates from public samples, declarative template
format, repository made public. Exit criteria: three non-technical testers go
from PDF to dashboard without help; an automated browser test proves no
network request is made after the page loads; the Content Security Policy
forbids outbound connections; the WebAssembly bundle loads in under 3 seconds
on a mid-range phone; PII scanner and issue forms in place before the
repository becomes public.

Phase 3, v0.3 community (open-ended). Change detection with seasonal
explanations, supplementary pension fund and seniority fields, optional
encryption, experimental local OCR. Exit criteria: at least five vendor
templates, at least two contributed externally; change notices measured on
fixtures with a documented false-positive rate.

Phase 4, later and optional. Self-hosted server mode, CU reconciliation,
minimum-pay tables for one or two widely used CCNLs maintained by the
community. Each needs its own ADR before work starts.

## 10. Risks and open questions

Bus factor and pace. One maintainer at 5 hours a week means any month of
absence stalls the project. Mitigation: small scope, good contributor docs,
templates as data so contributors need no Go.

Template drift. Vendors change layouts, and employers configure codes
differently. Fingerprinting and fixtures per template version catch it, but
template maintenance is the long-term cost of this design.

Data scarcity. Without a shared data set, coverage grows only as fast as
people contribute redacted layouts. The anonymizer is therefore not a
nice-to-have; it is the growth engine and belongs in v0.2.

False trust. A polished dashboard over a wrong number is worse than no tool.
Verified-only aggregates and visible cross-check failures are the defence.

Browser privacy claims must be verifiable. Browser extensions can read page
content and pdf.js workers must be self-hosted, not loaded from a CDN.
Publishing the CSP and offering an offline download makes the claim checkable.

Go in the browser. WebAssembly size and startup time on phones are unproven
for this workload; the phase 2 exit criteria measure it explicitly.

Name and trademark, vendor relations, and the copyright status of public sample
payslips all need a check before the repository is public.

Prior art exists and should be reviewed before phase 2, both to learn from it
and to avoid duplicating effort: for example MCP-Buste-Paga
(github.com/morettimarco/MCP-Buste-Paga), which parses INAZ payslips into a
local SQLite database for AI clients. Its license and approach have not been
checked yet.

Open questions for the maintainer:

- Whether templates should stay under AGPL-3.0-or-later (ADR 0002) or get a
  separate data license.
- Confirm the Go core boundary and PDF strategy (ADR 0004) after the spike.
- Which payroll software produces the maintainer's payslips, and which public
  samples are available, to choose templates two and three.
- Static web app or local binary for v0.2 (section 4), to be decided after v0.1.
- Whether the public app should be hosted on GitHub Pages or a project domain.
