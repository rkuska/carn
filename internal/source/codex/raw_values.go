package codex

import "bytes"

var (
	recordTypeSessionMetaRaw  = []byte(`"session_meta"`)
	recordTypeTurnContextRaw  = []byte(`"turn_context"`)
	recordTypeResponseItemRaw = []byte(`"response_item"`)
	recordTypeEventMsgRaw     = []byte(`"event_msg"`)

	responseTypeMessageRaw              = []byte(`"message"`)
	responseTypeReasoningRaw            = []byte(`"reasoning"`)
	responseTypeFunctionCallRaw         = []byte(`"function_call"`)
	responseTypeCustomToolCallRaw       = []byte(`"custom_tool_call"`)
	responseTypeWebSearchCallRaw        = []byte(`"web_search_call"`)
	responseTypeFunctionCallOutputRaw   = []byte(`"function_call_output"`)
	responseTypeCustomToolCallOutputRaw = []byte(`"custom_tool_call_output"`)

	responseRoleUserRaw      = []byte(`"user"`)
	responseRoleAssistantRaw = []byte(`"assistant"`)
	responseRoleDeveloperRaw = []byte(`"developer"`)

	eventTypeTokenCountRaw       = []byte(`"token_count"`)
	eventTypeUserMessageRaw      = []byte(`"user_message"`)
	eventTypeAgentMessageRaw     = []byte(`"agent_message"`)
	eventTypeAgentReasoningRaw   = []byte(`"agent_reasoning"`)
	eventTypeItemCompletedRaw    = []byte(`"item_completed"`)
	eventTypeTaskStartedRaw      = []byte(`"task_started"`)
	eventTypeTaskCompleteRaw     = []byte(`"task_complete"`)
	eventTypeTurnAbortedRaw      = []byte(`"turn_aborted"`)
	eventTypeContextCompactedRaw = []byte(`"context_compacted"`)

	contentTypeInputTextRaw  = []byte(`"input_text"`)
	contentTypeOutputTextRaw = []byte(`"output_text"`)

	reasoningSummaryTextRaw = []byte(`"summary_text"`)
	completedItemPlanRaw    = []byte(`"Plan"`)
)

func isKnownRecordTypeRaw(raw []byte) bool {
	return bytes.Equal(raw, recordTypeSessionMetaRaw) ||
		bytes.Equal(raw, recordTypeTurnContextRaw) ||
		bytes.Equal(raw, recordTypeResponseItemRaw) ||
		bytes.Equal(raw, recordTypeEventMsgRaw) ||
		codexKnownSchemaExtras.HasRaw("record_type", raw)
}

func isKnownResponseItemTypeRaw(raw []byte) bool {
	return bytes.Equal(raw, responseTypeMessageRaw) ||
		bytes.Equal(raw, responseTypeReasoningRaw) ||
		bytes.Equal(raw, responseTypeFunctionCallRaw) ||
		bytes.Equal(raw, responseTypeCustomToolCallRaw) ||
		bytes.Equal(raw, responseTypeWebSearchCallRaw) ||
		bytes.Equal(raw, responseTypeFunctionCallOutputRaw) ||
		bytes.Equal(raw, responseTypeCustomToolCallOutputRaw) ||
		codexKnownSchemaExtras.HasRaw("response_item_type", raw)
}

func isKnownRoleRaw(raw []byte) bool {
	return bytes.Equal(raw, responseRoleUserRaw) ||
		bytes.Equal(raw, responseRoleAssistantRaw) ||
		bytes.Equal(raw, responseRoleDeveloperRaw)
}

func isKnownEventTypeRaw(raw []byte) bool {
	return bytes.Equal(raw, eventTypeTokenCountRaw) ||
		bytes.Equal(raw, eventTypeUserMessageRaw) ||
		bytes.Equal(raw, eventTypeAgentMessageRaw) ||
		bytes.Equal(raw, eventTypeAgentReasoningRaw) ||
		bytes.Equal(raw, eventTypeItemCompletedRaw) ||
		bytes.Equal(raw, eventTypeTaskStartedRaw) ||
		bytes.Equal(raw, eventTypeTaskCompleteRaw) ||
		bytes.Equal(raw, eventTypeTurnAbortedRaw) ||
		bytes.Equal(raw, eventTypeContextCompactedRaw) ||
		codexKnownSchemaExtras.HasRaw("event_type", raw)
}

func isKnownContentBlockTypeRaw(raw []byte) bool {
	return bytes.Equal(raw, contentTypeInputTextRaw) ||
		bytes.Equal(raw, contentTypeOutputTextRaw) ||
		codexKnownSchemaExtras.HasRaw("content_block_type", raw)
}

func isKnownReasoningSummaryBlockTypeRaw(raw []byte) bool {
	return bytes.Equal(raw, reasoningSummaryTextRaw)
}

func isKnownCompletedItemTypeRaw(raw []byte) bool {
	return bytes.Equal(raw, completedItemPlanRaw)
}
