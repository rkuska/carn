package app

import (
	"testing"

	"github.com/stretchr/testify/assert"

	el "github.com/rkuska/carn/internal/app/elements"
)

func TestVersionInfoSyncsElementsBuildMetadata(t *testing.T) {
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
