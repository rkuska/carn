package config

// DefaultTemplate returns a TOML string with all default values commented out
// and explanatory comments. Suitable for writing to a new config file.
func DefaultTemplate() string {
	return `# càrn configuration
# https://github.com/rkuska/carn

# Source and archive directory paths.
[paths]
# archive_dir = "~/.local/share/carn"
# claude_source_dir = "~/.claude/projects"
# codex_source_dir = "~/.codex/sessions"
# log_file = "/tmp/carn.log"

# Display preferences.
[display]
# Go time format for timestamps shown in headers and metadata.
# timestamp_format = "2006-01-02 15:04"

# Number of loaded transcripts to keep in the browser cache.
# browser_cache_size = 20

# Search behavior.
[search]
# Milliseconds to wait before triggering a deep search query.
# deep_search_debounce_ms = 200
`
}
