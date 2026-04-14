package app

import (
	"sync"
	"testing"

	el "github.com/rkuska/carn/internal/app/elements"
	"github.com/stretchr/testify/assert"
)

var versionStateMu sync.Mutex

func TestVersionInfoSyncsElementsBuildMetadata(t *testing.T) {
	t.Parallel()

	versionStateMu.Lock()
	t.Cleanup(versionStateMu.Unlock)

	originalVersion := version
	originalCommit := commit
	originalDate := date
	t.Cleanup(func() {
		version = originalVersion
		commit = originalCommit
		date = originalDate
		syncVersionInfo()
	})

	version = "1.2.3"
	commit = "deadbeef"
	date = "2026-04-14"

	assert.Equal(t, "carn 1.2.3 (deadbeef, 2026-04-14)", VersionInfo())
	assert.Equal(t, "carn 1.2.3 (deadbeef, 2026-04-14)", el.VersionInfo())
}
