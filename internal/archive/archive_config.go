package archive

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	conv "github.com/rkuska/carn/internal/conversation"
)

const claudeProjectsDir = ".claude/projects"
const codexSessionsDir = ".codex/sessions"

type Config struct {
	SourceDir      string
	CodexSourceDir string
	ArchiveDir     string
}

type ImportAnalysis struct {
	SourceDir        string
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

func DefaultConfig() (Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}, fmt.Errorf("defaultConfig_os.UserHomeDir: %w", err)
	}

	sourceDir := os.Getenv("CARN_CLAUDE_SOURCE_DIR")
	if sourceDir == "" {
		sourceDir = os.Getenv("CARN_SOURCE_DIR")
	}
	if sourceDir == "" {
		sourceDir = filepath.Join(home, claudeProjectsDir)
	}

	codexSourceDir := os.Getenv("CARN_CODEX_SOURCE_DIR")
	if codexSourceDir == "" {
		codexSourceDir = filepath.Join(home, codexSessionsDir)
	}

	archiveDir := os.Getenv("CARN_ARCHIVE_DIR")
	if archiveDir == "" {
		archiveDir = filepath.Join(home, ".local", "share", "carn")
	}

	return Config{
		SourceDir:      sourceDir,
		CodexSourceDir: codexSourceDir,
		ArchiveDir:     archiveDir,
	}, nil
}

func (c Config) SourceDirFor(provider conv.Provider) string {
	switch provider {
	case conv.ProviderClaude:
		return c.SourceDir
	case conv.ProviderCodex:
		return c.CodexSourceDir
	default:
		return ""
	}
}
