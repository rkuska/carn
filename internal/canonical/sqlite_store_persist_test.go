package canonical

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingChunkExec struct {
	queries []string
	args    [][]any
}

func (r *recordingChunkExec) ExecContext(
	_ context.Context,
	query string,
	args ...any,
) (sql.Result, error) {
	argsCopy := append([]any(nil), args...)
	r.queries = append(r.queries, query)
	r.args = append(r.args, argsCopy)
	return recordingSQLResult(0), nil
}

type recordingSQLResult int64

func (r recordingSQLResult) LastInsertId() (int64, error) {
	return int64(r), nil
}

func (recordingSQLResult) RowsAffected() (int64, error) {
	return 1, nil
}

func TestInsertSQLiteSearchChunksBatchesRows(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		unitCount     int
		wantQueries   int
		wantFirstArgs int
		wantLastArgs  int
	}{
		{
			name:        "empty",
			unitCount:   0,
			wantQueries: 0,
		},
		{
			name:          "single_unit",
			unitCount:     1,
			wantQueries:   1,
			wantFirstArgs: 3,
			wantLastArgs:  3,
		},
		{
			name:          "exact_batch_size",
			unitCount:     searchChunkBatchSize,
			wantQueries:   1,
			wantFirstArgs: searchChunkBatchSize * 3,
			wantLastArgs:  searchChunkBatchSize * 3,
		},
		{
			name:          "batch_plus_one",
			unitCount:     searchChunkBatchSize + 1,
			wantQueries:   2,
			wantFirstArgs: searchChunkBatchSize * 3,
			wantLastArgs:  3,
		},
		{
			name:          "two_full_batches",
			unitCount:     searchChunkBatchSize * 2,
			wantQueries:   2,
			wantFirstArgs: searchChunkBatchSize * 3,
			wantLastArgs:  searchChunkBatchSize * 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			exec := &recordingChunkExec{}
			units := make([]searchUnit, 0, tt.unitCount)
			for i := range tt.unitCount {
				units = append(units, searchUnit{
					conversationID: "claude:demo",
					ordinal:        i,
					text:           "chunk",
				})
			}

			count, err := insertSQLiteSearchChunks(context.Background(), exec, 7, units)
			require.NoError(t, err)
			assert.Equal(t, tt.unitCount, count)
			require.Len(t, exec.queries, tt.wantQueries)
			if tt.wantQueries > 0 {
				assert.Len(t, exec.args[0], tt.wantFirstArgs)
				assert.Len(t, exec.args[len(exec.args)-1], tt.wantLastArgs)
				assert.Equal(t, int64(7), exec.args[0][0])
			}
		})
	}
}

func TestInsertSQLiteSearchChunksPreservesValues(t *testing.T) {
	t.Parallel()

	exec := &recordingChunkExec{}
	units := []searchUnit{
		{conversationID: "conv-1", ordinal: 0, text: "first chunk"},
		{conversationID: "conv-1", ordinal: 1, text: "second chunk"},
	}

	count, err := insertSQLiteSearchChunks(context.Background(), exec, 42, units)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
	require.Len(t, exec.args, 1)
	assert.Equal(t, int64(42), exec.args[0][0])
	assert.Equal(t, 0, exec.args[0][1])
	assert.Equal(t, "first chunk", exec.args[0][2])
	assert.Equal(t, int64(42), exec.args[0][3])
	assert.Equal(t, 1, exec.args[0][4])
	assert.Equal(t, "second chunk", exec.args[0][5])
}
