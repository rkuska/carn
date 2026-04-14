package app

import (
	"context"

	conv "github.com/rkuska/carn/internal/conversation"
)

type fakeBrowserStore struct {
	listResult              []conv.Conversation
	listErr                 error
	loadResult              conv.Session
	loadErr                 error
	loadCalls               int
	loadSessionResult       conv.Session
	loadSessionResults      map[string]conv.Session
	loadSessionErr          error
	loadSessionCalls        int
	loadSessionIDs          []string
	deepSearchCalls         int
	deepSearchResults       map[string][]conv.Conversation
	deepSearchErr           error
	sequenceErr             error
	sequenceRows            []conv.PerformanceSequenceSession
	sequenceRowsByKey       map[string][]conv.PerformanceSequenceSession
	turnMetricErr           error
	turnMetricRows          []conv.SessionTurnMetrics
	turnMetricRowsByKey     map[string][]conv.SessionTurnMetrics
	activityBucketErr       error
	activityBucketRows      []conv.ActivityBucketRow
	activityBucketRowsByKey map[string][]conv.ActivityBucketRow
}

func (s *fakeBrowserStore) List(context.Context, string) ([]conv.Conversation, error) {
	return s.listResult, s.listErr
}

func (s *fakeBrowserStore) Load(
	context.Context,
	string,
	conv.Conversation,
) (conv.Session, error) {
	s.loadCalls++
	return s.loadResult, s.loadErr
}

func (s *fakeBrowserStore) LoadSession(
	_ context.Context,
	_ conv.Conversation,
	sessionMeta conv.SessionMeta,
) (conv.Session, error) {
	s.loadSessionCalls++
	s.loadSessionIDs = append(s.loadSessionIDs, sessionMeta.ID)
	if s.loadSessionErr != nil {
		return conv.Session{}, s.loadSessionErr
	}
	if result, ok := s.loadSessionResults[sessionMeta.ID]; ok {
		return result, nil
	}
	return s.loadSessionResult, nil
}

func (s *fakeBrowserStore) DeepSearch(
	_ context.Context,
	_ string,
	query string,
	conversations []conv.Conversation,
) ([]conv.Conversation, error) {
	s.deepSearchCalls++
	if s.deepSearchErr != nil {
		return conversations, s.deepSearchErr
	}
	if query == "" {
		return conversations, nil
	}
	if results, ok := s.deepSearchResults[query]; ok {
		return results, nil
	}
	return nil, nil
}

func (s *fakeBrowserStore) QueryPerformanceSequence(
	_ context.Context,
	_ string,
	cacheKeys []string,
) ([]conv.PerformanceSequenceSession, error) {
	if s.sequenceErr != nil {
		return nil, s.sequenceErr
	}
	if len(s.sequenceRowsByKey) > 0 {
		rows := make([]conv.PerformanceSequenceSession, 0)
		for _, key := range cacheKeys {
			rows = append(rows, s.sequenceRowsByKey[key]...)
		}
		return rows, nil
	}
	return append([]conv.PerformanceSequenceSession(nil), s.sequenceRows...), nil
}

func (s *fakeBrowserStore) QueryTurnMetrics(
	_ context.Context,
	_ string,
	cacheKeys []string,
) ([]conv.SessionTurnMetrics, error) {
	if s.turnMetricErr != nil {
		return nil, s.turnMetricErr
	}
	if len(s.turnMetricRowsByKey) > 0 {
		rows := make([]conv.SessionTurnMetrics, 0)
		for _, key := range cacheKeys {
			rows = append(rows, s.turnMetricRowsByKey[key]...)
		}
		return rows, nil
	}
	return append([]conv.SessionTurnMetrics(nil), s.turnMetricRows...), nil
}

func (s *fakeBrowserStore) QueryActivityBuckets(
	_ context.Context,
	_ string,
	cacheKeys []string,
) ([]conv.ActivityBucketRow, error) {
	if s.activityBucketErr != nil {
		return nil, s.activityBucketErr
	}
	if len(s.activityBucketRowsByKey) > 0 {
		rows := make([]conv.ActivityBucketRow, 0)
		for _, key := range cacheKeys {
			rows = append(rows, s.activityBucketRowsByKey[key]...)
		}
		return rows, nil
	}
	return append([]conv.ActivityBucketRow(nil), s.activityBucketRows...), nil
}
