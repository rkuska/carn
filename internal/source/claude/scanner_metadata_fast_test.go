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

func TestVisitUserToolErrorsIgnoresRejectedToolUse(t *testing.T) {
	t.Parallel()

	raw := []byte(`[
		{
			"type":"tool_result",
			"tool_use_id":"toolu_1",
			"is_error":true,
			"content":"The user doesn't want to proceed with this tool use. The tool use was rejected."
		},
		{
			"type":"tool_result",
			"tool_use_id":"toolu_2",
			"is_error":true,
			"content":"file does not exist"
		}
	]`)

	var got []string
	ok := visitUserToolErrors(raw, func(toolUseID string) bool {
		got = append(got, toolUseID)
		return true
	})

	assert.True(t, ok)
	assert.Equal(t, []string{"toolu_2"}, got)
}

func TestInternClaudeToolNameNormalizesCaseVariants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Bash", internClaudeToolName([]byte("bash")))
	assert.Equal(t, "Read", internClaudeToolName([]byte("read")))
	assert.Equal(t, "ExitPlanMode", internClaudeToolName([]byte("exitplanmode")))
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

func TestParseUsageObjectWithNestedCacheCreationDetails(t *testing.T) {
	t.Parallel()

	raw := []byte(`{
		"input_tokens": 3,
		"cache_creation_input_tokens": 15402,
		"cache_read_input_tokens": 10681,
		"cache_creation": {
			"ephemeral_5m_input_tokens": 0,
			"ephemeral_1h_input_tokens": 15402
		},
		"output_tokens": 5,
		"service_tier": "standard"
	}`)

	assert.Equal(t, tokenUsage{
		InputTokens:              3,
		CacheCreationInputTokens: 15402,
		CacheReadInputTokens:     10681,
		OutputTokens:             5,
	}, parseUsageObject(raw))
}

func TestParseJSONUsageWithNestedCacheCreationDetails(t *testing.T) {
	t.Parallel()

	raw := []byte(`{
		"input_tokens": 3,
		"cache_creation_input_tokens": 15402,
		"cache_read_input_tokens": 10681,
		"cache_creation": {
			"ephemeral_5m_input_tokens": 0,
			"ephemeral_1h_input_tokens": 15402
		},
		"output_tokens": 5,
		"service_tier": "standard"
	}`)

	usage, err := parseJSONUsage(raw)
	assert.NoError(t, err)
	assert.Equal(t, &jsonUsage{
		InputTokens:              3,
		CacheCreationInputTokens: 15402,
		CacheReadInputTokens:     10681,
		OutputTokens:             5,
	}, usage)
}
