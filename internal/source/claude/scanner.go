package claude

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"iter"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"

	src "github.com/rkuska/carn/internal/source"
)

var metadataReaderPool = sync.Pool{
	New: func() any { return bufio.NewReaderSize(nil, jsonlMetadataBufferSize) },
}

var parseReaderPool = sync.Pool{
	New: func() any { return bufio.NewReaderSize(nil, jsonlParseBufferSize) },
}

var slugReaderPool = sync.Pool{
	New: func() any { return bufio.NewReaderSize(nil, jsonlSlugBufferSize) },
}

const (
	maxFirstMessage         = 200
	maxToolResultChars      = 500
	blockTypeText           = "text"
	blockTypeToolUse        = "tool_use"
	blockTypeThinking       = "thinking"
	jsonlMetadataBufferSize = 64 * 1024
	jsonlParseBufferSize    = 32 * 1024
	jsonlSlugBufferSize     = 32 * 1024
)

type sessionFile struct {
	path         string
	relPath      string
	srcInfo      fs.FileInfo
	project      project
	groupDirName string
	isSubagent   bool
}

type scannedSessionResult struct {
	session scannedSession
	ok      bool
}

type parsedSessionMessagesResult struct {
	messages []parsedMessage
	ok       bool
}

type parsedSessionProjectionResult struct {
	messages []message
	usage    tokenUsage
	ok       bool
}

func jsonlLines(br *bufio.Reader) iter.Seq2[[]byte, error] {
	return func(yield func([]byte, error) bool) {
		var overflow []byte

		for {
			line, nextOverflow, err := readJSONLLine(br, overflow)
			overflow = nextOverflow
			if len(line) > 0 && !yield(line, nil) {
				return
			}
			if err != nil {
				if err != io.EOF {
					yield(nil, err)
				}
				return
			}
		}
	}
}

func scanSessions(
	ctx context.Context,
	baseDir string,
) ([]scannedSession, src.DriftReport, src.MalformedDataReport, error) {
	if err := ctx.Err(); err != nil {
		return nil, src.DriftReport{}, src.MalformedDataReport{}, fmt.Errorf("scanSessions_ctx: %w", err)
	}

	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, src.DriftReport{}, src.MalformedDataReport{}, fmt.Errorf("os.ReadDir: %w", err)
	}

	log := zerolog.Ctx(ctx)
	var files []sessionFile
	for _, entry := range entries {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, src.DriftReport{}, src.MalformedDataReport{}, fmt.Errorf("scanSessions_ctx: %w", ctxErr)
		}
		if !entry.IsDir() {
			continue
		}

		projDir := filepath.Join(baseDir, entry.Name())
		proj := projectFromDirName(entry.Name())
		projectFiles, discoverErr := discoverProjectSessionFiles(
			projDir,
			project{DisplayName: proj.displayName},
			proj.dirName,
			baseDir,
		)
		if discoverErr != nil {
			log.Warn().Err(discoverErr).Msgf("discoverProjectSessionFiles failed for %s", projDir)
			continue
		}
		files = append(files, projectFiles...)
	}

	sessions, drift, malformedData, err := scanSessionFilesParallel(ctx, files)
	if err != nil {
		return nil, src.DriftReport{}, src.MalformedDataReport{}, fmt.Errorf("scanSessionFilesParallel: %w", err)
	}

	log.Info().Int("sessions", len(sessions)).Msg("claude source scan completed")
	return sessions, drift, malformedData, nil
}

func scanSessionFilesParallel(
	ctx context.Context,
	files []sessionFile,
) ([]scannedSession, src.DriftReport, src.MalformedDataReport, error) {
	log := zerolog.Ctx(ctx)
	results := make([]scannedSessionResult, len(files))
	malformedValues := make([]string, len(files))
	if len(files) == 0 {
		return nil, src.DriftReport{}, src.MalformedDataReport{}, nil
	}

	sem := semaphore.NewWeighted(int64(claudeScanParallelism(len(files))))
	group, groupCtx := errgroup.WithContext(ctx)

	for i := range files {
		index := i
		file := files[i]

		group.Go(func() error {
			if err := sem.Acquire(groupCtx, 1); err != nil {
				return fmt.Errorf("sem.Acquire_%s: %w", file.path, err)
			}
			defer sem.Release(1)

			result, malformedValue, err := scanSessionFileParallelResult(groupCtx, log, file)
			if err != nil {
				return err
			}
			results[index] = result
			malformedValues[index] = malformedValue
			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return nil, src.DriftReport{}, src.MalformedDataReport{}, fmt.Errorf("errgroup.Wait: %w", err)
	}

	sessions := make([]scannedSession, 0, len(files))
	drift := src.NewDriftReport()
	malformedData := src.NewMalformedDataReport()
	for i, result := range results {
		drift.Merge(result.session.drift)
		if result.ok {
			sessions = append(sessions, result.session)
		}
		malformedData.Record(malformedValues[i])
	}
	return sessions, drift, malformedData, nil
}

func scanSessionFileParallelResult(
	ctx context.Context,
	log *zerolog.Logger,
	file sessionFile,
) (scannedSessionResult, string, error) {
	scanned, err := scanSessionFile(ctx, file)
	if err == nil {
		return scannedSessionResult{session: scanned, ok: true}, "", nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return scannedSessionResult{}, "", fmt.Errorf("scanSessionFile_%s: %w", file.path, err)
	}

	malformedValue := ""
	if errors.Is(err, src.ErrMalformedRawData) {
		malformedValue = file.path
	}
	log.Debug().Err(err).Msgf("skipping %s", file.path)
	return scannedSessionResult{session: scanned}, malformedValue, nil
}

func projectFromDirName(dirName string) scannedProject {
	trimmed := strings.TrimPrefix(dirName, "-")
	display := dirName

	parts := strings.SplitN(trimmed, "-", 4)
	if len(parts) >= 3 {
		switch parts[0] {
		case "Users", "home":
			prefix := parts[0] + "-" + parts[1] + "-"
			rest := strings.TrimPrefix(trimmed, prefix)
			if rest != "" {
				display = rest
			}
		}
	}

	return scannedProject{
		dirName:     dirName,
		displayName: display,
	}
}

func scanSessionFile(ctx context.Context, file sessionFile) (scannedSession, error) {
	result, err := scanMetadataResult(ctx, file.path, file.project)
	if err != nil {
		return result, fmt.Errorf("scanMetadataResult: %w", err)
	}
	result.meta.IsSubagent = file.isSubagent
	result.groupKey = buildConversationGroupKey(file, result.meta)
	return result, nil
}

func buildConversationGroupKey(file sessionFile, meta sessionMeta) groupKey {
	if file.isSubagent || meta.Slug == "" {
		return groupKey{dirName: file.groupDirName, slug: filepath.ToSlash(file.relPath)}
	}
	return groupKey{dirName: file.groupDirName, slug: meta.Slug}
}

func discoverProjectSessionFiles(
	projDir string,
	proj project,
	groupDirName string,
	baseDir string,
) ([]sessionFile, error) {
	entries, err := os.ReadDir(projDir)
	if err != nil {
		return nil, fmt.Errorf("os.ReadDir: %w", err)
	}

	var files []sessionFile
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() {
			if strings.HasSuffix(name, ".jsonl") {
				info, err := entry.Info()
				if err != nil {
					return nil, fmt.Errorf("entry.Info_main: %w", err)
				}
				path := filepath.Join(projDir, name)
				relPath, err := filepath.Rel(baseDir, path)
				if err != nil {
					return nil, fmt.Errorf("filepath.Rel_main: %w", err)
				}
				files = append(files, sessionFile{
					path:         path,
					relPath:      relPath,
					srcInfo:      info,
					project:      proj,
					groupDirName: groupDirName,
				})
			}
			continue
		}

		subFiles, err := discoverSubagentFiles(
			filepath.Join(projDir, name, "subagents"),
			proj, groupDirName, baseDir,
		)
		if err != nil {
			return nil, err
		}
		files = append(files, subFiles...)
	}

	return files, nil
}

func discoverSubagentFiles(
	subagentDir string,
	proj project,
	groupDirName string,
	baseDir string,
) ([]sessionFile, error) {
	subEntries, err := os.ReadDir(subagentDir)
	if err != nil {
		return nil, nil // subagents dir may not exist
	}
	var files []sessionFile
	for _, sub := range subEntries {
		name := sub.Name()
		if sub.IsDir() || !strings.HasPrefix(name, "agent-") || !strings.HasSuffix(name, ".jsonl") {
			continue
		}
		info, err := sub.Info()
		if err != nil {
			return nil, fmt.Errorf("sub.Info: %w", err)
		}
		path := filepath.Join(subagentDir, name)
		relPath, err := filepath.Rel(baseDir, path)
		if err != nil {
			return nil, fmt.Errorf("filepath.Rel_subagent: %w", err)
		}
		files = append(files, sessionFile{
			path:         path,
			relPath:      relPath,
			srcInfo:      info,
			project:      proj,
			groupDirName: groupDirName,
			isSubagent:   true,
		})
	}
	return files, nil
}
