# càrn Vocabulary

Use these terms when referring to different parts of the project in prompts.

## Data Concepts

| Term | What it means |
|---|---|
| **Session** | A single `.jsonl` file from `~/.claude/projects/`. One continuous Claude interaction. Has metadata (`sessionMeta`) and optionally full messages (`sessionFull`). |
| **Conversation** | A group of sessions sharing the same **slug** + **project**. The primary browsable unit. Can have multiple "parts" (sessions). |
| **Slug** | User-assigned name for a session (from the JSONL). Used to group sessions into conversations. Falls back to first message or "untitled". |
| **Project** | The source directory a session was started from. Derived from the encoded path under `~/.claude/projects/`. |
| **Message** | A single turn in the transcript — either `user` or `assistant` role. Contains text, thinking, tool calls, tool results, and plans. |
| **Message Visibility** | Canonical visibility state for a message. Visible messages participate in default rendering, counts, and search; hidden system messages are preserved but suppressed by default. |
| **Hidden System Message** | A provider bootstrap or transport record preserved in canonical transcripts with hidden visibility. It can be shown with the viewer system toggle but does not affect default previews or search. |
| **Sidechain** | A background message stream (marked `isSidechain`). Not part of the main conversation flow. Counted separately in `mainMessageCount`. Can be hidden in the viewer. |
| **Subagent** | A spawned child session (`isSubagent: true`). May be shown standalone or grouped into the parent conversation, depending on provider-owned projection. |
| **Agent Divider** | A synthetic message (`isAgentDivider`) injected when subagent transcripts are merged into the parent via **projection**. Renders as a horizontal rule. |
| **Plan** | A structured document extracted from `ExitPlanMode` tool results. Has a `filePath` and `content`. Deduplicated by path (last write wins). |
| **Tool Call** | An assistant's invocation of a tool — `name` + optional `summary`. |
| **Tool Result** | The output returned to the user message — `toolName`, `content`, `isError`, and optionally a `structuredPatch` (for Edit diffs). |
| **Token Usage** | Input/output/cache token counts per message, aggregated per session as `totalUsage`. |
| **Tool Counts** | A map of tool name → invocation count per session. |

## Pipeline & Storage

| Term | What it means |
|---|---|
| **Archive** | The local cache directory (`~/.local/share/carn/`). Everything processed lives here. |
| **Source** | The raw input directory (`~/.claude/projects/`). Never modified. |
| **Sync** | Copying changed `.jsonl` files from **source** to `archive/provider/raw/`. A file "needs sync" if source is newer or different size. |
| **Scan** | Fast one-pass metadata extraction from JSONL files → `scannedSession`. No full message parsing. |
| **Parse** | Full detailed reading of a session → `sessionFull` with all messages. Slower than scan. |
| **Group** | Organizing scanned sessions by `groupKey` (dirName + slug) into conversations. |
| **Projection** | Converting intermediate `parsed*` types to final types (`message`, `toolCall`, etc.), including merging subagent transcripts at the right anchor point. |
| **Canonical Store** | The binary cache (`archive/store/v1/`) — the authoritative processed view of all data. |
| **Catalog** | `catalog.bin` — the index of all conversations with metadata. Enables fast listing without loading transcripts. |
| **Transcript** (file) | `transcripts/<key>.bin` — a single conversation's full `sessionFull`, loaded on demand. |
| **Search Corpus** | `search.bin` — chunked text index of all conversations for deep search. Made of `searchUnit`s (160-char chunks with 48-char overlap). |
| **Rebuild** | Regenerating the canonical store. Can be **full** (everything from scratch) or **incremental** (reuse unchanged transcripts). |
| **Ref** (`conversationRef`) | A stable identifier for a conversation: `provider` + `id`. Used as cache key and for loading transcripts. |
| **Provider Backend** | A source-specific implementation for one provider (for example Claude or Codex) that owns raw scan/load logic and provider-owned actions like resume. |
| **Resume Target** | The generic canonical data needed to reopen a session: `provider`, `id`, and `cwd`. The app passes this to a provider backend instead of constructing provider-specific CLI commands itself. |

## Import Wizard (first screen)

| Term | What it means |
|---|---|
| **Import Overview** | The initial TUI wizard that syncs files and builds the store. |
| **Phase** | The wizard progresses through: `analyzing` → `ready` → `syncing` → `done`. |
| **Sync Activity** | The active work inside the syncing phase: raw-file sync or local-store rebuild. |
| **Status Pill** | The colored badge showing current phase ("Analyzing", "Ready to Import", "Importing", "Complete"). |
| **Context Block** | Shows source and archive directory paths. |
| **Summary Block** | Metrics: project count, file count, conversation count. |
| **Detail Tokens** | Phase-specific stats (new/update/current counts, copied/failed/elapsed). |
| **Activity Block** | The syncing-phase status area. Raw-file sync shows a progress bar; local-store rebuild shows a spinner + message. |

## Browser (main screen)

| Term | What it means |
|---|---|
| **Browser** | The main view — conversation list + optional transcript pane. |
| **List Pane** | The left-side scrollable list of conversations. |
| **Transcript Pane** | The right-side (or fullscreen) conversation viewer. |
| **Focus** | Which pane receives keyboard input — `focusList` or `focusTranscript`. Toggled with Tab. |
| **Transcript Mode** | Layout state: `closed` (list only), `split` (side-by-side), `fullscreen` (transcript only). |
| **Delegate** | The renderer for individual list items. Controls title, description, match highlighting. |
| **Footer** | The bottom 2-line status bar. Top row: help hints or search input. Bottom row: status info and scroll %. |
| **Notification** | Transient message in the footer — `info` (gray, 3s), `success` (green, 3s), or `error` (red, 5s). |

## Search

| Term | What it means |
|---|---|
| **Metadata Search** | Default mode. Fuzzy matches on conversation title + description (model, tools, first message). Fast, client-side. |
| **Deep Search** | Full-text search across all message content via the search corpus. Toggled with `Ctrl+S`. |
| **Search Preview** | A context snippet (96 runes) around the match, shown in the list description during deep search results. |
| **Debounce** | 200ms delay before deep search fires, to avoid CPU hammering while typing. |
| **Match Ranges** | Character indices of query hits, used for highlighting matched text in list items. |

## Viewer (transcript reading)

| Term | What it means |
|---|---|
| **Viewer** | The read-only transcript display component. |
| **Viewport** | The scrollable content area within the viewer. |
| **Segment** | A visual chunk of the transcript. Types: `segmentRoleHeader`, `segmentMarkdown`, `segmentThinking`, `segmentToolCall`, `segmentToolResult`. |
| **Role Header** | A divider line ("User" or "Assistant") separating turns. |
| **Thinking Block** | Claude's extended thinking, rendered with a left `▎` border. |
| **Thinking Unavailable Note** | A Codex-only transcript block shown when reasoning existed for a reply but no readable thinking summary was stored. |
| **Initial Prompt** | The first user message, rendered with a distinctive left `▎` border. |
| **Toggles** | Visibility controls: `t` thinking, `T` tools, `R` tool results, `m` system messages, `p` plans, `s` sidechains. |
| **Transcript Search** | `/` within the viewer — searches the rendered transcript text. `n`/`N` to navigate matches. |

## Conversation Header (top of transcript)

| Term | What it means |
|---|---|
| **Badge** | Colored pill label: "subagent", "N parts", "N plans". |
| **Chip** | A label:value pair in the metadata row: "model claude-opus", "branch main", "msgs 45/120", "tokens 15k". |
| **Summary Chips** | First row: model, version, branch, duration, message counts, token count. |
| **Timing Chips** | Second row: started, last activity, resume ID. |
| **Tool Count Chips** | Row showing top tool invocations: "bash:5 grep:3 read:12". |

## Visual Primitives

| Term | What it means |
|---|---|
| **Framed Pane** | A bordered box with a title badge in the top border: `╭─ Title ──╮`. Used for list and transcript. |
| **Framed Box** | Like framed pane but non-scrollable. |
| **Inset Box** | Rounded border with padding, no custom title bar. Used for plan display. |
| **Chrome** | Non-content decoration: borders + footer. `framedChromeHeight = 4` (top border + 2 footer rows + bottom border). |
| **Key Hint** | Footer text where the key name is highlighted white: "Press `Enter` to open". |
