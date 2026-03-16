package canonical

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
)

const (
	searchPreviewMaxRunes                 = 96
	searchPreviewContextRunes             = 24
	searchPreviewFetchRowsPerConversation = 12
	searchPreviewMaxPerConversation       = 3
)

type rankedConversationMatch struct {
	id       int64
	cacheKey string
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

	conversationIDs := make([]int64, 0, len(matches))
	for _, match := range matches {
		conversationIDs = append(conversationIDs, match.id)
	}

	previews, err := readSearchPreviews(ctx, db, conversationIDs, matches, searchTerms(query))
	if err != nil {
		return nil, fmt.Errorf("readSearchPreviews: %w", err)
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
		conv.SetSearchPreview(strings.Join(previews[match.cacheKey], "\n"))
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
	rows, err := db.QueryContext(
		ctx,
		`SELECT c.id, c.cache_key
		   FROM search_fts
		   JOIN search_chunks sc ON sc.id = search_fts.rowid
		   JOIN conversations c ON c.id = sc.conversation_id
		  WHERE search_fts MATCH ?
		  GROUP BY c.id
		  ORDER BY MIN(sc.ordinal) ASC, c.last_timestamp_ns DESC`,
		ftsQuery,
	)
	if err != nil {
		return nil, fmt.Errorf("db.QueryContext: %w", err)
	}
	defer func() { _ = rows.Close() }()

	matches := make([]rankedConversationMatch, 0)
	for rows.Next() {
		var match rankedConversationMatch
		if err := rows.Scan(&match.id, &match.cacheKey); err != nil {
			return nil, fmt.Errorf("rows.Scan: %w", err)
		}
		matches = append(matches, match)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows.Err: %w", err)
	}
	return matches, nil
}

func readSearchPreviews(
	ctx context.Context,
	db *sql.DB,
	conversationIDs []int64,
	matches []rankedConversationMatch,
	terms []string,
) (map[string][]string, error) {
	if len(conversationIDs) == 0 {
		return nil, nil
	}

	query, args := buildSearchPreviewQuery(conversationIDs)
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("db.QueryContext: %w", err)
	}
	defer func() { _ = rows.Close() }()

	cacheKeysByConversationID := make(map[int64]string, len(matches))
	previews := make(map[string][]string, len(matches))
	for _, match := range matches {
		cacheKeysByConversationID[match.id] = match.cacheKey
	}

	for rows.Next() {
		var conversationID int64
		var text string
		if err := rows.Scan(&conversationID, &text); err != nil {
			return nil, fmt.Errorf("rows.Scan: %w", err)
		}
		appendSearchPreview(previews, cacheKeysByConversationID, conversationID, text, terms)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows.Err: %w", err)
	}
	return previews, nil
}

func buildSearchPreviewQuery(conversationIDs []int64) (string, []any) {
	args := make([]any, 0, len(conversationIDs)+1)
	placeholders := make([]string, 0, len(conversationIDs))
	for _, id := range conversationIDs {
		placeholders = append(placeholders, "?")
		args = append(args, id)
	}
	args = append(args, searchPreviewFetchRowsPerConversation)

	query := fmt.Sprintf(
		`WITH ranked_chunks AS (
			SELECT conversation_id, ordinal, text,
			       ROW_NUMBER() OVER (PARTITION BY conversation_id ORDER BY ordinal ASC) AS match_row
			  FROM search_chunks
			 WHERE conversation_id IN (%s)
		)
		SELECT conversation_id, text
		  FROM ranked_chunks
		 WHERE match_row <= ?
		 ORDER BY conversation_id, ordinal ASC`,
		strings.Join(placeholders, ", "),
	)
	return query, args
}

func appendSearchPreview(
	previews map[string][]string,
	cacheKeysByConversationID map[int64]string,
	conversationID int64,
	text string,
	terms []string,
) {
	cacheKey, ok := cacheKeysByConversationID[conversationID]
	if !ok || len(previews[cacheKey]) >= searchPreviewMaxPerConversation {
		return
	}

	lower := strings.ToLower(text)
	if !containsAllTermsLower(lower, terms) {
		return
	}

	preview := matchPreviewLower(text, lower, terms)
	if preview == "" || slices.Contains(previews[cacheKey], preview) {
		return
	}

	previews[cacheKey] = append(previews[cacheKey], preview)
}

func containsAllTermsLower(lower string, terms []string) bool {
	for _, term := range terms {
		if !strings.Contains(lower, strings.ToLower(term)) {
			return false
		}
	}
	return true
}

func matchPreviewLower(text, lower string, terms []string) string {
	if text == "" || len(terms) == 0 {
		return ""
	}

	bestIndex := -1
	bestTerm := ""
	for _, term := range terms {
		index := strings.Index(lower, strings.ToLower(term))
		if index < 0 {
			continue
		}
		if bestIndex == -1 || index < bestIndex {
			bestIndex = index
			bestTerm = term
		}
	}
	if bestIndex < 0 {
		return ""
	}

	startRunes := utf8.RuneCountInString(lower[:bestIndex])
	matchRunes := utf8.RuneCountInString(bestTerm)
	return compactPreview(text, startRunes, matchRunes)
}

func compactPreview(text string, startRunes, matchRunes int) string {
	runes := []rune(text)
	if len(runes) <= searchPreviewMaxRunes {
		return text
	}

	start := max(startRunes-searchPreviewContextRunes, 0)
	end := min(start+searchPreviewMaxRunes, len(runes))
	minEnd := min(startRunes+matchRunes+searchPreviewContextRunes, len(runes))
	if end < minEnd {
		end = minEnd
		start = max(end-searchPreviewMaxRunes, 0)
	}

	snippet := strings.TrimSpace(string(runes[start:end]))
	if start > 0 {
		snippet = "... " + strings.TrimLeft(snippet, " ")
	}
	if end < len(runes) {
		snippet = strings.TrimRight(snippet, " ") + " ..."
	}
	return snippet
}
