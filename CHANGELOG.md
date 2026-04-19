# v0.4.0

This release generalizes the stats dashboard's provider/version
grouping into a dimension-agnostic **Split by** filter row that
works across Sessions, Tools, and Cache tabs and can split by
Provider, Version, Model, or Project. The per-tab context panel
moves into a dynamic-height **Metric Detail** lane pinned between
the charts and the footer, so the explanation for the selected
lane stays on screen as charts scroll. The Cache tab gains a
Claude-only **first-turn cold cache** lane that groups cold-start
cache read rate by Claude version, answering whether a new Claude
release regresses cold-start priming.

In the browser, a new **Selection Mode** (`v` / `ctrl+v`) re-renders
the transcript without frame padding or trailing whitespace so
native terminal mouse drag produces clean copied text, and toggling
thinking, tools, results, plan, sidechain, or system content now
preserves the reader's place instead of scrolling back to the top.

Under the hood, the app package was split into `browser`, `stats`,
and `elements` ownership packages, a repo-owned `testsuite` command
with a committed coverage baseline gates regressions on every
commit, and Homebrew installs now use a rendered source formula
instead of a GoReleaser-managed cask so `brew install` works
without quarantined prebuilt binaries.

## Features

- **Split by filter row** — The stats filter overlay gains a new
  Split by row that single-selects across Provider, Version, Model,
  and Project. The values checked in each filter dimension both
  narrow the dataset and define which series the chosen split
  renders, so one set of checkboxes serves both purposes. Sessions,
  Tools, and Cache tabs reuse a single grouped vertical-bar layout
  that packs only active series into each bucket, preserves day
  separators, and keeps bar widths uniform. Tabs that cannot render
  the active split show a centered placeholder; each chart derives
  its legend from the data actually rendered rather than from the
  active filter selection. Replaces the former per-tab `v` versions
  toggle and the standalone provider/version scope overlay.

- **Metric Detail lane** — The per-tab Metric Detail context panel
  is promoted from a section inside the scrollable viewport into a
  dynamic-height lane pinned between the charts and the footer,
  clamped to three through ten rows. As the user scrolls chart
  content the lane stays anchored. The titled rule that separates
  the lane from the viewport carries scroll arrows, `j/k` /
  `g/G` help items, and the scroll percent when the viewport is
  scrollable, so hidden content is no longer invisible at the
  boundary that clips it.

- **First-turn cold cache by Claude version** — The Cache tab
  surfaces Claude-only first-turn cold cache rate grouped by
  Claude version with a min-sessions filter, recorded from the
  first assistant message's cache values per turn so follow-up
  messages reading from the cache the first one just created do
  not mask cold starts.

- **Session turn metric modes** — Session turn lanes add p50, p95,
  p99, and max modes alongside the existing avg. The active mode
  prefixes the lane title (`Avg`, `p99`, `max`, …) and the `m`
  shortcut only appears on lanes where the rendered metric changes.
  The renamed **Billed Tokens per Turn** lane (previously "Turn
  Cost") includes a Note in the metric inspector clarifying that
  the value sums every assistant API call's billed tokens in a
  single user turn, with cached prompts counted at face value on
  each step.

- **Selection Mode in the browser viewer** — A viewer-scoped `v` /
  `ctrl+v` toggle re-renders the transcript as plain text without
  frame, inner padding, or trailing whitespace so native terminal
  mouse drag produces clean output. The footer shows a `v select`
  help entry and a `[selection]` status chip while the mode is
  active.

- **Scroll-preserving toggle rebuilds** — Toggling thinking, tools,
  results, plan, sidechain, or system content now keeps the reader
  anchored to the nearest role header above the viewport top, so
  the message being read stays on-screen instead of jumping away
  when the transcript rebuilds.

- **Coverage baseline gate** — A repo-owned `./cmd/testsuite`
  command runs `go test -coverpkg=./...`, parses the coverprofile,
  and compares the result against a committed
  `COVERAGE_BASELINE.json` snapshot. Both repo total coverage and
  per-package coverage are tracked with small tolerances, refreshed
  with `go run ./cmd/testsuite -update`. Wired into `release.yml`
  and the committed `.githooks/pre-commit` hook.

## Fixes

- **Malformed raw data no longer blocks canonical rebuild** —
  Provider-owned scan and parse failures marked as malformed raw
  data now downgrade so rebuilds continue instead of failing with
  JSON parse errors like `Value is array, but can't find closing
  ']' symbol`. Last-good canonical conversations are retained
  during incremental rebuilds, and skipped items surface through
  rebuild warning notifications in the browser. Permission and
  other filesystem errors propagate unchanged; only missing raw
  files are treated as malformed.

- **Malformed-skip notification points at logs that actually
  contain detail** — The "skipped N malformed items in <provider>
  source (check logs)" notification now has matching WARN lines in
  `carn.log` with structured provider, path, and wrapped error
  fields. Previously the per-file skip was logged at debug and was
  silent at the default log level. Empty Claude session files with
  no metadata lines are no longer counted toward the malformed
  report; they get a distinct benign sentinel.

- **Image alt text in transcript viewer** — Image references
  rendered as `Image: <no value> ->` instead of the alt text
  because the Glamour template key `{{.Text}}` was uppercase and
  Go's `text/template` is case-sensitive on map keys. The template
  key is now lowercase and a render test asserts the alt text
  survives.

- **Homebrew installs via `brew install`** — Replaced the
  GoReleaser-managed Homebrew cask with a source formula rendered
  into `Formula/carn.rb` during tagged releases and published to
  the tap repository. Installs no longer ship quarantined prebuilt
  binaries through the tap.

## Improvements

- **Stats render hot path** — Chart styles, heatmap cells, and
  split color maps are cached outside the render call; per-split
  tools, cache, and color-map results are cached on the stats
  model and refreshed through the snapshot invalidation path so
  keystroke renders on split tabs no longer re-aggregate all
  filtered sessions on every paint. The Metric Detail lane caches
  split body lines so the active detail is not re-rendered twice
  per frame.

- **Streaming turn-token aggregation** —
  `ComputeTurnTokenMetrics` walks messages directly into position
  totals instead of building a per-session turn slice and then
  re-aggregating it. `BenchmarkComputeTokenGrowth` drops from
  640867 B/op and 1002 allocs/op to 1056 B/op and 3 allocs/op at
  the 1000-session size.

- **TUI module layout** — The app package is split into `browser`,
  `stats`, and `elements` ownership packages so the composition
  root stays focused on wiring. The theme is now threaded through
  browser, stats, import-overview, and shared elements as an
  explicit object created at app startup instead of a
  package-global palette, removing init-order coupling from the
  split.

- **gopls pre-commit check** — `.githooks/GOPLS_LINT` runs the
  info-level `gopls` diagnostics (unused declarations and
  parameters) that `golangci-lint` was not catching, so commits
  fail fast on dead code.

---

# v0.3.1

A small polish release that fixes a visual spacing issue in the stats
dashboard lane cards.

## Fixes

- **Stats lane card padding** — Chart bars and text in lane cards started
  immediately after the badge border with no vertical gap, making the content
  look cramped. Lane cards now include a line of breathing room between the
  title badge and the body.

---

# v0.3.0

This release adds a dedicated Cache stats tab, moves the stats dashboard onto
precomputed SQLite rows for faster rendering, and fixes token accounting across
both Claude and Codex providers. Activity tracking is now timezone-aware, prompt
growth charts focus on main-thread turns, and Codex token semantics are
normalized to match Anthropic's exclusive-bucket convention — eliminating
double-counting that made cross-provider comparisons misleading.

![Cache stats tab](https://raw.githubusercontent.com/rkuska/carn/v0.3.0/releases/assets/v0.3.0/cache.png)

## Features

- **Cache stats tab** — A new tab between Tools and Performance surfaces
  prompt-cache behavior with hit rate, miss rate, reuse ratio, daily trends,
  main-vs-subagent comparison, and duration histograms.

- **Precomputed stats rows** — Stats UI reads from canonical SQLite rows
  instead of loading transcript blobs. A one-time backfill runs on first
  launch after upgrade.

- **Message timestamps** — Per-message timestamps preserved through
  projection paths and canonical transcript blobs.

## Fixes

- **Codex token double-counting** — Codex parser normalizes OpenAI's
  inclusive token semantics to exclusive buckets at parse time.

- **Sidechain model metadata leak** — Primary sessions no longer inherit
  sidechain model identifiers.

- **UTC minute activity buckets** — Timezone-correct rebucketing and
  streak calculation.

- **Prompt growth aligned to main-thread turns** — Subagents, sidechains,
  and tool-only steps excluded from turn position.

## Improvements

- **Stats hot path overhead reduced** — Fixed-size top-five, indexed day
  buckets, preallocated slices, direct border rendering.

- **Degraded stats notification** — Error notification with rebuild
  guidance when precomputed row queries fail.

- **Full timestamps in carn.log** — RFC3339 format with timezone offset.

---

# v0.2.0

càrn v0.2.0 turns the transcript browser into an analytics surface. A new
fullscreen Stats view summarizes your local Claude and Codex archives with
activity heatmaps, session and tool histograms, token trends, and a
lane-driven performance scorecard that grades how your model runs behave
across accuracy, efficiency, robustness, and workflow.

Underneath, this release parses action and runtime metadata that earlier
versions ignored, reconstructs per-turn token usage for Codex sessions, and
persists tool outcomes directly in the canonical SQLite store so opening
the dashboard no longer reloads transcripts. A new source drift detector
warns after sync when provider formats grow unknown fields or record
types, keeping you ahead of silent schema changes.

Timestamps are now localized to your timezone with relative recency hints,
and dozens of smaller layout, coloring, and parser fixes land alongside
the features. The release workflow now ships with human-readable notes
instead of an auto-generated commit changelog.

![Stats dashboard — sessions tab](https://raw.githubusercontent.com/rkuska/carn/v0.2.0/releases/assets/v0.2.0/stats-sessions.png)

## Features

- **Fullscreen Stats dashboard** — Press `S` in the browser to open a
  fullscreen view with Overview, Activity, Sessions, Tools, and
  Performance tabs. Filters and range pickers reuse the same controls
  as the browser, charts use a consistent semantic palette (blue for
  tokens, light purple for time, green for counts), and histogram bars
  carry per-bucket labels so small differences stay readable. The
  activity heatmap folds 24 hourly rows into six 4-hour bands and
  compresses empty hour spans with an ellipsis marker for typical
  developer schedules.

- **Overview token trends and heavy sessions** — The overview tab
  compares the active range against the previous period of equal length
  and shows token chips as up, down, or flat. The token-heavy session
  table ranks the top five sessions and lets you open any of them with
  `1`–`5`; `q`/`esc` returns you to the same stats state.

- **Lane-driven performance scorecard** — A dedicated Performance tab
  shows a 2x2 lane grid (Accuracy, Efficiency, Robustness, Workflow)
  built from session-level and transcript-sequence metrics like
  first-pass resolution, blind edit rate, context growth, retry burden,
  and correction loops. Navigate with `h`/`l` across lanes and `m` to
  cycle metrics; the inspector shows each metric's question, formula,
  baseline delta, status, and a full-width trend sparkline. Scoring
  uses per-metric sample thresholds and weighted combination so
  diagnostic metrics don't dominate lane scores, and a scope gate
  prompts you to narrow the filter when multiple providers or models
  are mixed. Provider-specific metrics (abort rate for Codex, retry
  burden for Claude) surface only where they apply.

  ![Performance scorecard](https://raw.githubusercontent.com/rkuska/carn/v0.2.0/releases/assets/v0.2.0/stats-performance.png)

- **Action and runtime metadata parsing** — Claude and Codex import
  paths now extract normalized actions, tool outcomes, per-message
  stop reasons, reasoning effort, and per-session runtime counters
  (duration, retries, compaction, task lifecycle). The counters flow
  through the canonical codecs and SQLite store so stats can read
  them without touching transcripts.

- **Codex turn token reconstruction** — Codex sessions that only
  record `last_token_usage` now get per-turn usage reconstructed and
  attached to the assistant turn that owns the token event. The
  Sessions tab includes Codex conversations in turn-position charts
  with provider-neutral labels so mixed archives show the full story.

- **Persisted tool outcomes in canonical store** — Tool success,
  failure, and rejection counts are derived once during rebuild and
  stored in SQLite session rows and transcript blobs. The Tools tab
  reads them directly instead of lazy-loading transcripts, and a
  schema version bump rebuilds existing stores cleanly.

- **Source format drift detection** — A provider-neutral drift
  detector runs during source scans and reports unknown fields,
  record types, and content blocks through the import flow. After
  sync, càrn warns you if Claude or Codex started writing schema
  shapes the app does not yet model. A companion **known schema
  extras** catalog lets tolerated fields be documented explicitly
  so genuine novelty stays noisy.

- **Human-readable release notes workflow** — A new
  `/tag-and-release` skill drafts prose release notes grouped by
  user-facing outcome and commits them under `releases/`. The release
  GitHub Actions workflow picks up `releases/v{tag}.md` when present
  and passes it to GoReleaser via `--release-notes`, replacing the
  auto-generated commit changelog.

## Fixes

- **Timestamps are localized** — Canonical unix timestamps now decode
  into `time.Local` so browser and viewer titles reflect your
  timezone instead of forced UTC, and recent sessions get relative
  recency hints in the title.

- **Claude parser accepts null `stop_reason`** — Assistant messages
  where Claude stored `null` for `stop_reason` no longer break
  canonical rebuilds with a `Value is not a string: null` error.

- **Canonical rebuild no longer reparses sessions** — Tool outcome
  enrichment reuses source scan metadata when the provider reports
  scanned outcome support, and the rebuild path parses conversations
  before the SQLite write pass. Full rebuild dropped ~47%, parse
  conversations dropped ~57%, incremental rebuild dropped ~30%.

- **GoReleaser v2 config fields** — Replaced deprecated GoReleaser
  fields so the release pipeline runs cleanly on the v2 toolchain.

## Improvements

- **Hot-path performance pass** — Viewer render allocations dropped
  to zero, browser open-conversation dropped ~50%, canonical load
  transcript dropped ~55%, and Codex and Claude scan allocations
  dropped sharply. See `PERF_BASELINE.md` for the full benchmark
  table.
