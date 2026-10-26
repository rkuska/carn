package stats

import (
	"context"
	"time"

	el "github.com/rkuska/carn/internal/app/elements"
	conv "github.com/rkuska/carn/internal/conversation"
)

type Model = statsModel
type Store = browserStore

// NewModel builds the stats view. now anchors the relative time ranges
// (7d, 30d, 90d); the app passes time.Now, tests pass a fixed clock.
func NewModel(
	ctx context.Context,
	archiveDir string,
	conversations []conv.Conversation,
	store Store,
	width, height int,
	filter el.FilterState,
	theme *el.Theme,
	now func() time.Time,
) Model {
	model := newStatsModelWithTheme(conversations, store, width, height, filter, theme, now)
	model.ctx = ctx
	model.archiveDir = archiveDir
	return model.applyFilterChange()
}
