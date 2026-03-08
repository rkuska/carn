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
	archiveDir string // ~/.local/share/carn
}

type syncResult struct {
	copied  int
	skipped int
	failed  int
	elapsed time.Duration
	files   []syncFileResult

	storeBuilt bool
}

type syncProgress struct {
	current int
	total   int
	file    string // current file being copied (basename)
	copied  int
	failed  int

	stage string
}

func defaultArchiveConfig() (archiveConfig, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return archiveConfig{}, fmt.Errorf("os.UserHomeDir: %w", err)
	}

	sourceDir := os.Getenv("CARN_SOURCE_DIR")
	if sourceDir == "" {
		sourceDir = filepath.Join(home, claudeProjectsDir)
	}

	archiveDir := os.Getenv("CARN_ARCHIVE_DIR")
	if archiveDir == "" {
		archiveDir = filepath.Join(home, ".local", "share", "carn")
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

	filesToSync, err := collectFilesToSync(cfg)
	if err != nil {
		return syncResult{}, fmt.Errorf("collectFilesToSync: %w", err)
	}

	total := len(filesToSync)
	if total == 0 {
		result := syncResult{elapsed: time.Since(start)}
		log.Info().Msgf("archive sync: nothing to copy (all up to date)")
		return result, nil
	}

	result, err := syncFiles(ctx, cfg, filesToSync, onProgress)
	if err != nil {
		return syncResult{}, fmt.Errorf("syncFiles: %w", err)
	}

	log.Info().
		Int("copied", result.copied).
		Int("skipped", result.skipped).
		Int("failed", result.failed).
		Dur("elapsed", result.elapsed).
		Msgf("archive sync complete")

	return result, nil
}

func syncFiles(
	ctx context.Context,
	cfg archiveConfig,
	filesToSync []string,
	onProgress func(syncProgress),
) (syncResult, error) {
	start := time.Now()
	total := len(filesToSync)
	if total == 0 {
		return syncResult{elapsed: time.Since(start)}, nil
	}

	log := zerolog.Ctx(ctx)
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
						copied:  int(copied.Load()),
						failed:  int(failed.Load()),
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
					copied:  int(copied.Load()),
					failed:  int(failed.Load()),
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

	return syncResult{
		copied:  int(copied.Load()),
		skipped: 0,
		failed:  int(failed.Load()),
		elapsed: time.Since(start),
	}, nil
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
	sourceDir         string
	archiveDir        string
	filesInspected    int
	projects          int
	conversations     int      // total unique grouped conversations
	newConversations  int      // conversations with no archived files
	toUpdate          int      // conversations where source file is missing/stale in archive
	upToDate          int      // conversations fully synced
	filesToSync       []string // raw file paths needing copy (new + stale)
	legacyFilesToSync []string
	storeNeedsBuild   bool
	err               error
}

func (a importAnalysis) needsSync() bool {
	if a.err != nil {
		return false
	}
	return len(a.filesToSync) > 0 ||
		len(a.legacyFilesToSync) > 0 ||
		a.storeNeedsBuild
}

func (a importAnalysis) queuedFileCount() int {
	return len(a.filesToSync) + len(a.legacyFilesToSync)
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
		if rec.Slug != "" {
			return rec.Slug, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("scanner.Err: %w", err)
	}

	return "", nil
}

type classifiedFile struct {
	gk        groupKey
	needsSync bool
	dstExists bool
	srcPath   string
}

func classifyProjectFile(file sessionFile, cfg archiveConfig, dirName string) (classifiedFile, bool) {
	slug, slugErr := extractSessionSlug(file.path)
	if slugErr != nil {
		return classifiedFile{}, false
	}

	var gk groupKey
	if file.isSubagent || slug == "" {
		gk = groupKey{dirName: dirName, slug: file.path}
	} else {
		gk = groupKey{dirName: dirName, slug: slug}
	}

	rel, relErr := filepath.Rel(cfg.sourceDir, file.path)
	if relErr != nil {
		return classifiedFile{}, false
	}
	dstPath := filepath.Join(
		providerRawDir(cfg.archiveDir, conversationProviderClaude),
		rel,
	)

	info, statErr := os.Stat(file.path)
	if statErr != nil {
		return classifiedFile{}, false
	}

	needsSync := fileNeedsSync(info, dstPath)
	dstExists := true
	if _, dstErr := os.Stat(dstPath); os.IsNotExist(dstErr) {
		dstExists = false
	}

	return classifiedFile{
		gk:        gk,
		needsSync: needsSync,
		dstExists: dstExists,
		srcPath:   file.path,
	}, true
}

// analyzeProjectDir processes a single project directory during streaming analysis.
// It globs .jsonl files, classifies each one, and updates the running state.
func analyzeProjectDir(
	projDir string,
	cfg archiveConfig,
	seen map[groupKey]*conversationState,
	syncCandidates *[]string,
) (filesInspected int, err error) {
	dirName := filepath.Base(projDir)
	files, err := discoverProjectSessionFiles(
		projDir,
		project{displayName: projectFromDirName(dirName).displayName},
		dirName,
	)
	if err != nil {
		return 0, fmt.Errorf("discoverProjectSessionFiles: %w", err)
	}

	for _, file := range files {
		filesInspected++

		classified, ok := classifyProjectFile(file, cfg, dirName)
		if !ok {
			continue
		}

		state, exists := seen[classified.gk]
		if !exists {
			state = &conversationState{}
			seen[classified.gk] = state
		}

		if classified.needsSync {
			if !classified.dstExists && !state.hasUpToDate && !state.hasStale {
				state.allNew = true
			}
			state.hasStale = true
			state.allNew = state.allNew && !state.hasUpToDate
			*syncCandidates = append(*syncCandidates, classified.srcPath)
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
