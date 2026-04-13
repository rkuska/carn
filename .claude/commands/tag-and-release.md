# Tag and Release

Orchestrate a carn release with human-readable release notes.

Walk through each step below interactively. Do not skip steps or batch
multiple confirmations. Wait for user input where indicated.

## Step 0 — Preflight checks

Before starting, verify the working tree is ready for a release:

1. Run `git status` — the working tree must be clean (no staged,
   unstaged, or untracked changes). If dirty, stop and ask the user
   to commit or stash before continuing.
2. Run `git fetch origin` and compare `git rev-parse HEAD` with
   `git rev-parse @{upstream}`. If HEAD is behind the remote, stop
   and ask the user to pull. If ahead, warn that unpushed commits
   exist and confirm they should be included in the release.

## Step 1 — Gather context

Run these commands to understand what changed since the last release.

Find the last tag:

```bash
LAST_TAG=$(git describe --tags --abbrev=0)
```

Then list commits since that tag with full bodies:

```bash
git log "${LAST_TAG}..HEAD" --format='%h %s%n%b---'
```

Categorize commits by type (`feat`, `fix`, `refactor`, `perf`, `chore`,
`docs`, `style`) and by scope. Count features and fixes.

## Step 2 — Determine version

Present the user with a short summary of what changed (number of features,
fixes, improvements). Suggest a version number:

- Any `feat` commits → suggest **minor** bump (e.g. v0.1.0 → v0.2.0)
- Only `fix` commits → suggest **patch** bump (e.g. v0.1.0 → v0.1.1)
- Any commit body containing `BREAKING CHANGE` → suggest **major** bump

**Ask the user** what version this should be. Accept their explicit version
or their confirmation of the suggestion.

## Step 3 — Update project documents

**Skip this step for patch releases.** Fixes rarely introduce new types,
change package structure, or move performance baselines. Proceed directly
to Step 4.

For **minor** or **major** releases, update the project's baseline documents
so they reflect the code being tagged. Work through each document below
and make the necessary edits.

### PERF_BASELINE.md

Run the full benchmark suite from `CLAUDE.md` — one command at a time,
sequentially. Replace the results table with fresh numbers and update the
"Refreshed on" date. If any benchmark moved more than 5% in either
direction, update the Delta Summary section comparing old vs. new values.
Remove delta entries that are no longer relevant.

### CLAUDE.md

Check the **Package map** section against the actual files on disk:

1. Verify every file path listed still exists. Remove stale entries.
2. Glob `internal/**/*.go` (excluding `*_test.go`) and check for files
   that are not mentioned in the package map. Add missing entries under
   the appropriate group.
3. Check the **Core types** list against `internal/conversation/*.go`
   exports. Add new types and remove deleted ones.

### VOCABULARY.md

Review for completeness against changes since the last tag:

1. Check for new exported types, concepts, or UI elements introduced
   since `$LAST_TAG`.
2. Add any missing terms to the appropriate section.
3. Remove terms for features or concepts that no longer exist.

### Commit

If any documents changed, stage and commit them before proceeding:

```bash
TMPFILE=$(mktemp)
cat > "$TMPFILE" << EOF
docs: refresh baseline documents for release

Update PERF_BASELINE.md, CLAUDE.md, and VOCABULARY.md to reflect
the current state of the codebase ahead of tagging.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF

git add PERF_BASELINE.md CLAUDE.md VOCABULARY.md
git commit -F "$TMPFILE"
rm "$TMPFILE"
```

If nothing changed, skip the commit and continue.

## Step 4 — Draft release notes

Write human-readable release notes describing the **final shipped state**,
not the development journey.

### Grouping rules

- Fixes to a **new feature introduced in this release** are absorbed into
  that feature's description. For example, if stats is new in this release,
  stats chart color fixes belong in the stats feature entry, not in a
  separate Fixes section.
- The **Fixes** section only contains fixes to functionality that existed
  **before this release**.
- The **Improvements** section only covers changes to pre-existing behavior.

### Format

```markdown
## What's new in v{version}

<1-3 paragraphs of prose summary describing the release from a user
perspective. Focus on what changed and why it matters. Do not list commits.>

### Features

- **Feature name** — description of what it does and why it matters.
  Include relevant details that were refined during development.

### Fixes

- Description of what was broken and how it now works.

### Improvements

- Description of the improvement and its user-visible effect.

### Breaking Changes

- What changed and what users need to do.
```

Rules:
- Use `##` (H2) for the top heading (GitHub prepends the release name as H1)
- Bold lead-in phrases for each entry
- No commit hashes, author names, or PR numbers
- Omit empty sections entirely

**Present the full draft** and ask the user to review. Wait for them to say
"looks good" or request changes. Iterate until approved.

## Step 5 — Visual content

**Ask the user**: "Do you want to add screenshots, ASCII art, or neither
for this release?"

### If screenshots

- Ask for file paths
- Create `releases/assets/v{version}/` directory
- Copy files there
- Insert image references in the release notes using:
  ```
  ![description](https://raw.githubusercontent.com/rkuska/carn/v{version}/releases/assets/v{version}/filename.png)
  ```

### If ASCII art

Ask the user to provide the ASCII art (e.g. from https://monosketch.io/)
or offer to create a simple text representation. Inline it in a fenced
code block in the release notes.

### If neither

Proceed with prose only.

## Step 6 — Write and commit release notes

After the user approves the final draft:

1. Create `releases/` directory if it does not exist
2. Write the release notes to `releases/v{version}.md`
3. If `CHANGELOG.md` exists, prepend the new release content followed by
   a `---` separator before the existing content. If it does not exist,
   create it with just this release's content.
4. Copy any screenshot assets to `releases/assets/v{version}/`
5. Stage all release note files and commit:

Write the commit message to a temp file using `git commit -F` (project
convention — do not use `-m` for multi-line messages). Replace all
placeholders with actual values before running:

- `VERSION` — the chosen version (e.g. `v0.2.0`)
- `SCOPE` — a brief summary of the release scope (e.g. `stats dashboard,
  codex tokens, and format drift detection`)

```bash
TMPFILE=$(mktemp)
cat > "$TMPFILE" << EOF
docs(release): add ${VERSION} release notes

Add human-readable release notes for ${VERSION} covering
${SCOPE}. The release workflow reads this file and uses it
as the GitHub release body.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF

git add releases/${VERSION}.md CHANGELOG.md
# Also add releases/assets/${VERSION}/ if screenshots exist
git commit -F "$TMPFILE"
rm "$TMPFILE"
```

## Step 7 — Tag and push

First check that the tag does not already exist:

```bash
git tag -l "${VERSION}"
```

If the tag already exists, stop and warn the user. Do not overwrite
existing tags.

**Ask the user**: "Ready to tag ${VERSION} and push? This will trigger the
GitHub release workflow which builds binaries and publishes to Homebrew."

After confirmation:

```bash
git push origin HEAD
git tag "${VERSION}"
git push origin "${VERSION}"
```

Report the GitHub Actions workflow URL:
https://github.com/rkuska/carn/actions/workflows/release.yml
