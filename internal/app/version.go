package app

import (
	"sync"

	el "github.com/rkuska/carn/internal/app/elements"
)

var (
	version       = "dev"
	commit        = "unknown"
	date          = "unknown"
	versionInfoMu sync.Mutex
)

func VersionInfo() string {
	syncVersionInfo()
	return el.VersionInfo()
}

func syncVersionInfo() {
	versionInfoMu.Lock()
	defer versionInfoMu.Unlock()

	el.SetVersionDetails(version, commit, date)
}
