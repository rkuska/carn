# càrn

càrn is a tool that provides additional functionality over ~/.claude/projects files that hold sessions from claude locally.


## Architecture

All code lives in `internal/app/` (single flat package). Source: `~/.claude/projects/`, archive: `~/.local/share/carn/`.

### Data pipeline

```
~/.claude/projects/**/*.jsonl (raw sessions)
  → scanSessions()            scanner.go         fast metadata extraction
  → groupConversations()      conversation.go    group by slug+project
  → rebuildCanonicalStore()   canonical_store.go binary cache with search index
  → conversationRepository    conversation_repository.go load on demand
  → TUI display               browser.go → viewer.go → transcript.go
```

### File map

**Entry**: `run.go` (init, logger, Bubble Tea program)
**Data pipeline**: `scanner.go` (JSONL parse), `parser_types.go` (intermediate types), `conversation.go` (grouping), `conversation_repository.go` (data access)
**Store**: `canonical_store.go` (binary cache), `canonical_store_incremental.go` (incremental rebuild)
**Import**: `archive.go` (config, sync orchestration), `sync.go` (concurrent file sync), `import_pipeline.go` (pipeline logic), `import_overview.go` + `import_overview_view.go` (TUI wizard)
**TUI core**: `app.go` (root model, viewImportOverview → viewBrowser), `browser.go` (conversation list + preview), `viewer.go` (transcript reader), `transcript.go` (message → segments)
**TUI support**: `browser_search.go` + `browser_search_controller.go` (search), `search.go` + `search_preview.go` (deep search), `delegate.go` (list rendering), `styles.go` (theming), `keys.go` (keybindings), `commands.go` (Bubble Tea cmds), `footer.go`, `notifications.go`, `session.go` (session state), `conversation_header.go` (header rendering)
**Domain**: `types.go` (core types), `conversation_projection.go` (subagent transcript merge)

### Core types (types.go, conversation.go)

- `sessionMeta` — id, project, slug, timestamp, lastTimestamp, cwd, gitBranch, version, model, firstMessage, messageCount, mainMessageCount, totalUsage, toolCounts, isSubagent, filePath
- `sessionFull` — meta + []message
- `message` — role, text, thinking, []toolCall, []toolResult, isSidechain, isAgentDivider
- `toolCall` — name, summary
- `toolResult` — toolName, toolSummary, content, isError, []diffHunk (structuredPatch)
- `conversation` — ref, name (slug), project, []sessionMeta (chronological), searchPreview
- `tokenUsage` — inputTokens, cacheCreationInputTokens, cacheReadInputTokens, outputTokens
- `scannedSession` (scanner.go) — sessionMeta + groupKey + hasConversationContent

## How to write code

This is a custom tool and we don't plan it to be used as a package therefore most of the functionality can be written private.

We use flat layout wit no folders to not complicate things. We keep our files small in terms of line length and we group code by its functionality into separate files.

Write code using TDD platform. First write stubs. Then tests (following testing guidelines, scenarios are the most important tests) and implementation after. Write meaningful tests.

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

Make sure the code conforms the latest go code guidelines:
```bash
go fix ./...
```

Run performance benchmarks when touching runtime-sensitive paths:
```bash
go test -run '^$' -bench 'Benchmark(LoadCatalog|LoadSearchIndex|DeepSearchFuzzy|CanonicalTranscriptOpen|ViewerRenderContent|ViewerSearch|CollectFilesToSync|StreamImportAnalysis|CanonicalStoreScanSessions|CanonicalStoreParseConversationWithSubagents|CanonicalStoreParseConversations|CanonicalStoreFullRebuild|CanonicalStoreIncrementalRebuild)$' -benchmem ./internal/app
```

When identifying optimization candidates, always start with a benchmark for the target path and inspect a CPU profile before coding:
```bash
go test -run '^$' -bench '<benchmark>' -cpuprofile /tmp/carn.cpu ./internal/app
go tool pprof -top /tmp/carn.cpu
```

Keep benchmark scenarios in `perf_bench_test.go` and update `PERF_BASELINE.md`
when benchmark results change in a meaningful way.

Make sure to always start implementation with tests and scenarios. If doing refactor adjust the tests first to the expected
behavior if any test fails start with assumption that there is a bug in refactor rather than a wrong test.

### Unit tests

We write unit tests to test small (but complex) functions (both private and public). 

* don't use mocks for unit tests, when needed try to refactor the tested function instead
* write table driven tests
* avoid if's in tests (most common: `if testCase.ExpectError`) instead write separate tests
* use t.Run and t.Parallel

### Scenarions

We write scenarios as tests to test overall architecture. The test are written to test the whole flow of business
logic. They exist in scenarios/ folder. 

* avoid using mocks always prefer to use real dependency (not possible for llm for example)
* test via most outer layer 
* tests are run sequentially (no parallelism) and each is atomic (test shouldn't depend on values from other test)
* write generators for objects in `helpers/generators.go` when you need various objects for tests


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
