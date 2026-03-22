package canonical

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"

	"github.com/rs/zerolog"
)

func readSQLiteConversations(ctx context.Context, db *sql.DB) ([]conversation, error) {
	rows, err := db.QueryContext(
		ctx,
		`SELECT id, provider, provider_id, name, project_display_name, plan_count
		 FROM conversations
		 ORDER BY last_timestamp_ns DESC, cache_key`,
	)
	if err != nil {
		return nil, fmt.Errorf("db.QueryContext_conversations: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			zerolog.Ctx(ctx).Warn().Err(err).Msg("rows.Close")
		}
	}()

	conversations := make([]conversation, 0)
	byRowID := make(map[int64]int)
	for rows.Next() {
		var rowID int64
		var provider string
		var providerID string
		var name string
		var projectName string
		var planCount int
		if err := rows.Scan(&rowID, &provider, &providerID, &name, &projectName, &planCount); err != nil {
			return nil, fmt.Errorf("rows.Scan_conversations: %w", err)
		}
		byRowID[rowID] = len(conversations)
		conversations = append(conversations, conversation{
			Ref: conversationRef{
				Provider: conversationProvider(provider),
				ID:       providerID,
			},
			Name:      name,
			Project:   project{DisplayName: projectName},
			PlanCount: planCount,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows.Err_conversations: %w", err)
	}
	if len(conversations) == 0 {
		return nil, nil
	}

	if err := loadSQLiteSessions(ctx, db, conversations, byRowID); err != nil {
		return nil, fmt.Errorf("loadSQLiteSessions: %w", err)
	}
	return conversations, nil
}

func loadSQLiteSessions(
	ctx context.Context,
	db *sql.DB,
	conversations []conversation,
	byRowID map[int64]int,
) error {
	sessionRows, err := db.QueryContext(
		ctx,
		`SELECT conversation_id, session_id, slug, timestamp_ns, last_timestamp_ns,
		        cwd, git_branch, version, model, first_message, message_count, main_message_count,
		        user_message_count, assistant_message_count,
		        file_path, input_tokens, cache_creation_input_tokens, cache_read_input_tokens,
		        output_tokens, tool_counts_json, tool_error_counts_json, is_subagent
		   FROM conversation_sessions
		  ORDER BY conversation_id, ordinal`,
	)
	if err != nil {
		return fmt.Errorf("db.QueryContext_sessions: %w", err)
	}
	defer func() {
		if err := sessionRows.Close(); err != nil {
			zerolog.Ctx(ctx).Warn().Err(err).Msg("sessionRows.Close")
		}
	}()

	for sessionRows.Next() {
		var rowID int64
		var meta sessionMeta
		var timestampNS int64
		var lastTimestampNS int64
		var toolCountsJSON string
		var toolErrorCountsJSON string
		var isSubagent int
		if err := sessionRows.Scan(
			&rowID,
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
			&meta.UserMessageCount,
			&meta.AssistantMessageCount,
			&meta.FilePath,
			&meta.TotalUsage.InputTokens,
			&meta.TotalUsage.CacheCreationInputTokens,
			&meta.TotalUsage.CacheReadInputTokens,
			&meta.TotalUsage.OutputTokens,
			&toolCountsJSON,
			&toolErrorCountsJSON,
			&isSubagent,
		); err != nil {
			return fmt.Errorf("sessionRows.Scan: %w", err)
		}
		index, ok := byRowID[rowID]
		if !ok {
			return fmt.Errorf("loadSQLiteSessions: %w", errors.New("session references unknown conversation"))
		}

		if err := finalizeSessionMeta(
			&meta,
			timestampNS,
			lastTimestampNS,
			toolCountsJSON,
			toolErrorCountsJSON,
			isSubagent,
		); err != nil {
			return fmt.Errorf("finalizeSessionMeta: %w", err)
		}
		meta.Project = conversations[index].Project
		conversations[index].Sessions = append(conversations[index].Sessions, meta)
	}
	if err := sessionRows.Err(); err != nil {
		return fmt.Errorf("sessionRows.Err: %w", err)
	}
	return nil
}

func finalizeSessionMeta(
	meta *sessionMeta,
	timestampNS, lastTimestampNS int64,
	toolCountsJSON string,
	toolErrorCountsJSON string,
	isSubagent int,
) error {
	if timestampNS != 0 {
		meta.Timestamp = unixTime(timestampNS)
	}
	if lastTimestampNS != 0 {
		meta.LastTimestamp = unixTime(lastTimestampNS)
	}
	meta.IsSubagent = isSubagent == 1
	var err error
	meta.ToolCounts, err = unmarshalToolCounts(toolCountsJSON)
	if err != nil {
		return fmt.Errorf("unmarshalToolCounts: %w", err)
	}
	meta.ToolErrorCounts, err = unmarshalToolCounts(toolErrorCountsJSON)
	if err != nil {
		return fmt.Errorf("unmarshalToolCounts_toolErrorCounts: %w", err)
	}
	return nil
}

func readSQLiteTranscript(ctx context.Context, db *sql.DB, cacheKey string) (sessionFull, error) {
	var blob []byte
	if err := db.QueryRowContext(
		ctx,
		`SELECT transcript_blob FROM conversations WHERE cache_key = ?`,
		cacheKey,
	).Scan(&blob); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return sessionFull{}, fmt.Errorf("readSQLiteTranscript: %w", os.ErrNotExist)
		}
		return sessionFull{}, fmt.Errorf("QueryRowContext.Scan: %w", err)
	}
	session, err := decodeSessionBlob(blob)
	if err != nil {
		return sessionFull{}, fmt.Errorf("decodeSessionBlob: %w", err)
	}
	return session, nil
}

func validateSQLiteStoreCounts(ctx context.Context, db *sql.DB, want sqliteStoreCounts) error {
	for _, check := range []struct {
		query string
		want  int
	}{
		{query: `SELECT COUNT(*) FROM conversations`, want: want.conversations},
		{query: `SELECT COUNT(*) FROM conversation_sessions`, want: want.sessions},
		{query: `SELECT COUNT(*) FROM search_chunks`, want: want.searchChunks},
	} {
		var got int
		if err := db.QueryRowContext(ctx, check.query).Scan(&got); err != nil {
			return fmt.Errorf("QueryRowContext.Scan: %w", err)
		}
		if got != check.want {
			return fmt.Errorf("validateSQLiteStoreCounts: got %d want %d for %s", got, check.want, check.query)
		}
	}
	return nil
}
