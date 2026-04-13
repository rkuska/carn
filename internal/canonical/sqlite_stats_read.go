package canonical

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog"

	conv "github.com/rkuska/carn/internal/conversation"
)

func readStatsPerformanceSequence(
	ctx context.Context,
	db *sql.DB,
	cacheKeys []string,
) ([]conv.PerformanceSequenceSession, error) {
	rows, err := queryStatsRows(
		ctx,
		db,
		cacheKeys,
		`SELECT timestamp_ns, mutated, mutation_count, rewrite_count,
		        targeted_mutation_count, blind_mutation_count,
		        distinct_mutation_targets, patch_hunk_count,
		        verification_passed, first_pass_resolved,
		        correction_followups, reasoning_loop_count, action_count,
		        actions_before_first_mutation, tokens_before_first_mutation,
		        user_turns_before_first_mutation, assistant_turns,
		        visible_reasoning_chars, hidden_thinking_turns
		   FROM stats_performance_sequence
		  WHERE conversation_cache_key IN (%s)
		  ORDER BY timestamp_ns, conversation_cache_key, session_ordinal`,
	)
	if err != nil {
		return nil, fmt.Errorf("queryStatsRows: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			zerolog.Ctx(ctx).Warn().Err(err).Msg("rows.Close")
		}
	}()

	result := make([]conv.PerformanceSequenceSession, 0)
	for rows.Next() {
		var timestampNS int64
		var row conv.PerformanceSequenceSession
		var mutated int
		var verificationPassed int
		var firstPassResolved int
		if err := rows.Scan(
			&timestampNS,
			&mutated,
			&row.MutationCount,
			&row.RewriteCount,
			&row.TargetedMutationCount,
			&row.BlindMutationCount,
			&row.DistinctMutationTargets,
			&row.PatchHunkCount,
			&verificationPassed,
			&firstPassResolved,
			&row.CorrectionFollowups,
			&row.ReasoningLoopCount,
			&row.ActionCount,
			&row.ActionsBeforeFirstMutation,
			&row.TokensBeforeFirstMutation,
			&row.UserTurnsBeforeFirstMutation,
			&row.AssistantTurns,
			&row.VisibleReasoningChars,
			&row.HiddenThinkingTurns,
		); err != nil {
			return nil, fmt.Errorf("rows.Scan: %w", err)
		}
		if timestampNS != 0 {
			row.Timestamp = unixTime(timestampNS)
		}
		row.Mutated = mutated == 1
		row.VerificationPassed = verificationPassed == 1
		row.FirstPassResolved = firstPassResolved == 1
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows.Err: %w", err)
	}
	return result, nil
}

func readStatsTurnMetrics(
	ctx context.Context,
	db *sql.DB,
	cacheKeys []string,
) ([]conv.SessionTurnMetrics, error) {
	rows, err := queryStatsRows(
		ctx,
		db,
		cacheKeys,
		`SELECT timestamp_ns, turns_json
		   FROM stats_turn_metrics
		  WHERE conversation_cache_key IN (%s)
		  ORDER BY timestamp_ns, conversation_cache_key, session_ordinal`,
	)
	if err != nil {
		return nil, fmt.Errorf("queryStatsRows: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			zerolog.Ctx(ctx).Warn().Err(err).Msg("rows.Close")
		}
	}()

	result := make([]conv.SessionTurnMetrics, 0)
	for rows.Next() {
		var timestampNS int64
		var turnsJSON string
		row := conv.SessionTurnMetrics{}
		if err := rows.Scan(&timestampNS, &turnsJSON); err != nil {
			return nil, fmt.Errorf("rows.Scan: %w", err)
		}
		if timestampNS != 0 {
			row.Timestamp = unixTime(timestampNS)
		}
		turns, err := unmarshalTurnTokens(turnsJSON)
		if err != nil {
			return nil, fmt.Errorf("unmarshalTurnTokens: %w", err)
		}
		row.Turns = turns
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows.Err: %w", err)
	}
	return result, nil
}

func readStatsDailyTokens(
	ctx context.Context,
	db *sql.DB,
	cacheKeys []string,
) ([]conv.DailyTokenRow, error) {
	rows, err := queryStatsRows(
		ctx,
		db,
		cacheKeys,
		`SELECT date_key, provider, model, project,
		        SUM(session_count), SUM(message_count),
		        SUM(user_message_count), SUM(assistant_message_count),
		        SUM(input_tokens), SUM(cache_creation_tokens),
		        SUM(cache_read_tokens), SUM(output_tokens),
		        SUM(reasoning_output_tokens)
		   FROM stats_daily_tokens
		  WHERE conversation_cache_key IN (%s)
		  GROUP BY date_key, provider, model, project
		  ORDER BY date_key, provider, model, project`,
	)
	if err != nil {
		return nil, fmt.Errorf("queryStatsRows: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			zerolog.Ctx(ctx).Warn().Err(err).Msg("rows.Close")
		}
	}()

	result := make([]conv.DailyTokenRow, 0)
	for rows.Next() {
		var dateKey string
		row := conv.DailyTokenRow{}
		if err := rows.Scan(
			&dateKey,
			&row.Provider,
			&row.Model,
			&row.Project,
			&row.SessionCount,
			&row.MessageCount,
			&row.UserMessageCount,
			&row.AssistantMessageCount,
			&row.InputTokens,
			&row.CacheCreationTokens,
			&row.CacheReadTokens,
			&row.OutputTokens,
			&row.ReasoningOutputTokens,
		); err != nil {
			return nil, fmt.Errorf("rows.Scan: %w", err)
		}
		date, err := time.ParseInLocation("2006-01-02", dateKey, time.UTC)
		if err != nil {
			return nil, fmt.Errorf("time.ParseInLocation: %w", err)
		}
		row.Date = date
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows.Err: %w", err)
	}
	return result, nil
}

func queryStatsRows(
	ctx context.Context,
	db *sql.DB,
	cacheKeys []string,
	queryFormat string,
) (*sql.Rows, error) {
	args := make([]any, 0, len(cacheKeys))
	placeholders := make([]string, 0, len(cacheKeys))
	for _, key := range cacheKeys {
		placeholders = append(placeholders, "?")
		args = append(args, key)
	}

	rows, err := db.QueryContext(ctx, fmt.Sprintf(queryFormat, strings.Join(placeholders, ", ")), args...)
	if err != nil {
		return nil, fmt.Errorf("db.QueryContext: %w", err)
	}
	return rows, nil
}
