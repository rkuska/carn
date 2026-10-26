package conversation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveContextWindow(t *testing.T) {
	t.Parallel()

	recorded := func(tokens ...int) []SessionMeta {
		sessions := make([]SessionMeta, len(tokens))
		for i, window := range tokens {
			sessions[i].Performance.ModelContextWindow = window
		}
		return sessions
	}

	tests := []struct {
		name     string
		provider Provider
		sessions []SessionMeta
		peak     int
		want     ContextWindow
	}{
		{
			name:     "recorded window wins",
			provider: ProviderCodex,
			sessions: recorded(258_400),
			peak:     120_000,
			want:     ContextWindow{Tokens: 258_400, Known: true},
		},
		{
			name:     "largest recorded window across sessions",
			provider: ProviderCodex,
			sessions: recorded(128_000, 258_400, 0),
			peak:     10_000,
			want:     ContextWindow{Tokens: 258_400, Known: true},
		},
		{
			name:     "recorded window beats the claude tiers",
			provider: ProviderClaude,
			sessions: recorded(500_000),
			peak:     300_000,
			want:     ContextWindow{Tokens: 500_000, Known: true},
		},
		{
			name:     "claude peak inside the 200k tier",
			provider: ProviderClaude,
			sessions: recorded(0),
			peak:     47_483,
			want:     ContextWindow{Tokens: 200_000, Known: true},
		},
		{
			name:     "claude peak exactly 200k stays in the first tier",
			provider: ProviderClaude,
			sessions: recorded(0),
			peak:     200_000,
			want:     ContextWindow{Tokens: 200_000, Known: true},
		},
		{
			name:     "claude peak above 200k moves to the 1M tier",
			provider: ProviderClaude,
			sessions: recorded(0),
			peak:     556_675,
			want:     ContextWindow{Tokens: 1_000_000, Known: true},
		},
		{
			name:     "claude peak above 1M is unknown",
			provider: ProviderClaude,
			sessions: recorded(0),
			peak:     1_200_000,
			want:     ContextWindow{Tokens: 1_200_000},
		},
		{
			name:     "claude without steps has no window",
			provider: ProviderClaude,
			sessions: recorded(0),
			peak:     0,
			want:     ContextWindow{},
		},
		{
			name:     "codex without a recorded window is unknown",
			provider: ProviderCodex,
			sessions: recorded(0),
			peak:     90_000,
			want:     ContextWindow{Tokens: 90_000},
		},
		{
			name:     "unknown provider is unknown",
			provider: "",
			sessions: recorded(0),
			peak:     90_000,
			want:     ContextWindow{Tokens: 90_000},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			conversation := Conversation{Ref: Ref{Provider: tt.provider, ID: "w"}, Sessions: tt.sessions}

			assert.Equal(t, tt.want, resolveContextWindow(conversation, tt.peak))
		})
	}
}

func TestResolveContextWindowFallsBackToSessionProvider(t *testing.T) {
	t.Parallel()

	conversation := Conversation{Sessions: []SessionMeta{{Provider: ProviderClaude}}}

	assert.Equal(t, ContextWindow{Tokens: 200_000, Known: true}, resolveContextWindow(conversation, 50_000))
}
