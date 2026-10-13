---
name: new-adr
description: Create a new Architecture Decision Record from the project template, numbered after the latest one, and add it to the ADR index. Use when a significant technical decision needs to be recorded.
---

# New ADR

Follow docs/adr/README.md; this skill only sequences the steps.

1. Ask for the decision title if it was not given. Keep it short and
   affirmative (example: "Use PostgreSQL as the primary database").
2. Find the highest NNNN among docs/adr/NNNN-*.md and take the next number,
   zero-padded to four digits.
3. Copy docs/adr/0000-template.md to docs/adr/NNNN-short-kebab-title.md.
   Fill in the title, status (Proposed unless told Accepted), today's date,
   and the deciders. Fill Context, Decision, Options considered, and
   Consequences from the conversation; ask for what is missing rather than
   inventing it. Keep it to one page.
4. Add a row for the new ADR to the index table in docs/adr/README.md.
5. If the new ADR supersedes an older one, set the old one's status to
   "Superseded by [ADR-NNNN](NNNN-short-kebab-title.md)" and update its
   index row. Never change the old ADR's substance.
6. Commit as "docs: add ADR NNNN on <topic>" on a topic branch and open a
   pull request, unless the change belongs to a pull request already in
   progress.
