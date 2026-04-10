package app

func performanceChartHelpItems() []helpItem {
	return []helpItem{
		{
			key:  "Lane Cards",
			desc: "scorecards",
			detail: "Shows outcome, discipline, efficiency, and robustness as " +
				"separate scorecards. Each card lists the key metrics that drove the " +
				"lane and a compact trend sparkline for the active time slice.",
		},
		{
			key:  "Metric Detail",
			desc: "inspector",
			detail: "Shows the selected metric question, formula, baseline delta, " +
				"and a full-width trend chart so the scorecard stays readable without hiding the explanation.",
		},
		{
			key:  "Diagnostics",
			desc: "causes",
			detail: "Shows likely causes from regressing score-driving metrics and " +
				"supporting provider signals such as hidden thinking, cache efficiency, stop reasons, server tool use, " +
				"and effort mode without letting them dominate the score.",
		},
	}
}

func performanceSummaryHelpItems() []helpItem {
	return []helpItem{
		{
			key:  "overall",
			desc: "score",
			detail: "Shows the blended health score across all lane metrics that " +
				"have enough current and baseline data to compare.",
		},
		{
			key:  "outcome",
			desc: "lane score",
			detail: "Tracks whether changes stick, get verified, and avoid " +
				"follow-up correction work.",
		},
		{
			key:  "discipline",
			desc: "lane score",
			detail: "Tracks whether the model reads and searches before it " +
				"mutates, and whether it falls into rewrite-heavy or looping behavior.",
		},
		{
			key:  "efficiency",
			desc: "lane score",
			detail: "Tracks token and action cost per user direction, plus " +
				"provider-specific reasoning spend when the slice is provider-pure.",
		},
		{
			key:  "robustness",
			desc: "lane score",
			detail: "Tracks tool failures, rejections, aborts, retries, and " +
				"context-pressure signals.",
		},
	}
}

func performanceNavigationHelpItems() []helpItem {
	return []helpItem{
		{
			key:    "h/l",
			desc:   "lane",
			detail: "move focus between outcome, discipline, efficiency, and robustness",
		},
		{
			key:    "m",
			desc:   "metric",
			detail: "cycle the selected lane metric shown in the detail inspector",
		},
	}
}
