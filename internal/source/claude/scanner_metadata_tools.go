package claude

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/buger/jsonparser"

	conv "github.com/rkuska/carn/internal/conversation"
)

var (
	assistantToolUseMarker    = []byte(`"tool_use"`)
	userToolResultMarker      = []byte(`"tool_result"`)
	userToolResultErrorMarker = []byte(`"is_error"`)
	toolUseTypeMarker         = []byte(`"type":"tool_use"`)
	toolResultTypeMarker      = []byte(`"type":"tool_result"`)
	isErrorTrueMarker         = []byte(`"is_error":true`)
	nameFieldMarker           = []byte(`"name":"`)
	idFieldMarker             = []byte(`"id":"`)
	toolUseIDFieldMarker      = []byte(`"tool_use_id":"`)
	contentFieldMarker        = []byte(`"content":"`)
	claudeToolNamesByLower    = map[string]string{
		"read":            "Read",
		"write":           "Write",
		"edit":            "Edit",
		"bash":            "Bash",
		"glob":            "Glob",
		"grep":            "Grep",
		"webfetch":        "WebFetch",
		"websearch":       "WebSearch",
		"toolsearch":      "ToolSearch",
		"agent":           "Agent",
		"askuserquestion": "AskUserQuestion",
		"skill":           "Skill",
		"task":            "Task",
		"taskcreate":      "TaskCreate",
		"taskupdate":      "TaskUpdate",
		"taskget":         "TaskGet",
		"tasklist":        "TaskList",
		"taskoutput":      "TaskOutput",
		"notebookedit":    "NotebookEdit",
		"enterworktree":   "EnterWorktree",
		"enterplanmode":   "EnterPlanMode",
		"exitplanmode":    "ExitPlanMode",
	}
)

func visitAssistantToolUses(raw json.RawMessage, yield func(name, id string) bool) bool {
	return visitJSONArrayObjects(raw, func(value []byte) bool {
		name, id, ok := assistantToolUseFields(value)
		if !ok {
			return true
		}
		return yield(name, id)
	})
}

func visitUserToolOutcomes(raw json.RawMessage, yield func(toolUseID string, rejected bool) bool) bool {
	if !bytes.Contains(raw, userToolResultMarker) || !bytes.Contains(raw, userToolResultErrorMarker) {
		return true
	}

	return visitJSONArrayObjects(raw, func(value []byte) bool {
		toolUseID, rejected, ok := userToolOutcome(value)
		if !ok {
			return true
		}
		return yield(toolUseID, rejected)
	})
}

func visitUserToolErrors(raw json.RawMessage, yield func(toolUseID string) bool) bool {
	return visitUserToolOutcomes(raw, func(toolUseID string, rejected bool) bool {
		if rejected {
			return true
		}
		return yield(toolUseID)
	})
}

func visitJSONArrayObjects(raw json.RawMessage, visit func([]byte) bool) bool {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] != '[' {
		return false
	}

	parseOK := true
	_, err := jsonparser.ArrayEach(raw, func(value []byte, dataType jsonparser.ValueType, _ int, err error) {
		if !parseOK {
			return
		}
		if err != nil || dataType != jsonparser.Object {
			parseOK = false
			return
		}
		if !visit(value) {
			parseOK = false
		}
	})
	return err == nil && parseOK
}

func assistantToolUseFields(value []byte) (string, string, bool) {
	if !bytes.Contains(value, toolUseTypeMarker) {
		return "", "", false
	}

	nameRaw, ok := extractFastJSONStringFieldBytes(value, nameFieldMarker)
	if !ok || len(nameRaw) == 0 {
		return "", "", false
	}
	idRaw, _ := extractFastJSONStringFieldBytes(value, idFieldMarker)
	return internClaudeToolName(nameRaw), string(idRaw), true
}

func userToolOutcome(value []byte) (string, bool, bool) {
	if !bytes.Contains(value, toolResultTypeMarker) ||
		!bytes.Contains(value, isErrorTrueMarker) {
		return "", false, false
	}

	toolUseIDRaw, ok := extractFastJSONStringFieldBytes(value, toolUseIDFieldMarker)
	if !ok || len(toolUseIDRaw) == 0 {
		return "", false, false
	}
	contentRaw, _ := extractFastJSONStringFieldBytes(value, contentFieldMarker)
	return string(toolUseIDRaw), conv.IsRejectedToolResultContent(bytesToStringView(contentRaw)), true
}

func extractFastJSONStringFieldBytes(raw []byte, marker []byte) ([]byte, bool) {
	idx := bytes.Index(raw, marker)
	if idx == -1 {
		return nil, false
	}
	start := idx + len(marker)
	end := bytes.IndexByte(raw[start:], '"')
	if end == -1 {
		return nil, false
	}
	return raw[start : start+end], true
}

func internClaudeToolName(raw []byte) string {
	name := string(raw)
	if canonical, ok := claudeToolNamesByLower[strings.ToLower(name)]; ok {
		return canonical
	}
	return name
}
