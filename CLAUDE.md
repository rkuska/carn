# càrn

càrn is a tool that provides additional functionality over local Claude and
Codex session archives.

## Architecture

Code is split by ownership under `internal/`.

- `internal/app/` — composition root and TUI layer only.
- `internal/conversation/` — shared conversation/session/message/plan types
  and presentation helpers. Shared types live here; godoc is authoritative.
- `internal/source/claude/` — Claude raw-source scan, parse, grouping,
  projection, source-side import analysis, and incremental rebuild
  resolution.
- `internal/source/codex/` — Codex raw-source scan, parse, grouping,
  projection, source-side import analysis, and incremental rebuild
  resolution.
- `internal/source/` — provider-neutral backend contract shared by app,
  archive, and canonical.
- `internal/config/` — config-file path resolution, defaults, template
  rendering, parsing, and config state (`missing`, `loaded`, `invalid`).
- `internal/canonical/` — SQLite Canonical Store, rebuild/load, transcript
  codecs, transcript persistence, incremental rebuilds, FTS deep search.
- `internal/archive/` — sync, import analysis, pipeline orchestration.
- `internal/stats/` — stats computation: overview aggregates, activity
  heatmaps, histograms, tool breakdowns, token trends, streaks,
  performance scorecard and lanes, trends, transcript-sequence metrics.

Defaults: sources `~/.claude/projects/`, `~/.codex/sessions/`; archive
`~/.local/share/carn/`. Runtime paths come from the user config file
resolved by `internal/config`.

Data pipeline: raw sessions → `archive.Pipeline` → `source.Backend.Scan` /
`ResolveIncremental` → `canonical.Store.RebuildAll` →
`canonical.Store.List` / `Load` / `DeepSearch` → app browser/viewer/stats.

Use `VOCABULARY.md` as the source of truth for naming packages, files, and
refactor targets. If a needed term is missing, add it there first.

## How to write code

This is a custom tool not published as a package, so most functionality can
stay private.

Keep `internal/app/` focused on app wiring and the TUI. When a file mixes
backend responsibilities or approaches the complexity cap, split it by
responsibility and move backend code into vocabulary-named packages under
`internal/`. Keep files small and group by functionality.

When handling objects or creating signatures, follow "Parse, don't validate".

Prefer streaming aggregation: walk records directly into the final
accumulator instead of materialising a per-item slice and re-folding it.
*Why:* `ComputeTurnTokenMetrics` dropped from 640 KB / 1002 allocs per op
to 1 KB / 3 allocs at 1000 sessions (`e5dc05b`).

### Dependencies

When adding new dependencies, always use the latest major version. Check
git tags or the repository for a `vX` folder (where X is a number).

### Errors

- Always wrap errors with `fmt.Errorf("%w", ...)` when forwarding (use
  `errors.New` when there are no arguments).
- Wrap with the format `"methodThatFailed: %w"`. If a method calls the same
  function multiple times, differentiate with context:
  `"methodThatFailed_context: %w"`.
- Logging an error counts as handling it: don't log and forward.
- Don't capitalize error messages.
- Log errors that originate in `defer`, don't propagate them.
- When wrapping an error with a classification sentinel (e.g.
  `src.MarkMalformedRawData`), gate the wrap on `errors.Is` for the narrow
  error types that actually match. Propagate every other error unchanged.
  *Why:* wrapping every `os.Open` failure as "malformed raw data" hid
  permission and I/O errors and dropped them via the malformed-skip path
  (`df712e3`, `e436c5a`). Only `fs.ErrNotExist` means "the raw file is
  gone".
- Use a distinct package-local sentinel for *benign* edge cases (empty
  files, missing metadata). Do not reuse a corruption sentinel just because
  both lead to "skip this item". *Why:* claude sessions with no metadata
  lines were flagged as malformed and counted toward the user-facing
  notification (`fa997b9`).

### Logging

- Don't use `fmt.Print*`.
- Use zerolog with `zerolog.Ctx`.
- Use `Msgf` with enough detail.
- Use fields (`.Str` etc.) only for things we expect to search for in
  structured logging.
- Any log line a user-facing notification points users at ("check logs",
  "see logs for details") must be at `Warn` or `Error`. The default zerolog
  level is `Info`, so `Debug` lines are silent in the shipped log. *Why:*
  the malformed-items notification pointed users at a log file where
  per-file detail was at `Debug` (`fa997b9`).

### Pointers

- Only use pointers for shared objects (a `Client`, shared state).
- Pass by value for schemas like value objects or entities.
- If the object is meant to be used as a pointer, all methods should have
  pointer receivers. And vice versa. The only exception is marshal
  functions.

### Interfaces

- Define interfaces in the module where they are needed.

### Concurrency

- Use `errgroup.WithContext(ctx)` for goroutine management.
- Use `semaphore.NewWeighted()` for rate limiting.
- Always cancel contexts when done:
  `ctx, cancel := context.WithCancel(ctx); defer cancel()`.
- Write channels so `for item := range channel` can always finish (drain on
  error paths).
- Use `select` with context cancellation when sending to a channel.
- Ensure goroutines aren't leaking.

### Generics

- When writing the same code for different types, reach for generics.
- When a method accepts `any`, check whether generics can restore type
  safety.
- Prefer `any` over `interface{}`.
- Generics are defined on functions without receivers, or on both the
  receiver and method when used on methods.

### Iterators

- Use range-over-func for custom iterators.
- Consider them when handling large data where pipelining helps.

### TUI development

- For layout changes, verify height/width arithmetic accounts for all
  decorations. Document: `total = top_decoration + content +
  bottom_decoration = terminal_dimension`.
- When a new UI element replaces an existing one, remove or disable the old
  element. Verify no visual duplication.
- Apply styling changes to the exact scope described — "make X white" means
  only X, not surrounding text.
- Use `lipgloss.Width()` (not `len()`) for width calculations — it accounts
  for ANSI escape codes and Unicode.
- After implementing state transitions, test the full user flow manually
  (start app → sync → navigate → view → back).
- When integrating with a templating engine (Glamour, `text/template`),
  verify the exact context keys the engine passes, and assert the rendered
  output does not contain a missing-value sentinel (e.g. `<no value>`) in a
  golden test. *Why:* `{{.Text}}` produced `<no value>` because Glamour
  passes `text` lowercase (`e436c5a`).
- For TUI render paths on the hot bench list (`BenchmarkStats*Render`,
  `BenchmarkViewerRenderContent`), cache style and primitive construction
  outside the render call, and rerun the stats benches after changes.
  *Why:* chart styles and heatmap cells were rebuilt on every call; only
  rerunning the benches sequentially surfaced the churn (`391a2a1`).

### Session data

- Before filtering or removing records from the pipeline, scan real data to
  verify what information would be lost. Never assume fields are empty
  without checking.

### Format drift and known schema extras

Format drift workflow lives in `DRIFT_WORKFLOW.md`. Trigger it when the
task mentions format drift, schema drift, unknown fields, unknown record
types, `carn.log` drift warnings, or `known_schema_extras`.

### Review and planning workflow

- When asked to review code, present all findings first. Wait for
  confirmation before making changes.
- When a fix reintroduces a previous bug, stop and find the root cause
  instead of adding workarounds.
- When introducing a sentinel/classifier that drives a user-visible
  notification, dry-run the change against real session data before
  calling it done: trigger the notification path, read the actual log
  output, and confirm only matching items appear. *Why:* two follow-up
  fixes (`fa997b9`, `df712e3`) corrected the initial `MarkMalformedRawData`
  rollout because the original change was not exercised against real
  sessions.

## Testing

Follow TDD: stubs first, tests next (see guidelines below — scenarios are
the most important), implementation last. Write meaningful tests. When
refactoring, adjust tests to the expected behavior first; if a test fails
during refactor, assume a bug in the refactor rather than a wrong test.

Common commands:

```bash
golangci-lint run ./...        # lint
go run ./cmd/testsuite          # canonical suite with coverage gating
go run ./cmd/testsuite -update  # refresh committed coverage baseline
go test -race ./...             # race detector
go fix ./...                    # latest Go guidelines
```

Enable the repo hook with:

```bash
git config core.hooksPath .githooks
```

The committed `pre-commit` hook runs `go fix ./...`, `golangci-lint run
./...`, and `go run ./cmd/testsuite` in that order. It requires a clean
working tree because `go fix ./...` may rewrite tracked files. The hook
stages those changes before lint and coverage run.

Complexity is guarded by `TestFileComplexityGuard` and
`TestModuleComplexityGuard` across `internal/**/*.go`. Rules, limits, and
split order live in `COMPLEXITY_GUIDE.md`. Refresh the baseline with:

```bash
go test ./internal/app -run TestComplexityBaselineDocument -count=1 -update
```

Benchmarks live next to the package they measure. Full bench commands and
current baseline are in `PERF_BASELINE.md` — run them one at a time, never
in parallel. For optimisation work, start with a bench and a CPU profile:

```bash
go test -run '^$' -bench '<name>' -cpuprofile /tmp/carn.cpu <pkg>
go tool pprof -top /tmp/carn.cpu
```

Use [goperf.dev](https://goperf.dev/) for implementation ideas once the
profile identifies the hot path. Update `PERF_BASELINE.md` when benchmark
commands or results change meaningfully.

### Unit tests

Unit tests cover small (but complex) functions, private or public.

- Don't mock; refactor the tested function instead if mocking seems
  necessary.
- Write table-driven tests.
- Avoid `if`s in tests (especially `if testCase.ExpectError`) — split into
  separate tests.
- Use `t.Run` and `t.Parallel`.

### Scenarios

Scenarios live under `scenarios/` and test overall architecture by
exercising the whole business flow.

- Prefer real dependencies over mocks (exception: LLMs).
- Test via the outermost layer.
- Tests run sequentially, and each is atomic (no shared state between
  tests).

### Test data and fixtures

- Tests must not read pre-existing files from `~/.claude/projects`,
  `~/.codex/sessions`, `$HOME`, or any other host-specific location.
- Use committed sanitised fixtures under `testdata/` for raw session
  coverage.
- `t.TempDir()` is allowed for scratch space, but source data copied into
  it must come from the repository.
- Fixtures and goldens must not contain author-specific usernames, local
  project names, private transcript content, or machine-specific temp
  paths.
- When snapshotting UI output, normalise unstable paths so goldens stay
  portable.
- Write object generators in `scenarios/helpers/generators.go`.

### Shell scripts

- `.githooks/*` scripts must run under bash 3.2 (macOS default) and stay
  shellcheck-clean: no SC2207, every expansion quoted. Build sorted arrays
  with a `while IFS= read -r` loop, not `arr=($(... | sort))`. *Why:*
  `GOPLS_LINT` shipped an orphan apostrophe and an unquoted command
  substitution (`e436c5a`).

## Commit messages

Pattern: `type(scope): message` where type is `fix`, `feat`, `chore`,
`docs`, `refactor`, or `style`; scope marks what is affected; message is a
short lowercased description.

Keep subject and body lines under 72 characters. Always write a body
explaining what, how, and why in continuous human-readable text. For fixes
include the error message being fixed. No first-person speech. Re-read the
actual git diff before writing — the message must describe what changed,
not what was planned.

### Commit command

Pipe the message to git via a single-quoted heredoc on stdin — one
command, real newlines, nothing to clean up:

```bash
git commit -F - <<'EOF'
type(scope): subject line

Body paragraph explaining what, how, and why.
EOF
```

Use `git commit --amend -F -` the same way for amends. The single-quoted
`'EOF'` delimiter disables shell expansion, so backticks, `$`, and
backslashes in the message are preserved literally. Do not use `-m
"...\n..."` with escaped `\n` sequences — those store backslashes
literally and produce a one-line body.
