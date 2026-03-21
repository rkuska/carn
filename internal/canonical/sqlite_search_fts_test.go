package canonical

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ftsTestDB(tb testing.TB) *sql.DB {
	tb.Helper()

	ctx := context.Background()
	db, err := openSQLiteDB(ctx, filepath.Join(tb.TempDir(), "fts.db"), false)
	require.NoError(tb, err)
	tb.Cleanup(func() {
		if closeErr := db.Close(); closeErr != nil {
			tb.Log(closeErr)
		}
	})

	require.NoError(tb, ensureSQLiteSchemaBase(ctx, db))
	return db
}

func insertFTSText(tb testing.TB, db *sql.DB, rowID int, text string) {
	tb.Helper()

	_, err := db.ExecContext(
		context.Background(),
		`INSERT INTO search_fts(rowid, text) VALUES (?, ?)`,
		rowID,
		text,
	)
	require.NoError(tb, err)
}

func ftsMatchCount(tb testing.TB, db *sql.DB, query string) int {
	tb.Helper()

	var count int
	err := db.QueryRowContext(
		context.Background(),
		`SELECT COUNT(*) FROM search_fts WHERE search_fts MATCH ?`,
		query,
	).Scan(&count)
	require.NoError(tb, err)
	return count
}

func TestFTSTokenizerMatchesDiacriticVariants(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		text  string
		query string
		rowID int
	}{
		{
			name:  "resume",
			text:  "résumé",
			query: "resume",
			rowID: 1,
		},
		{
			name:  "cafe",
			text:  "café",
			query: "cafe",
			rowID: 2,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			db := ftsTestDB(t)
			insertFTSText(t, db, testCase.rowID, testCase.text)

			assert.Equal(t, 1, ftsMatchCount(t, db, buildFTSQuery(testCase.query)))
		})
	}
}

func TestFTSTokenizerSplitsOnSeparators(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		text  string
		query string
		rowID int
	}{
		{
			name:  "underscore left side",
			text:  "generate_uuid",
			query: "generate",
			rowID: 1,
		},
		{
			name:  "underscore right side",
			text:  "generate_uuid",
			query: "uuid",
			rowID: 2,
		},
		{
			name:  "slash left side",
			text:  "foo/bar",
			query: "foo",
			rowID: 3,
		},
		{
			name:  "slash right side",
			text:  "foo/bar",
			query: "bar",
			rowID: 4,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			db := ftsTestDB(t)
			insertFTSText(t, db, testCase.rowID, testCase.text)

			assert.Equal(t, 1, ftsMatchCount(t, db, buildFTSQuery(testCase.query)))
		})
	}
}

func TestFTSTokenizerPrefixMatching(t *testing.T) {
	t.Parallel()

	db := ftsTestDB(t)
	insertFTSText(t, db, 1, "authentication")

	assert.Equal(t, 1, ftsMatchCount(t, db, `"auth"*`))
}

func TestFTSTokenizerCaseInsensitive(t *testing.T) {
	t.Parallel()

	db := ftsTestDB(t)
	insertFTSText(t, db, 1, "Hello World")

	assert.Equal(t, 1, ftsMatchCount(t, db, buildFTSQuery("hello")))
	assert.Equal(t, 1, ftsMatchCount(t, db, buildFTSQuery("HELLO")))
}

func TestFTSTokenizerCJKContent(t *testing.T) {
	t.Parallel()

	db := ftsTestDB(t)
	insertFTSText(t, db, 1, "東京 deploy 完了")

	assert.Equal(t, 1, ftsMatchCount(t, db, buildFTSQuery("deploy")))
}

func TestFTSTokenizerEmojiDoesNotCorruptTokenization(t *testing.T) {
	t.Parallel()

	db := ftsTestDB(t)
	insertFTSText(t, db, 1, "deploy the fix 🚀 immediately")

	assert.Equal(t, 1, ftsMatchCount(t, db, buildFTSQuery("deploy")))
}
