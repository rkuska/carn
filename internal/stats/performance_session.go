package stats

import (
	"slices"
	"time"

	conv "github.com/rkuska/carn/internal/conversation"
)

type performanceSession struct {
	provider conv.Provider
	meta     *conv.SessionMeta
}

func (s performanceSession) sessionTimestamp() time.Time {
	if s.meta == nil {
		return time.Time{}
	}
	return s.meta.Timestamp
}

type performanceAggregate struct {
	sessionCount          int
	userTurns             int
	totalTokens           int
	totalActions          int
	readCount             int
	searchCount           int
	webCount              int
	mutateCount           int
	rewriteCount          int
	executeCount          int
	testCount             int
	buildCount            int
	actionErrorCount      int
	actionRejectCount     int
	mutatedSessions       int
	verifiedSessions      int
	abortCount            int
	startedTurnCount      int
	contextPressureCount  int
	apiErrorCount         int
	retryAttemptCount     int
	retryDelayMS          int
	reasoningOutputTokens int
	codexSessionCount     int
	hiddenThinkingCount   int
	reasoningBlockCount   int
	maxThinkingTokens     int
	maxThinkingSamples    int
	cachePromptTokens     int
	cacheReadTokens       int
	outputTokens          int
	inputTokens           int
	stopReasonCounts      map[string]int
	serverToolUseCounts   map[string]int
	effortCounts          map[string]int
}

type performanceBucketAggregate struct {
	bucket    performanceBucket
	aggregate performanceAggregate
}

type performanceMetricContext struct {
	currentSampleCount  int
	baselineSampleCount int
	bucketAggregates    []performanceBucketAggregate
}

func flattenPerformanceSessions(conversations []conv.Conversation) []performanceSession {
	if len(conversations) == 0 {
		return nil
	}
	count := 0
	for _, conversation := range conversations {
		count += len(conversation.Sessions)
	}
	sessions := make([]performanceSession, 0, count)
	for _, conversation := range conversations {
		for i := range conversation.Sessions {
			sessions = append(sessions, performanceSession{
				provider: conversation.Ref.Provider,
				meta:     &conversation.Sessions[i],
			})
		}
	}
	return sessions
}

func newPerformanceMetricContext(
	timeRange TimeRange,
	currentSampleCount, baselineSampleCount int,
	sessions []performanceSession,
) performanceMetricContext {
	return performanceMetricContext{
		currentSampleCount:  currentSampleCount,
		baselineSampleCount: baselineSampleCount,
		bucketAggregates:    buildPerformanceBucketAggregates(sessions, timeRange),
	}
}

func buildPerformanceScope(
	sessions []performanceSession,
	timeRange TimeRange,
	baselineRange TimeRange,
	baselineSessionCount int,
	sequenceSampleCount int,
) PerformanceScope {
	var providers map[string]struct{}
	var models map[string]struct{}
	sessionCount := 0
	for _, session := range sessions {
		if !timeRangeContains(timeRange, session.meta.Timestamp) {
			continue
		}
		sessionCount++
		if label := session.provider.Label(); label != "" {
			if providers == nil {
				providers = make(map[string]struct{})
			}
			providers[label] = struct{}{}
		}
		if model := session.meta.Model; model != "" {
			if models == nil {
				models = make(map[string]struct{})
			}
			models[model] = struct{}{}
		}
	}
	providerLabels := sortedStringKeys(providers)
	modelLabels := sortedStringKeys(models)
	singleProvider := len(providerLabels) == 1
	singleModel := len(modelLabels) == 1
	primaryProvider := ""
	if singleProvider {
		primaryProvider = providerLabels[0]
	}
	primaryModel := ""
	if singleModel {
		primaryModel = modelLabels[0]
	}
	return PerformanceScope{
		SessionCount:         sessionCount,
		Providers:            providerLabels,
		Models:               modelLabels,
		PrimaryProvider:      primaryProvider,
		PrimaryModel:         primaryModel,
		SingleProvider:       singleProvider,
		SingleModel:          singleModel,
		SingleFamily:         singleProvider && singleModel,
		CurrentRange:         timeRange,
		BaselineRange:        baselineRange,
		SequenceLoaded:       sequenceSampleCount > 0,
		SequenceSampleCount:  sequenceSampleCount,
		BaselineSessionCount: baselineSessionCount,
	}
}

func aggregatePerformanceSessionsInRange(
	sessions []performanceSession,
	timeRange TimeRange,
) performanceAggregate {
	var agg performanceAggregate
	for _, session := range sessions {
		if !timeRangeContains(timeRange, session.meta.Timestamp) {
			continue
		}
		addPerformanceSession(&agg, session)
	}
	return agg
}

func addPerformanceSession(agg *performanceAggregate, session performanceSession) {
	if session.meta == nil {
		return
	}
	agg.sessionCount++
	agg.userTurns += session.meta.UserMessageCount
	agg.totalTokens += performanceSessionTotalTokens(session)
	agg.totalActions += sumCountMap(session.meta.ActionCounts)
	agg.readCount += actionCount(session.meta.ActionCounts, conv.NormalizedActionRead)
	agg.searchCount += actionCount(session.meta.ActionCounts, conv.NormalizedActionSearch)
	agg.webCount += actionCount(session.meta.ActionCounts, conv.NormalizedActionWeb)
	agg.mutateCount += actionCount(session.meta.ActionCounts, conv.NormalizedActionMutate)
	agg.rewriteCount += actionCount(session.meta.ActionCounts, conv.NormalizedActionRewrite)
	agg.executeCount += actionCount(session.meta.ActionCounts, conv.NormalizedActionExecute)
	agg.testCount += actionCount(session.meta.ActionCounts, conv.NormalizedActionTest)
	agg.buildCount += actionCount(session.meta.ActionCounts, conv.NormalizedActionBuild)
	agg.actionErrorCount += sumCountMap(session.meta.ActionErrorCounts)
	agg.actionRejectCount += sumCountMap(session.meta.ActionRejectCounts)
	agg.abortCount += session.meta.Performance.AbortCount
	agg.startedTurnCount += max(session.meta.Performance.TaskStartedCount, session.meta.Performance.TaskCompleteCount)
	agg.retryAttemptCount += session.meta.Performance.RetryAttemptCount
	agg.retryDelayMS += session.meta.Performance.RetryDelayMS
	agg.apiErrorCount += sumCountMap(session.meta.Performance.APIErrorCounts)
	agg.reasoningOutputTokens += session.meta.TotalUsage.ReasoningOutputTokens
	agg.hiddenThinkingCount += session.meta.Performance.ReasoningRedactionCount
	agg.reasoningBlockCount += session.meta.Performance.ReasoningBlockCount
	agg.inputTokens += session.meta.TotalUsage.InputTokens
	agg.outputTokens += session.meta.TotalUsage.OutputTokens
	agg.cachePromptTokens += performanceSessionPromptTokens(session)
	agg.cacheReadTokens += session.meta.TotalUsage.CacheReadInputTokens
	if session.meta.Performance.MaxThinkingTokens > 0 {
		agg.maxThinkingTokens += session.meta.Performance.MaxThinkingTokens
		agg.maxThinkingSamples++
	}
	if session.provider == conv.ProviderCodex {
		agg.codexSessionCount++
	}
	if hasMutation(session.meta.ActionCounts) {
		agg.mutatedSessions++
		if verificationPassed(*session.meta) {
			agg.verifiedSessions++
		}
	}
	if hasContextPressure(session.meta.Performance) {
		agg.contextPressureCount++
	}
	addCountMap(&agg.stopReasonCounts, session.meta.Performance.StopReasonCounts)
	addCountMap(&agg.serverToolUseCounts, session.meta.Performance.ServerToolUseCounts)
	addCountMap(&agg.effortCounts, session.meta.Performance.EffortCounts)
}

func buildPerformanceBucketAggregates(
	sessions []performanceSession,
	timeRange TimeRange,
) []performanceBucketAggregate {
	buckets := performanceBuckets(timeRange)
	if len(buckets) == 0 {
		return nil
	}

	aggregates := make([]performanceBucketAggregate, len(buckets))
	for i, bucket := range buckets {
		aggregates[i].bucket = bucket
	}
	step := performanceBucketStep(buckets)
	for _, session := range sessions {
		timestamp := session.meta.Timestamp
		if !timeRangeContains(timeRange, timestamp) {
			continue
		}
		index := performanceBucketIndex(timestamp, buckets[0].start, step, len(buckets))
		addPerformanceSession(&aggregates[index].aggregate, session)
	}
	return aggregates
}

func performanceBucketStep(buckets []performanceBucket) time.Duration {
	if len(buckets) == 0 {
		return 0
	}
	return buckets[0].end.Sub(buckets[0].start) + time.Nanosecond
}

func performanceBucketIndex(
	timestamp, start time.Time,
	step time.Duration,
	bucketCount int,
) int {
	if bucketCount <= 1 || step <= 0 {
		return 0
	}
	index := int(timestamp.Sub(start) / step)
	if index < 0 {
		return 0
	}
	if index >= bucketCount {
		return bucketCount - 1
	}
	return index
}

func actionCount(counts map[string]int, action conv.NormalizedActionType) int {
	if len(counts) == 0 {
		return 0
	}
	return counts[string(action)]
}

func hasMutation(counts map[string]int) bool {
	return actionCount(counts, conv.NormalizedActionMutate)+actionCount(counts, conv.NormalizedActionRewrite) > 0
}

func verificationPassed(meta conv.SessionMeta) bool {
	total := actionCount(meta.ActionCounts, conv.NormalizedActionTest) +
		actionCount(meta.ActionCounts, conv.NormalizedActionBuild)
	errors := actionCount(meta.ActionErrorCounts, conv.NormalizedActionTest) +
		actionCount(meta.ActionErrorCounts, conv.NormalizedActionBuild)
	rejects := actionCount(meta.ActionRejectCounts, conv.NormalizedActionTest) +
		actionCount(meta.ActionRejectCounts, conv.NormalizedActionBuild)
	return total > 0 && total > errors+rejects
}

func hasContextPressure(meta conv.SessionPerformanceMeta) bool {
	return meta.CompactionCount > 0 ||
		meta.MicroCompactionCount > 0 ||
		meta.ContextCompactedCount > 0
}

func sumCountMap(counts map[string]int) int {
	total := 0
	for _, value := range counts {
		total += value
	}
	return total
}

func addCountMap(dst *map[string]int, counts map[string]int) {
	if len(counts) == 0 {
		return
	}
	if *dst == nil {
		*dst = make(map[string]int, len(counts))
	}
	for key, value := range counts {
		(*dst)[key] += value
	}
}

func topCountEntry(counts map[string]int) (string, int) {
	if len(counts) == 0 {
		return "", 0
	}
	names := make([]string, 0, len(counts))
	for name := range counts {
		names = append(names, name)
	}
	slices.Sort(names)
	bestName := ""
	bestValue := 0
	for _, name := range names {
		if counts[name] > bestValue {
			bestName = name
			bestValue = counts[name]
		}
	}
	return bestName, bestValue
}

func singleProviderInRange(sessions []performanceSession, timeRange TimeRange) conv.Provider {
	provider := conv.Provider("")
	for _, session := range sessions {
		if !timeRangeContains(timeRange, session.meta.Timestamp) {
			continue
		}
		if provider == "" {
			provider = session.provider
			continue
		}
		if session.provider != provider {
			return ""
		}
	}
	return provider
}

func allPerformanceMetrics(performance Performance) []PerformanceMetric {
	metrics := make([]PerformanceMetric, 0,
		len(performance.Outcome.Metrics)+
			len(performance.Discipline.Metrics)+
			len(performance.Efficiency.Metrics)+
			len(performance.Robustness.Metrics),
	)
	metrics = append(metrics, performance.Outcome.Metrics...)
	metrics = append(metrics, performance.Discipline.Metrics...)
	metrics = append(metrics, performance.Efficiency.Metrics...)
	metrics = append(metrics, performance.Robustness.Metrics...)
	return metrics
}
