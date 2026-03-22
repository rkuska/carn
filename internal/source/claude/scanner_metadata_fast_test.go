package claude

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVisitAssistantToolUses(t *testing.T) {
	t.Parallel()

	raw := []byte(`[
		{"type":"text","text":"reply"},
		{"type":"tool_use","id":"toolu_1","name":"Read","input":{"file_path":"/tmp/a.go"}},
		{"type":"tool_use","id":"toolu_2","name":"Bash","input":{"command":"go test ./..."}}
	]`)

	type toolUse struct {
		name string
		id   string
	}

	var got []toolUse
	ok := visitAssistantToolUses(raw, func(name, id string) bool {
		got = append(got, toolUse{name: name, id: id})
		return true
	})

	assert.True(t, ok)
	assert.Equal(t, []toolUse{
		{name: "Read", id: "toolu_1"},
		{name: "Bash", id: "toolu_2"},
	}, got)
}

func TestVisitUserToolErrors(t *testing.T) {
	t.Parallel()

	raw := []byte(`[
		{"type":"text","text":"follow up"},
		{"type":"tool_result","tool_use_id":"toolu_1","is_error":false,"content":"ok"},
		{"type":"tool_result","tool_use_id":"toolu_2","is_error":true,"content":"failed"}
	]`)

	var got []string
	ok := visitUserToolErrors(raw, func(toolUseID string) bool {
		got = append(got, toolUseID)
		return true
	})

	assert.True(t, ok)
	assert.Equal(t, []string{"toolu_2"}, got)
}

func TestParseUsageObject(t *testing.T) {
	t.Parallel()

	raw := []byte(`{
		"input_tokens": 120,
		"cache_creation_input_tokens": 7,
		"cache_read_input_tokens": 13,
		"output_tokens": 42
	}`)

	assert.Equal(t, tokenUsage{
		InputTokens:              120,
		CacheCreationInputTokens: 7,
		CacheReadInputTokens:     13,
		OutputTokens:             42,
	}, parseUsageObject(raw))
}
