package claude

import (
	"encoding/json"
	"testing"

	conv "github.com/rkuska/carn/internal/conversation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSummarizeToolCallFast(t *testing.T) {
	t.Parallel()

	longCommand := "go test ./... && go test -race ./... && golangci-lint run ./... && echo done"

	testCases := []struct {
		name  string
		tool  string
		input map[string]any
		want  string
	}{
		{
			name:  "read file path",
			tool:  "Read",
			input: map[string]any{"file_path": "/tmp/main.go"},
			want:  "/tmp/main.go",
		},
		{
			name:  "bash command is truncated",
			tool:  "Bash",
			input: map[string]any{"command": longCommand},
			want:  conv.Truncate(longCommand, 80),
		},
		{
			name:  "plan mode constant",
			tool:  "EnterPlanMode",
			input: map[string]any{},
			want:  "enter plan mode",
		},
		{
			name:  "mcp query",
			tool:  "mcp__context7__resolve-library-id",
			input: map[string]any{"query": "zod"},
			want:  "zod",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			raw, err := json.Marshal(testCase.input)
			require.NoError(t, err)
			assert.Equal(t, testCase.want, summarizeToolCallFast(testCase.tool, raw))
		})
	}
}
