package app

import (
	"testing"
	"time"

	conv "github.com/rkuska/carn/internal/conversation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testFilterConversation(provider conv.Provider, project, model, branch string) conv.Conversation {
	return conv.Conversation{
		Ref:     conv.Ref{Provider: provider, ID: project + "-" + model},
		Project: conv.Project{DisplayName: project},
		Sessions: []conv.SessionMeta{
			{
				ID:        project + "-" + model,
				Model:     model,
				GitBranch: branch,
				Project:   conv.Project{DisplayName: project},
				Timestamp: time.Now(),
			},
		},
	}
}

func testFilterConversationWithPlans(provider conv.Provider, project string, planCount int) conv.Conversation {
	c := testFilterConversation(provider, project, "opus", "main")
	c.PlanCount = planCount
	return c
}

func testFilterConversationMultiPart(provider conv.Provider, project string) conv.Conversation {
	c := testFilterConversation(provider, project, "opus", "main")
	c.Sessions = append(c.Sessions, conv.SessionMeta{
		ID:        project + "-part2",
		Model:     "opus",
		Timestamp: time.Now(),
	})
	return c
}

func TestExtractFilterValuesCollectsDistinctSortedValues(t *testing.T) {
	t.Parallel()

	conversations := []conv.Conversation{
		testFilterConversation(conv.ProviderClaude, "alpha", "opus", "main"),
		testFilterConversation(conv.ProviderCodex, "beta", "sonnet", "feat/x"),
		testFilterConversation(conv.ProviderClaude, "alpha", "haiku", "main"),
		testFilterConversation(conv.ProviderClaude, "gamma", "opus", "develop"),
	}

	values := extractFilterValues(conversations)

	assert.Equal(t, []string{"Claude", "Codex"}, values[filterDimProvider])
	assert.Equal(t, []string{"alpha", "beta", "gamma"}, values[filterDimProject])
	assert.Equal(t, []string{"haiku", "opus", "sonnet"}, values[filterDimModel])
	assert.Equal(t, []string{"develop", "feat/x", "main"}, values[filterDimGitBranch])
	assert.Equal(t, []string{"yes", "no"}, values[filterDimHasPlans])
	assert.Equal(t, []string{"yes", "no"}, values[filterDimMultiPart])
}

func TestExtractFilterValuesSkipsEmptyFields(t *testing.T) {
	t.Parallel()

	conversations := []conv.Conversation{
		testFilterConversation(conv.ProviderClaude, "alpha", "", ""),
	}

	values := extractFilterValues(conversations)

	assert.Equal(t, []string{"Claude"}, values[filterDimProvider])
	assert.Equal(t, []string{"alpha"}, values[filterDimProject])
	assert.Empty(t, values[filterDimModel])
	assert.Empty(t, values[filterDimGitBranch])
}

func TestApplyStructuredFiltersNoActiveReturnsAll(t *testing.T) {
	t.Parallel()

	conversations := []conv.Conversation{
		testFilterConversation(conv.ProviderClaude, "alpha", "opus", "main"),
		testFilterConversation(conv.ProviderCodex, "beta", "sonnet", "feat"),
	}

	var dims [filterDimCount]dimensionFilter
	result := applyStructuredFilters(conversations, dims)
	assert.Len(t, result, 2)
}

func TestApplyStructuredFiltersSingleDimensionExactMatch(t *testing.T) {
	t.Parallel()

	conversations := []conv.Conversation{
		testFilterConversation(conv.ProviderClaude, "alpha", "opus", "main"),
		testFilterConversation(conv.ProviderCodex, "beta", "sonnet", "feat"),
		testFilterConversation(conv.ProviderClaude, "gamma", "haiku", "main"),
	}

	var dims [filterDimCount]dimensionFilter
	dims[filterDimProvider] = dimensionFilter{
		selected: map[string]bool{"Claude": true},
	}

	result := applyStructuredFilters(conversations, dims)
	require.Len(t, result, 2)
	assert.Equal(t, "Claude", result[0].Ref.Provider.Label())
	assert.Equal(t, "Claude", result[1].Ref.Provider.Label())
}

func TestApplyStructuredFiltersMultipleValuesWithinDimensionOR(t *testing.T) {
	t.Parallel()

	conversations := []conv.Conversation{
		testFilterConversation(conv.ProviderClaude, "alpha", "opus", "main"),
		testFilterConversation(conv.ProviderCodex, "beta", "sonnet", "feat"),
		testFilterConversation(conv.ProviderClaude, "gamma", "haiku", "main"),
	}

	var dims [filterDimCount]dimensionFilter
	dims[filterDimProject] = dimensionFilter{
		selected: map[string]bool{"alpha": true, "beta": true},
	}

	result := applyStructuredFilters(conversations, dims)
	require.Len(t, result, 2)
	assert.Equal(t, "alpha", result[0].Project.DisplayName)
	assert.Equal(t, "beta", result[1].Project.DisplayName)
}

func TestApplyStructuredFiltersMultipleDimensionsAND(t *testing.T) {
	t.Parallel()

	conversations := []conv.Conversation{
		testFilterConversation(conv.ProviderClaude, "alpha", "opus", "main"),
		testFilterConversation(conv.ProviderCodex, "beta", "sonnet", "feat"),
		testFilterConversation(conv.ProviderClaude, "gamma", "haiku", "main"),
	}

	var dims [filterDimCount]dimensionFilter
	dims[filterDimProvider] = dimensionFilter{
		selected: map[string]bool{"Claude": true},
	}
	dims[filterDimProject] = dimensionFilter{
		selected: map[string]bool{"alpha": true},
	}

	result := applyStructuredFilters(conversations, dims)
	require.Len(t, result, 1)
	assert.Equal(t, "alpha", result[0].Project.DisplayName)
}

func TestApplyStructuredFiltersRegexMode(t *testing.T) {
	t.Parallel()

	conversations := []conv.Conversation{
		testFilterConversation(conv.ProviderClaude, "alpha-api", "opus", "main"),
		testFilterConversation(conv.ProviderClaude, "beta-web", "sonnet", "feat"),
		testFilterConversation(conv.ProviderClaude, "gamma-api", "haiku", "main"),
	}

	var dims [filterDimCount]dimensionFilter
	dims[filterDimProject] = dimensionFilter{
		useRegex: true,
		regex:    ".*-api$",
	}

	result := applyStructuredFilters(conversations, dims)
	require.Len(t, result, 2)
	assert.Equal(t, "alpha-api", result[0].Project.DisplayName)
	assert.Equal(t, "gamma-api", result[1].Project.DisplayName)
}

func TestApplyStructuredFiltersInvalidRegexPassesAll(t *testing.T) {
	t.Parallel()

	conversations := []conv.Conversation{
		testFilterConversation(conv.ProviderClaude, "alpha", "opus", "main"),
		testFilterConversation(conv.ProviderCodex, "beta", "sonnet", "feat"),
	}

	var dims [filterDimCount]dimensionFilter
	dims[filterDimProject] = dimensionFilter{
		useRegex: true,
		regex:    "[invalid",
	}

	result := applyStructuredFilters(conversations, dims)
	assert.Len(t, result, 2)
}

func TestApplyStructuredFiltersBooleanHasPlans(t *testing.T) {
	t.Parallel()

	conversations := []conv.Conversation{
		testFilterConversationWithPlans(conv.ProviderClaude, "alpha", 3),
		testFilterConversationWithPlans(conv.ProviderClaude, "beta", 0),
		testFilterConversationWithPlans(conv.ProviderCodex, "gamma", 1),
	}

	tests := []struct {
		name      string
		state     boolFilterState
		wantCount int
	}{
		{name: "any", state: boolFilterAny, wantCount: 3},
		{name: "yes", state: boolFilterYes, wantCount: 2},
		{name: "no", state: boolFilterNo, wantCount: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var dims [filterDimCount]dimensionFilter
			dims[filterDimHasPlans] = dimensionFilter{boolState: tt.state}
			result := applyStructuredFilters(conversations, dims)
			assert.Len(t, result, tt.wantCount)
		})
	}
}

func TestApplyStructuredFiltersBooleanMultiPart(t *testing.T) {
	t.Parallel()

	conversations := []conv.Conversation{
		testFilterConversationMultiPart(conv.ProviderClaude, "alpha"),
		testFilterConversation(conv.ProviderClaude, "beta", "opus", "main"),
	}

	var dims [filterDimCount]dimensionFilter
	dims[filterDimMultiPart] = dimensionFilter{boolState: boolFilterYes}

	result := applyStructuredFilters(conversations, dims)
	require.Len(t, result, 1)
	assert.Equal(t, "alpha", result[0].Project.DisplayName)
}

func TestCycleBoolFilter(t *testing.T) {
	t.Parallel()

	assert.Equal(t, boolFilterYes, cycleBoolFilter(boolFilterAny))
	assert.Equal(t, boolFilterNo, cycleBoolFilter(boolFilterYes))
	assert.Equal(t, boolFilterAny, cycleBoolFilter(boolFilterNo))
}

func TestFilterBadgesFormatsActiveDimensions(t *testing.T) {
	t.Parallel()

	var dims [filterDimCount]dimensionFilter
	dims[filterDimProvider] = dimensionFilter{
		selected: map[string]bool{"Claude": true},
	}
	dims[filterDimHasPlans] = dimensionFilter{boolState: boolFilterYes}

	badges := filterBadges(dims)
	require.Len(t, badges, 2)
	assert.Equal(t, "provider:Claude", badges[0])
	assert.Equal(t, "has plans:yes", badges[1])
}

func TestFilterBadgesRegexFormat(t *testing.T) {
	t.Parallel()

	var dims [filterDimCount]dimensionFilter
	dims[filterDimProject] = dimensionFilter{
		useRegex: true,
		regex:    "alpha.*",
	}

	badges := filterBadges(dims)
	require.Len(t, badges, 1)
	assert.Equal(t, "project:/alpha.*/", badges[0])
}

func TestFilterBadgesMultipleSelectedSorted(t *testing.T) {
	t.Parallel()

	var dims [filterDimCount]dimensionFilter
	dims[filterDimModel] = dimensionFilter{
		selected: map[string]bool{"sonnet": true, "haiku": true},
	}

	badges := filterBadges(dims)
	require.Len(t, badges, 1)
	assert.Equal(t, "model:haiku,sonnet", badges[0])
}

func TestFilterBadgesEmptyWhenNoFilters(t *testing.T) {
	t.Parallel()

	var dims [filterDimCount]dimensionFilter
	badges := filterBadges(dims)
	assert.Empty(t, badges)
}

func TestDimensionFilterIsActive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		filter dimensionFilter
		want   bool
	}{
		{name: "empty", filter: dimensionFilter{}, want: false},
		{name: "selected", filter: dimensionFilter{selected: map[string]bool{"x": true}}, want: true},
		{name: "regex", filter: dimensionFilter{useRegex: true, regex: "x"}, want: true},
		{name: "regex empty", filter: dimensionFilter{useRegex: true, regex: ""}, want: false},
		{name: "bool yes", filter: dimensionFilter{boolState: boolFilterYes}, want: true},
		{name: "bool no", filter: dimensionFilter{boolState: boolFilterNo}, want: true},
		{name: "bool any", filter: dimensionFilter{boolState: boolFilterAny}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.filter.isActive())
		})
	}
}

func TestFilterMatchCountReflectsActiveFilters(t *testing.T) {
	t.Parallel()

	conversations := []conv.Conversation{
		testFilterConversation(conv.ProviderClaude, "alpha", "opus", "main"),
		testFilterConversation(conv.ProviderCodex, "beta", "sonnet", "feat"),
		testFilterConversation(conv.ProviderClaude, "gamma", "haiku", "main"),
	}

	state := newBrowserFilterState()
	assert.Equal(t, 3, state.matchCount(conversations))

	state.dimensions[filterDimProvider] = dimensionFilter{
		selected: map[string]bool{"Claude": true},
	}
	assert.Equal(t, 2, state.matchCount(conversations))
}

func TestFilterFooterItemsCategoricalDimOmitsSpaceToggle(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	b.filter.active = true
	b.filter.cursor = int(filterDimProvider)
	items := b.filterFooterItems()
	keys := helpItemKeys(items)

	assert.Contains(t, keys, "enter")
	assert.Contains(t, keys, "/")
	assert.NotContains(t, keys, "space")
}

func TestFilterFooterItemsBoolDimOmitsEnterSelectAndRegex(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	b.filter.active = true
	b.filter.cursor = int(filterDimHasPlans)
	items := b.filterFooterItems()
	keys := helpItemKeys(items)

	assert.Contains(t, keys, "space")
	assert.NotContains(t, keys, "enter")
	assert.NotContains(t, keys, "/")
}

func TestFilterFooterItemsShowsClearOnlyWhenActive(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	b.filter.active = true
	b.filter.cursor = int(filterDimProvider)

	keys := helpItemKeys(b.filterFooterItems())
	assert.NotContains(t, keys, "x")
	assert.NotContains(t, keys, "X")

	b.filter.dimensions[filterDimProvider] = dimensionFilter{
		selected: map[string]bool{"Claude": true},
	}
	keys = helpItemKeys(b.filterFooterItems())
	assert.Contains(t, keys, "x")
	assert.Contains(t, keys, "X")
}

func TestFilterFooterItemsExpandedState(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	b.filter.active = true
	b.filter.expanded = int(filterDimProject)
	items := b.filterFooterItems()
	keys := helpItemKeys(items)

	assert.Equal(t, []string{"j/k", "space", "enter", "/", "x", "esc"}, keys)
}

func TestFilterFooterItemsRegexState(t *testing.T) {
	t.Parallel()

	b := testBrowser(t)
	b.filter.active = true
	b.filter.regexEditing = true
	items := b.filterFooterItems()
	keys := helpItemKeys(items)

	assert.Equal(t, []string{"enter", "esc"}, keys)
}

func TestFilterDimensionLabels(t *testing.T) {
	t.Parallel()

	for i := range filterDimCount {
		label := filterDimensionLabel(filterDimension(i))
		assert.NotEmpty(t, label, "dimension %d should have a label", i)
	}
}

func TestFilterDimensionIsBool(t *testing.T) {
	t.Parallel()

	assert.False(t, filterDimensionIsBool(filterDimProvider))
	assert.False(t, filterDimensionIsBool(filterDimProject))
	assert.False(t, filterDimensionIsBool(filterDimModel))
	assert.False(t, filterDimensionIsBool(filterDimGitBranch))
	assert.True(t, filterDimensionIsBool(filterDimHasPlans))
	assert.True(t, filterDimensionIsBool(filterDimMultiPart))
}
