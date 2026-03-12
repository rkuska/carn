package archive

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const claudeProjectsDir = ".claude/projects"

type Config struct {
	SourceDir  string
	ArchiveDir string
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

type SyncProgress struct {
	Current int
	Total   int
	File    string
	Copied  int
	Failed  int
	Stage   string
}

func DefaultConfig() (Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}, fmt.Errorf("defaultConfig_os.UserHomeDir: %w", err)
	}

	sourceDir := os.Getenv("CARN_SOURCE_DIR")
	if sourceDir == "" {
		sourceDir = filepath.Join(home, claudeProjectsDir)
	}

	archiveDir := os.Getenv("CARN_ARCHIVE_DIR")
	if archiveDir == "" {
		archiveDir = filepath.Join(home, ".local", "share", "carn")
	}

	return Config{
		SourceDir:  sourceDir,
		ArchiveDir: archiveDir,
	}, nil
}
