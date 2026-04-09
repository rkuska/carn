package stats

import (
	"strconv"
	"testing"

	conv "github.com/rkuska/carn/internal/conversation"
)

func BenchmarkComputePerformance(b *testing.B) {
	for _, size := range []int{100, 1000} {
		conversations := makeBenchPerformanceConversations(size)
		timeRange := benchConversationTimeRange(conversations)
		b.Run(strconv.Itoa(size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = ComputePerformance(conversations, timeRange, nil)
			}
		})
	}
}

func BenchmarkComputePerformanceWithSequence(b *testing.B) {
	for _, size := range []int{100, 1000} {
		conversations, sessions := makeBenchPerformanceData(size)
		timeRange := benchConversationTimeRange(conversations)
		sequence := CollectPerformanceSequenceSessions(sessions)
		b.Run(strconv.Itoa(size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = ComputePerformance(conversations, timeRange, sequence)
			}
		})
	}
}

func BenchmarkCollectPerformanceSequenceSessions(b *testing.B) {
	for _, size := range []int{100, 1000} {
		_, sessions := makeBenchPerformanceData(size)
		b.Run(strconv.Itoa(size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = CollectPerformanceSequenceSessions(sessions)
			}
		})
	}
}

func makeBenchPerformanceConversations(n int) []conversation {
	conversations, _ := makeBenchPerformanceData(n)
	return conversations
}

func makeBenchPerformanceData(n int) ([]conversation, []session) {
	metas := makeBenchPerformanceMetas(n)
	conversations := make([]conversation, 0, len(metas))
	sessions := make([]session, 0, len(metas))

	for i, meta := range metas {
		provider := conv.ProviderClaude
		if i%3 == 0 {
			provider = conv.ProviderCodex
		}

		conversations = append(conversations, conversation{
			Ref:      conv.Ref{Provider: provider, ID: meta.ID},
			Name:     meta.ID,
			Project:  meta.Project,
			Sessions: []sessionMeta{meta},
		})
		sessions = append(sessions, session{
			Meta:     meta,
			Messages: makeBenchPerformanceMessages(i),
		})
	}
	return conversations, sessions
}

func makeBenchPerformanceMetas(n int) []sessionMeta {
	metas := makeBenchSessionMetas(n)
	for i := range metas {
		metas[i].ActionCounts = map[string]int{
			string(conv.NormalizedActionRead):   1 + i%3,
			string(conv.NormalizedActionSearch): i % 2,
			string(conv.NormalizedActionMutate): 1 + i%2,
			string(conv.NormalizedActionTest):   1,
		}
		if i%7 == 0 {
			metas[i].ActionCounts[string(conv.NormalizedActionRewrite)] = 1
		}
		if i%5 == 0 {
			metas[i].ActionErrorCounts = map[string]int{
				string(conv.NormalizedActionTest): 1,
			}
		}
		if i%9 == 0 {
			metas[i].ActionRejectCounts = map[string]int{
				string(conv.NormalizedActionMutate): 1,
			}
		}
		metas[i].Performance.TaskStartedCount = 2
		metas[i].Performance.TaskCompleteCount = 1
		metas[i].Performance.AbortCount = i % 2
		metas[i].Performance.RetryAttemptCount = i % 3
		metas[i].Performance.APIErrorCounts = map[string]int{"transport": i % 2}
		if i%6 == 0 {
			metas[i].Performance.CompactionCount = 1
		}
		if i%3 == 0 {
			metas[i].Performance.EffortCounts = map[string]int{"medium": 1}
			metas[i].TotalUsage.ReasoningOutputTokens = 80 + i%40
			continue
		}
		metas[i].Performance.StopReasonCounts = map[string]int{"end_turn": 1}
		metas[i].Performance.MaxThinkingTokens = 2048 + i%512
		metas[i].Performance.ReasoningBlockCount = 1
		metas[i].Performance.ReasoningRedactionCount = i % 2
		metas[i].Performance.ServerToolUseCounts = map[string]int{"web_search": i % 2}
	}
	return metas
}

func makeBenchPerformanceMessages(i int) []message {
	filePath := "/repo/file" + strconv.Itoa(i%32) + ".go"
	messages := []message{
		{Role: conv.RoleUser, Text: "fix benchmark issue"},
	}
	if i%3 != 0 {
		messages = append(messages, message{
			Role:     conv.RoleAssistant,
			Thinking: "inspect first",
			ToolCalls: []conv.ToolCall{{
				Name: "Read",
				Action: conv.NormalizedAction{
					Type:    conv.NormalizedActionRead,
					Targets: []conv.ActionTarget{{Type: conv.ActionTargetFilePath, Value: filePath}},
				},
			}},
		})
	}
	messages = append(messages, message{
		Role:              conv.RoleAssistant,
		HasHiddenThinking: i%4 == 0,
		ToolCalls: []conv.ToolCall{{
			Name: "Edit",
			Action: conv.NormalizedAction{
				Type:    conv.NormalizedActionMutate,
				Targets: []conv.ActionTarget{{Type: conv.ActionTargetFilePath, Value: filePath}},
			},
		}},
	})
	messages = append(messages, message{
		Role: conv.RoleUser,
		ToolResults: []conv.ToolResult{{
			ToolName: "Edit",
			IsError:  i%5 == 0,
			Action: conv.NormalizedAction{
				Type:    conv.NormalizedActionMutate,
				Targets: []conv.ActionTarget{{Type: conv.ActionTargetFilePath, Value: filePath}},
			},
			StructuredPatch: []conv.DiffHunk{
				{OldStart: 1, OldLines: 1, NewStart: 1, NewLines: 1},
				{OldStart: 3, OldLines: 1, NewStart: 3, NewLines: 2},
			},
		}},
	})
	if i%2 == 0 {
		messages = append(messages,
			message{
				Role: conv.RoleAssistant,
				ToolCalls: []conv.ToolCall{{
					Name: "Bash",
					Action: conv.NormalizedAction{
						Type:    conv.NormalizedActionTest,
						Targets: []conv.ActionTarget{{Type: conv.ActionTargetCommand, Value: "go test ./..."}},
					},
				}},
			},
			message{
				Role: conv.RoleUser,
				ToolResults: []conv.ToolResult{{
					ToolName: "Bash",
					Action: conv.NormalizedAction{
						Type:    conv.NormalizedActionTest,
						Targets: []conv.ActionTarget{{Type: conv.ActionTargetCommand, Value: "go test ./..."}},
					},
				}},
			},
		)
		return messages
	}

	messages = append(messages,
		message{Role: conv.RoleUser, Text: "tighten the fix"},
		message{
			Role: conv.RoleAssistant,
			ToolCalls: []conv.ToolCall{{
				Name: "Edit",
				Action: conv.NormalizedAction{
					Type:    conv.NormalizedActionMutate,
					Targets: []conv.ActionTarget{{Type: conv.ActionTargetFilePath, Value: filePath}},
				},
			}},
		},
	)
	return messages
}

func benchConversationTimeRange(conversations []conversation) TimeRange {
	if len(conversations) == 0 {
		return TimeRange{}
	}

	start := conversations[0].Sessions[0].Timestamp
	end := conversations[0].Sessions[0].Timestamp
	for _, conversation := range conversations[1:] {
		for _, session := range conversation.Sessions {
			if session.Timestamp.Before(start) {
				start = session.Timestamp
			}
			if session.Timestamp.After(end) {
				end = session.Timestamp
			}
		}
	}
	return TimeRange{Start: start, End: end}
}
