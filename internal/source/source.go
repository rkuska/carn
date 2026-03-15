package source

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	conv "github.com/rkuska/carn/internal/conversation"
)

var (
	ErrResumeTargetIDEmpty = errors.New("resume target id is empty")
	ErrResumeDirEmpty      = errors.New("resume directory is empty")
	ErrResumeDirNotDir     = errors.New("resume directory is not a directory")
)

// Analysis describes provider-local import state before syncing raw files.
type Analysis struct {
	UnitsTotal       int
	FilesInspected   int
	Conversations    int
	NewConversations int
	ToUpdate         int
	UpToDate         int
	SyncCandidates   []string
}

// SyncCandidate is a backend-owned raw-file sync plan item.
type SyncCandidate struct {
	SourcePath string
	DestPath   string
	DestExists bool
}

// Progress reports provider-local analysis progress using provider-neutral terms.
type Progress struct {
	Provider         conv.Provider
	UnitsCompleted   int
	UnitsTotal       int
	FilesInspected   int
	Conversations    int
	NewConversations int
	ToUpdate         int
	CurrentUnit      string
	Err              error
}

// Backend is the generic provider contract used by archive, canonical, and app.
type Backend interface {
	Provider() conv.Provider
	Scan(ctx context.Context, rawDir string) ([]conv.Conversation, error)
	Load(ctx context.Context, conversation conv.Conversation) (conv.Session, error)
	Analyze(ctx context.Context, sourceDir, rawDir string, onProgress func(Progress)) (Analysis, error)
	SyncCandidates(ctx context.Context, sourceDir, rawDir string) ([]SyncCandidate, error)
	ResumeCommand(target conv.ResumeTarget) (*exec.Cmd, error)
}

// IncrementalLookup provides canonical-store conversation lookups for
// provider-local targeted rebuild resolution.
type IncrementalLookup interface {
	ConversationByFilePath(ctx context.Context, provider conv.Provider, filePath string) (conv.Conversation, bool, error)
	ConversationBySessionID(ctx context.Context, provider conv.Provider, sessionID string) (conv.Conversation, bool, error)
	ConversationByCacheKey(ctx context.Context, cacheKey string) (conv.Conversation, bool, error)
}

// IncrementalResolution describes the exact conversations that should be
// replaced during a targeted canonical rebuild.
type IncrementalResolution struct {
	Conversations    []conv.Conversation
	ReplaceCacheKeys []string
}

// IncrementalResolver is an optional provider hook for targeted canonical
// rebuilds driven by changed raw paths.
type IncrementalResolver interface {
	ResolveIncremental(
		ctx context.Context,
		rawDir string,
		changedRawPaths []string,
		lookup IncrementalLookup,
	) (IncrementalResolution, error)
}

// Dedupe returns a stable deduplicated copy of values.
func Dedupe[T comparable](values []T) []T {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[T]struct{}, len(values))
	deduped := make([]T, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		deduped = append(deduped, value)
	}
	return deduped
}

// DedupeAndSort returns a sorted, deduplicated copy of values, skipping empty strings.
func DedupeAndSort(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	deduped := make([]string, 0, len(values))
	for _, value := range Dedupe(values) {
		if value == "" {
			continue
		}
		deduped = append(deduped, value)
	}
	sort.Strings(deduped)
	return deduped
}

// SortedKeys returns the keys of a set as a sorted slice.
func SortedKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// FileNeedsSync returns true if the destination is missing, different, or older.
func FileNeedsSync(srcInfo os.FileInfo, dstPath string) bool {
	dstInfo, err := os.Stat(dstPath)
	if err != nil {
		return true
	}
	if srcInfo.Size() != dstInfo.Size() {
		return true
	}
	return srcInfo.ModTime().After(dstInfo.ModTime())
}

// StatDir returns whether path exists and is a directory.
func StatDir(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}

// ProviderRawDir returns the provider-specific raw archive path.
func ProviderRawDir(archiveDir string, provider conv.Provider) string {
	return filepath.Join(archiveDir, string(provider), "raw")
}

// ValidateResumeTarget applies the shared strict resume policy.
func ValidateResumeTarget(target conv.ResumeTarget) error {
	if target.ID == "" {
		return fmt.Errorf("validateResumeTarget: %w", ErrResumeTargetIDEmpty)
	}
	if target.CWD == "" {
		return fmt.Errorf("validateResumeTarget: %w", ErrResumeDirEmpty)
	}

	info, err := os.Stat(target.CWD)
	if err != nil {
		return fmt.Errorf("validateResumeTarget_osStat: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("validateResumeTarget: %w", ErrResumeDirNotDir)
	}
	return nil
}
