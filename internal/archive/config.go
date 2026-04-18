package archive

import (
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
	Copied        int
	Skipped       int
	Failed        int
	Elapsed       time.Duration
	StoreBuilt    bool
	Drift         src.ProviderDriftReports
	MalformedData src.ProviderMalformedDataReports

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

func (c Config) SourceDirFor(provider conv.Provider) string {
	if c.SourceDirs == nil {
		return ""
	}
	return c.SourceDirs[provider]
}
