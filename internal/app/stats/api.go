package stats

import (
	"context"

	el "github.com/rkuska/carn/internal/app/elements"
	conv "github.com/rkuska/carn/internal/conversation"
)

type Model = statsModel
type Store = browserStore

func NewModel(
	ctx context.Context,
	archiveDir string,
	conversations []conv.Conversation,
	store Store,
	width, height int,
	filter el.FilterState,
	theme *el.Theme,
) Model {
	model := newStatsModelWithTheme(conversations, store, width, height, filter, theme)
	model.ctx = ctx
	model.archiveDir = archiveDir
	return model.applyFilterChange()
}
