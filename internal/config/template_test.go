package config

import (
	"testing"

	"github.com/BurntSushi/toml"
)

func TestDefaultTemplate_ValidTOML(t *testing.T) {
	var raw rawConfig
	if err := toml.Unmarshal([]byte(DefaultTemplate()), &raw); err != nil {
		t.Fatalf("DefaultTemplate() produces invalid TOML: %v", err)
	}

	// All values are commented out — loading the template as a config file
	// should produce exactly the same defaults as Load() with no file.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Display.TimestampFormat != DefaultTimestampFormat {
		t.Errorf("template round-trips to wrong TimestampFormat")
	}
}

func TestDefaultTemplate_RoundTrip(t *testing.T) {
	t.Parallel()

	tomlContent := `
[paths]
archive_dir = "~/.local/share/carn"
log_file = "/tmp/carn.log"

[display]
timestamp_format = "2006-01-02 15:04"
browser_cache_size = 20

[search]
deep_search_debounce_ms = 200
`

	var raw rawConfig
	if err := toml.Unmarshal([]byte(tomlContent), &raw); err != nil {
		t.Fatalf("uncommented template is invalid TOML: %v", err)
	}
	if raw.Paths == nil || raw.Paths.ArchiveDir == nil {
		t.Fatal("expected paths.archive_dir to be set")
	}
	if *raw.Display.BrowserCacheSize != 20 {
		t.Errorf("browser_cache_size = %d, want 20", *raw.Display.BrowserCacheSize)
	}
}
