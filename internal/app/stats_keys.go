package app

import (
	"regexp"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/rkuska/carn/internal/stats"
)

type statsKeyMap struct {
	NextTab key.Binding
	PrevTab key.Binding
	Range   key.Binding
	Filter  key.Binding
	Metric  key.Binding
	Help    key.Binding
	Close   key.Binding
}

var statsKeys = statsKeyMap{
	NextTab: key.NewBinding(
		key.WithKeys("ctrl+f"),
		key.WithHelp("ctrl+f", "next tab"),
	),
	PrevTab: key.NewBinding(
		key.WithKeys("ctrl+b"),
		key.WithHelp("ctrl+b", "prev tab"),
	),
	Range: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "range"),
	),
	Filter: key.NewBinding(
		key.WithKeys("f"),
		key.WithHelp("f", "filter"),
	),
	Metric: key.NewBinding(
		key.WithKeys("m"),
		key.WithHelp("m", "metric"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Close: key.NewBinding(
		key.WithKeys("q", "esc"),
		key.WithHelp("q/esc", "close"),
	),
}

func (m statsModel) handleStatsKey(msg tea.KeyPressMsg) (statsModel, tea.Cmd) {
	if next, cmd, handled := m.handleStatsActionKey(msg); handled {
		return next, cmd
	}
	if next, cmd, handled := m.handleStatsOpenSessionKey(msg); handled {
		return next, cmd
	}
	if next, handled := m.handleStatsJumpKey(msg); handled {
		return next, nil
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m statsModel) handleStatsActionKey(msg tea.KeyPressMsg) (statsModel, tea.Cmd, bool) {
	switch {
	case key.Matches(msg, statsKeys.NextTab):
		m.tab = nextStatsTab(m.tab)
		next, cmd := m.renderViewportContentAndMaybeLoad(true)
		return next, cmd, true
	case key.Matches(msg, statsKeys.PrevTab):
		m.tab = prevStatsTab(m.tab)
		next, cmd := m.renderViewportContentAndMaybeLoad(true)
		return next, cmd, true
	case key.Matches(msg, statsKeys.Range):
		next, cmd := m.handleStatsRangeAction()
		return next, cmd, true
	case key.Matches(msg, statsKeys.Filter):
		return m.openFilterOverlay(), nil, true
	case key.Matches(msg, statsKeys.Metric):
		if m.tab != statsTabActivity {
			return m, nil, true
		}
		m.activityMetric = nextActivityMetric(m.activityMetric)
		return m.renderViewportContent(true), nil, true
	case key.Matches(msg, statsKeys.Help):
		m.helpOpen = true
		return m, nil, true
	case key.Matches(msg, statsKeys.Close):
		return m, updateCloseStatsCmd(), true
	default:
		return m, nil, false
	}
}

func (m statsModel) handleStatsRangeAction() (statsModel, tea.Cmd) {
	m.timeRange = nextStatsTimeRange(m.timeRange)
	m.snapshot = stats.ComputeSnapshot(m.filteredSessions(), m.timeRange)
	claudeTurnMetricsCached := m.claudeTurnMetricsSourceKey == m.claudeTurnMetricsSourceCacheKey()

	if claudeTurnMetricsCached {
		m.claudeTurnMetrics = stats.ComputeTurnTokenMetricsForRange(m.claudeTurnMetricSessions, m.timeRange)
		m.snapshot.Sessions.ClaudeTurnMetrics = m.claudeTurnMetrics
	} else {
		m.claudeTurnMetrics = nil
	}
	if m.tab == statsTabSessions && !claudeTurnMetricsCached {
		return m.renderViewportContentAndMaybeLoad(true)
	}
	return m.renderViewportContent(true), nil
}

func (m statsModel) handleStatsOpenSessionKey(msg tea.KeyPressMsg) (statsModel, tea.Cmd, bool) {
	rank, ok := heavySessionRankFromKey(msg.Text)
	if !ok {
		return m, nil, false
	}
	next, cmd := m.openHeavySession(rank)
	return next, cmd, true
}

func (m statsModel) handleStatsJumpKey(msg tea.KeyPressMsg) (statsModel, bool) {
	switch {
	case msg.Text == "g" || msg.Code == tea.KeyHome:
		m.viewport.GotoTop()
		return m, true
	case msg.Text == "G" || msg.Code == tea.KeyEnd:
		m.viewport.GotoBottom()
		return m, true
	default:
		return m, false
	}
}

func heavySessionRankFromKey(text string) (int, bool) {
	switch text {
	case "1", "2", "3", "4", "5":
		return int(text[0] - '1'), true
	default:
		return 0, false
	}
}

func (m statsModel) openFilterOverlay() statsModel {
	m.filter.active = true
	m.filter.expanded = -1
	m.filter.regexEditing = false
	m.filter.regexInput.Blur()
	return m
}

func (m statsModel) closeFilterOverlay() statsModel {
	m.filter.active = false
	m.filter.expanded = -1
	m.filter.regexEditing = false
	m.filter.regexInput.Blur()
	return m
}

func (m statsModel) handleFilterKey(msg tea.KeyPressMsg) (statsModel, tea.Cmd) {
	if m.filter.regexEditing {
		return m.handleFilterRegexKey(msg)
	}
	if m.filter.expanded >= 0 {
		return m.handleFilterExpandedKey(msg)
	}
	return m.handleFilterDimensionKey(msg)
}

func (m statsModel) handleFilterDimensionKey(msg tea.KeyPressMsg) (statsModel, tea.Cmd) {
	dim := filterDimension(m.filter.cursor)

	if updated, cmd, handled := m.handleFilterNavigation(msg); handled {
		return updated, cmd
	}

	return m.handleFilterDimensionAction(msg, dim)
}

func (m statsModel) handleFilterNavigation(msg tea.KeyPressMsg) (statsModel, tea.Cmd, bool) {
	switch {
	case msg.Code == tea.KeyEscape || msg.Text == "q":
		return m.closeFilterOverlay(), nil, true
	case msg.Text == "j" || msg.Code == tea.KeyDown:
		if m.filter.cursor < int(filterDimCount)-1 {
			m.filter.cursor++
		}
		return m, nil, true
	case msg.Text == "k" || msg.Code == tea.KeyUp:
		if m.filter.cursor > 0 {
			m.filter.cursor--
		}
		return m, nil, true
	}
	return m, nil, false
}

func (m statsModel) handleFilterDimensionAction(
	msg tea.KeyPressMsg,
	dim filterDimension,
) (statsModel, tea.Cmd) {
	switch {
	case msg.Code == tea.KeyEnter || msg.Text == "l" || msg.Code == tea.KeyRight:
		return m.filterExpandOrCycleBool(dim)
	case msg.Text == " ":
		return m.filterToggleBool(dim)
	case msg.Text == "x":
		m.filter.dimensions[dim] = dimensionFilter{}
		return m.applyFilterChangeAndMaybeLoad()
	case msg.Text == "X":
		return m.filterClearAll()
	case msg.Text == "/":
		return m.filterStartRegex(dim)
	}
	return m, nil
}

func (m statsModel) filterExpandOrCycleBool(dim filterDimension) (statsModel, tea.Cmd) {
	if filterDimensionIsBool(dim) {
		m.filter.dimensions[dim] = dimensionFilter{
			boolState: cycleBoolFilter(m.filter.dimensions[dim].boolState),
		}
		return m.applyFilterChangeAndMaybeLoad()
	}
	m.filter.expanded = m.filter.cursor
	m.filter.expandedCursor = 0
	m.filter.expandedScroll = 0
	return m, nil
}

func (m statsModel) filterToggleBool(dim filterDimension) (statsModel, tea.Cmd) {
	if filterDimensionIsBool(dim) {
		m.filter.dimensions[dim] = dimensionFilter{
			boolState: cycleBoolFilter(m.filter.dimensions[dim].boolState),
		}
		return m.applyFilterChangeAndMaybeLoad()
	}
	return m, nil
}

func (m statsModel) filterClearAll() (statsModel, tea.Cmd) {
	for i := range filterDimCount {
		m.filter.dimensions[i] = dimensionFilter{}
	}
	m, cmd := m.applyFilterChangeAndMaybeLoad()
	return m.closeFilterOverlay(), cmd
}

func (m statsModel) filterStartRegex(dim filterDimension) (statsModel, tea.Cmd) {
	if filterDimensionIsBool(dim) {
		return m, nil
	}
	m.filter.regexEditing = true
	m.filter.regexInput.SetValue(m.filter.dimensions[dim].regex)
	m.filter.regexInput.Focus()
	return m, textinput.Blink
}

func (m statsModel) handleFilterExpandedKey(msg tea.KeyPressMsg) (statsModel, tea.Cmd) {
	dim := filterDimension(m.filter.expanded)
	values := m.filter.values[dim]

	if updated, cmd, handled := m.handleFilterExpandedNav(msg, len(values)); handled {
		return updated, cmd
	}
	return m.handleFilterExpandedAction(msg, dim, values)
}

func (m statsModel) handleFilterExpandedNav(msg tea.KeyPressMsg, count int) (statsModel, tea.Cmd, bool) {
	if isFilterCollapseKey(msg) {
		m.filter.expanded = -1
		return m, nil, true
	}
	switch {
	case msg.Text == "j" || msg.Code == tea.KeyDown:
		if m.filter.expandedCursor < count-1 {
			m.filter.expandedCursor++
		}
		return m, nil, true
	case msg.Text == "k" || msg.Code == tea.KeyUp:
		if m.filter.expandedCursor > 0 {
			m.filter.expandedCursor--
		}
		return m, nil, true
	}
	return m, nil, false
}

func (m statsModel) handleFilterExpandedAction(
	msg tea.KeyPressMsg,
	dim filterDimension,
	values []string,
) (statsModel, tea.Cmd) {
	switch msg.Text {
	case " ":
		return m.filterToggleValue(dim, values)
	case "/":
		m.filter.regexEditing = true
		m.filter.expanded = -1
		m.filter.regexInput.SetValue(m.filter.dimensions[dim].regex)
		m.filter.regexInput.Focus()
		return m, textinput.Blink
	case "x":
		m.filter.dimensions[dim] = dimensionFilter{}
		m.filter.expanded = -1
		return m.applyFilterChangeAndMaybeLoad()
	}
	return m, nil
}

func (m statsModel) filterToggleValue(
	dim filterDimension,
	values []string,
) (statsModel, tea.Cmd) {
	if m.filter.expandedCursor >= len(values) {
		return m, nil
	}
	value := values[m.filter.expandedCursor]
	filter := m.filter.dimensions[dim]
	if filter.useRegex {
		filter = dimensionFilter{}
	}
	if filter.selected == nil {
		filter.selected = make(map[string]bool)
	}
	if filter.selected[value] {
		delete(filter.selected, value)
	} else {
		filter.selected[value] = true
	}
	filter.useRegex = false
	m.filter.dimensions[dim] = filter
	return m.applyFilterChangeAndMaybeLoad()
}

func (m statsModel) handleFilterRegexKey(msg tea.KeyPressMsg) (statsModel, tea.Cmd) {
	dim := filterDimension(m.filter.cursor)

	switch msg.Code {
	case tea.KeyEnter:
		return m.filterApplyRegex(dim)
	case tea.KeyEscape:
		m.filter.regexEditing = false
		m.filter.regexInput.Blur()
		return m, nil
	}

	var cmd tea.Cmd
	m.filter.regexInput, cmd = m.filter.regexInput.Update(msg)
	return m, cmd
}

func (m statsModel) filterApplyRegex(dim filterDimension) (statsModel, tea.Cmd) {
	regex := m.filter.regexInput.Value()
	if regex != "" {
		re, err := regexp.Compile(regex)
		if err == nil {
			m.filter.dimensions[dim] = dimensionFilter{
				useRegex:   true,
				regex:      regex,
				compiledRe: re,
			}
			returned, cmd := m.applyFilterChangeAndMaybeLoad()
			m = returned
			m.filter.regexEditing = false
			m.filter.regexInput.Blur()
			return m, cmd
		}
	}
	m.filter.regexEditing = false
	m.filter.regexInput.Blur()
	return m, nil
}
