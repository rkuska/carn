package canonical

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/rs/zerolog"

	conv "github.com/rkuska/carn/internal/conversation"
)

type sqliteStoreCounts struct {
	conversations int
	sessions      int
	searchChunks  int
}

type sqliteExecContext interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

const searchChunkBatchSize = 64

var searchChunkInsertQueryCache sync.Map

func replaceSQLiteStoreContents(
	ctx context.Context,
	db *sql.DB,
	conversations []conversation,
	transcripts map[string]sessionFull,
	corpus searchCorpus,
	statsData map[string][]conv.SessionStatsData,
	dailyTokenRows map[string][]conv.DailyTokenRow,
) (sqliteStoreCounts, error) {
	if err := ensureSQLiteSchemaBase(ctx, db); err != nil {
		return sqliteStoreCounts{}, fmt.Errorf("ensureSQLiteSchemaBase: %w", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return sqliteStoreCounts{}, fmt.Errorf("db.BeginTx: %w", err)
	}
	defer func() {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			zerolog.Ctx(ctx).Warn().Err(rbErr).Msg("tx.Rollback")
		}
	}()

	if err = clearSQLiteStoreTables(ctx, tx); err != nil {
		return sqliteStoreCounts{}, fmt.Errorf("clearSQLiteStoreTables: %w", err)
	}

	counts, err := insertSQLiteConversations(
		ctx,
		tx,
		conversations,
		transcripts,
		corpus.byConversation,
		statsData,
		dailyTokenRows,
	)
	if err != nil {
		return sqliteStoreCounts{}, fmt.Errorf("insertSQLiteConversations: %w", err)
	}
	if err := finalizeSQLiteStoreContents(ctx, tx); err != nil {
		return sqliteStoreCounts{}, fmt.Errorf("finalizeSQLiteStoreContents: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return sqliteStoreCounts{}, fmt.Errorf("tx.Commit: %w", err)
	}
	return counts, nil
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
	statsData map[string][]conv.SessionStatsData,
	dailyTokenRows map[string][]conv.DailyTokenRow,
) (sqliteStoreCounts, error) {
	convStmt, sessionStmt, err := prepareSQLiteConversationStatements(ctx, tx)
	if err != nil {
		return sqliteStoreCounts{}, fmt.Errorf("prepareSQLiteConversationStatements: %w", err)
	}
	defer func() {
		if err := convStmt.Close(); err != nil {
			zerolog.Ctx(ctx).Warn().Err(err).Msg("convStmt.Close")
		}
	}()
	defer func() {
		if err := sessionStmt.Close(); err != nil {
			zerolog.Ctx(ctx).Warn().Err(err).Msg("sessionStmt.Close")
		}
	}()

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
			tx,
			conv,
			session,
			groupedUnits[conv.CacheKey()],
			statsData[conv.CacheKey()],
			dailyTokenRows[conv.CacheKey()],
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
	tx *sql.Tx,
	conv conversation,
	session sessionFull,
	units []searchUnit,
	statsData []conv.SessionStatsData,
	dailyTokenRows []conv.DailyTokenRow,
) (sqliteStoreCounts, error) {
	conversationID, err := insertSQLiteConversationRow(ctx, convStmt, conv, session)
	if err != nil {
		return sqliteStoreCounts{}, fmt.Errorf("insertSQLiteConversationRow: %w", err)
	}

	sessionCount, err := insertSQLiteSessionRows(ctx, sessionStmt, conversationID, conv)
	if err != nil {
		return sqliteStoreCounts{}, fmt.Errorf("insertSQLiteSessionRows: %w", err)
	}
	searchChunkCount, err := insertSQLiteSearchChunks(ctx, tx, conversationID, units)
	if err != nil {
		return sqliteStoreCounts{}, fmt.Errorf("insertSQLiteSearchChunks: %w", err)
	}
	if err := insertSQLiteConversationStats(ctx, tx, conv.CacheKey(), statsData, dailyTokenRows); err != nil {
		return sqliteStoreCounts{}, fmt.Errorf("insertSQLiteConversationStats: %w", err)
	}

	return sqliteStoreCounts{
		conversations: 1,
		sessions:      sessionCount,
		searchChunks:  searchChunkCount,
	}, nil
}

func insertSQLiteConversationStats(
	ctx context.Context,
	tx *sql.Tx,
	cacheKey string,
	statsData []conv.SessionStatsData,
	dailyTokenRows []conv.DailyTokenRow,
) error {
	for ordinal, row := range statsData {
		if err := writeStatsPerformanceSequence(ctx, tx, cacheKey, ordinal, row.PerformanceSequence); err != nil {
			return fmt.Errorf("writeStatsPerformanceSequence: %w", err)
		}
		if err := writeStatsTurnMetrics(ctx, tx, cacheKey, ordinal, row.TurnMetrics); err != nil {
			return fmt.Errorf("writeStatsTurnMetrics: %w", err)
		}
	}
	if err := writeStatsDailyTokens(ctx, tx, cacheKey, dailyTokenRows); err != nil {
		return fmt.Errorf("writeStatsDailyTokens: %w", err)
	}
	return nil
}

func insertSQLiteConversationRow(
	ctx context.Context,
	stmt *sql.Stmt,
	conv conversation,
	session sessionFull,
) (int64, error) {
	usage := conv.TotalTokenUsage()
	var conversationID int64
	if err := withEncodedSessionBlob(session, func(transcriptBlob []byte) error {
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
			usage.ReasoningOutputTokens,
			conv.TotalMessageCount(),
			conv.MainMessageCount(),
			transcriptBlob,
		)
		if err != nil {
			return fmt.Errorf("stmt.ExecContext: %w", err)
		}

		conversationID, err = result.LastInsertId()
		if err != nil {
			return fmt.Errorf("result.LastInsertId: %w", err)
		}
		return nil
	}); err != nil {
		return 0, fmt.Errorf("withEncodedSessionBlob: %w", err)
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
		toolErrorCountsJSON := marshalToolCountsCached(meta.ToolErrorCounts)
		toolRejectCountsJSON := marshalToolCountsCached(meta.ToolRejectCounts)
		actionCountsJSON := marshalToolCountsCached(meta.ActionCounts)
		actionErrorCountsJSON := marshalToolCountsCached(meta.ActionErrorCounts)
		actionRejectCountsJSON := marshalToolCountsCached(meta.ActionRejectCounts)
		performanceMetaJSON, err := marshalSessionPerformanceMeta(meta.Performance)
		if err != nil {
			return 0, fmt.Errorf("marshalSessionPerformanceMeta: %w", err)
		}
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
			meta.UserMessageCount,
			meta.AssistantMessageCount,
			meta.FilePath,
			meta.TotalUsage.InputTokens,
			meta.TotalUsage.CacheCreationInputTokens,
			meta.TotalUsage.CacheReadInputTokens,
			meta.TotalUsage.OutputTokens,
			meta.TotalUsage.ReasoningOutputTokens,
			toolCountsJSON,
			toolErrorCountsJSON,
			toolRejectCountsJSON,
			actionCountsJSON,
			actionErrorCountsJSON,
			actionRejectCountsJSON,
			performanceMetaJSON,
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
	exec sqliteExecContext,
	conversationID int64,
	units []searchUnit,
) (int, error) {
	count := 0
	args := make([]any, 0, searchChunkBatchSize*3)
	batchExec := newSQLiteChunkBatchExec(exec)
	defer func() {
		if err := batchExec.close(); err != nil {
			zerolog.Ctx(ctx).Warn().Err(err).Msg("batchExec.close")
		}
	}()
	for start := 0; start < len(units); start += searchChunkBatchSize {
		end := min(start+searchChunkBatchSize, len(units))
		args = args[:0]
		for _, unit := range units[start:end] {
			args = append(args, conversationID, unit.ordinal, unit.text)
		}
		if err := batchExec.execBatch(ctx, end-start, args); err != nil {
			return 0, fmt.Errorf("batchExec.execBatch: %w", err)
		}
		count += end - start
	}
	return count, nil
}

func searchChunkInsertQuery(batchSize int) string {
	if cached, ok := searchChunkInsertQueryCache.Load(batchSize); ok {
		if s, ok := cached.(string); ok {
			return s
		}
	}

	var sb strings.Builder
	sb.Grow(len(`INSERT INTO search_chunks(conversation_id, ordinal, text) VALUES `) + batchSize*10)
	sb.WriteString(`INSERT INTO search_chunks(conversation_id, ordinal, text) VALUES `)
	for i := range batchSize {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString("(?, ?, ?)")
	}
	query := sb.String()
	actual, _ := searchChunkInsertQueryCache.LoadOrStore(batchSize, query)
	if s, ok := actual.(string); ok {
		return s
	}
	return query
}
