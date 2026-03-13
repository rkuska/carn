package canonical

import (
	"bufio"
	"fmt"
)

func writeMessage(w *bufio.Writer, msg message) error {
	bw := binWriter{w: w}
	bw.writeString(string(msg.Role))
	bw.writeString(string(msg.Visibility))
	bw.writeString(msg.Text)
	bw.writeString(msg.Thinking)
	bw.writeUint(uint64(len(msg.ToolCalls)))
	for _, call := range msg.ToolCalls {
		bw.writeString(call.Name)
		bw.writeString(call.Summary)
	}
	bw.writeUint(uint64(len(msg.ToolResults)))
	for _, result := range msg.ToolResults {
		if bw.err != nil {
			break
		}
		if err := writeToolResult(w, result); err != nil {
			bw.err = fmt.Errorf("writeToolResult: %w", err)
		}
	}
	bw.writeBool(msg.IsSidechain)
	bw.writeBool(msg.IsAgentDivider)
	bw.writeUint(uint64(len(msg.Plans)))
	for _, plan := range msg.Plans {
		if bw.err != nil {
			break
		}
		if err := writePlan(w, plan); err != nil {
			bw.err = fmt.Errorf("writePlan: %w", err)
		}
	}
	if bw.err != nil {
		return fmt.Errorf("writeMessage: %w", bw.err)
	}
	return nil
}

func readMessage(r *bufio.Reader) (message, error) {
	br := binReader{r: r}
	roleValue := br.readString()
	visibilityValue := br.readString()
	text := br.readString()
	thinking := br.readString()
	callCount := br.readUint()
	toolCalls := make([]toolCall, 0, callCount)
	for range callCount {
		name := br.readString()
		summary := br.readString()
		toolCalls = append(toolCalls, toolCall{Name: name, Summary: summary})
	}

	resultCount := br.readUint()
	if br.err != nil {
		return message{}, fmt.Errorf("readMessage: %w", br.err)
	}
	toolResults := make([]toolResult, 0, resultCount)
	for range resultCount {
		result, err := readToolResult(r)
		if err != nil {
			return message{}, fmt.Errorf("readMessage_toolResult: %w", err)
		}
		toolResults = append(toolResults, result)
	}

	isSidechain := br.readBool()
	isAgentDivider := br.readBool()
	planCount := br.readUint()
	if br.err != nil {
		return message{}, fmt.Errorf("readMessage: %w", br.err)
	}
	plans := make([]plan, 0, planCount)
	for range planCount {
		plan, err := readPlan(r)
		if err != nil {
			return message{}, fmt.Errorf("readMessage_plan: %w", err)
		}
		plans = append(plans, plan)
	}
	return message{
		Role:           role(roleValue),
		Visibility:     convMessageVisibility(visibilityValue),
		Text:           text,
		Thinking:       thinking,
		ToolCalls:      toolCalls,
		ToolResults:    toolResults,
		Plans:          plans,
		IsSidechain:    isSidechain,
		IsAgentDivider: isAgentDivider,
	}, nil
}

func convMessageVisibility(value string) messageVisibility {
	return messageVisibility(value)
}

func writeToolResult(w *bufio.Writer, result toolResult) error {
	if err := writeString(w, result.ToolName); err != nil {
		return fmt.Errorf("writeString_toolName: %w", err)
	}
	if err := writeString(w, result.ToolSummary); err != nil {
		return fmt.Errorf("writeString_toolSummary: %w", err)
	}
	if err := writeString(w, result.Content); err != nil {
		return fmt.Errorf("writeString_content: %w", err)
	}
	if err := writeBool(w, result.IsError); err != nil {
		return fmt.Errorf("writeBool_isError: %w", err)
	}
	if err := writeUint(w, uint64(len(result.StructuredPatch))); err != nil {
		return fmt.Errorf("writeUint_structuredPatch: %w", err)
	}
	for _, hunk := range result.StructuredPatch {
		if err := writeDiffHunk(w, hunk); err != nil {
			return fmt.Errorf("writeDiffHunk: %w", err)
		}
	}
	return nil
}

func readToolResult(r *bufio.Reader) (toolResult, error) {
	toolName, err := readString(r)
	if err != nil {
		return toolResult{}, fmt.Errorf("readString_toolName: %w", err)
	}
	toolSummary, err := readString(r)
	if err != nil {
		return toolResult{}, fmt.Errorf("readString_toolSummary: %w", err)
	}
	content, err := readString(r)
	if err != nil {
		return toolResult{}, fmt.Errorf("readString_content: %w", err)
	}
	isError, err := readBool(r)
	if err != nil {
		return toolResult{}, fmt.Errorf("readBool_isError: %w", err)
	}
	hunkCount, err := readUint(r)
	if err != nil {
		return toolResult{}, fmt.Errorf("readUint_structuredPatch: %w", err)
	}

	patch := make([]diffHunk, 0, hunkCount)
	for range hunkCount {
		hunk, err := readDiffHunk(r)
		if err != nil {
			return toolResult{}, fmt.Errorf("readDiffHunk: %w", err)
		}
		patch = append(patch, hunk)
	}
	return toolResult{
		ToolName:        toolName,
		ToolSummary:     toolSummary,
		Content:         content,
		IsError:         isError,
		StructuredPatch: patch,
	}, nil
}

func writeDiffHunk(w *bufio.Writer, hunk diffHunk) error {
	for _, value := range []int{hunk.OldStart, hunk.OldLines, hunk.NewStart, hunk.NewLines} {
		if err := writeInt(w, int64(value)); err != nil {
			return fmt.Errorf("writeInt: %w", err)
		}
	}
	if err := writeUint(w, uint64(len(hunk.Lines))); err != nil {
		return fmt.Errorf("writeUint_lines: %w", err)
	}
	for _, line := range hunk.Lines {
		if err := writeString(w, line); err != nil {
			return fmt.Errorf("writeString_line: %w", err)
		}
	}
	return nil
}

func readDiffHunk(r *bufio.Reader) (diffHunk, error) {
	oldStart, err := readInt(r)
	if err != nil {
		return diffHunk{}, fmt.Errorf("readInt_oldStart: %w", err)
	}
	oldLines, err := readInt(r)
	if err != nil {
		return diffHunk{}, fmt.Errorf("readInt_oldLines: %w", err)
	}
	newStart, err := readInt(r)
	if err != nil {
		return diffHunk{}, fmt.Errorf("readInt_newStart: %w", err)
	}
	newLines, err := readInt(r)
	if err != nil {
		return diffHunk{}, fmt.Errorf("readInt_newLines: %w", err)
	}
	lineCount, err := readUint(r)
	if err != nil {
		return diffHunk{}, fmt.Errorf("readUint_lines: %w", err)
	}

	lines := make([]string, 0, lineCount)
	for range lineCount {
		line, err := readString(r)
		if err != nil {
			return diffHunk{}, fmt.Errorf("readString_line: %w", err)
		}
		lines = append(lines, line)
	}
	return diffHunk{
		OldStart: int(oldStart),
		OldLines: int(oldLines),
		NewStart: int(newStart),
		NewLines: int(newLines),
		Lines:    lines,
	}, nil
}

func writeTokenUsage(w *bufio.Writer, usage tokenUsage) error {
	for _, value := range []int{
		usage.InputTokens,
		usage.CacheCreationInputTokens,
		usage.CacheReadInputTokens,
		usage.OutputTokens,
	} {
		if err := writeUint(w, uint64(value)); err != nil {
			return fmt.Errorf("writeUint: %w", err)
		}
	}
	return nil
}

func readTokenUsage(r *bufio.Reader) (tokenUsage, error) {
	values := make([]uint64, 4)
	for i := range values {
		value, err := readUint(r)
		if err != nil {
			return tokenUsage{}, fmt.Errorf("readUint: %w", err)
		}
		values[i] = value
	}
	return tokenUsage{
		InputTokens:              int(values[0]),
		CacheCreationInputTokens: int(values[1]),
		CacheReadInputTokens:     int(values[2]),
		OutputTokens:             int(values[3]),
	}, nil
}
