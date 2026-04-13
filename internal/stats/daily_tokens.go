package stats

import (
	"slices"
	"time"

	conv "github.com/rkuska/carn/internal/conversation"
)

type dailyTokenKey struct {
	dateKey  string
	provider string
	model    string
	project  string
}

func AggregateDailyTokens(sessions []conv.Session) []conv.DailyTokenRow {
	if len(sessions) == 0 {
		return nil
	}

	rows := make(map[dailyTokenKey]conv.DailyTokenRow)
	for _, session := range sessions {
		startDay := dailyTokenDay(session.Meta.Timestamp)
		if !startDay.IsZero() {
			row := rows[dailyTokenKeyForSession(session, startDay)]
			row = mergeSessionActivityCounts(row, session.Meta, startDay)
			rows[dailyTokenKeyForSession(session, startDay)] = row
		}

		for _, msg := range session.Messages {
			day := dailyTokenMessageDay(session.Meta.Timestamp, msg.Timestamp)
			if day.IsZero() {
				continue
			}

			key := dailyTokenKeyForSession(session, day)
			row := rows[key]
			row.Date = day
			row.Provider = string(session.Meta.Provider)
			row.Model = session.Meta.Model
			row.Project = session.Meta.Project.DisplayName
			row.InputTokens += msg.Usage.InputTokens
			row.CacheCreationTokens += msg.Usage.CacheCreationInputTokens
			row.CacheReadTokens += msg.Usage.CacheReadInputTokens
			row.OutputTokens += msg.Usage.OutputTokens
			row.ReasoningOutputTokens += msg.Usage.ReasoningOutputTokens
			rows[key] = row
		}
	}

	return sortDailyTokenRows(rows)
}

func dailyTokenKeyForSession(session conv.Session, day time.Time) dailyTokenKey {
	return dailyTokenKey{
		dateKey:  day.Format("2006-01-02"),
		provider: string(session.Meta.Provider),
		model:    session.Meta.Model,
		project:  session.Meta.Project.DisplayName,
	}
}

func dailyTokenMessageDay(sessionStart, messageTimestamp time.Time) time.Time {
	if !messageTimestamp.IsZero() {
		return dailyTokenDay(messageTimestamp)
	}
	return dailyTokenDay(sessionStart)
}

func dailyTokenDay(timestamp time.Time) time.Time {
	if timestamp.IsZero() {
		return time.Time{}
	}
	year, month, day := timestamp.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, timestamp.Location())
}

func mergeSessionActivityCounts(
	row conv.DailyTokenRow,
	meta conv.SessionMeta,
	day time.Time,
) conv.DailyTokenRow {
	row.Date = day
	row.Provider = string(meta.Provider)
	row.Model = meta.Model
	row.Project = meta.Project.DisplayName
	row.SessionCount++
	row.MessageCount += sessionMessageCount(meta)
	row.UserMessageCount += meta.UserMessageCount
	row.AssistantMessageCount += meta.AssistantMessageCount
	return row
}

func sortDailyTokenRows(rows map[dailyTokenKey]conv.DailyTokenRow) []conv.DailyTokenRow {
	if len(rows) == 0 {
		return nil
	}

	result := make([]conv.DailyTokenRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, row)
	}
	slices.SortFunc(result, func(left, right conv.DailyTokenRow) int {
		switch {
		case left.Date.Before(right.Date):
			return -1
		case left.Date.After(right.Date):
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
	})
	return result
}
