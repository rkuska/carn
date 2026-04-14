package app

import el "github.com/rkuska/carn/internal/app/elements"

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func VersionInfo() string {
	syncVersionInfo()
	return el.VersionInfo()
}

func syncVersionInfo() {
	el.SetVersionDetails(version, commit, date)
}
