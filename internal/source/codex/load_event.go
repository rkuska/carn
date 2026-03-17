package codex

func classifyLoadedEventMessage(payload []byte, itemType string) (visibleMessage, bool) {
	switch itemType {
	case eventTypeUserMessage:
		return classifyLoadedEventUserMessage(payload), true
	case eventTypeAgentMessage:
		return classifyLoadedEventAssistantMessage(payload), true
	case eventTypeTaskComplete:
		return classifyLoadedTaskCompleteMessage(payload), true
	default:
		return visibleMessage{}, false
	}
}

func classifyLoadedEventUserMessage(payload []byte) visibleMessage {
	message, ok := extractTopLevelRawJSONStringFieldByMarker(payload, messageFieldMarker)
	if !ok {
		return visibleMessage{}
	}
	classified, _ := classifyEventUserMessage(message)
	return classified
}

func classifyLoadedEventAssistantMessage(payload []byte) visibleMessage {
	message, ok := extractTopLevelRawJSONStringFieldByMarker(payload, messageFieldMarker)
	if !ok {
		return visibleMessage{}
	}
	classified, _ := classifyEventAssistantMessage(message)
	return classified
}

func classifyLoadedTaskCompleteMessage(payload []byte) visibleMessage {
	message, ok := extractTopLevelRawJSONStringFieldByMarker(payload, lastAgentMessageFieldMarker)
	if !ok {
		return visibleMessage{}
	}
	classified, _ := classifyTaskCompleteMessage(message)
	return classified
}
