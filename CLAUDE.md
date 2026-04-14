# càrn

càrn is a tool that provides additional functionality over local Claude and
Codex session archives.


## Architecture

Code is split by ownership under `internal/`.

- `internal/app/` is the composition root and TUI layer only.
- `internal/conversation/` owns shared conversation/session/message/plan types
  and presentation helpers.
- `internal/source/claude/` owns Claude raw-source scan, parse, grouping,
  projection, source-side import analysis, and targeted incremental rebuild
  resolution.
- `internal/source/codex/` owns Codex raw-source scan, parse, grouping,
  projection, source-side import analysis, and targeted incremental rebuild
  resolution.
- `internal/source/` owns the provider-neutral backend contract shared by app,
  archive, and canonical.
- `internal/config/` owns config-file path resolution, default values, template
  rendering, parsing, and config state (`missing`, `loaded`, `invalid`).
- `internal/canonical/` owns the SQLite Canonical Store, rebuild/load logic,
  transcript codecs, transcript persistence, targeted incremental rebuilds,
  and FTS deep search.
- `internal/archive/` owns sync, import analysis, and pipeline orchestration.
- `internal/stats/` owns stats computation: overview aggregates, activity
  heatmaps, session histograms, tool breakdowns, token trends, streak
  tracking, performance scorecard computation, performance lanes, performance
  trends, and transcript-sequence metrics.

Defaults: sources `~/.claude/projects/`, `~/.codex/sessions/`, archive:
`~/.local/share/carn/`. Runtime paths come from the user config file resolved
by `internal/config`.

### Data pipeline

```
<provider raw sessions>
  → archive.Pipeline.Analyze/Run()
                                 provider-aware import analysis and sync flow
  → source.Backend.Scan()/ResolveIncremental()
                                 raw scan, metadata extraction, grouping
  → canonical.Store.RebuildAll() SQLite canonical store with transcript blobs
                                 and FTS search data
  → canonical.Store.List/Load()  browser list and transcript open
  → canonical.Store.DeepSearch() canonical deep search
  → app browser/viewer/transcript TUI
```

### Package map

**Entry and composition**: `internal/app/run.go`, `internal/app/app.go`

**TUI core**: `internal/app/browser_*.go`, `internal/app/viewer_*.go`,
`internal/app/transcript_*.go`, `internal/app/import_overview*.go`,
`internal/app/import_sync_activity.go`, `internal/app/stats_*.go`

**TUI support**: `internal/app/commands.go`, `internal/app/config_reload.go`,
`internal/app/delegate.go`, `internal/app/export_names.go`,
`internal/app/footer.go`, `internal/app/help.go`,
`internal/app/help_overlay.go`, `internal/app/help_fit.go`,
`internal/app/keys.go`, `internal/app/markdown_style.go`,
`internal/app/notifications.go`, `internal/app/provider_display.go`,
`internal/app/resync.go`, `internal/app/search_preview.go`,
`internal/app/session_launcher.go`, `internal/app/styles.go`,
`internal/app/tool_result_style.go`,
`internal/app/conversation_header.go`, `internal/app/browser_store.go`,
`internal/app/import_pipeline_binding.go`,
`internal/app/drift_notification.go`, `internal/app/logging.go`,
`internal/app/version.go`

**Shared conversation model**: `internal/conversation/*.go`

**Claude source backend**: `internal/source/claude/*.go`

**Codex source backend**: `internal/source/codex/*.go`

**Provider-neutral source contract**: `internal/source/source.go`

**Config backend**: `internal/config/*.go`

**Canonical Store backend**: `internal/canonical/*.go`

**Stats backend**: `internal/stats/*.go`

**Archive/import backend**: `internal/archive/*.go`

### Core types (`internal/conversation`)

- `Provider` — claude/codex enum
- `Role` — user/assistant/system enum
- `MessageVisibility` — visible/hidden-system enum
- `Project` — display name wrapper
- `ResumeTarget` — provider, id, cwd for session resume
- `Ref` — stable conversation identifier (provider + id)
- `SessionMeta` — id, project, slug, timestamps, cwd, git branch, version,
  model, first message, counts, token usage, tool counts, subagent flag,
  file path
- `Session` — `Meta` plus ordered `Messages`
- `Message` — role, text, thinking, hidden thinking flag, visibility,
  tool calls, tool results, plans, sidechain flag, agent divider flag
- `ToolCall` — name, summary, and normalized action
- `ToolResult` — tool name, summary, content, error flag, structured patch,
  and normalized action
- `Conversation` — ref, name, project, chronological sessions, plan count,
  search preview
- `TokenUsage` — input, cache creation, cache read, output, and reasoning
  output token counts
- `DiffHunk`, `Plan`
- `NormalizedActionType` — provider-neutral action classification enum
- `ActionTargetType` — action target classification enum
- `ActionTarget` — target type and value pair
- `NormalizedAction` — action type with targets
- `MessagePerformanceMeta` — per-message reasoning, stop reason, phase, effort
- `SessionPerformanceMeta` — per-session reasoning, thinking, model context,
  duration, retries, compaction, task lifecycle, error, and provider counters
- `ToolOutcomeCounts` — per-tool success, error, and rejection counts
- `ActionOutcomeCounts` — per-action success, error, and rejection counts
- `ActivityBucketRow` — per-minute activity bucket with turn and token counts
- `PerformanceSequenceSession` — per-session performance sequence data point
- `SessionStatsData` — bundled per-session stats for canonical store rebuild
- `SessionTurnMetrics` — per-session turn-level token metrics
- `TurnTokens` — prompt, completion, and reasoning tokens for a single turn

## How to write code

This is a custom tool and we don't plan it to be used as a package therefore most of the functionality can be written private.

Keep `internal/app/` focused on app wiring and the TUI. When a file mixes
backend responsibilities or approaches the complexity cap, split it by
responsibility and move backend code into vocabulary-named packages under
`internal/` when needed. Keep files small and group code by functionality.

Write code using TDD platform. First write stubs. Then tests (following testing guidelines, scenarios are the most important tests) and implementation after. Write meaningful tests.

When handling objects or creating signatures follow the mantra "Parse, don’t validate".

Use `VOCABULARY.md` as the source of truth for naming packages, files, and
refactor targets. If a needed term is missing, add it there first.


### Dependencies

When adding new dependencies always make sure you are using the latest major version. Either check git tags or the repository for vX (where X is a number) folder.

### Errors

* Always wrap errors with fmt.Errorf using the %w directive when forwarding errors (use errors.New when there are no arguments to be passed to construct errors).
* For wrapping, include the name of the method that failed in a format "methodThatFailed: %w".
* If a single method calls the same function multiple times, differentiate the message with additional context "methodThatFailed_context: %w".
* Logging an error is considered handling the error; don't log and forward the error to other functions.
- Don't capitalize error messages
* Log errors that originate in defer functions, don't propagate them


### Logging.

* Don't use fmt.Print* methods.
* Use zerolog for logging with zerolog.Ctx
* Use Msgf to print messages with enough details
* Use fields (like .Str and similair) only for things we expect to search for in structured logging

### Pointers

* Only use pointers for shared objects (for example, a Client or shared state)
* Pass by reference when using schemas like value objects or entities 

If the object is meant to be used as pointer then all methods should have pointer receiver. And vice versa. Only exception are awalys marshal functions.

### Interfaces

* Define interfaces in the module where they are needed

### Concurrency

* Use `errgroup.WithContext(ctx)` for goroutine management
* Use `semaphore.NewWeighted()` for rate limiting concurrent operations
* Always cancel contexts when done: `ctx, cancel := context.WithCancel(ctx); defer cancel()`
* Make sure to write code in a way that you can always drain a channel so `for item := range channel` will always finish
* Use `select` when sending to a channel with a check for context cancellation
* Make sure the goroutines are not leaking

### Generics

* when you find yourself writing the same code for different types use generics
* also when your method accepts `any` or `interface{}` investigate if it can be hardened with use of generics (avoid losing type safety)
* prefer `any` instead of `interface{}`
* generics can be only defined over functions without receivers or they have to be defined also on the receiver itself in order to be used on methods


Example:

```
// Generic helper functions
func Filter[T any](slice []T, predicate func(T) bool) []T {
    result := make([]T, 0, len(slice))
    for _, v := range slice {
        if predicate(v) {
            result = append(result, v)
        }
    }
    return result
}

func Map[T, U any](slice []T, transform func(T) U) []U {
    result := make([]U, len(slice))
    for i, v := range slice {
        result[i] = transform(v)
    }
    return result
}
```

### Iterators

- you can use range-over-func to create custom iterators
- considers this when you handle lot of data and want to speed up processing with pipelining

Example:
```
// Iterator function signature
type Seq[V any] func(yield func(V) bool)
type Seq2[K, V any] func(yield func(K, V) bool)

// Custom iterator
func Backward[T any](s []T) func(yield func(int, T) bool) {
    return func(yield func(int, T) bool) {
        for i := len(s) - 1; i >= 0; i-- {
            if !yield(i, s[i]) {
                return
            }
        }
    }
}

// Usage
for i, v := range Backward(items) {
    fmt.Println(i, v)
}
```

### TUI development

* For layout changes, verify height/width arithmetic accounts for all decorations (borders, padding, footer, status bar). Document: total = top_decoration + content + bottom_decoration = terminal_dimension.
* When adding a UI element that replaces an existing one, remove or disable the old element. Verify no visual duplication.
* Apply styling changes to the exact scope described — "make X white" means only X, not surrounding text.
* Use `lipgloss.Width()` (not `len()`) for width calculations — it accounts for ANSI escape codes and Unicode.
* After implementing TUI state transitions, test the full user flow manually (start app → sync → navigate → view → back).

### Session data

* Before filtering or removing records from the pipeline, scan real data to verify what information would be lost. Never assume fields are empty without checking.

### Format drift and known schema extras

* Trigger this workflow when the task mentions format drift, schema drift, unknown fields, unknown record types, extra fields, `carn.log` drift warnings, or `known_schema_extras`.
* First read `VOCABULARY.md` so drift terms stay aligned with the project vocabulary.
* Then read the provider-owned drift files under `internal/source/<provider>/`, especially `drift*.go`, `raw_values.go`, and any existing `known_schema_extras.go`, `known_schema_extras.json`, and `known_schema_extras_test.go`.
* Keep **Known Schema** for fields and values the app actively models or depends on.
* Use **Known Schema Extras** only for observed provider-owned fields or values that are intentionally tolerated but not yet modeled.
* When adding a known schema extra, update the provider catalog entry with `status`, `path`, `record_types`, `description`, `future_use`, `first_seen`, and `example`.
* Keep examples small but real enough to contain the declared path and document the raw shape that triggered the warning.
* If the app starts parsing or depending on an extra field or value, move it from the known schema extras catalog into the provider's compile-time known schema maps and update tests accordingly.
* Add or update tests that prove documented extras suppress drift warnings and do not duplicate the compile-time known schema.
* When a new drift-related term or workflow appears, update `VOCABULARY.md` before adding new package, file, or helper names.

### Review and planning workflow

* When asked to review code, present all findings first. Wait for confirmation before making changes.
* When a fix reintroduces a previous bug, stop and find the root cause instead of adding workarounds.

## Testing

Make sure the new code passes linting:
```bash
golangci-lint run ./...
```

Run tests with the Go test command:
```bash
go test ./...
```

Run tests with the race detector:
```bash
go test -race ./...
```

Make sure the code conforms the latest go code guidelines:
```bash
go fix ./...
```

Run performance benchmarks when touching runtime-sensitive paths:
```bash
go test -run '^$' -bench 'Benchmark(CanonicalStoreScanSessions|CanonicalStoreParseConversationWithSubagents)$' -benchmem ./internal/source/claude
go test -run '^$' -bench 'Benchmark(ScanRollouts|LoadConversation)$' -benchmem ./internal/source/codex
go test -run '^$' -bench 'Benchmark(CanonicalStoreListCold|CanonicalStoreListWarm|CanonicalStoreSearchChunkCountQuery|CanonicalStoreDeepSearch|CanonicalStoreLoadTranscript|CanonicalStoreFullRebuild|CanonicalStoreIncrementalRebuild|CanonicalStoreParseConversations)$' -benchmem ./internal/canonical
go test -run '^$' -bench 'Benchmark(CollectFilesToSync|StreamImportAnalysis)$' -benchmem ./internal/archive
go test -run '^$' -bench 'Benchmark(ComputeOverview|ComputeActivity|ComputeTokenGrowth|ComputeStreaks|ToolAggregation|ComputeCache|ComputePerformance|ComputePerformanceWithSequence|CollectPerformanceSequenceSessions)$' -benchmem ./internal/stats
go test -run '^$' -bench 'Benchmark(BrowserLoadSessionsCold|BrowserLoadSessionsWarm|BrowserOpenConversationWarm|BrowserDeepSearchWarm|ViewerRenderContent|ViewerSearch)$' -benchmem ./internal/app
go test -run '^$' -bench 'Benchmark(StatsOverviewRender|StatsHeatmapRender|StatsHistogramRender|StatsCacheRender|StatsPerformanceRender)$' -benchmem ./internal/app
```

Run these benchmark commands one at a time. Do not execute multiple
`go test -bench` invocations in parallel.

When identifying optimization candidates, always start with a benchmark for
the target path and inspect a CPU profile before coding. Use
[goperf.dev](https://goperf.dev/) for implementation ideas after the profile
identifies the hot path:
```bash
go test -run '^$' -bench '<benchmark>' -cpuprofile /tmp/carn.cpu <package>
go tool pprof -top /tmp/carn.cpu
```

Keep benchmarks with the package that owns the runtime path and update
`PERF_BASELINE.md` with the full benchmark suite when benchmark commands or
results change in a meaningful way.

Complexity thresholds are enforced by `TestFileComplexityGuard` and
`TestModuleComplexityGuard` in `internal/app/complexity_guard_test.go`
(runs with `go test ./...`) across `internal/**/*.go`.

Hard limits:

* source files: complexity `<=80`, code lines `<=400`
* test files: code lines `<=800`
* modules: source complexity `<=1200`, source code lines `<=6000`

There are no file-level or module-level exceptions. When the guard fails, use
`COMPLEXITY_GUIDE.md` and refresh `COMPLEXITY_BASELINE.md` with:

```bash
go test ./internal/app -run TestComplexityBaselineDocument -count=1 -update
```

Make sure to always start implementation with tests and scenarios. If doing refactor adjust the tests first to the expected
behavior if any test fails start with assumption that there is a bug in refactor rather than a wrong test.

### Unit tests

We write unit tests to test small (but complex) functions (both private and public). 

* don't use mocks for unit tests, when needed try to refactor the tested function instead
* write table driven tests
* avoid if's in tests (most common: `if testCase.ExpectError`) instead write separate tests
* use t.Run and t.Parallel

### Scenarios

We write scenarios as tests to test overall architecture. The test are written to test the whole flow of business
logic. They exist in scenarios/ folder. 

* avoid using mocks always prefer to use real dependency (not possible for llm for example)
* test via most outer layer 
* tests are run sequentially (no parallelism) and each is atomic (test shouldn't depend on values from other test)

### Test data and fixtures

* tests must not read pre-existing files from `~/.claude/projects`, `~/.codex/sessions`, the current home directory, or any other host-specific location
* use committed sanitized fixtures under `testdata/` when you need raw session coverage
* `t.TempDir()` is allowed for scratch space, but the source data copied into it must come from the repository
* test fixtures and goldens must not contain author-specific usernames, local project names, private transcript content, or machine-specific temp paths
* when snapshotting UI output, normalize unstable paths so golden files stay portable across machines
* write generators for objects in `scenarios/helpers/generators.go` when you need various objects for tests


## Commit messages

Write commits using following pattern:
type(scope): message

where:
* type is either fix, feat, chore, docs, refactor, style
* scope marks what is affected
* message is short description of a change, lowercased

Keep the commit message and body under 72 characters.
Also keep the body of the commit message compact and readable.

Always write body with the commit message where you thoroughly explain
what was done, how was it done and why it was done in a continuous human
readable text.
For fixes try to include an error message that you are fixing.
Don't use first person speech.
Before writing a commit message, re-read the actual git diff. The message must describe what was changed, not what was planned.

### Commit command

When creating or amending commits, write the message to a temporary file
with real newlines and use `git commit -F <file>` or
`git commit --amend -F <file>`.

Do not use a single `-m` argument with escaped `\n` sequences for a
multi-line message. That stores the backslash characters literally and
produces a one-line body in git history.
