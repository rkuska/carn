package canonical

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

func deleteSQLiteSearchIndexForCacheKeys(
	ctx context.Context,
	tx *sql.Tx,
	cacheKeys []string,
) error {
	if len(cacheKeys) == 0 {
		return nil
	}

	query, args := sqliteSearchChunksByCacheKeyQuery(
		`INSERT INTO search_fts(search_fts, rowid, text)
		 SELECT 'delete', sc.id, sc.text
		   FROM search_chunks sc
		   JOIN conversations c ON c.id = sc.conversation_id`,
		cacheKeys,
	)
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("tx.ExecContext: %w", err)
	}
	return nil
}

func populateSQLiteSearchIndexForCacheKeys(
	ctx context.Context,
	tx *sql.Tx,
	cacheKeys []string,
) error {
	if len(cacheKeys) == 0 {
		return nil
	}

	query, args := sqliteSearchChunksByCacheKeyQuery(
		`INSERT INTO search_fts(rowid, text)
		 SELECT sc.id, sc.text
		   FROM search_chunks sc
		   JOIN conversations c ON c.id = sc.conversation_id`,
		cacheKeys,
	)
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("tx.ExecContext: %w", err)
	}
	return nil
}

func sqliteSearchChunksByCacheKeyQuery(prefix string, cacheKeys []string) (string, []any) {
	placeholders := make([]string, 0, len(cacheKeys))
	args := make([]any, 0, len(cacheKeys))
	for _, key := range cacheKeys {
		placeholders = append(placeholders, "?")
		args = append(args, key)
	}

	var query strings.Builder
	query.Grow(len(prefix) + len(placeholders)*3 + 48)
	query.WriteString(prefix)
	query.WriteString(` WHERE c.cache_key IN (`)
	query.WriteString(strings.Join(placeholders, ", "))
	query.WriteString(`) ORDER BY sc.id`)
	return query.String(), args
}
