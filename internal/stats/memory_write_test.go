package stats

import (
	"testing"

	"github.com/stretchr/testify/assert"

	conv "github.com/rkuska/carn/internal/conversation"
)

func TestIsMemoryWriteCall(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		call conv.ToolCall
		want bool
	}{
		{
			name: "write to memory subdir",
			call: conv.ToolCall{
				Action: conv.NormalizedAction{
					Type: conv.NormalizedActionRewrite,
					Targets: []conv.ActionTarget{
						{Type: conv.ActionTargetFilePath, Value: "/u/proj/memory/foo.md"},
					},
				},
			},
			want: true,
		},
		{
			name: "edit to memory index",
			call: conv.ToolCall{
				Action: conv.NormalizedAction{
					Type: conv.NormalizedActionMutate,
					Targets: []conv.ActionTarget{
						{Type: conv.ActionTargetFilePath, Value: "/u/proj/memory/MEMORY.md"},
					},
				},
			},
			want: true,
		},
		{
			name: "case-insensitive /Memory/Notes.md",
			call: conv.ToolCall{
				Action: conv.NormalizedAction{
					Type: conv.NormalizedActionRewrite,
					Targets: []conv.ActionTarget{
						{Type: conv.ActionTargetFilePath, Value: "/u/proj/Memory/Notes.md"},
					},
				},
			},
			want: true,
		},
		{
			name: "root MEMORY.md",
			call: conv.ToolCall{
				Action: conv.NormalizedAction{
					Type: conv.NormalizedActionRewrite,
					Targets: []conv.ActionTarget{
						{Type: conv.ActionTargetFilePath, Value: "/MEMORY.md"},
					},
				},
			},
			want: true,
		},
		{
			name: "CLAUDE.md is not memory",
			call: conv.ToolCall{
				Action: conv.NormalizedAction{
					Type: conv.NormalizedActionRewrite,
					Targets: []conv.ActionTarget{
						{Type: conv.ActionTargetFilePath, Value: "/u/proj/CLAUDE.md"},
					},
				},
			},
			want: false,
		},
		{
			name: "non-md file in memory dir does not match",
			call: conv.ToolCall{
				Action: conv.NormalizedAction{
					Type: conv.NormalizedActionRewrite,
					Targets: []conv.ActionTarget{
						{Type: conv.ActionTargetFilePath, Value: "/u/proj/memory/notes.txt"},
					},
				},
			},
			want: false,
		},
		{
			name: "nested path under memory dir does not match",
			call: conv.ToolCall{
				Action: conv.NormalizedAction{
					Type: conv.NormalizedActionRewrite,
					Targets: []conv.ActionTarget{
						{Type: conv.ActionTargetFilePath, Value: "/u/proj/memory/sub/foo.md"},
					},
				},
			},
			want: false,
		},
		{
			name: "read action against memory file is not a write",
			call: conv.ToolCall{
				Action: conv.NormalizedAction{
					Type: conv.NormalizedActionRead,
					Targets: []conv.ActionTarget{
						{Type: conv.ActionTargetFilePath, Value: "/u/proj/memory/foo.md"},
					},
				},
			},
			want: false,
		},
		{
			name: "empty targets",
			call: conv.ToolCall{
				Action: conv.NormalizedAction{Type: conv.NormalizedActionRewrite},
			},
			want: false,
		},
		{
			name: "non-file-path target is ignored",
			call: conv.ToolCall{
				Action: conv.NormalizedAction{
					Type: conv.NormalizedActionRewrite,
					Targets: []conv.ActionTarget{
						{Type: conv.ActionTargetCommand, Value: "echo /memory/foo.md"},
					},
				},
			},
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, IsMemoryWriteCall(tc.call))
		})
	}
}
