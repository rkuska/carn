package stats

import conv "github.com/rkuska/carn/internal/conversation"

func FilterByTimeRange(sessions []conv.SessionMeta, tr TimeRange) []conv.SessionMeta {
	if len(sessions) == 0 {
		return nil
	}
	if tr.Start.IsZero() && tr.End.IsZero() {
		return append([]conv.SessionMeta(nil), sessions...)
	}

	filtered := make([]conv.SessionMeta, 0, len(sessions))
	for _, session := range sessions {
		if !tr.Start.IsZero() && session.Timestamp.Before(tr.Start) {
			continue
		}
		if !tr.End.IsZero() && session.Timestamp.After(tr.End) {
			continue
		}
		filtered = append(filtered, session)
	}
	return filtered
}
