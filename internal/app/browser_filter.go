package app

import (
	"regexp"
	"slices"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	conv "github.com/rkuska/carn/internal/conversation"
)

type filterDimension int

const (
	filterDimProvider filterDimension = iota
	filterDimProject
	filterDimModel
	filterDimGitBranch
	filterDimHasPlans
	filterDimMultiPart
	filterDimCount
)

const (
	boolValueYes = "yes"
	boolValueNo  = "no"
)

type boolFilterState int

const (
	boolFilterAny boolFilterState = iota
	boolFilterYes
	boolFilterNo
)

type dimensionFilter struct {
	selected  map[string]bool
	regex     string
	useRegex  bool
	boolState boolFilterState
}

func (f dimensionFilter) isActive() bool {
	if f.useRegex && f.regex != "" {
		return true
	}
	if f.boolState != boolFilterAny {
		return true
	}
	return len(f.selected) > 0
}

type browserFilterState struct {
	active         bool
	cursor         int
	expanded       int
	expandedCursor int
	expandedScroll int
	regexEditing   bool
	regexInput     textinput.Model
	dimensions     [filterDimCount]dimensionFilter
	values         [filterDimCount][]string
}

func newBrowserFilterState() browserFilterState {
	ti := textinput.New()
	ti.Prompt = "/"
	ti.CharLimit = 100
	ti.Blur()
	return browserFilterState{
		expanded:   -1,
		regexInput: ti,
	}
}

func (f browserFilterState) hasActiveFilters() bool {
	for i := range filterDimCount {
		if f.dimensions[i].isActive() {
			return true
		}
	}
	return false
}

func (f browserFilterState) matchCount(conversations []conv.Conversation) int {
	if !f.hasActiveFilters() {
		return len(conversations)
	}
	count := 0
	for _, c := range conversations {
		if matchesAllFilters(c, f.dimensions) {
			count++
		}
	}
	return count
}

func filterDimensionLabel(dim filterDimension) string {
	switch dim { //nolint:exhaustive // filterDimCount is a sentinel
	case filterDimProvider:
		return "Provider"
	case filterDimProject:
		return "Project"
	case filterDimModel:
		return "Model"
	case filterDimGitBranch:
		return "Git Branch"
	case filterDimHasPlans:
		return "Has Plans"
	case filterDimMultiPart:
		return "Multi-part"
	default:
		return ""
	}
}

func filterDimensionIsBool(dim filterDimension) bool {
	return dim == filterDimHasPlans || dim == filterDimMultiPart
}

func extractFilterValues(conversations []conv.Conversation) [filterDimCount][]string {
	var result [filterDimCount][]string
	sets := [filterDimCount]map[string]bool{}
	for i := range filterDimCount {
		sets[i] = make(map[string]bool)
	}

	for _, c := range conversations {
		if label := c.Ref.Provider.Label(); label != "" {
			sets[filterDimProvider][label] = true
		}
		if p := c.Project.DisplayName; p != "" {
			sets[filterDimProject][p] = true
		}
		if m := c.Model(); m != "" {
			sets[filterDimModel][m] = true
		}
		if b := c.GitBranch(); b != "" {
			sets[filterDimGitBranch][b] = true
		}
	}

	for i := range filterDimCount {
		if filterDimensionIsBool(filterDimension(i)) {
			result[i] = []string{boolValueYes, boolValueNo}
			continue
		}
		vals := make([]string, 0, len(sets[i]))
		for v := range sets[i] {
			vals = append(vals, v)
		}
		slices.Sort(vals)
		result[i] = vals
	}

	return result
}

func conversationDimensionValue(c conv.Conversation, dim filterDimension) string {
	switch dim { //nolint:exhaustive // filterDimCount is a sentinel
	case filterDimProvider:
		return c.Ref.Provider.Label()
	case filterDimProject:
		return c.Project.DisplayName
	case filterDimModel:
		return c.Model()
	case filterDimGitBranch:
		return c.GitBranch()
	case filterDimHasPlans:
		if c.PlanCount > 0 {
			return boolValueYes
		}
		return boolValueNo
	case filterDimMultiPart:
		if c.PartCount() > 1 {
			return boolValueYes
		}
		return boolValueNo
	default:
		return ""
	}
}

func matchesDimensionFilter(c conv.Conversation, dim filterDimension, f dimensionFilter) bool {
	if !f.isActive() {
		return true
	}

	value := conversationDimensionValue(c, dim)

	if filterDimensionIsBool(dim) {
		switch f.boolState {
		case boolFilterYes:
			return value == boolValueYes
		case boolFilterNo:
			return value == boolValueNo
		case boolFilterAny:
			return true
		}
	}

	if f.useRegex && f.regex != "" {
		re, err := regexp.Compile(f.regex)
		if err != nil {
			return true
		}
		return re.MatchString(value)
	}

	if len(f.selected) > 0 {
		return f.selected[value]
	}

	return true
}

func matchesAllFilters(c conv.Conversation, dims [filterDimCount]dimensionFilter) bool {
	for i := range filterDimCount {
		if !matchesDimensionFilter(c, filterDimension(i), dims[i]) {
			return false
		}
	}
	return true
}

func applyStructuredFilters(
	conversations []conv.Conversation,
	dims [filterDimCount]dimensionFilter,
) []conv.Conversation {
	hasActive := false
	for i := range filterDimCount {
		if dims[i].isActive() {
			hasActive = true
			break
		}
	}
	if !hasActive {
		return conversations
	}

	result := make([]conv.Conversation, 0, len(conversations))
	for _, c := range conversations {
		if matchesAllFilters(c, dims) {
			result = append(result, c)
		}
	}
	return result
}

func filterBadges(dims [filterDimCount]dimensionFilter) []string {
	badges := make([]string, 0, int(filterDimCount))
	for i := range filterDimCount {
		dim := filterDimension(i)
		f := dims[i]
		if !f.isActive() {
			continue
		}
		label := strings.ToLower(filterDimensionLabel(dim))
		if filterDimensionIsBool(dim) {
			switch f.boolState {
			case boolFilterYes:
				badges = append(badges, label+":"+boolValueYes)
			case boolFilterNo:
				badges = append(badges, label+":"+boolValueNo)
			case boolFilterAny:
				// not active, handled above
			}
			continue
		}
		if f.useRegex {
			badges = append(badges, label+":/"+f.regex+"/")
			continue
		}
		selected := make([]string, 0, len(f.selected))
		for v := range f.selected {
			selected = append(selected, v)
		}
		slices.Sort(selected)
		badges = append(badges, label+":"+strings.Join(selected, ","))
	}
	return badges
}

func (m browserModel) applyFilterChange(cmds *[]tea.Cmd) browserModel {
	filtered := applyStructuredFilters(m.mainConversations, m.filter.dimensions)
	m.search.baseConversations = filtered
	return m.refreshSearchResults(cmds)
}

func cycleBoolFilter(state boolFilterState) boolFilterState {
	switch state {
	case boolFilterAny:
		return boolFilterYes
	case boolFilterYes:
		return boolFilterNo
	case boolFilterNo:
		return boolFilterAny
	}
	return boolFilterAny
}
