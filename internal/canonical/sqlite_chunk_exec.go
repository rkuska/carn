package canonical

import (
	"context"
	"database/sql"
	"fmt"
)

type sqlitePrepareExecContext interface {
	sqliteExecContext
	PrepareContext(context.Context, string) (*sql.Stmt, error)
}

type sqliteChunkBatchExec struct {
	exec    sqliteExecContext
	prepare sqlitePrepareExecContext
	stmts   map[int]*sql.Stmt
}

func newSQLiteChunkBatchExec(exec sqliteExecContext) sqliteChunkBatchExec {
	batchExec := sqliteChunkBatchExec{exec: exec}
	if prepare, ok := exec.(sqlitePrepareExecContext); ok {
		batchExec.prepare = prepare
		batchExec.stmts = make(map[int]*sql.Stmt)
	}
	return batchExec
}

func (e *sqliteChunkBatchExec) close() error {
	var closeErr error
	for size, stmt := range e.stmts {
		if err := stmt.Close(); err != nil && closeErr == nil {
			closeErr = fmt.Errorf("stmt.Close_%d: %w", size, err)
		}
	}
	return closeErr
}

func (e *sqliteChunkBatchExec) execBatch(ctx context.Context, batchSize int, args []any) error {
	query := searchChunkInsertQuery(batchSize)
	if e.prepare == nil {
		if _, err := e.exec.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("exec.ExecContext: %w", err)
		}
		return nil
	}

	stmt, ok := e.stmts[batchSize]
	if !ok {
		prepared, err := e.prepare.PrepareContext(ctx, query)
		if err != nil {
			return fmt.Errorf("prepare.PrepareContext: %w", err)
		}
		stmt = prepared
		e.stmts[batchSize] = stmt
	}

	if _, err := stmt.ExecContext(ctx, args...); err != nil {
		return fmt.Errorf("stmt.ExecContext: %w", err)
	}
	return nil
}
