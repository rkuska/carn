package app

import (
	"testing"
	"time"

	conv "github.com/rkuska/carn/internal/conversation"
	"github.com/rkuska/carn/scenarios/helpers"
)

func BenchmarkStatsOverviewRender(b *testing.B) {
	model := newBenchStatsModel()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = model.renderOverviewTab(120)
	}
}

func BenchmarkStatsHeatmapRender(b *testing.B) {
	model := newBenchStatsModel()
	heatmap := model.snapshot.Activity.Heatmap

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = renderActivityHeatmap("Activity Heatmap", heatmap, 120)
	}
}

func BenchmarkStatsHistogramRender(b *testing.B) {
	model := newBenchStatsModel()
	buckets := make([]histBucket, 0, len(model.snapshot.Sessions.DurationHistogram))
	for _, bucket := range model.snapshot.Sessions.DurationHistogram {
		buckets = append(buckets, histBucket{Label: bucket.Label, Count: bucket.Count})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = renderVerticalHistogram("Session Duration", buckets, 58, 8)
	}
}

func newBenchStatsModel() statsModel {
	return newStatsModel(
		makeBenchStatsConversations(300),
		&fakeBrowserStore{},
		120,
		40,
		newBrowserFilterState(),
	)
}

func makeBenchStatsConversations(n int) []conv.Conversation {
	specs := helpers.GenerateSessionSpecs(n)
	models := []string{"claude-opus-4-1", "claude-sonnet-4", "gpt-5"}
	conversations := make([]conv.Conversation, 0, len(specs))

	for i, spec := range specs {
		mainMessages := 6 + i%32
		meta := conv.SessionMeta{
			ID:                    spec.SessionID,
			Slug:                  spec.Slug,
			Project:               conv.Project{DisplayName: spec.Project},
			Timestamp:             spec.Timestamp,
			LastTimestamp:         spec.Timestamp.Add(time.Duration(5+i%120) * time.Minute),
			Model:                 models[i%len(models)],
			MessageCount:          mainMessages,
			MainMessageCount:      mainMessages,
			UserMessageCount:      mainMessages / 2,
			AssistantMessageCount: mainMessages - mainMessages/2,
			TotalUsage: conv.TokenUsage{
				InputTokens:              1400 + (i%240)*6,
				CacheCreationInputTokens: 30 + i%25,
				CacheReadInputTokens:     80 + i%70,
				OutputTokens:             320 + (i%100)*4,
			},
			ToolCounts: map[string]int{
				"Read":  2 + i%4,
				"Write": 1 + i%3,
				"Bash":  1 + i%5,
			},
		}

		conversations = append(conversations, conv.Conversation{
			Ref:      conv.Ref{Provider: conv.ProviderClaude, ID: spec.SessionID},
			Name:     spec.Slug,
			Project:  meta.Project,
			Sessions: []conv.SessionMeta{meta},
		})
	}

	return conversations
}
