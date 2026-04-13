package canonical

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"

	conv "github.com/rkuska/carn/internal/conversation"
)

func (s *Store) QueryPerformanceSequence(
	ctx context.Context,
	archiveDir string,
	cacheKeys []string,
) ([]conv.PerformanceSequenceSession, error) {
	rows, err := queryStoreStatsRows(s, ctx, archiveDir, cacheKeys, readStatsPerformanceSequence)
	if err != nil {
		return nil, fmt.Errorf("queryStatsRows_performanceSequence: %w", err)
	}
	return rows, nil
}

func (s *Store) QueryTurnMetrics(
	ctx context.Context,
	archiveDir string,
	cacheKeys []string,
) ([]conv.SessionTurnMetrics, error) {
	rows, err := queryStoreStatsRows(s, ctx, archiveDir, cacheKeys, readStatsTurnMetrics)
	if err != nil {
		return nil, fmt.Errorf("queryStatsRows_turnMetrics: %w", err)
	}
	return rows, nil
}

func (s *Store) QueryActivityBuckets(
	ctx context.Context,
	archiveDir string,
	cacheKeys []string,
) ([]conv.ActivityBucketRow, error) {
	rows, err := queryStoreStatsRows(s, ctx, archiveDir, cacheKeys, readStatsActivityBuckets)
	if err != nil {
		return nil, fmt.Errorf("queryStatsRows_activityBuckets: %w", err)
	}
	return rows, nil
}

func queryStoreStatsRows[T any](
	store *Store,
	ctx context.Context,
	archiveDir string,
	cacheKeys []string,
	read func(context.Context, *sql.DB, []string) ([]T, error),
) ([]T, error) {
	if len(cacheKeys) == 0 {
		return nil, nil
	}

	db, err := store.loadDB(ctx, archiveDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("loadDB: %w", err)
	}

	rows, err := read(ctx, db, cacheKeys)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}
	return rows, nil
}

func (s *Store) loadDB(ctx context.Context, archiveDir string) (*sql.DB, error) {
	path := canonicalStorePath(archiveDir)
	if db, ok := s.cachedDB(path); ok {
		return db, nil
	}

	exists, err := pathExists(path)
	if err != nil {
		return nil, fmt.Errorf("pathExists: %w", err)
	}
	if !exists {
		return nil, fs.ErrNotExist
	}

	db, err := openSQLiteDB(ctx, path, true)
	if err != nil {
		return nil, fmt.Errorf("openSQLiteDB: %w", err)
	}
	return s.cacheDB(path, db)
}

func (s *Store) needsRebuild(ctx context.Context, archiveDir string) (bool, error) {
	path := canonicalStorePath(archiveDir)
	exists, err := pathExists(path)
	if err != nil {
		return true, fmt.Errorf("pathExists: %w", err)
	}
	if !exists {
		return true, nil
	}

	db, err := s.loadDB(ctx, archiveDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return true, nil
		}
		return true, fmt.Errorf("loadDB: %w", err)
	}

	meta, err := readSQLiteMeta(ctx, db)
	if err != nil {
		return true, fmt.Errorf("readSQLiteMeta: %w", err)
	}
	return !sqliteMetaCurrent(meta), nil
}
