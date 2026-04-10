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
