package elements

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

var versionDetailsMu sync.Mutex

func TestSetVersionDetailsUpdatesVersionInfo(t *testing.T) {
	t.Parallel()

	versionDetailsMu.Lock()
	t.Cleanup(versionDetailsMu.Unlock)

	originalVersion := version
	originalCommit := commit
	originalDate := date
	t.Cleanup(func() {
		SetVersionDetails(originalVersion, originalCommit, originalDate)
	})

	SetVersionDetails("1.2.3", "deadbeef", "2026-04-14")

	assert.Equal(t, "1.2.3", Version())
	assert.Equal(t, "deadbeef", Commit())
	assert.Equal(t, "carn 1.2.3 (deadbeef, 2026-04-14)", VersionInfo())
}
