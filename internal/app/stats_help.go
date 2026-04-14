package app

func (m statsModel) helpSections() []helpSection {
	return []helpSection{m.navigationHelpSection()}
}

func (m statsModel) navigationHelpSection() helpSection {
	return helpSection{
		title: "Navigation",
		items: m.statsNavigationHelpItems(),
	}
}

func (m statsModel) statsNavigationHelpItems() []helpItem {
	filterDesc := "filter"
	filterDetail := "open the filter overlay and narrow the stats dataset"
	if m.performanceScopeGateActive() {
		filterDesc = "fix scope"
		filterDetail = "open the filter overlay and narrow performance to one provider and one model"
	}

	items := []helpItem{
		{
			key:      "ctrl+f/b",
			desc:     "tabs",
			detail:   "switch between overview, activity, sessions, tools, cache, and performance",
			priority: helpPriorityLow,
		},
		{
			key:      "r",
			desc:     "range",
			detail:   "cycle the active time range through 7d, 30d, 90d, and All",
			priority: helpPriorityLow,
		},
		{
			key:      "f",
			desc:     filterDesc,
			detail:   filterDetail,
			glow:     m.performanceScopeGateActive() || m.filter.hasActiveFilters(),
			priority: helpPriorityNormal,
		},
	}
	if m.statsContentScrollable() {
		items = append(items,
			helpItem{
				key:      "j/k",
				desc:     "scroll",
				detail:   "scroll the current stats content up or down",
				priority: helpPriorityLow,
			},
			helpItem{
				key:      "g/G",
				desc:     "jump",
				detail:   "jump to the top or bottom of the current stats content",
				priority: helpPriorityLow,
			},
		)
	}
	if len(m.activeStatsLanes()) > 1 {
		items = append(items, helpItem{
			key:      "h/l",
			desc:     "lane",
			detail:   "move focus between the current tab's metric-detail lanes",
			priority: helpPriorityHigh,
		})
	}
	if item := m.activeTabGroupHelpItem(); item.key != "" {
		item.priority = helpPriorityHigh
		items = append(items, item)
	}
	if item := m.activeLaneMetricHelpItem(); item.key != "" {
		item.detail = m.activeLaneMetricHelpDetail()
		item.priority = helpPriorityHigh
		items = append(items, item)
	}
	if m.activeLaneSupportsOpen() {
		items = append(items, helpItem{
			key:      "enter",
			desc:     "open",
			detail:   "open the selected token-heavy session in the viewer and return here with q/esc",
			priority: helpPriorityHigh,
		})
	}
	items = append(items,
		helpItem{key: "?", desc: "help", detail: "toggle the stats help overlay", priority: helpPriorityEssential},
		helpItem{
			key:      "q/esc",
			desc:     "close",
			detail:   "return to the browser without changing its state",
			priority: helpPriorityHigh,
		},
	)
	return items
}

func (m statsModel) activeLaneMetricHelpDetail() string {
	lane, _, ok := m.selectedStatsLane()
	if !ok {
		return ""
	}

	if lane.id == statsLaneOverviewTop {
		return "cycle the selected token-heavy session row"
	}
	if lane.id == statsLaneActivityDaily {
		return "cycle the daily chart between sessions, messages, and tokens"
	}
	if lane.id == statsLaneCacheDaily {
		return "cycle between daily hit rate and reuse ratio"
	}
	if lane.id == statsLanePerformanceOutcome ||
		lane.id == statsLanePerformanceDiscipline ||
		lane.id == statsLanePerformanceEfficiency ||
		lane.id == statsLanePerformanceRobustness {
		return "cycle the selected lane metric shown in the detail inspector"
	}
	return ""
}

func (m statsModel) activeTabGroupHelpItem() helpItem {
	switch m.tab {
	case statsTabSessions:
		return helpItem{
			key:      "v",
			desc:     "versions",
			detail:   "toggle grouped version bars and open the provider/version scope when needed",
			glow:     m.sessionsGrouped,
			priority: helpPriorityNormal,
		}
	case statsTabOverview,
		statsTabActivity,
		statsTabTools,
		statsTabCache,
		statsTabPerformance:
		return helpItem{}
	}
	return helpItem{}
}
