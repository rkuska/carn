package canonical

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	conv "github.com/rkuska/carn/internal/conversation"
)

var errSourceNotRegistered = errors.New("source is not registered")

func writeStatsPerformanceSequence(
	ctx context.Context,
	tx *sql.Tx,
	cacheKey string,
	ordinal int,
	row conv.PerformanceSequenceSession,
) error {
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO stats_performance_sequence(
			conversation_cache_key, session_ordinal, timestamp_ns, mutated,
			mutation_count, rewrite_count, targeted_mutation_count,
			blind_mutation_count, distinct_mutation_targets, patch_hunk_count,
			verification_passed, first_pass_resolved, correction_followups,
			reasoning_loop_count, action_count, actions_before_first_mutation,
			tokens_before_first_mutation, user_turns_before_first_mutation,
			assistant_turns, visible_reasoning_chars, hidden_thinking_turns
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		cacheKey,
		ordinal,
		timeToUnixNano(row.Timestamp),
		boolToInt(row.Mutated),
		row.MutationCount,
		row.RewriteCount,
		row.TargetedMutationCount,
		row.BlindMutationCount,
		row.DistinctMutationTargets,
		row.PatchHunkCount,
		boolToInt(row.VerificationPassed),
		boolToInt(row.FirstPassResolved),
		row.CorrectionFollowups,
		row.ReasoningLoopCount,
		row.ActionCount,
		row.ActionsBeforeFirstMutation,
		row.TokensBeforeFirstMutation,
		row.UserTurnsBeforeFirstMutation,
		row.AssistantTurns,
		row.VisibleReasoningChars,
		row.HiddenThinkingTurns,
	); err != nil {
		return fmt.Errorf("tx.ExecContext: %w", err)
	}
	return nil
}

func writeStatsTurnMetrics(
	ctx context.Context,
	tx *sql.Tx,
	cacheKey string,
	ordinal int,
	row conv.SessionTurnMetrics,
) error {
	turnsJSON, err := marshalTurnTokens(row.Turns)
	if err != nil {
		return fmt.Errorf("marshalTurnTokens: %w", err)
	}
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO stats_turn_metrics(
			conversation_cache_key, session_ordinal, timestamp_ns, turns_json
		) VALUES (?, ?, ?, ?)`,
		cacheKey,
		ordinal,
		timeToUnixNano(row.Timestamp),
		turnsJSON,
	); err != nil {
		return fmt.Errorf("tx.ExecContext: %w", err)
	}
	return nil
}

func writeStatsActivityBuckets(
	ctx context.Context,
	tx *sql.Tx,
	cacheKey string,
	rows []conv.ActivityBucketRow,
) error {
	for _, row := range rows {
		if row.BucketStart.IsZero() {
			continue
		}
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO stats_activity_buckets(
				conversation_cache_key, bucket_start_ns, provider, model, project,
				session_count, message_count, user_message_count,
				assistant_message_count, input_tokens, cache_creation_tokens,
				cache_read_tokens, output_tokens, reasoning_output_tokens
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			cacheKey,
			timeToUnixNano(row.BucketStart),
			row.Provider,
			row.Model,
			row.Project,
			row.SessionCount,
			row.MessageCount,
			row.UserMessageCount,
			row.AssistantMessageCount,
			row.InputTokens,
			row.CacheCreationTokens,
			row.CacheReadTokens,
			row.OutputTokens,
			row.ReasoningOutputTokens,
		); err != nil {
			return fmt.Errorf("tx.ExecContext: %w", err)
		}
	}
	return nil
}

func deleteStatsByCacheKeys(ctx context.Context, tx *sql.Tx, cacheKeys []string) error {
	if len(cacheKeys) == 0 {
		return nil
	}

	args := make([]any, 0, len(cacheKeys))
	placeholders := make([]string, 0, len(cacheKeys))
	for _, key := range cacheKeys {
		placeholders = append(placeholders, "?")
		args = append(args, key)
	}

	for _, table := range []string{
		"stats_performance_sequence",
		"stats_turn_metrics",
		"stats_activity_buckets",
	} {
		query := fmt.Sprintf(
			`DELETE FROM %s WHERE conversation_cache_key IN (%s)`,
			table,
			strings.Join(placeholders, ", "),
		)
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("tx.ExecContext_%s: %w", table, err)
		}
	}
	return nil
}

func marshalTurnTokens(turns []conv.TurnTokens) (string, error) {
	if turns == nil {
		turns = []conv.TurnTokens{}
	}
	encoded, err := json.Marshal(turns)
	if err != nil {
		return "", fmt.Errorf("json.Marshal: %w", err)
	}
	return string(encoded), nil
}

func unmarshalTurnTokens(raw string) ([]conv.TurnTokens, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	var turns []conv.TurnTokens
	if err := json.Unmarshal([]byte(raw), &turns); err != nil {
		return nil, fmt.Errorf("json.Unmarshal: %w", err)
	}
	return turns, nil
}

type activityBucketRowKey struct {
	bucketStartNS int64
	provider      string
	model         string
	project       string
}

func collectConversationStatsData(
	ctx context.Context,
	sources sourceRegistry,
	collector StatsCollector,
	convValue conversation,
	preloadedSessions []sessionFull,
) ([]conv.SessionStatsData, []conv.ActivityBucketRow, error) {
	source, ok := sources.lookup(conversationProvider(convValue.Ref.Provider))
	if !ok {
		return nil, nil, fmt.Errorf("collectConversationStatsData: %w", errSourceNotRegistered)
	}

	if len(preloadedSessions) == len(convValue.Sessions) && len(preloadedSessions) > 0 {
		sessions := statsSessionsWithMeta(convValue, preloadedSessions)
		return collectLoadedConversationStats(collector, sessions), aggregateActivityBuckets(sessions), nil
	}

	statsData := make([]conv.SessionStatsData, 0, len(convValue.Sessions))
	loadedSessions := make([]sessionFull, 0, len(convValue.Sessions))
	for _, meta := range convValue.Sessions {
		session, err := source.LoadSession(ctx, convValue, meta)
		if err != nil {
			return nil, nil, fmt.Errorf("source.LoadSession_%s: %w", meta.ID, err)
		}
		session = statsSessionWithMeta(session, convValue, meta)
		loadedSessions = append(loadedSessions, session)
		if sessionStats, ok := collectSessionStatsData(collector, session); ok {
			statsData = append(statsData, sessionStats)
		}
	}
	return statsData, aggregateActivityBuckets(loadedSessions), nil
}

func collectLoadedConversationStats(
	collector StatsCollector,
	sessions []sessionFull,
) []conv.SessionStatsData {
	if collector == nil || len(sessions) == 0 {
		return nil
	}
	statsData := make([]conv.SessionStatsData, 0, len(sessions))
	for _, session := range sessions {
		if sessionStats, ok := collectSessionStatsData(collector, session); ok {
			statsData = append(statsData, sessionStats)
		}
	}
	return statsData
}

func statsSessionWithMeta(session sessionFull, convValue conversation, meta sessionMeta) sessionFull {
	session.Meta = meta
	session.Meta.Provider = convValue.Ref.Provider
	session.Meta.Project = convValue.Project
	return session
}

func statsSessionsWithMeta(convValue conversation, loadedSessions []sessionFull) []sessionFull {
	sessions := make([]sessionFull, len(loadedSessions))
	for i := range loadedSessions {
		sessions[i] = statsSessionWithMeta(loadedSessions[i], convValue, convValue.Sessions[i])
	}
	return sessions
}

func collectSessionStatsData(collector StatsCollector, session sessionFull) (conv.SessionStatsData, bool) {
	if collector == nil {
		return conv.SessionStatsData{}, false
	}
	return collector.CollectSessionStats(session), true
}

func aggregateActivityBuckets(sessions []sessionFull) []conv.ActivityBucketRow {
	if len(sessions) == 0 {
		return nil
	}

	rows := make(map[activityBucketRowKey]conv.ActivityBucketRow)
	for _, session := range sessions {
		aggregateSessionActivityRow(rows, session)
		aggregateSessionMessageTokens(rows, session)
	}

	result := make([]conv.ActivityBucketRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, row)
	}
	slices.SortFunc(result, compareActivityBucketRow)
	return result
}

func aggregateSessionActivityRow(rows map[activityBucketRowKey]conv.ActivityBucketRow, session sessionFull) {
	bucketStart := statsBucketStart(session.Meta.Timestamp)
	if bucketStart.IsZero() {
		return
	}

	key := activityBucketKeyForMeta(session.Meta, bucketStart)
	row := activityBucketBaseRow(session.Meta, bucketStart, rows[key])
	row.SessionCount++
	row.MessageCount += statsSessionMessageCount(session.Meta)
	row.UserMessageCount += session.Meta.UserMessageCount
	row.AssistantMessageCount += session.Meta.AssistantMessageCount
	rows[key] = row
}

func aggregateSessionMessageTokens(rows map[activityBucketRowKey]conv.ActivityBucketRow, session sessionFull) {
	for _, msg := range session.Messages {
		bucketStart := statsMessageBucketStart(session.Meta.Timestamp, msg.Timestamp)
		if bucketStart.IsZero() {
			continue
		}

		key := activityBucketKeyForMeta(session.Meta, bucketStart)
		row := activityBucketBaseRow(session.Meta, bucketStart, rows[key])
		row.InputTokens += msg.Usage.InputTokens
		row.CacheCreationTokens += msg.Usage.CacheCreationInputTokens
		row.CacheReadTokens += msg.Usage.CacheReadInputTokens
		row.OutputTokens += msg.Usage.OutputTokens
		row.ReasoningOutputTokens += msg.Usage.ReasoningOutputTokens
		rows[key] = row
	}
}

func activityBucketBaseRow(meta sessionMeta, bucketStart time.Time, row conv.ActivityBucketRow) conv.ActivityBucketRow {
	row.BucketStart = bucketStart
	row.Provider = string(meta.Provider)
	row.Model = meta.Model
	row.Project = meta.Project.DisplayName
	return row
}

func compareActivityBucketRow(left, right conv.ActivityBucketRow) int {
	switch {
	case left.BucketStart.Before(right.BucketStart):
		return -1
	case left.BucketStart.After(right.BucketStart):
		return 1
	case left.Provider < right.Provider:
		return -1
	case left.Provider > right.Provider:
		return 1
	case left.Model < right.Model:
		return -1
	case left.Model > right.Model:
		return 1
	case left.Project < right.Project:
		return -1
	case left.Project > right.Project:
		return 1
	default:
		return 0
	}
}

func activityBucketKeyForMeta(meta sessionMeta, bucketStart time.Time) activityBucketRowKey {
	return activityBucketRowKey{
		bucketStartNS: bucketStart.UnixNano(),
		provider:      string(meta.Provider),
		model:         meta.Model,
		project:       meta.Project.DisplayName,
	}
}

func statsSessionMessageCount(meta sessionMeta) int {
	if meta.IsSubagent && meta.MessageCount > 0 {
		return meta.MessageCount
	}
	return meta.MainMessageCount
}

func statsMessageBucketStart(sessionStart, messageTimestamp time.Time) time.Time {
	if !messageTimestamp.IsZero() {
		return statsBucketStart(messageTimestamp)
	}
	return statsBucketStart(sessionStart)
}

func statsBucketStart(timestamp time.Time) time.Time {
	if timestamp.IsZero() {
		return time.Time{}
	}
	return timestamp.UTC().Truncate(time.Minute)
}
