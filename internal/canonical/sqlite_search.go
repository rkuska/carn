package canonical

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

const (
	searchPreviewSnippetTokens      = 16
	searchPreviewMaxPerConversation = 3
)

var deepSearchQuery = fmt.Sprintf(
	`WITH matching_chunks AS (
		SELECT c.id AS conversation_id,
		       c.cache_key,
		       c.last_timestamp_ns,
		       sc.ordinal,
		       TRIM(snippet(search_fts, 0, '', '', '...', %d)) AS preview
		  FROM search_fts
		  JOIN search_chunks sc ON sc.id = search_fts.rowid
		  JOIN conversations c ON c.id = sc.conversation_id
		 WHERE search_fts MATCH ?
	),
	ranked_conversations AS (
		SELECT conversation_id,
		       cache_key,
		       last_timestamp_ns,
		       MIN(ordinal) AS first_ordinal
		  FROM matching_chunks
		 GROUP BY conversation_id
	),
	unique_previews AS (
		SELECT conversation_id,
		       preview,
		       MIN(ordinal) AS first_ordinal
		  FROM matching_chunks
		 WHERE preview <> ''
		 GROUP BY conversation_id, preview
	),
	ranked_previews AS (
		SELECT conversation_id,
		       preview,
		       first_ordinal,
		       ROW_NUMBER() OVER (PARTITION BY conversation_id ORDER BY first_ordinal ASC) AS preview_row
		  FROM unique_previews
	)
	SELECT rc.conversation_id,
	       rc.cache_key,
	       rp.preview
	  FROM ranked_conversations rc
	  LEFT JOIN ranked_previews rp
	    ON rp.conversation_id = rc.conversation_id
	   AND rp.preview_row <= ?
	 ORDER BY rc.first_ordinal ASC, rc.last_timestamp_ns DESC, rp.first_ordinal ASC`,
	searchPreviewSnippetTokens,
)

type rankedConversationMatch struct {
	id       int64
	cacheKey string
	previews []string
}

func runSQLiteDeepSearch(
	ctx context.Context,
	db *sql.DB,
	query string,
	mainConversations []conversation,
) ([]conversation, error) {
	ftsQuery := buildFTSQuery(query)
	if ftsQuery == "" {
		return nil, nil
	}

	matches, err := readRankedConversationMatches(ctx, db, ftsQuery)
	if err != nil {
		return nil, fmt.Errorf("readRankedConversationMatches: %w", err)
	}
	if len(matches) == 0 {
		return []conversation{}, nil
	}

	byKey := make(map[string]conversation, len(mainConversations))
	for _, conv := range mainConversations {
		byKey[conv.CacheKey()] = conv
	}

	results := make([]conversation, 0, len(matches))
	for _, match := range matches {
		conv, ok := byKey[match.cacheKey]
		if !ok {
			continue
		}
		conv.SetSearchPreview(strings.Join(match.previews, "\n"))
		results = append(results, conv)
	}
	return results, nil
}

func buildFTSQuery(query string) string {
	terms := searchTerms(query)
	if len(terms) == 0 {
		return ""
	}

	parts := make([]string, 0, len(terms))
	for _, term := range terms {
		parts = append(parts, `"`+strings.ReplaceAll(term, `"`, `""`)+`"*`)
	}
	return strings.Join(parts, " AND ")
}

func searchTerms(query string) []string {
	tokens := strings.Fields(query)
	if len(tokens) == 0 {
		return nil
	}

	terms := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if token == "" {
			continue
		}
		terms = append(terms, token)
	}
	return terms
}

func readRankedConversationMatches(
	ctx context.Context,
	db *sql.DB,
	ftsQuery string,
) ([]rankedConversationMatch, error) {
	query, args := buildDeepSearchQuery(ftsQuery)
	rows, err := db.QueryContext(
		ctx,
		query,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("db.QueryContext: %w", err)
	}
	defer func() { _ = rows.Close() }()

	matches := make([]rankedConversationMatch, 0)
	matchIndexByID := make(map[int64]int)
	for rows.Next() {
		var conversationID int64
		var cacheKey string
		var preview sql.NullString
		if err := rows.Scan(&conversationID, &cacheKey, &preview); err != nil {
			return nil, fmt.Errorf("rows.Scan: %w", err)
		}
		index, ok := matchIndexByID[conversationID]
		if !ok {
			matchIndexByID[conversationID] = len(matches)
			matches = append(matches, rankedConversationMatch{
				id:       conversationID,
				cacheKey: cacheKey,
			})
			index = len(matches) - 1
		}
		if preview.Valid && preview.String != "" {
			matches[index].previews = append(matches[index].previews, preview.String)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows.Err: %w", err)
	}
	return matches, nil
}

func buildDeepSearchQuery(ftsQuery string) (string, []any) {
	return deepSearchQuery, []any{ftsQuery, searchPreviewMaxPerConversation}
}
