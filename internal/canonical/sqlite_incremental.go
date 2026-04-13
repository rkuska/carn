package canonical

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/rs/zerolog"

	conv "github.com/rkuska/carn/internal/conversation"
	src "github.com/rkuska/carn/internal/source"
)

func applySQLiteIncrementalRebuild(
	ctx context.Context,
	db *sql.DB,
	replaceCacheKeys []string,
	conversations []conversation,
	transcripts map[string]sessionFull,
	groupedUnits map[string][]searchUnit,
	statsData map[string][]conv.SessionStatsData,
	activityBucketRows map[string][]conv.ActivityBucketRow,
) error {
	if err := ensureSQLiteSchemaBase(ctx, db); err != nil {
		return fmt.Errorf("ensureSQLiteSchemaBase: %w", err)
	}
	replaceCacheKeys = src.DedupeAndSort(replaceCacheKeys)

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("db.BeginTx: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			zerolog.Ctx(ctx).Warn().Err(err).Msg("tx.Rollback")
		}
	}()

	if err := applySQLiteIncrementalRebuildTx(
		ctx,
		tx,
		replaceCacheKeys,
		conversations,
		transcripts,
		groupedUnits,
		statsData,
		activityBucketRows,
	); err != nil {
		return fmt.Errorf("applySQLiteIncrementalRebuildTx: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("tx.Commit: %w", err)
	}
	return nil
}

func applySQLiteIncrementalRebuildTx(
	ctx context.Context,
	tx *sql.Tx,
	replaceCacheKeys []string,
	conversations []conversation,
	transcripts map[string]sessionFull,
	groupedUnits map[string][]searchUnit,
	statsData map[string][]conv.SessionStatsData,
	activityBucketRows map[string][]conv.ActivityBucketRow,
) error {
	if err := dropSQLiteSearchTriggers(ctx, tx); err != nil {
		return fmt.Errorf("dropSQLiteSearchTriggers: %w", err)
	}
	if err := deleteSQLiteSearchIndexForCacheKeys(ctx, tx, replaceCacheKeys); err != nil {
		return fmt.Errorf("deleteSQLiteSearchIndexForCacheKeys: %w", err)
	}
	if err := deleteStatsByCacheKeys(ctx, tx, replaceCacheKeys); err != nil {
		return fmt.Errorf("deleteStatsByCacheKeys: %w", err)
	}
	if err := deleteSQLiteConversations(ctx, tx, replaceCacheKeys); err != nil {
		return fmt.Errorf("deleteSQLiteConversations: %w", err)
	}
	if _, err := insertSQLiteConversations(
		ctx,
		tx,
		conversations,
		transcripts,
		groupedUnits,
		statsData,
		activityBucketRows,
	); err != nil {
		return fmt.Errorf("insertSQLiteConversations: %w", err)
	}
	if err := populateSQLiteSearchIndexForCacheKeys(ctx, tx, replaceCacheKeys); err != nil {
		return fmt.Errorf("populateSQLiteSearchIndexForCacheKeys: %w", err)
	}
	if err := ensureSQLiteSearchTriggers(ctx, tx); err != nil {
		return fmt.Errorf("ensureSQLiteSearchTriggers: %w", err)
	}
	if err := writeSQLiteMeta(ctx, tx); err != nil {
		return fmt.Errorf("writeSQLiteMeta: %w", err)
	}
	return nil
}

func deleteSQLiteConversations(ctx context.Context, tx *sql.Tx, cacheKeys []string) error {
	if len(cacheKeys) == 0 {
		return nil
	}

	args := make([]any, 0, len(cacheKeys))
	placeholders := make([]string, 0, len(cacheKeys))
	for _, key := range cacheKeys {
		placeholders = append(placeholders, "?")
		args = append(args, key)
	}

	query := fmt.Sprintf(
		`DELETE FROM conversations WHERE cache_key IN (%s)`,
		strings.Join(placeholders, ", "),
	)
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("tx.ExecContext: %w", err)
	}
	return nil
}
