package canonical

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/rs/zerolog"

	conv "github.com/rkuska/carn/internal/conversation"
)

func writeCanonicalStoreAtomically(
	ctx context.Context,
	archiveDir string,
	conversations []conversation,
	transcripts map[string]sessionFull,
	corpus searchCorpus,
	statsData map[string][]conv.SessionStatsData,
	activityBucketRows map[string][]conv.ActivityBucketRow,
) error {
	return withCanonicalStoreTempDB(ctx, archiveDir, func(tempPath string) error {
		db, err := openSQLiteDB(ctx, tempPath, false)
		if err != nil {
			return fmt.Errorf("openSQLiteDB: %w", err)
		}
		defer func() {
			if closeErr := db.Close(); closeErr != nil {
				zerolog.Ctx(ctx).Warn().Err(closeErr).Msg("db.Close")
			}
		}()

		if err = configureSQLiteBulkLoadDB(ctx, db); err != nil {
			return fmt.Errorf("configureSQLiteBulkLoadDB: %w", err)
		}

		counts, err := replaceSQLiteStoreContents(
			ctx,
			db,
			conversations,
			transcripts,
			corpus,
			statsData,
			activityBucketRows,
		)
		if err != nil {
			return fmt.Errorf("replaceSQLiteStoreContents: %w", err)
		}
		if err := validateSQLiteStoreCounts(ctx, db, counts); err != nil {
			return fmt.Errorf("validateSQLiteStoreCounts: %w", err)
		}
		if err := db.Close(); err != nil {
			return fmt.Errorf("db.Close: %w", err)
		}
		if err := replaceSQLiteFile(tempPath, canonicalStorePath(archiveDir)); err != nil {
			return fmt.Errorf("replaceSQLiteFile: %w", err)
		}
		return nil
	})
}

func withCanonicalStoreTempDB(
	ctx context.Context,
	archiveDir string,
	run func(tempPath string) error,
) error {
	storeDir := canonicalStoreDir(archiveDir)
	if err := os.MkdirAll(storeDir, 0o755); err != nil {
		return fmt.Errorf("os.MkdirAll: %w", err)
	}

	tempFile, err := os.CreateTemp(storeDir, "canonical-*.sqlite")
	if err != nil {
		return fmt.Errorf("os.CreateTemp: %w", err)
	}
	tempPath := tempFile.Name()
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("tempFile.Close: %w", err)
	}
	defer func() {
		if err := os.Remove(tempPath); err != nil {
			zerolog.Ctx(ctx).Warn().Err(err).Msg("os.Remove")
		}
	}()
	if err := run(tempPath); err != nil {
		return err
	}
	return nil
}

func configureSQLiteBulkLoadDB(ctx context.Context, db *sql.DB) error {
	for _, stmt := range []string{
		"PRAGMA journal_mode = MEMORY",
		"PRAGMA synchronous = OFF",
		"PRAGMA temp_store = MEMORY",
	} {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("db.ExecContext_%s: %w", stmt, err)
		}
	}
	return nil
}

func clearSQLiteStoreTables(ctx context.Context, tx *sql.Tx) error {
	for _, stmt := range []string{
		`DELETE FROM stats_performance_sequence`,
		`DELETE FROM stats_turn_metrics`,
		`DELETE FROM stats_activity_buckets`,
		`DELETE FROM conversation_sessions`,
		`DELETE FROM search_chunks`,
		`DELETE FROM conversations`,
		`DELETE FROM meta`,
	} {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("tx.ExecContext_clear: %w", err)
		}
	}
	return nil
}

func finalizeSQLiteStoreContents(ctx context.Context, tx *sql.Tx) error {
	if err := populateSQLiteSearchIndex(ctx, tx); err != nil {
		return fmt.Errorf("populateSQLiteSearchIndex: %w", err)
	}
	if err := ensureSQLiteSearchTriggers(ctx, tx); err != nil {
		return fmt.Errorf("ensureSQLiteSearchTriggers: %w", err)
	}
	if err := writeSQLiteMeta(ctx, tx); err != nil {
		return fmt.Errorf("writeSQLiteMeta: %w", err)
	}
	return nil
}

func prepareSQLiteConversationStatements(
	ctx context.Context,
	tx *sql.Tx,
) (*sql.Stmt, *sql.Stmt, error) {
	convStmt, err := tx.PrepareContext(ctx, `INSERT INTO conversations(
		provider, provider_id, cache_key, name, project_display_name,
		first_timestamp_ns, last_timestamp_ns, first_message, plan_count,
		total_input_tokens, total_cache_creation_tokens, total_cache_read_tokens, total_output_tokens,
		total_reasoning_output_tokens,
		total_message_count, total_main_message_count, transcript_blob
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return nil, nil, fmt.Errorf("tx.PrepareContext_conversations: %w", err)
	}

	sessionStmt, err := tx.PrepareContext(ctx, `INSERT INTO conversation_sessions(
		conversation_id, ordinal, session_id, slug, timestamp_ns, last_timestamp_ns,
		cwd, git_branch, version, model, first_message, message_count, main_message_count,
		user_message_count, assistant_message_count,
		file_path, input_tokens, cache_creation_input_tokens, cache_read_input_tokens,
		output_tokens, reasoning_output_tokens,
		tool_counts_json, tool_error_counts_json, tool_reject_counts_json,
		action_counts_json, action_error_counts_json, action_reject_counts_json,
		performance_meta_json, is_subagent
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		if closeErr := convStmt.Close(); closeErr != nil {
			zerolog.Ctx(ctx).Warn().Err(closeErr).Msg("convStmt.Close")
		}
		return nil, nil, fmt.Errorf("tx.PrepareContext_sessions: %w", err)
	}
	return convStmt, sessionStmt, nil
}
