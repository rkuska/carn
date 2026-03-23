package claude

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	src "github.com/rkuska/carn/internal/source"
)

func TestDetectLineDrift(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		raw  map[string]any
		want []src.DriftFinding
	}{
		{
			name: "known user record",
			raw: map[string]any{
				"type":      "user",
				"sessionId": "session-1",
				"slug":      "demo",
				"timestamp": "2026-03-21T10:00:00Z",
				"message": map[string]any{
					"role":    "user",
					"content": "hello",
				},
			},
			want: []src.DriftFinding{},
		},
		{
			name: "unknown envelope field",
			raw: map[string]any{
				"type":      "user",
				"sessionId": "session-1",
				"slug":      "demo",
				"timestamp": "2026-03-21T10:00:00Z",
				"transport": "v2",
				"message": map[string]any{
					"role":    "user",
					"content": "hello",
				},
			},
			want: []src.DriftFinding{
				{Category: "envelope_field", Value: "transport"},
			},
		},
		{
			name: "unknown message field",
			raw: map[string]any{
				"type":      "assistant",
				"sessionId": "session-1",
				"slug":      "demo",
				"timestamp": "2026-03-21T10:00:01Z",
				"message": map[string]any{
					"role":      "assistant",
					"model":     "claude-sonnet-4",
					"content":   []map[string]any{{"type": "text", "text": "hello"}},
					"transport": "v2",
				},
			},
			want: []src.DriftFinding{
				{Category: "message_field", Value: "transport"},
			},
		},
		{
			name: "unknown usage field",
			raw: map[string]any{
				"type":      "assistant",
				"sessionId": "session-1",
				"slug":      "demo",
				"timestamp": "2026-03-21T10:00:01Z",
				"message": map[string]any{
					"role":    "assistant",
					"model":   "claude-sonnet-4",
					"content": []map[string]any{{"type": "text", "text": "hello"}},
					"usage": map[string]any{
						"input_tokens":                1,
						"output_tokens":               2,
						"cache_write_input_tokens":    3,
						"cache_read_input_tokens":     4,
						"cache_creation_input_tokens": 5,
					},
				},
			},
			want: []src.DriftFinding{
				{Category: "usage_field", Value: "cache_write_input_tokens"},
			},
		},
		{
			name: "unknown record type",
			raw: map[string]any{
				"type":      "transport-summary",
				"sessionId": "session-1",
				"slug":      "demo",
				"timestamp": "2026-03-21T10:00:00Z",
				"message": map[string]any{
					"role":    "assistant",
					"content": "hello",
				},
			},
			want: []src.DriftFinding{
				{Category: "record_type", Value: "transport-summary"},
			},
		},
		{
			name: "unknown content block type",
			raw: map[string]any{
				"type":      "assistant",
				"sessionId": "session-1",
				"slug":      "demo",
				"timestamp": "2026-03-21T10:00:01Z",
				"message": map[string]any{
					"role":  "assistant",
					"model": "claude-sonnet-4",
					"content": []map[string]any{
						{"type": "attachment", "text": "hello"},
					},
				},
			},
			want: []src.DriftFinding{
				{Category: "content_block_type", Value: "attachment"},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			raw, err := json.Marshal(testCase.raw)
			require.NoError(t, err)

			report := src.NewDriftReport()
			detectLineDrift(raw, &report)

			assert.Equal(t, testCase.want, report.Findings())
		})
	}
}

func TestScanMetadataResultKeepsFixtureCorpusDriftFree(t *testing.T) {
	t.Parallel()

	baseDir := copyScannerFixtureCorpus(t)
	path := filepath.Join(baseDir, "project-a", "session-with-tools.jsonl")

	result, err := scanMetadataResult(t.Context(), path, project{DisplayName: "demo"})
	require.NoError(t, err)
	assert.True(t, result.drift.Empty())
}

func TestKnownClaudeSchemaCoversParseRecordJSONTags(t *testing.T) {
	t.Parallel()

	assert.ElementsMatch(t, []string{
		"type",
		"sessionId",
		"slug",
		"cwd",
		"gitBranch",
		"version",
		"timestamp",
		"isSidechain",
		"isMeta",
		"toolUseResult",
		"message",
	}, setKeys(knownEnvelopeFields))

	assert.Subset(t, setKeys(knownMessageFields), []string{
		"role",
		"content",
		"model",
		"usage",
	})

	assert.ElementsMatch(t, jsonTagsOfType(reflect.TypeFor[jsonUsage]()), setKeys(knownUsageFields))
	assert.ElementsMatch(t, []string{"user", "assistant"}, setKeys(knownRecordTypes))
	assert.ElementsMatch(
		t,
		[]string{"text", "tool_use", "tool_result", "thinking"},
		setKeys(knownContentBlockTypes),
	)
}

func jsonTagsOfType(t reflect.Type) []string {
	fields := make([]string, 0, t.NumField())
	for i := range t.NumField() {
		tag := t.Field(i).Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		fields = append(fields, tag)
	}
	return fields
}

func setKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}
