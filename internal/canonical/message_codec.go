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
	bw.writeBool(msg.HasHiddenThinking)
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
	bw.writeTokenUsage(msg.Usage)
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
	var values [4]uint64
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
