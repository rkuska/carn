package canonical

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/rs/zerolog"
	_ "modernc.org/sqlite"
)

const (
	sqliteDriverName  = "sqlite"
	metaSchemaKey     = "schema_version"
	metaProjectionKey = "projection_version"
	metaSearchKey     = "search_version"
)

var sqliteSchemaStatements = []string{
	`CREATE TABLE IF NOT EXISTS meta (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS conversations (
		id INTEGER PRIMARY KEY,
		provider TEXT NOT NULL,
		provider_id TEXT NOT NULL,
		cache_key TEXT NOT NULL UNIQUE,
		name TEXT NOT NULL,
		project_display_name TEXT NOT NULL,
		first_timestamp_ns INTEGER NOT NULL,
		last_timestamp_ns INTEGER NOT NULL,
		first_message TEXT NOT NULL,
		plan_count INTEGER NOT NULL,
		total_input_tokens INTEGER NOT NULL,
		total_cache_creation_tokens INTEGER NOT NULL,
		total_cache_read_tokens INTEGER NOT NULL,
		total_output_tokens INTEGER NOT NULL,
		total_reasoning_output_tokens INTEGER NOT NULL DEFAULT 0,
		total_message_count INTEGER NOT NULL,
		total_main_message_count INTEGER NOT NULL,
		transcript_blob BLOB NOT NULL,
		UNIQUE(provider, provider_id)
	)`,
	`CREATE TABLE IF NOT EXISTS conversation_sessions (
		conversation_id INTEGER NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
		ordinal INTEGER NOT NULL,
		session_id TEXT NOT NULL,
		slug TEXT NOT NULL,
		timestamp_ns INTEGER NOT NULL,
		last_timestamp_ns INTEGER NOT NULL,
		cwd TEXT NOT NULL,
		git_branch TEXT NOT NULL,
		version TEXT NOT NULL,
		model TEXT NOT NULL,
		first_message TEXT NOT NULL,
		message_count INTEGER NOT NULL,
		main_message_count INTEGER NOT NULL,
		user_message_count INTEGER NOT NULL DEFAULT 0,
		assistant_message_count INTEGER NOT NULL DEFAULT 0,
		file_path TEXT NOT NULL,
		input_tokens INTEGER NOT NULL,
		cache_creation_input_tokens INTEGER NOT NULL,
		cache_read_input_tokens INTEGER NOT NULL,
		output_tokens INTEGER NOT NULL,
		reasoning_output_tokens INTEGER NOT NULL DEFAULT 0,
		tool_counts_json TEXT NOT NULL,
		tool_error_counts_json TEXT NOT NULL DEFAULT '{}',
		tool_reject_counts_json TEXT NOT NULL DEFAULT '{}',
		action_counts_json TEXT NOT NULL DEFAULT '{}',
		action_error_counts_json TEXT NOT NULL DEFAULT '{}',
		action_reject_counts_json TEXT NOT NULL DEFAULT '{}',
		performance_meta_json TEXT NOT NULL DEFAULT '{}',
		is_subagent INTEGER NOT NULL,
		PRIMARY KEY(conversation_id, ordinal)
	)`,
	`CREATE TABLE IF NOT EXISTS search_chunks (
		id INTEGER PRIMARY KEY,
		conversation_id INTEGER NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
		ordinal INTEGER NOT NULL,
		text TEXT NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS stats_performance_sequence (
		conversation_cache_key TEXT NOT NULL,
		session_ordinal INTEGER NOT NULL,
		timestamp_ns INTEGER NOT NULL,
		mutated INTEGER NOT NULL DEFAULT 0,
		mutation_count INTEGER NOT NULL DEFAULT 0,
		rewrite_count INTEGER NOT NULL DEFAULT 0,
		targeted_mutation_count INTEGER NOT NULL DEFAULT 0,
		blind_mutation_count INTEGER NOT NULL DEFAULT 0,
		distinct_mutation_targets INTEGER NOT NULL DEFAULT 0,
		patch_hunk_count INTEGER NOT NULL DEFAULT 0,
		verification_passed INTEGER NOT NULL DEFAULT 0,
		first_pass_resolved INTEGER NOT NULL DEFAULT 0,
		correction_followups INTEGER NOT NULL DEFAULT 0,
		reasoning_loop_count INTEGER NOT NULL DEFAULT 0,
		action_count INTEGER NOT NULL DEFAULT 0,
		actions_before_first_mutation INTEGER NOT NULL DEFAULT 0,
		tokens_before_first_mutation INTEGER NOT NULL DEFAULT 0,
		user_turns_before_first_mutation INTEGER NOT NULL DEFAULT 0,
		assistant_turns INTEGER NOT NULL DEFAULT 0,
		visible_reasoning_chars INTEGER NOT NULL DEFAULT 0,
		hidden_thinking_turns INTEGER NOT NULL DEFAULT 0,
		PRIMARY KEY (conversation_cache_key, session_ordinal)
	)`,
	`CREATE TABLE IF NOT EXISTS stats_turn_metrics (
		conversation_cache_key TEXT NOT NULL,
		session_ordinal INTEGER NOT NULL,
		timestamp_ns INTEGER NOT NULL,
		turns_json TEXT NOT NULL DEFAULT '[]',
		PRIMARY KEY (conversation_cache_key, session_ordinal)
	)`,
	`CREATE TABLE IF NOT EXISTS stats_daily_tokens (
		conversation_cache_key TEXT NOT NULL,
		date_key TEXT NOT NULL,
		provider TEXT NOT NULL,
		model TEXT NOT NULL,
		project TEXT NOT NULL,
		session_count INTEGER NOT NULL DEFAULT 0,
		message_count INTEGER NOT NULL DEFAULT 0,
		user_message_count INTEGER NOT NULL DEFAULT 0,
		assistant_message_count INTEGER NOT NULL DEFAULT 0,
		input_tokens INTEGER NOT NULL DEFAULT 0,
		cache_creation_tokens INTEGER NOT NULL DEFAULT 0,
		cache_read_tokens INTEGER NOT NULL DEFAULT 0,
		output_tokens INTEGER NOT NULL DEFAULT 0,
		reasoning_output_tokens INTEGER NOT NULL DEFAULT 0,
		PRIMARY KEY (
			conversation_cache_key,
			date_key,
			provider,
			model,
			project
		)
	)`,
	`CREATE VIRTUAL TABLE IF NOT EXISTS search_fts USING fts5(
		text,
		content='search_chunks',
		content_rowid='id',
		tokenize='unicode61 remove_diacritics 2',
		detail='full'
	)`,
	`CREATE INDEX IF NOT EXISTS conversations_last_timestamp_idx ON conversations(last_timestamp_ns DESC)`,
	`CREATE INDEX IF NOT EXISTS conversation_sessions_conversation_idx ON conversation_sessions(conversation_id, ordinal)`,
	`CREATE INDEX IF NOT EXISTS conversation_sessions_file_path_idx ON conversation_sessions(file_path)`,
	`CREATE INDEX IF NOT EXISTS conversation_sessions_session_id_idx ON conversation_sessions(session_id)`,
	`CREATE INDEX IF NOT EXISTS search_chunks_conversation_idx ON search_chunks(conversation_id, ordinal)`,
}

var sqliteSearchTriggerStatements = []string{
	`CREATE TRIGGER IF NOT EXISTS search_chunks_ai AFTER INSERT ON search_chunks BEGIN
		INSERT INTO search_fts(rowid, text) VALUES (new.id, new.text);
	END`,
	`CREATE TRIGGER IF NOT EXISTS search_chunks_ad AFTER DELETE ON search_chunks BEGIN
		INSERT INTO search_fts(search_fts, rowid, text) VALUES ('delete', old.id, old.text);
	END`,
	`CREATE TRIGGER IF NOT EXISTS search_chunks_au AFTER UPDATE ON search_chunks BEGIN
		INSERT INTO search_fts(search_fts, rowid, text) VALUES ('delete', old.id, old.text);
		INSERT INTO search_fts(rowid, text) VALUES (new.id, new.text);
	END`,
}

var sqliteSearchTriggerNames = []string{
	"search_chunks_ai",
	"search_chunks_ad",
	"search_chunks_au",
}

func openSQLiteDB(ctx context.Context, path string, useWAL bool) (*sql.DB, error) {
	db, err := sql.Open(sqliteDriverName, path)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}
	db.SetMaxOpenConns(1)

	if err := configureSQLiteDB(ctx, db, useWAL); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			zerolog.Ctx(ctx).Warn().Err(closeErr).Msg("db.Close")
		}
		return nil, fmt.Errorf("configureSQLiteDB: %w", err)
	}
	return db, nil
}

func configureSQLiteDB(ctx context.Context, db *sql.DB, useWAL bool) error {
	journalMode := "DELETE"
	if useWAL {
		journalMode = "WAL"
	}

	for _, stmt := range []string{
		"PRAGMA journal_mode = " + journalMode,
		"PRAGMA synchronous = NORMAL",
		"PRAGMA foreign_keys = ON",
	} {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("db.ExecContext_%s: %w", stmt, err)
		}
	}
	return nil
}

func ensureSQLiteSchema(ctx context.Context, db *sql.DB) error {
	if err := ensureSQLiteSchemaBase(ctx, db); err != nil {
		return fmt.Errorf("ensureSQLiteSchemaBase: %w", err)
	}
	if err := ensureSQLiteSearchTriggers(ctx, db); err != nil {
		return fmt.Errorf("ensureSQLiteSearchTriggers: %w", err)
	}
	return nil
}

func ensureSQLiteSchemaBase(ctx context.Context, db *sql.DB) error {
	for _, stmt := range sqliteSchemaStatements {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("db.ExecContext: %w", err)
		}
	}
	return nil
}

type sqliteExecer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func ensureSQLiteSearchTriggers(ctx context.Context, execer sqliteExecer) error {
	for _, stmt := range sqliteSearchTriggerStatements {
		if _, err := execer.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("execer.ExecContext: %w", err)
		}
	}
	return nil
}

func dropSQLiteSearchTriggers(ctx context.Context, execer sqliteExecer) error {
	for _, name := range sqliteSearchTriggerNames {
		if _, err := execer.ExecContext(ctx, `DROP TRIGGER IF EXISTS `+name); err != nil {
			return fmt.Errorf("execer.ExecContext_%s: %w", name, err)
		}
	}
	return nil
}

func writeSQLiteMeta(ctx context.Context, tx *sql.Tx) error {
	for key, value := range map[string]string{
		metaSchemaKey:     strconv.Itoa(storeSchemaVersion),
		metaProjectionKey: strconv.Itoa(storeProjectionVersion),
		metaSearchKey:     strconv.Itoa(storeSearchCorpusVersion),
	} {
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO meta(key, value) VALUES (?, ?)
			 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
			key,
			value,
		); err != nil {
			return fmt.Errorf("tx.ExecContext_%s: %w", key, err)
		}
	}
	return nil
}

func readSQLiteMeta(ctx context.Context, db *sql.DB) (map[string]string, error) {
	rows, err := db.QueryContext(ctx, `SELECT key, value FROM meta`)
	if err != nil {
		return nil, fmt.Errorf("db.QueryContext: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			zerolog.Ctx(ctx).Warn().Err(err).Msg("rows.Close")
		}
	}()

	meta := make(map[string]string, 3)
	for rows.Next() {
		var key string
		var value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("rows.Scan: %w", err)
		}
		meta[key] = value
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows.Err: %w", err)
	}
	return meta, nil
}

func sqliteMetaCurrent(meta map[string]string) bool {
	if meta[metaSchemaKey] != strconv.Itoa(storeSchemaVersion) {
		return false
	}
	if meta[metaProjectionKey] != strconv.Itoa(storeProjectionVersion) {
		return false
	}
	if meta[metaSearchKey] != strconv.Itoa(storeSearchCorpusVersion) {
		return false
	}
	return true
}

func removeSQLiteSidecars(path string) error {
	for _, suffix := range []string{"-wal", "-shm"} {
		if err := os.Remove(path + suffix); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("os.Remove_%s: %w", suffix, err)
		}
	}
	return nil
}

func replaceSQLiteFile(tempPath, finalPath string) error {
	if err := os.MkdirAll(filepath.Dir(finalPath), 0o755); err != nil {
		return fmt.Errorf("os.MkdirAll: %w", err)
	}

	exists, err := pathExists(finalPath)
	if err != nil {
		return fmt.Errorf("pathExists: %w", err)
	}
	if !exists {
		return replaceMissingSQLiteFile(tempPath, finalPath)
	}
	return replaceExistingSQLiteFile(tempPath, finalPath)
}

func replaceMissingSQLiteFile(tempPath, finalPath string) error {
	if err := removeSQLiteSidecars(finalPath); err != nil {
		return fmt.Errorf("removeSQLiteSidecars_new: %w", err)
	}
	if err := os.Rename(tempPath, finalPath); err != nil {
		return fmt.Errorf("os.Rename_new: %w", err)
	}
	return nil
}

func replaceExistingSQLiteFile(tempPath, finalPath string) error {
	backupPath, err := reserveTempPath(filepath.Dir(finalPath), filepath.Base(finalPath)+"-backup-*")
	if err != nil {
		return fmt.Errorf("reserveTempPath: %w", err)
	}
	if err := removeSQLiteSidecars(finalPath); err != nil {
		return fmt.Errorf("removeSQLiteSidecars_existing: %w", err)
	}
	if err := os.Rename(finalPath, backupPath); err != nil {
		return fmt.Errorf("os.Rename_backup: %w", err)
	}
	if err := os.Rename(tempPath, finalPath); err != nil {
		if restoreErr := os.Rename(backupPath, finalPath); restoreErr != nil {
			return fmt.Errorf("os.Rename_restore: %v (original: %w)", restoreErr, err)
		}
		return fmt.Errorf("os.Rename_final: %w", err)
	}
	if err := os.Remove(backupPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("os.Remove_backup: %w", err)
	}
	return nil
}
