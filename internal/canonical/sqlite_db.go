package canonical

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

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
		file_path TEXT NOT NULL,
		input_tokens INTEGER NOT NULL,
		cache_creation_input_tokens INTEGER NOT NULL,
		cache_read_input_tokens INTEGER NOT NULL,
		output_tokens INTEGER NOT NULL,
		tool_counts_json TEXT NOT NULL,
		is_subagent INTEGER NOT NULL,
		PRIMARY KEY(conversation_id, ordinal)
	)`,
	`CREATE TABLE IF NOT EXISTS search_chunks (
		id INTEGER PRIMARY KEY,
		conversation_id INTEGER NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
		ordinal INTEGER NOT NULL,
		text TEXT NOT NULL
	)`,
	`CREATE VIRTUAL TABLE IF NOT EXISTS search_fts USING fts5(
		text,
		content='search_chunks',
		content_rowid='id',
		tokenize='unicode61 remove_diacritics 2',
		detail='none'
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

func openSQLiteDB(ctx context.Context, path string, useWAL bool) (*sql.DB, error) {
	db, err := sql.Open(sqliteDriverName, path)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}
	db.SetMaxOpenConns(1)

	if err := configureSQLiteDB(ctx, db, useWAL); err != nil {
		_ = db.Close()
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
	defer func() { _ = rows.Close() }()

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

func storeNeedsRebuild(ctx context.Context, archiveDir string) (bool, error) {
	path := canonicalStorePath(archiveDir)
	exists, err := pathExists(path)
	if err != nil {
		return true, fmt.Errorf("pathExists: %w", err)
	}
	if !exists {
		return true, nil
	}

	db, err := openSQLiteDB(ctx, path, false)
	if err != nil {
		return true, nil
	}
	defer func() { _ = db.Close() }()

	meta, err := readSQLiteMeta(ctx, db)
	if err != nil {
		return true, nil
	}
	if meta[metaSchemaKey] != strconv.Itoa(storeSchemaVersion) {
		return true, nil
	}
	if meta[metaProjectionKey] != strconv.Itoa(storeProjectionVersion) {
		return true, nil
	}
	if meta[metaSearchKey] != strconv.Itoa(storeSearchCorpusVersion) {
		return true, nil
	}
	return false, nil
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
