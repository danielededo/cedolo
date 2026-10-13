---
name: init-from-template
description: Turn a fresh copy of the scaffold template into a real project. Replaces placeholders, applies the chosen license, rewrites the README from the template, removes unwanted optional files, resets the changelog, and deletes itself. Use once, right after creating a repository from the template.
---

# Initialize a project from the template

Follow the "How to use it as a template" section of README.md; this skill
only sequences the steps and the questions.

1. Collect the inputs, asking for anything not already known: project name,
   GitHub owner handle, security contact email, code of conduct contact
   email (may be the same), copyright holder, and whether the repository is
   public, private, or private for now with a possible public future.
2. Choose the license from the three scenarios in docs/LICENSE-GUIDE.md and
   confirm the choice with the user. Copy the matching file from licenses/
   to LICENSE at the root, fill in <YEAR> and <COPYRIGHT_HOLDER>, remove
   the template-user note at the bottom of the proprietary notice if that
   one was chosen, then delete licenses/ and docs/LICENSE-GUIDE.md.
3. Replace <PROJECT_NAME>, <OWNER_HANDLE>, <SECURITY_CONTACT_EMAIL>, and
   <CONTACT_EMAIL> across the repository. Leave every [TO BE FILLED IN]
   marker in place: those need project knowledge the user will add later.
4. Replace README.md with the skeleton from docs/README-TEMPLATE.md
   (everything below the horizontal rule), fill the one-paragraph
   description from the conversation, then delete docs/README-TEMPLATE.md.
5. Ask which optional files to keep: CODE_OF_CONDUCT.md (drop for private
   or strictly personal repositories), .github/CODEOWNERS (drop for a single
   maintainer), docs/adr/ (keep only if decisions will be recorded; if kept,
   delete 0001-trunk-based-development.md, which is the template's own
   decision, and reset the index). Remove what is not wanted and update the
   structure tree in README.md if one is kept there.
6. Rewrite AGENTS.md for the new project using the skeleton in
   docs/AI-AGENTS.md: real commands, real layout, real boundaries. Remove
   the scaffold-specific boundaries (canonical texts, language-agnostic rule).
7. Reset CHANGELOG.md to a single empty Unreleased section and fix the
   links at the bottom with the new owner and project name.
8. Delete this skill, both .agents/skills/init-from-template/ and the
   .claude/skills/init-from-template symlink, and keep new-adr and release
   only if the user wants them.
9. Point the user to docs/REPO-SETTINGS.md for the GitHub settings that
   files cannot carry, and commit everything as
   "chore: initialize project from scaffold template".
