package elements

import (
	"regexp"
	"slices"
	"strings"

	"charm.land/bubbles/v2/textinput"

	conv "github.com/rkuska/carn/internal/conversation"
)

type FilterDimension int

const (
	FilterDimProvider FilterDimension = iota
	FilterDimProject
	FilterDimModel
	FilterDimGitBranch
	FilterDimHasPlans
	FilterDimMultiPart
	FilterDimCount
)

const (
	BoolValueYes = "yes"
	BoolValueNo  = "no"
)

type BoolFilterState int

const (
	BoolFilterAny BoolFilterState = iota
	BoolFilterYes
	BoolFilterNo
)

type DimensionFilter struct {
	Selected   map[string]bool
	Regex      string
	CompiledRe *regexp.Regexp
	UseRegex   bool
	BoolState  BoolFilterState
}

func (f DimensionFilter) IsActive() bool {
	if f.UseRegex && f.Regex != "" {
		return true
	}
	if f.BoolState != BoolFilterAny {
		return true
	}
	return len(f.Selected) > 0
}

type FilterState struct {
	Active         bool
	Cursor         int
	Expanded       int
	ExpandedCursor int
	ExpandedScroll int
	RegexEditing   bool
	RegexInput     textinput.Model
	Dimensions     [FilterDimCount]DimensionFilter
	Values         [FilterDimCount][]string
}

func NewFilterState() FilterState {
	ti := textinput.New()
	ti.Prompt = "/"
	ti.CharLimit = 100
	ti.Blur()
	return FilterState{
		Expanded:   -1,
		RegexInput: ti,
	}
}

func (f FilterState) HasActiveFilters() bool {
	for i := range FilterDimCount {
		if f.Dimensions[i].IsActive() {
			return true
		}
	}
	return false
}

func (f FilterState) MatchCount(conversations []conv.Conversation) int {
	if !f.HasActiveFilters() {
		return len(conversations)
	}
	count := 0
	for _, c := range conversations {
		if matchesAllFilters(c, f.Dimensions) {
			count++
		}
	}
	return count
}

func FilterDimensionLabel(dim FilterDimension) string {
	switch dim { //nolint:exhaustive // FilterDimCount is a sentinel
	case FilterDimProvider:
		return "Provider"
	case FilterDimProject:
		return "Project"
	case FilterDimModel:
		return "Model"
	case FilterDimGitBranch:
		return "Git Branch"
	case FilterDimHasPlans:
		return "Has Plans"
	case FilterDimMultiPart:
		return "Multi-part"
	default:
		return ""
	}
}

func FilterDimensionIsBool(dim FilterDimension) bool {
	return dim == FilterDimHasPlans || dim == FilterDimMultiPart
}

func ExtractFilterValues(conversations []conv.Conversation) [FilterDimCount][]string {
	var result [FilterDimCount][]string
	sets := [FilterDimCount]map[string]bool{}
	for i := range FilterDimCount {
		sets[i] = make(map[string]bool)
	}

	for _, c := range conversations {
		if label := c.Ref.Provider.Label(); label != "" {
			sets[FilterDimProvider][label] = true
		}
		if p := c.Project.DisplayName; p != "" {
			sets[FilterDimProject][p] = true
		}
		if m := c.Model(); m != "" {
			sets[FilterDimModel][m] = true
		}
		if b := c.GitBranch(); b != "" {
			sets[FilterDimGitBranch][b] = true
		}
	}

	for i := range FilterDimCount {
		if FilterDimensionIsBool(FilterDimension(i)) {
			result[i] = []string{BoolValueYes, BoolValueNo}
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

func conversationDimensionValue(c conv.Conversation, dim FilterDimension) string {
	switch dim { //nolint:exhaustive // FilterDimCount is a sentinel
	case FilterDimProvider:
		return c.Ref.Provider.Label()
	case FilterDimProject:
		return c.Project.DisplayName
	case FilterDimModel:
		return c.Model()
	case FilterDimGitBranch:
		return c.GitBranch()
	case FilterDimHasPlans:
		if c.PlanCount > 0 {
			return BoolValueYes
		}
		return BoolValueNo
	case FilterDimMultiPart:
		if c.PartCount() > 1 {
			return BoolValueYes
		}
		return BoolValueNo
	default:
		return ""
	}
}

func (f DimensionFilter) matchesRegex(value string) bool {
	re := f.CompiledRe
	if re == nil {
		var err error
		re, err = regexp.Compile(f.Regex)
		if err != nil {
			return true
		}
	}
	return re.MatchString(value)
}

func matchesDimensionFilter(c conv.Conversation, dim FilterDimension, f DimensionFilter) bool {
	if !f.IsActive() {
		return true
	}

	value := conversationDimensionValue(c, dim)

	if FilterDimensionIsBool(dim) {
		switch f.BoolState {
		case BoolFilterYes:
			return value == BoolValueYes
		case BoolFilterNo:
			return value == BoolValueNo
		case BoolFilterAny:
			return true
		}
	}

	if f.UseRegex && f.Regex != "" {
		return f.matchesRegex(value)
	}

	if len(f.Selected) > 0 {
		return f.Selected[value]
	}

	return true
}

func matchesAllFilters(c conv.Conversation, dims [FilterDimCount]DimensionFilter) bool {
	for i := range FilterDimCount {
		if !matchesDimensionFilter(c, FilterDimension(i), dims[i]) {
			return false
		}
	}
	return true
}

func ApplyStructuredFilters(
	conversations []conv.Conversation,
	dims [FilterDimCount]DimensionFilter,
) []conv.Conversation {
	hasActive := false
	for i := range FilterDimCount {
		if dims[i].IsActive() {
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

func FilterBadges(dims [FilterDimCount]DimensionFilter) []string {
	badges := make([]string, 0, int(FilterDimCount))
	for i := range FilterDimCount {
		dim := FilterDimension(i)
		f := dims[i]
		if !f.IsActive() {
			continue
		}
		label := strings.ToLower(FilterDimensionLabel(dim))
		if FilterDimensionIsBool(dim) {
			switch f.BoolState {
			case BoolFilterYes:
				badges = append(badges, label+":"+BoolValueYes)
			case BoolFilterNo:
				badges = append(badges, label+":"+BoolValueNo)
			case BoolFilterAny:
				// not active, handled above
			}
			continue
		}
		if f.UseRegex {
			badges = append(badges, label+":/"+f.Regex+"/")
			continue
		}
		selected := make([]string, 0, len(f.Selected))
		for v := range f.Selected {
			selected = append(selected, v)
		}
		slices.Sort(selected)
		badges = append(badges, label+":"+strings.Join(selected, ","))
	}
	return badges
}

func CycleBoolFilter(state BoolFilterState) BoolFilterState {
	switch state {
	case BoolFilterAny:
		return BoolFilterYes
	case BoolFilterYes:
		return BoolFilterNo
	case BoolFilterNo:
		return BoolFilterAny
	}
	return BoolFilterAny
}
