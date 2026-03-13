package archive

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	conv "github.com/rkuska/carn/internal/conversation"
	src "github.com/rkuska/carn/internal/source"
)

type Config struct {
	SourceDirs map[conv.Provider]string
	ArchiveDir string
}

type ImportAnalysis struct {
	ArchiveDir       string
	FilesInspected   int
	Projects         int
	Conversations    int
	NewConversations int
	ToUpdate         int
	UpToDate         int
	QueuedFiles      []string
	StoreNeedsBuild  bool
	Err              error
}

func (a ImportAnalysis) NeedsSync() bool {
	if a.Err != nil {
		return false
	}
	return len(a.QueuedFiles) > 0 || a.StoreNeedsBuild
}

func (a ImportAnalysis) QueuedFileCount() int {
	return len(a.QueuedFiles)
}

type ImportProgress struct {
	Provider          conv.Provider
	ProjectsCompleted int
	ProjectsTotal     int
	FilesInspected    int
	Conversations     int
	NewConversations  int
	ToUpdate          int
	CurrentProject    string
	Err               error
}

type SyncResult struct {
	Copied     int
	Skipped    int
	Failed     int
	Elapsed    time.Duration
	StoreBuilt bool

	files []syncFileResult
}

type SyncActivity string

const (
	SyncActivitySyncingFiles    SyncActivity = "syncing_files"
	SyncActivityRebuildingStore SyncActivity = "rebuilding_store"
)

type SyncProgress struct {
	Provider conv.Provider
	Current  int
	Total    int
	File     string
	Copied   int
	Failed   int
	Activity SyncActivity
}

func DefaultConfig(backends ...src.Backend) (Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}, fmt.Errorf("defaultConfig_os.UserHomeDir: %w", err)
	}

	archiveDir := os.Getenv("CARN_ARCHIVE_DIR")
	if archiveDir == "" {
		archiveDir = filepath.Join(home, ".local", "share", "carn")
	}

	return Config{
		SourceDirs: resolveSourceDirs(home, backends...),
		ArchiveDir: archiveDir,
	}, nil
}

func (c Config) SourceDirFor(provider conv.Provider) string {
	if c.SourceDirs == nil {
		return ""
	}
	return c.SourceDirs[provider]
}

func resolveSourceDirs(home string, backends ...src.Backend) map[conv.Provider]string {
	sourceDirs := make(map[conv.Provider]string, len(backends))
	for _, backend := range backends {
		if backend == nil {
			continue
		}

		sourceDir := firstConfiguredSourceDir(backend, home)
		if sourceDir == "" {
			continue
		}
		sourceDirs[backend.Provider()] = sourceDir
	}
	return sourceDirs
}

func firstConfiguredSourceDir(backend src.Backend, home string) string {
	for _, envVar := range backend.SourceEnvVars() {
		if value := os.Getenv(envVar); value != "" {
			return value
		}
	}
	return backend.DefaultSourceDir(home)
}
