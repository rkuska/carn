package canonical

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/rs/zerolog"
)

func writeCanonicalStoreStreamingAtomically(
	ctx context.Context,
	archiveDir string,
	conversations []conversation,
	sources sourceRegistry,
) error {
	return withCanonicalStoreTempDB(ctx, archiveDir, func(dbPath string) error {
		db, err := openSQLiteDB(ctx, dbPath, false)
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

		counts, err := replaceSQLiteStoreContentsStreaming(ctx, db, conversations, sources)
		if err != nil {
			return fmt.Errorf("replaceSQLiteStoreContentsStreaming: %w", err)
		}
		if err := validateSQLiteStoreCounts(ctx, db, counts); err != nil {
			return fmt.Errorf("validateSQLiteStoreCounts: %w", err)
		}
		if err := db.Close(); err != nil {
			return fmt.Errorf("db.Close: %w", err)
		}
		if err := replaceSQLiteFile(dbPath, canonicalStorePath(archiveDir)); err != nil {
			return fmt.Errorf("replaceSQLiteFile: %w", err)
		}
		return nil
	})
}

func replaceSQLiteStoreContentsStreaming(
	ctx context.Context,
	db *sql.DB,
	conversations []conversation,
	sources sourceRegistry,
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

	counts, err := insertStreamingSQLiteConversations(ctx, tx, conversations, sources)
	if err != nil {
		return sqliteStoreCounts{}, fmt.Errorf("insertStreamingSQLiteConversations: %w", err)
	}

	if err := finalizeSQLiteStoreContents(ctx, tx); err != nil {
		return sqliteStoreCounts{}, fmt.Errorf("finalizeSQLiteStoreContents: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return sqliteStoreCounts{}, fmt.Errorf("tx.Commit: %w", err)
	}
	return counts, nil
}

func insertStreamingSQLiteConversations(
	ctx context.Context,
	tx *sql.Tx,
	conversations []conversation,
	sources sourceRegistry,
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
		inserted, err := insertStreamingSQLiteConversation(ctx, convStmt, sessionStmt, tx, conv, sources)
		if err != nil {
			return sqliteStoreCounts{}, fmt.Errorf("insertStreamingSQLiteConversation: %w", err)
		}
		counts.conversations += inserted.conversations
		counts.sessions += inserted.sessions
		counts.searchChunks += inserted.searchChunks
	}
	return counts, nil
}

func insertStreamingSQLiteConversation(
	ctx context.Context,
	convStmt *sql.Stmt,
	sessionStmt *sql.Stmt,
	tx *sql.Tx,
	conv conversation,
	sources sourceRegistry,
) (sqliteStoreCounts, error) {
	session, err := loadConversationSession(ctx, sources, conv)
	if err != nil {
		return sqliteStoreCounts{}, fmt.Errorf("loadConversationSession_%s: %w", conv.CacheKey(), err)
	}
	conv, session, err = enrichConversationToolOutcomes(ctx, sources, conv, session)
	if err != nil {
		return sqliteStoreCounts{}, fmt.Errorf("enrichConversationToolOutcomes_%s: %w", conv.CacheKey(), err)
	}

	conv.PlanCount = countPlansInMessages(session.Messages)
	inserted, err := insertSQLiteConversationStreaming(
		ctx,
		convStmt,
		sessionStmt,
		tx,
		conv,
		session,
	)
	if err != nil {
		return sqliteStoreCounts{}, fmt.Errorf("insertSQLiteConversationStreaming: %w", err)
	}
	return inserted, nil
}
