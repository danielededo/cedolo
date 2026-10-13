# License Guide

How to pick a license for a new project based on this template. The candidates
live in [`licenses/`](../licenses/): copy the right one to the repository root as
`LICENSE`, fill in `<YEAR>` and `<COPYRIGHT_HOLDER>`, and delete the `licenses/`
directory.

> This is practical guidance for personal projects, not legal advice.

## Decision by scenario

### 1. Public project I want to open source

Pick a real open-source license, because a public repo **without** a license is *not*
open source: by default others have no right to use, modify, or redistribute
your code.

- **MIT** (`licenses/MIT.txt`): the default choice for most personal projects.
  Short, universally understood, maximally permissive: anyone can do anything as
  long as they keep the copyright notice. Choose it when you just want the code
  to be useful and don't care what people do with it.
- **Apache-2.0** (`licenses/APACHE-2.0.txt`): also permissive, but longer and
  more "corporate-grade". It adds an **explicit patent grant** (and a patent
  retaliation clause) and requires stating changes made to the files. Choose it
  when the project could plausibly involve patentable techniques, when you expect
  corporate adoption or contributions, or when you want the extra legal clarity.

Rule of thumb: MIT for a small library, tool, or experiment; Apache-2.0 for
something bigger that companies might build on. If you ever want a copyleft license
(GPL family), that's a deliberate philosophical choice this template doesn't
bundle; fetch the text from [choosealicense.com](https://choosealicense.com).

For Apache-2.0, optionally add a `NOTICE` file with the project name and
copyright line; downstream users must preserve it.

### 2. Private / personal project

If the repo is private and will stay private, a license mostly documents intent:

- **PROPRIETARY** (`licenses/PROPRIETARY.txt`): an "all rights reserved" notice.
  Use it for client work or anything you explicitly do not want reused. For
  client projects, remember the contract with the client governs ownership; this
  file is just the in-repo marker.
- **No license file at all** is also acceptable for throwaway personal repos:
  legally, "all rights reserved" is already the default. Add the PROPRIETARY
  notice anyway if other people (collaborators, clients) will ever see the code;
  it removes ambiguity.

### 3. Private project that might become public later

Plan for the flip from day one, because relicensing *later* is easy only if you
stay the sole author:

- Start with **PROPRIETARY** (or no license) while private.
- **Keep the copyright clean**: if anyone else contributes while the repo is
  private, agree explicitly (even just in writing in an issue) that you may
  relicense their contribution, or you'll need their consent at flip time.
- Avoid pasting in code under incompatible licenses (especially copyleft); it
  constrains what you can relicense to.
- When you make it public, swap `LICENSE` for MIT or Apache-2.0 (see scenario 1)
  in the same commit that flips visibility, so there is no public window where
  the repo has no license.

## Quick comparison

| | MIT | Apache-2.0 | PROPRIETARY |
| --- | --- | --- | --- |
| Others can use/modify/sell | yes | yes | no |
| Patent grant | implicit at best | explicit | n/a |
| Must state changes | no | yes | n/a |
| Complexity | very low | medium | very low |
| Typical use | small OSS projects | OSS with corporate reach | private/client work |

## Checklist when applying a license

- [ ] Copy the chosen file to `/LICENSE` (keep the `.txt` content, the name
      `LICENSE` is the GitHub convention)
- [ ] Replace `<YEAR>` and `<COPYRIGHT_HOLDER>`
- [ ] For Apache-2.0: consider adding a `NOTICE` file
- [ ] Mention the license in the README (a badge or a "License" section)
- [ ] Delete the `licenses/` directory and this guide from the new project
