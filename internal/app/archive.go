package app

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

type archiveConfig struct {
	sourceDir  string // ~/.claude/projects
	archiveDir string // ~/.local/share/cldrsrch
}

type syncResult struct {
	copied  int
	skipped int
	failed  int
	elapsed time.Duration
}

type syncProgress struct {
	current int
	total   int
	file    string // current file being copied (basename)
}

func defaultArchiveConfig() (archiveConfig, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return archiveConfig{}, fmt.Errorf("os.UserHomeDir: %w", err)
	}

	sourceDir := os.Getenv("CLDSRCH_SOURCE_DIR")
	if sourceDir == "" {
		sourceDir = filepath.Join(home, claudeProjectsDir)
	}

	archiveDir := os.Getenv("CLDSRCH_ARCHIVE_DIR")
	if archiveDir == "" {
		archiveDir = filepath.Join(home, ".local", "share", "cldrsrch")
	}

	return archiveConfig{
		sourceDir:  sourceDir,
		archiveDir: archiveDir,
	}, nil
}

func syncArchive(ctx context.Context, cfg archiveConfig, onProgress func(syncProgress)) (syncResult, error) {
	log := zerolog.Ctx(ctx)
	start := time.Now()

	// Check if source dir exists
	if _, err := os.Stat(cfg.sourceDir); os.IsNotExist(err) {
		log.Warn().Msgf("source directory does not exist: %s", cfg.sourceDir)
		return syncResult{elapsed: time.Since(start)}, nil
	}

	// Phase 1: Walk source to collect .jsonl files needing sync
	var filesToSync []string
	err := filepath.WalkDir(cfg.sourceDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip inaccessible entries
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".jsonl") {
			return nil
		}

		rel, err := filepath.Rel(cfg.sourceDir, path)
		if err != nil {
			return nil
		}
		dstPath := filepath.Join(cfg.archiveDir, rel)

		info, err := d.Info()
		if err != nil {
			return nil
		}

		if fileNeedsSync(info, dstPath) {
			filesToSync = append(filesToSync, path)
		}

		return nil
	})
	if err != nil {
		return syncResult{}, fmt.Errorf("filepath.WalkDir: %w", err)
	}

	total := len(filesToSync)
	if total == 0 {
		result := syncResult{elapsed: time.Since(start)}
		log.Info().Msgf("archive sync: nothing to copy (all up to date)")
		return result, nil
	}

	// Phase 2: Copy concurrently
	var copied atomic.Int64
	var failed atomic.Int64
	var completed atomic.Int64
	var progressMu sync.Mutex

	sem := semaphore.NewWeighted(int64(runtime.NumCPU()))
	g, gctx := errgroup.WithContext(ctx)

	for _, src := range filesToSync {
		g.Go(func() error {
			if err := sem.Acquire(gctx, 1); err != nil {
				return fmt.Errorf("sem.Acquire: %w", err)
			}
			defer sem.Release(1)

			rel, _ := filepath.Rel(cfg.sourceDir, src)
			dst := filepath.Join(cfg.archiveDir, rel)

			if err := copyFile(src, dst); err != nil {
				log.Warn().Err(err).Msgf("failed to copy %s", src)
				failed.Add(1)
				if onProgress != nil {
					progress := syncProgress{
						current: int(completed.Add(1)),
						total:   total,
						file:    filepath.Base(src),
					}
					progressMu.Lock()
					onProgress(progress)
					progressMu.Unlock()
				}
				return nil // continue other copies
			}

			copied.Add(1)

			if onProgress != nil {
				progress := syncProgress{
					current: int(completed.Add(1)),
					total:   total,
					file:    filepath.Base(src),
				}
				progressMu.Lock()
				onProgress(progress)
				progressMu.Unlock()
			}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return syncResult{}, fmt.Errorf("errgroup.Wait: %w", err)
	}

	result := syncResult{
		copied:  int(copied.Load()),
		skipped: 0,
		failed:  int(failed.Load()),
		elapsed: time.Since(start),
	}

	log.Info().
		Int("copied", result.copied).
		Int("skipped", result.skipped).
		Int("failed", result.failed).
		Dur("elapsed", result.elapsed).
		Msgf("archive sync complete")

	return result, nil
}

func fileNeedsSync(srcInfo os.FileInfo, dstPath string) bool {
	dstInfo, err := os.Stat(dstPath)
	if err != nil {
		return true // dst missing
	}
	if srcInfo.Size() != dstInfo.Size() {
		return true // size differs
	}
	if srcInfo.ModTime().After(dstInfo.ModTime()) {
		return true // src is newer
	}
	return false
}

// importAnalysis holds the final result of the streaming analysis.
type importAnalysis struct {
	sourceDir        string
	archiveDir       string
	filesInspected   int
	projects         int
	conversations    int      // total unique grouped conversations
	newConversations int      // conversations with no archived files
	toUpdate         int      // conversations where source file is missing/stale in archive
	upToDate         int      // conversations fully synced
	filesToSync      []string // raw file paths needing copy (new + stale)
}

func (a importAnalysis) needsSync() bool {
	return len(a.filesToSync) > 0
}

// importProgress is emitted during streaming analysis.
type importProgress struct {
	filesInspected   int
	conversations    int // discovered so far
	newConversations int
	toUpdate         int
	currentProject   string // project being analyzed
	err              error
}

// conversationState tracks the sync classification of a conversation during analysis.
type conversationState struct {
	hasUpToDate bool // at least one file is fully synced in archive
	hasStale    bool // at least one source file needs sync
	allNew      bool // no archived files exist at all for this conversation
}

// extractSessionSlug reads only the first user record from a JSONL file
// and returns the slug. Uses extractType for fast record identification
// and targeted JSON unmarshal for just the slug field.
func extractSessionSlug(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("os.Open: %w", err)
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 512*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		if extractType(line) != "user" {
			continue
		}

		var rec struct {
			Slug string `json:"slug"`
		}
		if err := json.Unmarshal(line, &rec); err != nil {
			return "", fmt.Errorf("json.Unmarshal: %w", err)
		}
		return rec.Slug, nil
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("scanner.Err: %w", err)
	}

	return "", nil
}

// analyzeProjectDir processes a single project directory during streaming analysis.
// It globs .jsonl files, classifies each one, and updates the running state.
func analyzeProjectDir(
	projDir string,
	cfg archiveConfig,
	seen map[groupKey]*conversationState,
	syncCandidates *[]string,
) (filesInspected int, err error) {
	proj := filepath.Base(projDir)

	// Glob main session files
	mainFiles, err := filepath.Glob(filepath.Join(projDir, "*.jsonl"))
	if err != nil {
		return 0, fmt.Errorf("filepath.Glob_main: %w", err)
	}

	// Glob subagent files
	subFiles, err := filepath.Glob(filepath.Join(projDir, "*/subagents/agent-*.jsonl"))
	if err != nil {
		return 0, fmt.Errorf("filepath.Glob_subagent: %w", err)
	}

	allFiles := make([]string, 0, len(mainFiles)+len(subFiles))
	allFiles = append(allFiles, mainFiles...)
	allFiles = append(allFiles, subFiles...)

	for _, f := range allFiles {
		filesInspected++

		// Determine if this is a subagent
		isSubagent := strings.Contains(f, "/subagents/")

		// Extract slug
		slug, slugErr := extractSessionSlug(f)
		if slugErr != nil {
			// Skip files we can't read
			continue
		}

		// Build group key — subagents and empty slugs get unique keys
		// (matching groupConversations logic)
		var gk groupKey
		if isSubagent || slug == "" {
			gk = groupKey{dirName: proj, slug: f} // unique per file
		} else {
			gk = groupKey{dirName: proj, slug: slug}
		}

		// Classify this file
		rel, relErr := filepath.Rel(cfg.sourceDir, f)
		if relErr != nil {
			continue
		}
		dstPath := filepath.Join(cfg.archiveDir, rel)

		info, statErr := os.Stat(f)
		if statErr != nil {
			continue
		}

		needsSync := fileNeedsSync(info, dstPath)

		state, exists := seen[gk]
		if !exists {
			state = &conversationState{}
			seen[gk] = state
		}

		if needsSync {
			// Check if dst exists at all
			if _, dstErr := os.Stat(dstPath); os.IsNotExist(dstErr) {
				if !state.hasUpToDate && !state.hasStale {
					state.allNew = true
				}
			}
			state.hasStale = true
			state.allNew = state.allNew && !state.hasUpToDate
			*syncCandidates = append(*syncCandidates, f)
		} else {
			state.hasUpToDate = true
			state.allNew = false
		}
	}

	return filesInspected, nil
}

// classifyConversations counts how many conversations are new, to-update, or up-to-date.
func classifyConversations(seen map[groupKey]*conversationState) (newConvs, toUpdate, upToDate int) {
	for _, state := range seen {
		switch {
		case state.allNew:
			newConvs++
		case state.hasStale:
			toUpdate++
		default:
			upToDate++
		}
	}
	return newConvs, toUpdate, upToDate
}

// listProjectDirs returns top-level directories in sourceDir.
func listProjectDirs(sourceDir string) ([]string, error) {
	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		return nil, nil
	}

	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return nil, fmt.Errorf("os.ReadDir: %w", err)
	}

	dirs := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, filepath.Join(sourceDir, e.Name()))
		}
	}
	return dirs, nil
}

// collectFilesToSync walks the source dir and returns paths of .jsonl files needing sync.
func collectFilesToSync(cfg archiveConfig) ([]string, error) {
	if _, err := statDir(cfg.sourceDir); err != nil {
		return nil, nil
	}

	var files []string
	err := filepath.WalkDir(cfg.sourceDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".jsonl") {
			return nil
		}

		rel, err := filepath.Rel(cfg.sourceDir, path)
		if err != nil {
			return nil
		}
		dstPath := filepath.Join(cfg.archiveDir, rel)

		info, err := d.Info()
		if err != nil {
			return nil
		}

		if fileNeedsSync(info, dstPath) {
			files = append(files, path)
		}
		return nil
	})

	return files, err
}

func statDir(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("os.Open: %w", err)
	}
	defer func() { _ = srcFile.Close() }()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("srcFile.Stat: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("os.MkdirAll: %w", err)
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("os.Create: %w", err)
	}
	defer func() { _ = dstFile.Close() }()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("io.Copy: %w", err)
	}

	if err := dstFile.Close(); err != nil {
		return fmt.Errorf("dstFile.Close: %w", err)
	}

	if err := os.Chtimes(dst, srcInfo.ModTime(), srcInfo.ModTime()); err != nil {
		return fmt.Errorf("os.Chtimes: %w", err)
	}

	return nil
}
