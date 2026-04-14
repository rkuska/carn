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
	return laneMetricHelpDetail(m, lane)
}

func laneMetricHelpDetail(m statsModel, lane statsLane) string {
	if lane.id == statsLaneOverviewTop {
		return "cycle the selected token-heavy session row"
	}
	if lane.id == statsLaneActivityDaily {
		return "cycle the daily chart between sessions, messages, and tokens"
	}
	if lane.id == statsLaneCacheDaily {
		return cacheLaneMetricHelpDetail(m.cacheGrouped)
	}
	if detail := sessionLaneMetricHelpDetail(lane.id); detail != "" {
		return detail
	}
	if isPerformanceMetricLane(lane.id) {
		return "cycle the selected lane metric shown in the detail inspector"
	}
	return ""
}

func cacheLaneMetricHelpDetail(grouped bool) string {
	if grouped {
		return "cycle between daily cache read share and write share"
	}
	return "cycle between daily hit rate and reuse ratio"
}

func sessionLaneMetricHelpDetail(id statsLaneID) string {
	if id == statsLaneSessionsContext || id == statsLaneSessionsTurnCost {
		return "cycle the selected statistic used by the chart and detail inspector"
	}
	return ""
}

func isPerformanceMetricLane(id statsLaneID) bool {
	return id == statsLanePerformanceOutcome ||
		id == statsLanePerformanceDiscipline ||
		id == statsLanePerformanceEfficiency ||
		id == statsLanePerformanceRobustness
}

func (m statsModel) activeTabGroupHelpItem() helpItem {
	if !m.versionGroupingSupportedTab() {
		return helpItem{}
	}
	return helpItem{
		key:      "v",
		desc:     "versions",
		detail:   "toggle grouped version bars and open the provider/version scope when needed",
		glow:     m.versionGroupingActive(),
		priority: helpPriorityNormal,
	}
}
