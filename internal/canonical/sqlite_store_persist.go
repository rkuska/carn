package canonical

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
)

type sqliteStoreCounts struct {
	conversations int
	sessions      int
	searchChunks  int
}

func writeCanonicalStoreAtomically(
	ctx context.Context,
	archiveDir string,
	conversations []conversation,
	transcripts map[string]sessionFull,
	corpus searchCorpus,
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
	defer func() { _ = os.Remove(tempPath) }()

	db, err := openSQLiteDB(ctx, tempPath, false)
	if err != nil {
		return fmt.Errorf("openSQLiteDB: %w", err)
	}
	defer func() { _ = db.Close() }()

	if err := configureSQLiteBulkLoadDB(ctx, db); err != nil {
		return fmt.Errorf("configureSQLiteBulkLoadDB: %w", err)
	}

	counts, err := replaceSQLiteStoreContents(ctx, db, conversations, transcripts, corpus)
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
}

func replaceSQLiteStoreContents(
	ctx context.Context,
	db *sql.DB,
	conversations []conversation,
	transcripts map[string]sessionFull,
	corpus searchCorpus,
) (sqliteStoreCounts, error) {
	if err := ensureSQLiteSchemaBase(ctx, db); err != nil {
		return sqliteStoreCounts{}, fmt.Errorf("ensureSQLiteSchemaBase: %w", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return sqliteStoreCounts{}, fmt.Errorf("db.BeginTx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, stmt := range []string{
		`DELETE FROM conversation_sessions`,
		`DELETE FROM search_chunks`,
		`DELETE FROM conversations`,
		`DELETE FROM meta`,
	} {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return sqliteStoreCounts{}, fmt.Errorf("tx.ExecContext_clear: %w", err)
		}
	}

	groupedUnits := groupSearchUnitsByConversation(corpus, len(conversations))
	counts, err := insertSQLiteConversations(ctx, tx, conversations, transcripts, groupedUnits)
	if err != nil {
		return sqliteStoreCounts{}, fmt.Errorf("insertSQLiteConversations: %w", err)
	}
	if err := populateSQLiteSearchIndex(ctx, tx); err != nil {
		return sqliteStoreCounts{}, fmt.Errorf("populateSQLiteSearchIndex: %w", err)
	}
	if err := ensureSQLiteSearchTriggers(ctx, tx); err != nil {
		return sqliteStoreCounts{}, fmt.Errorf("ensureSQLiteSearchTriggers: %w", err)
	}
	if err := writeSQLiteMeta(ctx, tx); err != nil {
		return sqliteStoreCounts{}, fmt.Errorf("writeSQLiteMeta: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return sqliteStoreCounts{}, fmt.Errorf("tx.Commit: %w", err)
	}
	return counts, nil
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

func populateSQLiteSearchIndex(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO search_fts(rowid, text)
		 SELECT id, text
		   FROM search_chunks
		  ORDER BY id`,
	); err != nil {
		return fmt.Errorf("tx.ExecContext: %w", err)
	}
	return nil
}

func insertSQLiteConversations(
	ctx context.Context,
	tx *sql.Tx,
	conversations []conversation,
	transcripts map[string]sessionFull,
	groupedUnits map[string][]searchUnit,
) (sqliteStoreCounts, error) {
	convStmt, err := tx.PrepareContext(ctx, `INSERT INTO conversations(
		provider, provider_id, cache_key, name, project_display_name,
		first_timestamp_ns, last_timestamp_ns, first_message, plan_count,
		total_input_tokens, total_cache_creation_tokens, total_cache_read_tokens, total_output_tokens,
		total_message_count, total_main_message_count, transcript_blob
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return sqliteStoreCounts{}, fmt.Errorf("tx.PrepareContext_conversations: %w", err)
	}
	defer func() { _ = convStmt.Close() }()

	sessionStmt, err := tx.PrepareContext(ctx, `INSERT INTO conversation_sessions(
		conversation_id, ordinal, session_id, slug, timestamp_ns, last_timestamp_ns,
		cwd, git_branch, version, model, first_message, message_count, main_message_count,
		file_path, input_tokens, cache_creation_input_tokens, cache_read_input_tokens,
		output_tokens, tool_counts_json, is_subagent
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return sqliteStoreCounts{}, fmt.Errorf("tx.PrepareContext_sessions: %w", err)
	}
	defer func() { _ = sessionStmt.Close() }()

	chunkStmt, err := tx.PrepareContext(ctx, `INSERT INTO search_chunks(conversation_id, ordinal, text) VALUES (?, ?, ?)`)
	if err != nil {
		return sqliteStoreCounts{}, fmt.Errorf("tx.PrepareContext_chunks: %w", err)
	}
	defer func() { _ = chunkStmt.Close() }()

	var counts sqliteStoreCounts
	for _, conv := range conversations {
		session, ok := transcripts[conv.CacheKey()]
		if !ok {
			return sqliteStoreCounts{}, fmt.Errorf(
				"insertSQLiteConversations: %w",
				errors.New("missing transcript for conversation"),
			)
		}

		inserted, err := insertSQLiteConversation(
			ctx,
			convStmt,
			sessionStmt,
			chunkStmt,
			conv,
			session,
			groupedUnits[conv.CacheKey()],
		)
		if err != nil {
			return sqliteStoreCounts{}, fmt.Errorf("insertSQLiteConversation: %w", err)
		}
		counts.conversations += inserted.conversations
		counts.sessions += inserted.sessions
		counts.searchChunks += inserted.searchChunks
	}

	return counts, nil
}

func insertSQLiteConversation(
	ctx context.Context,
	convStmt *sql.Stmt,
	sessionStmt *sql.Stmt,
	chunkStmt *sql.Stmt,
	conv conversation,
	session sessionFull,
	units []searchUnit,
) (sqliteStoreCounts, error) {
	conversationID, err := insertSQLiteConversationRow(ctx, convStmt, conv, session)
	if err != nil {
		return sqliteStoreCounts{}, fmt.Errorf("insertSQLiteConversationRow: %w", err)
	}

	sessionCount, err := insertSQLiteSessionRows(ctx, sessionStmt, conversationID, conv)
	if err != nil {
		return sqliteStoreCounts{}, fmt.Errorf("insertSQLiteSessionRows: %w", err)
	}
	searchChunkCount, err := insertSQLiteSearchChunks(ctx, chunkStmt, conversationID, units)
	if err != nil {
		return sqliteStoreCounts{}, fmt.Errorf("insertSQLiteSearchChunks: %w", err)
	}

	return sqliteStoreCounts{
		conversations: 1,
		sessions:      sessionCount,
		searchChunks:  searchChunkCount,
	}, nil
}

func insertSQLiteConversationRow(
	ctx context.Context,
	stmt *sql.Stmt,
	conv conversation,
	session sessionFull,
) (int64, error) {
	transcriptBlob, err := encodeSessionBlob(session)
	if err != nil {
		return 0, fmt.Errorf("encodeSessionBlob: %w", err)
	}

	usage := conv.TotalTokenUsage()
	result, err := stmt.ExecContext(
		ctx,
		string(conv.Ref.Provider),
		conv.Ref.ID,
		conv.CacheKey(),
		conv.Name,
		conv.Project.DisplayName,
		timeToUnixNano(conv.Timestamp()),
		timeToUnixNano(conversationLastTimestamp(conv)),
		conv.FirstMessage(),
		conv.PlanCount,
		usage.InputTokens,
		usage.CacheCreationInputTokens,
		usage.CacheReadInputTokens,
		usage.OutputTokens,
		conv.TotalMessageCount(),
		conv.MainMessageCount(),
		transcriptBlob,
	)
	if err != nil {
		return 0, fmt.Errorf("stmt.ExecContext: %w", err)
	}
	conversationID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("result.LastInsertId: %w", err)
	}
	return conversationID, nil
}

func insertSQLiteSessionRows(
	ctx context.Context,
	stmt *sql.Stmt,
	conversationID int64,
	conv conversation,
) (int, error) {
	count := 0
	for ordinal, meta := range conv.Sessions {
		toolCountsJSON := marshalToolCountsCached(meta.ToolCounts)
		if _, err := stmt.ExecContext(
			ctx,
			conversationID,
			ordinal,
			meta.ID,
			meta.Slug,
			timeToUnixNano(meta.Timestamp),
			timeToUnixNano(meta.LastTimestamp),
			meta.CWD,
			meta.GitBranch,
			meta.Version,
			meta.Model,
			meta.FirstMessage,
			meta.MessageCount,
			meta.MainMessageCount,
			meta.FilePath,
			meta.TotalUsage.InputTokens,
			meta.TotalUsage.CacheCreationInputTokens,
			meta.TotalUsage.CacheReadInputTokens,
			meta.TotalUsage.OutputTokens,
			toolCountsJSON,
			boolToInt(meta.IsSubagent),
		); err != nil {
			return 0, fmt.Errorf("stmt.ExecContext: %w", err)
		}
		count++
	}
	return count, nil
}

func insertSQLiteSearchChunks(
	ctx context.Context,
	stmt *sql.Stmt,
	conversationID int64,
	units []searchUnit,
) (int, error) {
	count := 0
	for _, unit := range units {
		if _, err := stmt.ExecContext(ctx, conversationID, unit.ordinal, unit.text); err != nil {
			return 0, fmt.Errorf("stmt.ExecContext: %w", err)
		}
		count++
	}
	return count, nil
}
