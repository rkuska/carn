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
) Model {
	model := newStatsModel(conversations, store, width, height, filter)
	model.ctx = ctx
	model.archiveDir = archiveDir
	return model.applyFilterChange()
}
