package app

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
	model := newViewerModel(session, singleSessionConversation(session.Meta), "dark", 140, 45)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		model.renderContent()
	}
}

func BenchmarkViewerSearch(b *testing.B) {
	session := benchViewerSession(200, true)
	model := newViewerModel(session, singleSessionConversation(session.Meta), "dark", 140, 45)
	model.searchQuery = "IMPORTANT_NEEDLE"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		model.performSearch()
	}
}
