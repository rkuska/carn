package app

import (
	"context"

	appbrowser "github.com/rkuska/carn/internal/app/browser"
)

func newTestBrowserModel(ctx context.Context, archiveDir string, store appbrowser.Store) appbrowser.Model {
	return appbrowser.NewModelWithStore(
		ctx,
		archiveDir,
		"",
		"dark",
		"2006-01-02 15:04",
		20,
		200,
		nil,
		store,
	).SetSize(120, 40)
}
