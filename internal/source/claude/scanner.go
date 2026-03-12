package claude

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

const (
	claudeProjectsDir   = ".claude/projects"
	maxFirstMessage     = 200
	maxToolResultChars  = 500
	blockTypeText       = "text"
	jsonlScanBufferSize = 512 * 1024
	jsonlSlugBufferSize = 64 * 1024
)

type sessionFile struct {
	path         string
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

func jsonlLines(r io.Reader, bufferSize int) iter.Seq2[[]byte, error] {
	return func(yield func([]byte, error) bool) {
		br := bufio.NewReaderSize(r, bufferSize)
		var overflow []byte

		yieldLine := func(line []byte) bool {
			line = bytes.TrimRight(line, "\n\r")
			if len(line) == 0 {
				return true
			}
			return yield(line, nil)
		}

		for {
			line, err := br.ReadSlice('\n')
			if err == bufio.ErrBufferFull {
				overflow = append(overflow[:0], line...)
				for err == bufio.ErrBufferFull {
					var more []byte
					more, err = br.ReadSlice('\n')
					overflow = append(overflow, more...)
				}
				line = overflow
			}

			if len(line) > 0 && !yieldLine(line) {
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

func scanSessions(ctx context.Context, baseDir string) ([]scannedSession, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("scanSessions_ctx: %w", err)
	}

	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, fmt.Errorf("os.ReadDir: %w", err)
	}

	log := zerolog.Ctx(ctx)
	var files []sessionFile
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("scanSessions_ctx: %w", err)
		}
		if !entry.IsDir() {
			continue
		}

		projDir := filepath.Join(baseDir, entry.Name())
		proj := projectFromDirName(entry.Name())
		projectFiles, err := discoverProjectSessionFiles(
			projDir,
			project{DisplayName: proj.displayName},
			proj.dirName,
		)
		if err != nil {
			log.Warn().Err(err).Msgf("discoverProjectSessionFiles failed for %s", projDir)
			continue
		}
		files = append(files, projectFiles...)
	}

	sessions, err := scanSessionFilesParallel(ctx, files)
	if err != nil {
		return nil, fmt.Errorf("scanSessionFilesParallel: %w", err)
	}
	return sessions, nil
}

func scanSessionFilesParallel(ctx context.Context, files []sessionFile) ([]scannedSession, error) {
	log := zerolog.Ctx(ctx)
	results := make([]scannedSessionResult, len(files))
	if len(files) == 0 {
		return nil, nil
	}

	sem := semaphore.NewWeighted(int64(runtime.NumCPU()))
	group, groupCtx := errgroup.WithContext(ctx)

	for i := range files {
		index := i
		file := files[i]

		group.Go(func() error {
			if err := sem.Acquire(groupCtx, 1); err != nil {
				return fmt.Errorf("sem.Acquire_%s: %w", file.path, err)
			}
			defer sem.Release(1)

			scanned, err := scanSessionFile(groupCtx, file)
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return fmt.Errorf("scanSessionFile_%s: %w", file.path, err)
				}
				log.Debug().Err(err).Msgf("skipping %s", file.path)
				return nil
			}

			results[index] = scannedSessionResult{session: scanned, ok: true}
			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return nil, fmt.Errorf("errgroup.Wait: %w", err)
	}

	sessions := make([]scannedSession, 0, len(files))
	for _, result := range results {
		if result.ok {
			sessions = append(sessions, result.session)
		}
	}
	return sessions, nil
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
		return scannedSession{}, fmt.Errorf("scanMetadataResult: %w", err)
	}
	result.meta.IsSubagent = file.isSubagent
	result.groupKey = buildConversationGroupKey(file, result.meta)
	return result, nil
}

func buildConversationGroupKey(file sessionFile, meta sessionMeta) groupKey {
	if file.isSubagent || meta.Slug == "" {
		return groupKey{dirName: file.groupDirName, slug: file.path}
	}
	return groupKey{dirName: file.groupDirName, slug: meta.Slug}
}

func discoverProjectSessionFiles(projDir string, proj project, groupDirName string) ([]sessionFile, error) {
	mainFiles, err := filepath.Glob(filepath.Join(projDir, "*.jsonl"))
	if err != nil {
		return nil, fmt.Errorf("filepath.Glob_main: %w", err)
	}

	subagentFiles, err := filepath.Glob(filepath.Join(projDir, "*/subagents/agent-*.jsonl"))
	if err != nil {
		return nil, fmt.Errorf("filepath.Glob_subagent: %w", err)
	}

	files := make([]sessionFile, 0, len(mainFiles)+len(subagentFiles))
	for _, path := range mainFiles {
		files = append(files, sessionFile{
			path:         path,
			project:      proj,
			groupDirName: groupDirName,
		})
	}
	for _, path := range subagentFiles {
		files = append(files, sessionFile{
			path:         path,
			project:      proj,
			groupDirName: groupDirName,
			isSubagent:   true,
		})
	}

	return files, nil
}

func displayNameFromCWD(cwd string) string {
	parts := strings.Split(filepath.ToSlash(cwd), "/")
	if len(parts) >= 2 {
		return strings.Join(parts[len(parts)-2:], "/")
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return cwd
}

func truncate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}

func truncatePreserveNewlines(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\r", "")
	if len(s) > maxLen {
		return s[:maxLen] + "\n..."
	}
	return s
}
