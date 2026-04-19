package stats

import statspkg "github.com/rkuska/carn/internal/stats"

func (m statsModel) helpSections() []helpSection {
	return []helpSection{m.navigationHelpSection()}
}

func (m statsModel) navigationHelpSection() helpSection {
	return helpSection{
		Title: "Navigation",
		Items: m.statsFullHelpItems(),
	}
}

// statsNavigationHelpItems returns the footer help items. Scroll keys are
// surfaced via the Metric detail lane rule and intentionally omitted here.
func (m statsModel) statsNavigationHelpItems() []helpItem {
	return m.buildStatsNavigationItems(false)
}

// statsFullHelpItems returns the full help overlay items, including scroll
// keys when the active tab's content is scrollable.
func (m statsModel) statsFullHelpItems() []helpItem {
	return m.buildStatsNavigationItems(true)
}

func (m statsModel) buildStatsNavigationItems(includeScroll bool) []helpItem {
	filterDesc := "filter"
	filterDetail := "open the filter overlay and narrow the stats dataset"
	if m.performanceScopeGateActive() {
		filterDesc = "fix scope"
		filterDetail = "open the filter overlay and narrow performance to one provider and one model"
	}

	items := []helpItem{
		{
			Key:      "ctrl+f/b",
			Desc:     "tabs",
			Detail:   "switch between overview, activity, sessions, tools, cache, and performance",
			Priority: helpPriorityLow,
		},
		{
			Key:      "r",
			Desc:     "range",
			Detail:   "cycle the active time range through 7d, 30d, 90d, and All",
			Priority: helpPriorityLow,
		},
		{
			Key:      "f",
			Desc:     filterDesc,
			Detail:   filterDetail,
			Glow:     m.performanceScopeGateActive() || m.filter.HasActiveFilters() || m.splitBy.IsActive(),
			Priority: helpPriorityNormal,
		},
	}
	if includeScroll && m.statsContentScrollable() {
		items = append(items,
			helpItem{
				Key:      "j/k",
				Desc:     "scroll",
				Detail:   "scroll the current stats content up or down",
				Priority: helpPriorityLow,
			},
			helpItem{
				Key:      "g/G",
				Desc:     "jump",
				Detail:   "jump to the top or bottom of the current stats content",
				Priority: helpPriorityLow,
			},
		)
	}
	if len(m.activeStatsLanes()) > 1 {
		items = append(items, helpItem{
			Key:      "h/l",
			Desc:     "lane",
			Detail:   "move focus between the current tab's metric-detail lanes",
			Priority: helpPriorityHigh,
		})
	}
	if item := m.activeLaneMetricHelpItem(); item.Key != "" {
		item.Detail = m.activeLaneMetricHelpDetail()
		item.Priority = helpPriorityHigh
		items = append(items, item)
	}
	if m.activeLaneSupportsOpen() {
		items = append(items, helpItem{
			Key:      "enter",
			Desc:     "open",
			Detail:   "open the selected token-heavy session in the viewer and return here with q/esc",
			Priority: helpPriorityHigh,
		})
	}
	items = append(items,
		helpItem{Key: "?", Desc: "help", Detail: "toggle the stats help overlay", Priority: helpPriorityEssential},
		helpItem{
			Key:      "q/esc",
			Desc:     "close",
			Detail:   "return to the browser without changing its state",
			Priority: helpPriorityHigh,
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
		return cacheLaneMetricHelpDetail(m.splitBy)
	}
	if detail := sessionLaneMetricHelpDetail(lane.id); detail != "" {
		return detail
	}
	if isPerformanceMetricLane(lane.id) {
		return "cycle the selected lane metric shown in the detail inspector"
	}
	return ""
}

func cacheLaneMetricHelpDetail(dim statspkg.SplitDimension) string {
	if dim.IsActive() {
		return "cycle between daily cache read rate and write rate"
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
