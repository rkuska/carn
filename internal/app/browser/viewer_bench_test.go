package browser

import (
	"fmt"
	"testing"
	"time"

	conv "github.com/rkuska/carn/internal/conversation"
)

func benchViewerSession(messages int, withNeedle bool) conv.Session {
	msgs := make([]conv.Message, 0, messages*2)
	for i := range messages {
		msgs = append(msgs, conv.Message{Role: conv.RoleUser, Text: fmt.Sprintf("user line %d", i)})
		text := fmt.Sprintf("assistant line %d", i)
		if withNeedle && i%25 == 0 {
			text += " IMPORTANT_NEEDLE"
		}
		msgs = append(msgs, conv.Message{Role: conv.RoleAssistant, Text: text})
	}

	return conv.Session{
		Meta: conv.SessionMeta{
			ID:        "bench-viewer",
			Timestamp: time.Now(),
			Project:   conv.Project{DisplayName: "bench/project"},
		},
		Messages: msgs,
	}
}

func BenchmarkViewerRenderContent(b *testing.B) {
	session := benchViewerSession(200, false)
	model := newViewerModel(session, singleSessionConversation(session.Meta), "dark", "2006-01-02 15:04", 140, 45)

	for b.Loop() {
		model = model.renderContent()
	}
}

func BenchmarkViewerSearch(b *testing.B) {
	session := benchViewerSession(200, true)
	model := newViewerModel(session, singleSessionConversation(session.Meta), "dark", "2006-01-02 15:04", 140, 45)
	model.searchQuery = "IMPORTANT_NEEDLE"

	for b.Loop() {
		model = model.performSearch()
	}
}

// benchContextSession is a long conversation whose requests carry usage, so the
// Turn Ledger and the Context Dock have work to do.
func benchContextSession(requests int) conv.Session {
	msgs := make([]conv.Message, 0, requests*3)
	prompt := 20_000
	for i := range requests {
		prompt += 900 + (i%7)*400
		if prompt > 180_000 {
			prompt = 25_000
		}
		call := conv.ToolCall{Name: "Read", Summary: fmt.Sprintf("internal/file%d.go", i)}
		msgs = append(msgs,
			conv.Message{Role: conv.RoleUser, Text: fmt.Sprintf("user line %d", i)},
			conv.Message{
				Role: conv.RoleAssistant, Text: fmt.Sprintf("assistant line %d", i),
				ToolCalls: []conv.ToolCall{call}, Usage: contextUsage(prompt),
			},
			conv.Message{Role: conv.RoleUser, ToolResults: []conv.ToolResult{{ToolName: "Read", Content: "ok"}}},
		)
	}
	return conv.Session{
		Meta: conv.SessionMeta{
			ID:          "bench-context",
			Provider:    conv.ProviderClaude,
			Timestamp:   time.Now(),
			Project:     conv.Project{DisplayName: "bench/project"},
			Performance: conv.SessionPerformanceMeta{ModelContextWindow: 200_000},
		},
		Messages: msgs,
	}
}

// BenchmarkViewerRenderContentLedger renders cold (render cache dropped) with the
// Turn Ledger on and off; markdown and role header caches stay warm.
func BenchmarkViewerRenderContentLedger(b *testing.B) {
	session := benchContextSession(200)
	conversation := contextFixtureConversation(session)
	for _, ledger := range []bool{false, true} {
		b.Run(fmt.Sprintf("ledger=%t", ledger), func(b *testing.B) {
			model := newViewerModel(session, conversation, "dark", "2006-01-02 15:04", 140, 45)
			model.opts = allTranscriptOptions()
			model.opts.showContextLedger = ledger
			model = model.renderContent()

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				model.renderCache = nil
				model = model.renderContent()
			}
		})
	}
}

// BenchmarkViewerDockView renders the Context Dock for a scrolled viewer.
func BenchmarkViewerDockView(b *testing.B) {
	session := benchContextSession(200)
	model := newViewerModel(session, contextFixtureConversation(session), "dark", "2006-01-02 15:04", 140, 45).
		SetFullscreen(true)
	model.dockOpen = true
	model = model.applyViewportHeight()
	model.viewport.SetYOffset(model.viewport.TotalLineCount() / 2)

	b.ReportAllocs()

	for b.Loop() {
		_ = model.dockView()
	}
}
