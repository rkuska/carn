package conversation

import (
	"fmt"
	"testing"
)

func benchContextMessages(count int) []Message {
	messages := make([]Message, 0, count)
	prompt := 20_000
	for i := range count {
		switch i % 4 {
		case 0:
			messages = append(messages, Message{Role: RoleUser, Text: "prompt"})
		case 1:
			prompt += 1_500
			if prompt > 180_000 {
				prompt = 20_000
			}
			messages = append(messages, Message{Role: RoleAssistant, Text: "reply", Usage: ctxUsage(prompt)})
		default:
			messages = append(messages, Message{Role: RoleAssistant, Text: "continued"})
		}
	}
	return messages
}

func BenchmarkContextSeries(b *testing.B) {
	conversation := ctxClaudeConversation()
	for _, count := range []int{1_000, 10_000} {
		messages := benchContextMessages(count)
		b.Run(fmt.Sprintf("%d", count), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = NewContextSeries(conversation, messages)
			}
		})
	}
}
