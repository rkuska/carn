package canonical

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	conv "github.com/rkuska/carn/internal/conversation"
	src "github.com/rkuska/carn/internal/source"
)

var _ src.IncrementalLookup = sqliteIncrementalLookup{}

type sqliteIncrementalLookup struct {
	db *sql.DB
}

func (l sqliteIncrementalLookup) ConversationByFilePath(
	ctx context.Context,
	provider conv.Provider,
	filePath string,
) (conv.Conversation, bool, error) {
	rowID, ok, err := lookupSQLiteConversationRowIDByFilePath(ctx, l.db, provider, filePath)
	if err != nil {
		return conversation{}, false, fmt.Errorf("lookupSQLiteConversationRowIDByFilePath: %w", err)
	}
	if !ok {
		return conversation{}, false, nil
	}
	convValue, ok, err := readSQLiteConversationByRowID(ctx, l.db, rowID)
	if err != nil {
		return conversation{}, false, fmt.Errorf("readSQLiteConversationByRowID: %w", err)
	}
	return convValue, ok, nil
}

func (l sqliteIncrementalLookup) ConversationBySessionID(
	ctx context.Context,
	provider conv.Provider,
	sessionID string,
) (conv.Conversation, bool, error) {
	rowID, ok, err := lookupSQLiteConversationRowIDBySessionID(ctx, l.db, provider, sessionID)
	if err != nil {
		return conversation{}, false, fmt.Errorf("lookupSQLiteConversationRowIDBySessionID: %w", err)
	}
	if !ok {
		return conversation{}, false, nil
	}
	convValue, ok, err := readSQLiteConversationByRowID(ctx, l.db, rowID)
	if err != nil {
		return conversation{}, false, fmt.Errorf("readSQLiteConversationByRowID: %w", err)
	}
	return convValue, ok, nil
}

func (l sqliteIncrementalLookup) ConversationByCacheKey(
	ctx context.Context,
	cacheKey string,
) (conv.Conversation, bool, error) {
	rowID, ok, err := lookupSQLiteConversationRowIDByCacheKey(ctx, l.db, cacheKey)
	if err != nil {
		return conversation{}, false, fmt.Errorf("lookupSQLiteConversationRowIDByCacheKey: %w", err)
	}
	if !ok {
		return conversation{}, false, nil
	}
	convValue, ok, err := readSQLiteConversationByRowID(ctx, l.db, rowID)
	if err != nil {
		return conversation{}, false, fmt.Errorf("readSQLiteConversationByRowID: %w", err)
	}
	return convValue, ok, nil
}

func lookupSQLiteConversationRowIDByFilePath(
	ctx context.Context,
	db *sql.DB,
	provider conv.Provider,
	filePath string,
) (int64, bool, error) {
	var rowID int64
	err := db.QueryRowContext(
		ctx,
		`SELECT c.id
		   FROM conversations c
		   JOIN conversation_sessions s ON s.conversation_id = c.id
		  WHERE c.provider = ?
		    AND s.file_path = ?
		  LIMIT 1`,
		string(provider),
		filePath,
	).Scan(&rowID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("QueryRowContext.Scan: %w", err)
	}
	return rowID, true, nil
}

func lookupSQLiteConversationRowIDBySessionID(
	ctx context.Context,
	db *sql.DB,
	provider conv.Provider,
	sessionID string,
) (int64, bool, error) {
	var rowID int64
	err := db.QueryRowContext(
		ctx,
		`SELECT c.id
		   FROM conversations c
		   JOIN conversation_sessions s ON s.conversation_id = c.id
		  WHERE c.provider = ?
		    AND s.session_id = ?
		  LIMIT 1`,
		string(provider),
		sessionID,
	).Scan(&rowID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("QueryRowContext.Scan: %w", err)
	}
	return rowID, true, nil
}

func lookupSQLiteConversationRowIDByCacheKey(
	ctx context.Context,
	db *sql.DB,
	cacheKey string,
) (int64, bool, error) {
	var rowID int64
	err := db.QueryRowContext(
		ctx,
		`SELECT id FROM conversations WHERE cache_key = ?`,
		cacheKey,
	).Scan(&rowID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("QueryRowContext.Scan: %w", err)
	}
	return rowID, true, nil
}

func readSQLiteConversationByRowID(
	ctx context.Context,
	db *sql.DB,
	rowID int64,
) (conversation, bool, error) {
	var provider string
	var providerID string
	var name string
	var projectName string
	var planCount int
	err := db.QueryRowContext(
		ctx,
		`SELECT provider, provider_id, name, project_display_name, plan_count
		   FROM conversations
		  WHERE id = ?`,
		rowID,
	).Scan(&provider, &providerID, &name, &projectName, &planCount)
	if errors.Is(err, sql.ErrNoRows) {
		return conversation{}, false, nil
	}
	if err != nil {
		return conversation{}, false, fmt.Errorf("QueryRowContext.Scan: %w", err)
	}

	convValue := conversation{
		Ref: conversationRef{
			Provider: conversationProvider(provider),
			ID:       providerID,
		},
		Name:      name,
		Project:   project{DisplayName: projectName},
		PlanCount: planCount,
	}
	if err := loadSQLiteConversationSessionsByRowID(ctx, db, rowID, &convValue); err != nil {
		return conversation{}, false, fmt.Errorf("loadSQLiteConversationSessionsByRowID: %w", err)
	}
	return convValue, true, nil
}

func loadSQLiteConversationSessionsByRowID(
	ctx context.Context,
	db *sql.DB,
	rowID int64,
	convValue *conversation,
) error {
	rows, err := db.QueryContext(
		ctx,
		`SELECT session_id, slug, timestamp_ns, last_timestamp_ns,
		        cwd, git_branch, version, model, first_message, message_count, main_message_count,
		        file_path, input_tokens, cache_creation_input_tokens, cache_read_input_tokens,
		        output_tokens, tool_counts_json, is_subagent
		   FROM conversation_sessions
		  WHERE conversation_id = ?
		  ORDER BY ordinal`,
		rowID,
	)
	if err != nil {
		return fmt.Errorf("db.QueryContext: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var meta sessionMeta
		var timestampNS int64
		var lastTimestampNS int64
		var toolCountsJSON string
		var isSubagent int
		if err := rows.Scan(
			&meta.ID,
			&meta.Slug,
			&timestampNS,
			&lastTimestampNS,
			&meta.CWD,
			&meta.GitBranch,
			&meta.Version,
			&meta.Model,
			&meta.FirstMessage,
			&meta.MessageCount,
			&meta.MainMessageCount,
			&meta.FilePath,
			&meta.TotalUsage.InputTokens,
			&meta.TotalUsage.CacheCreationInputTokens,
			&meta.TotalUsage.CacheReadInputTokens,
			&meta.TotalUsage.OutputTokens,
			&toolCountsJSON,
			&isSubagent,
		); err != nil {
			return fmt.Errorf("rows.Scan: %w", err)
		}

		if err := finalizeSessionMeta(&meta, timestampNS, lastTimestampNS, toolCountsJSON, isSubagent); err != nil {
			return fmt.Errorf("finalizeSessionMeta: %w", err)
		}
		meta.Project = convValue.Project
		convValue.Sessions = append(convValue.Sessions, meta)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("rows.Err: %w", err)
	}
	return nil
}
