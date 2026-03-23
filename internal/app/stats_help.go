package app

import (
	"fmt"

	statspkg "github.com/rkuska/carn/internal/stats"
)

func (m statsModel) helpSections() []helpSection {
	sections := []helpSection{
		m.summaryHelpSection(),
		m.chartHelpSection(),
		m.navigationHelpSection(),
	}
	return sections
}

func (m statsModel) chartHelpSection() helpSection {
	switch m.tab {
	case statsTabOverview:
		return helpSection{
			title: "Charts",
			items: []helpItem{
				{
					key:  "Tokens by Model",
					desc: "bars",
					detail: "Shows which models are driving token use. Y-axis lists models; " +
						"X-axis shows total tokens in the active slice.",
				},
				{
					key:  "Tokens by Project",
					desc: "bars",
					detail: "Shows which projects are consuming the most context. Y-axis " +
						"lists projects; X-axis shows total tokens.",
				},
				{
					key:  "Most Token-Heavy Sessions",
					desc: "table",
					detail: "Shows the heaviest individual sessions, not grouped " +
						"conversations. Columns read left to right as project, slug, date, " +
						"messages, duration, and total tokens.",
				},
			},
		}
	case statsTabActivity:
		return helpSection{
			title: "Charts",
			items: []helpItem{
				{
					key:  "Daily Activity",
					desc: "line chart",
					detail: "Shows whether activity is steady, spiky, or fading across the " +
						"range. X-axis is calendar day; Y-axis is the selected metric: " +
						"sessions, messages, or tokens.",
				},
				{
					key:  "Activity Heatmap",
					desc: "heatmap",
					detail: "Shows when work tends to happen. Rows are weekdays, columns are " +
						"hours, and darker cells mean more sessions in that slot.",
				},
			},
		}
	case statsTabSessions:
		return helpSection{
			title: "Charts",
			items: []helpItem{
				{
					key:  "Session Duration",
					desc: "histogram",
					detail: "Shows whether sessions are mostly quick checks or long runs. " +
						"X-axis is duration bucket; Y-axis is session count.",
				},
				{
					key:  "Messages per Session",
					desc: "histogram",
					detail: "Shows whether sessions stay short or turn into long exchanges. " +
						"X-axis is message-count bucket; Y-axis is session count.",
				},
				{
					key:  statsClaudeContextGrowthTitle,
					desc: "line chart",
					detail: "Shows how context tends to accumulate as sessions go " +
						"deeper. X-axis is usage-bearing turn number; Y-axis is average input " +
						"tokens at that turn. Read it as context depth: rising lines mean " +
						"later turns are reading more prompt context. Turns with fewer than " +
						"three contributing sessions are omitted.",
				},
				{
					key:  statsClaudeTurnCostTitle,
					desc: "line chart",
					detail: "Shows how expensive each turn becomes once prompt and " +
						"response are counted together. X-axis is usage-bearing turn number; " +
						"Y-axis is average input+output tokens at that turn. Read it as total " +
						"turn cost rather than pure context size. Turns with fewer than three " +
						"contributing sessions are omitted.",
				},
			},
		}
	case statsTabTools:
		return helpSection{
			title: "Charts",
			items: []helpItem{
				{
					key:  "Top Tools",
					desc: "bars",
					detail: "Shows which tools dominate the workflow. Y-axis lists tools; " +
						"X-axis shows total calls.",
				},
				{
					key:  "Tool Calls/Session",
					desc: "histogram",
					detail: "Shows whether tool use is light and frequent or concentrated in " +
						"a few heavy sessions. X-axis is call-count bucket; Y-axis is " +
						"session count.",
				},
				{
					key:  "Tool Error Rate",
					desc: "bars",
					detail: "Shows which tools fail often enough to inspect once they are " +
						"actually run. Y-axis lists tools; X-axis shows error rate percent " +
						"with the absolute error count alongside it. Tools with fewer than " +
						"three errors are omitted. User-declined suggestions are excluded " +
						"here and shown separately below.",
				},
				{
					key:  statsRejectedSuggestionsTitle,
					desc: "bars",
					detail: "Shows which suggested tools users push back on before they run. " +
						"Y-axis lists tools; X-axis shows rejected-share percent. Read it as " +
						"user resistance to the assistant's proposed tool choice, not as an execution failure.",
				},
			},
		}
	default:
		return helpSection{}
	}
}

func (m statsModel) summaryHelpSection() helpSection {
	switch m.tab {
	case statsTabOverview:
		return helpSection{
			title: "Summary Chips",
			items: []helpItem{
				{
					key:  "sessions",
					desc: "count",
					detail: "Sets the size of the slice you are looking at. This is the " +
						"number of sessions in the active range.",
				},
				{
					key:  "messages",
					desc: "total",
					detail: "Shows how chat-heavy the range was. This is the total " +
						"main-thread message count.",
				},
				{
					key:  "tokens",
					desc: "total",
					detail: "Shows overall token burn. This combines input, output, and " +
						"cache token usage. On 7d, 30d, and 90d ranges it also shows the " +
						"change versus the previous period of the same length.",
				},
				{
					key:  "input/output",
					desc: "breakdown",
					detail: "Shows where token spend is going. It compares prompt tokens to " +
						"generated completion tokens.",
				},
				{
					key:  "cache-rd/cache-wr",
					desc: "cache",
					detail: "Shows how much reuse the provider reported. Read means cache " +
						"hits; write means cache population.",
				},
			},
		}
	case statsTabActivity:
		return helpSection{
			title: "Summary Chips",
			items: []helpItem{
				{
					key:  "active days",
					desc: "days used",
					detail: "Shows how consistently the tool was used. It counts days with " +
						"at least one session out of all days in range.",
				},
				{
					key:  "current streak",
					desc: "streak",
					detail: "Shows whether activity is still ongoing. This is the run of " +
						"consecutive active days ending at the range boundary.",
				},
				{
					key:  "longest streak",
					desc: "best streak",
					detail: "Shows the strongest burst of consistency in the range. This is " +
						"the longest run of active days.",
				},
			},
		}
	case statsTabSessions:
		return helpSection{
			title: "Summary Chips",
			items: []helpItem{
				{
					key:  "avg duration",
					desc: "duration",
					detail: "Shows the typical session length. It is the mean time from first " +
						"activity to last activity.",
				},
				{
					key:  "avg messages",
					desc: "messages",
					detail: "Shows how long conversations tend to run. It is the mean " +
						"main-thread message count per session.",
				},
				{
					key:  "user:assistant",
					desc: "ratio",
					detail: "Shows who is carrying the conversation. It compares user " +
						"messages to assistant messages across sessions.",
				},
				{
					key:  "abandoned",
					desc: "drop-off",
					detail: "Shows how often work stops before it really starts. It counts " +
						"sessions with fewer than three messages or shorter than one minute.",
				},
				{
					key:  statsClaudeContextEarlyLabel,
					desc: "early context",
					detail: "Shows the baseline context depth early in a session. It " +
						"is the average input tokens across turns one through five.",
				},
				{
					key:  statsClaudeContextLateLabel,
					desc: "late context",
					detail: "Shows how much prompt context late turns are carrying. " +
						"It is the average input tokens from turn twenty onward.",
				},
				{
					key:  statsClaudeContextFactorLabel,
					desc: "context factor",
					detail: "Shows how much the prompt context swells from early to late " +
						"turns. It is the late-context average divided by the first-five average.",
				},
				{
					key:  statsClaudeTurnCostEarlyLabel,
					desc: "early turn cost",
					detail: "Shows the typical total cost of early turns. It is the " +
						"average input+output tokens across turns one through five.",
				},
				{
					key:  statsClaudeTurnCostLateLabel,
					desc: "late turn cost",
					detail: "Shows how expensive deep turns become in total. It is " +
						"the average input+output tokens from turn twenty onward.",
				},
				{
					key:  statsClaudeTurnCostFactorLabel,
					desc: "cost factor",
					detail: "Shows how much total turn cost expands from early to " +
						"late turns. It is the late turn-cost average divided by the first-five average.",
				},
			},
		}
	case statsTabTools:
		return helpSection{
			title: "Summary Chips",
			items: []helpItem{
				{
					key:  "total calls",
					desc: "invocations",
					detail: "Shows overall tool dependence. This is the total number of tool " +
						"calls across matching sessions.",
				},
				{
					key:  "avg/session",
					desc: "mean calls",
					detail: "Shows how tool-heavy a typical session is. This is the average " +
						"number of tool calls per session.",
				},
				{
					key:  "error rate",
					desc: "errors",
					detail: "Shows how noisy tool execution is overall. This is the share of " +
						"tool calls that ended in real execution errors, with user rejections excluded.",
				},
				{
					key:  "rejected",
					desc: "user pushback",
					detail: "Shows how often the user declines a suggested tool before it " +
						"runs. This is the share of tool calls that were explicitly rejected by the user.",
				},
				{
					key:  "read:write:bash",
					desc: "ratio",
					detail: "Shows the working style behind the sessions. It compares read " +
						"tools, write tools, and Bash on a normalized scale.",
				},
			},
		}
	default:
		return helpSection{}
	}
}

func (m statsModel) navigationHelpSection() helpSection {
	items := []helpItem{
		{key: "ctrl+f/b", desc: "tabs", detail: "switch between overview, activity, sessions, and tools"},
		{key: "r", desc: "range", detail: "cycle the active time range through 7d, 30d, 90d, and All"},
		{key: "f", desc: "filter", detail: "open the filter overlay and narrow the stats dataset"},
		{key: "j/k", desc: "scroll", detail: "scroll the current stats content up or down"},
		{key: "g/G", desc: "jump", detail: "jump to the top or bottom of the current stats content"},
		{key: "?", desc: "help", detail: "toggle the stats help overlay"},
		{key: "q/esc", desc: "close", detail: "return to the browser without changing its state"},
	}
	if m.tab == statsTabActivity {
		items = append(items, helpItem{
			key:    "m",
			desc:   "metric",
			detail: "cycle the daily chart between sessions, messages, and tokens",
		})
	}
	return helpSection{title: "Navigation", items: items}
}

func formatRatio(value float64) string {
	return fmt.Sprintf("%.1f:1", value)
}

func formatToolRatio(ratio statspkg.ToolCategoryRatio) string {
	return fmt.Sprintf("%.1f:%.1f:%.1f", ratio.Read, ratio.Write, ratio.Bash)
}
