package elements

import "fmt"

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

// VersionInfo returns a formatted version string for CLI output.
func VersionInfo() string {
	return fmt.Sprintf("carn %s (%s, %s)", version, commit, date)
}

func SetVersionDetails(versionValue, commitValue, dateValue string) {
	version = versionValue
	commit = commitValue
	date = dateValue
}

func Version() string {
	return version
}

func Commit() string {
	return commit
}
