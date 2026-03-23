package conversation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeriveToolOutcomeCounts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		messages []Message
		want     ToolOutcomeCounts
	}{
		{
			name: "counts calls errors and rejections",
			messages: []Message{
				{
					ToolCalls: []ToolCall{
						{Name: "Read"},
						{Name: "Read"},
						{Name: "Bash"},
					},
				},
				{
					ToolResults: []ToolResult{
						{
							ToolName: "Read",
							IsError:  true,
							Content:  "file missing",
						},
						{
							ToolName: "Bash",
							IsError:  true,
							Content:  "User rejected tool use",
						},
					},
				},
			},
			want: ToolOutcomeCounts{
				Calls:      map[string]int{"Read": 2, "Bash": 1},
				Errors:     map[string]int{"Read": 1},
				Rejections: map[string]int{"Bash": 1},
			},
		},
		{
			name: "ignores empty tool names and nil slices",
			messages: []Message{
				{
					ToolCalls: []ToolCall{
						{Name: ""},
					},
					ToolResults: []ToolResult{
						{
							ToolName: "",
							IsError:  true,
							Content:  "failed",
						},
					},
				},
			},
			want: ToolOutcomeCounts{},
		},
		{
			name: "does not count non errors as rejections",
			messages: []Message{
				{
					ToolResults: []ToolResult{
						{
							ToolName: "Edit",
							IsError:  false,
							Content:  "The tool use was rejected by the user.",
						},
					},
				},
			},
			want: ToolOutcomeCounts{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, DeriveToolOutcomeCounts(tt.messages))
		})
	}
}
