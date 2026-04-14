package stats

import (
	"cmp"
	"slices"
	"time"

	conv "github.com/rkuska/carn/internal/conversation"
)

type activityBucketKey struct {
	bucketStartNS int64
	provider      string
	version       string
	model         string
	project       string
}

func AggregateActivityBuckets(sessions []conv.Session) []conv.ActivityBucketRow {
	if len(sessions) == 0 {
		return nil
	}

	rows := make(map[activityBucketKey]conv.ActivityBucketRow)
	for _, session := range sessions {
		bucketStart := activityBucketStart(session.Meta.Timestamp)
		if !bucketStart.IsZero() {
			key := activityBucketKeyForSession(session, bucketStart)
			rows[key] = mergeSessionActivityCounts(rows[key], session.Meta, bucketStart)
		}

		for _, msg := range session.Messages {
			bucketStart := activityBucketMessageStart(session.Meta.Timestamp, msg.Timestamp)
			if bucketStart.IsZero() {
				continue
			}

			key := activityBucketKeyForSession(session, bucketStart)
			row := rows[key]
			row.BucketStart = bucketStart
			row.Provider = string(session.Meta.Provider)
			row.Version = session.Meta.Version
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

	return sortActivityBucketRows(rows)
}

func activityBucketKeyForSession(session conv.Session, bucketStart time.Time) activityBucketKey {
	return activityBucketKey{
		bucketStartNS: bucketStart.UnixNano(),
		provider:      string(session.Meta.Provider),
		version:       session.Meta.Version,
		model:         session.Meta.Model,
		project:       session.Meta.Project.DisplayName,
	}
}

func activityBucketMessageStart(sessionStart, messageTimestamp time.Time) time.Time {
	if !messageTimestamp.IsZero() {
		return activityBucketStart(messageTimestamp)
	}
	return activityBucketStart(sessionStart)
}

func activityBucketStart(timestamp time.Time) time.Time {
	if timestamp.IsZero() {
		return time.Time{}
	}
	return timestamp.UTC().Truncate(time.Minute)
}

func mergeSessionActivityCounts(
	row conv.ActivityBucketRow,
	meta conv.SessionMeta,
	bucketStart time.Time,
) conv.ActivityBucketRow {
	row.BucketStart = bucketStart
	row.Provider = string(meta.Provider)
	row.Version = meta.Version
	row.Model = meta.Model
	row.Project = meta.Project.DisplayName
	row.SessionCount++
	row.MessageCount += sessionMessageCount(meta)
	row.UserMessageCount += meta.UserMessageCount
	row.AssistantMessageCount += meta.AssistantMessageCount
	return row
}

func sortActivityBucketRows(rows map[activityBucketKey]conv.ActivityBucketRow) []conv.ActivityBucketRow {
	if len(rows) == 0 {
		return nil
	}

	result := make([]conv.ActivityBucketRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, row)
	}
	slices.SortFunc(result, compareActivityBucketRows)
	return result
}

func compareActivityBucketRows(left, right conv.ActivityBucketRow) int {
	return firstCompare(
		cmp.Compare(left.BucketStart.UnixNano(), right.BucketStart.UnixNano()),
		cmp.Compare(left.Provider, right.Provider),
		cmp.Compare(left.Version, right.Version),
		cmp.Compare(left.Model, right.Model),
		cmp.Compare(left.Project, right.Project),
	)
}

func firstCompare(values ...int) int {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}
