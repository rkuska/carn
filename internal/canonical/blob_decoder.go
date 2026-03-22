package canonical

import (
	"encoding/binary"
	"errors"
	"fmt"
	"time"
	"unsafe"
)

var (
	errBlobDecoderInvalidVarint = errors.New("invalid blob varint")
	errBlobDecoderTruncated     = errors.New("truncated blob")
	errBlobDecoderCountOverflow = errors.New("blob count overflow")
)

type blobDecoder struct {
	data []byte
	pos  int
	err  error
}

func decodeSessionBlobFast(blob []byte) (sessionFull, error) {
	decoder := blobDecoder{data: blob}
	session := decoder.readSessionFull()
	if decoder.err != nil {
		return sessionFull{}, fmt.Errorf("decodeSessionBlobFast: %w", decoder.err)
	}
	return session, nil
}

func (d *blobDecoder) readSessionFull() sessionFull {
	meta := d.readSessionMeta()
	messageCount := d.readUint()
	messages := makeDecodedSlice[message](d, messageCount)
	for i := range messages {
		messages[i] = d.readMessage()
	}
	return sessionFull{Meta: meta, Messages: messages}
}

func (d *blobDecoder) readSessionMeta() sessionMeta {
	id := d.readString()
	projectName := d.readString()
	slug := d.readString()
	timestampValue := d.readInt()
	lastTimestampValue := d.readInt()
	cwd := d.readString()
	gitBranch := d.readString()
	version := d.readString()
	model := d.readString()
	firstMessage := d.readString()
	messageCount := d.readIntCount(d.readUint())
	mainMessageCount := d.readIntCount(d.readUint())
	userMessageCount := d.readIntCount(d.readUint())
	assistantMessageCount := d.readIntCount(d.readUint())
	filePath := d.readString()
	usage := d.readTokenUsage()
	toolCounts := d.readStringIntMap()
	toolErrorCounts := d.readStringIntMap()
	isSubagent := d.readBool()

	meta := sessionMeta{
		ID:                    id,
		Project:               project{DisplayName: projectName},
		Slug:                  slug,
		CWD:                   cwd,
		GitBranch:             gitBranch,
		Version:               version,
		Model:                 model,
		FirstMessage:          firstMessage,
		MessageCount:          messageCount,
		MainMessageCount:      mainMessageCount,
		UserMessageCount:      userMessageCount,
		AssistantMessageCount: assistantMessageCount,
		FilePath:              filePath,
		TotalUsage:            usage,
		ToolCounts:            toolCounts,
		ToolErrorCounts:       toolErrorCounts,
		IsSubagent:            isSubagent,
	}

	if timestampValue != 0 {
		meta.Timestamp = unixTime(timestampValue)
	}
	if lastTimestampValue != 0 {
		meta.LastTimestamp = unixTime(lastTimestampValue)
	}
	return meta
}

func (d *blobDecoder) readMessage() message {
	roleValue := d.readString()
	visibilityValue := d.readString()
	msg := message{
		Role:              role(roleValue),
		Visibility:        convMessageVisibility(visibilityValue),
		Text:              d.readString(),
		Thinking:          d.readString(),
		HasHiddenThinking: d.readBool(),
	}

	callCount := d.readUint()
	msg.ToolCalls = makeDecodedSlice[toolCall](d, callCount)
	for i := range msg.ToolCalls {
		msg.ToolCalls[i] = toolCall{
			Name:    d.readString(),
			Summary: d.readString(),
		}
	}

	resultCount := d.readUint()
	msg.ToolResults = makeDecodedSlice[toolResult](d, resultCount)
	for i := range msg.ToolResults {
		msg.ToolResults[i] = d.readToolResult()
	}

	msg.IsSidechain = d.readBool()
	msg.IsAgentDivider = d.readBool()
	msg.Usage = d.readTokenUsage()

	planCount := d.readUint()
	msg.Plans = makeDecodedSlice[plan](d, planCount)
	for i := range msg.Plans {
		msg.Plans[i] = d.readPlan()
	}

	return msg
}

func (d *blobDecoder) readToolResult() toolResult {
	result := toolResult{
		ToolName:    d.readString(),
		ToolSummary: d.readString(),
		Content:     d.readString(),
		IsError:     d.readBool(),
	}

	hunkCount := d.readUint()
	result.StructuredPatch = makeDecodedSlice[diffHunk](d, hunkCount)
	for i := range result.StructuredPatch {
		result.StructuredPatch[i] = d.readDiffHunk()
	}
	return result
}

func (d *blobDecoder) readDiffHunk() diffHunk {
	hunk := diffHunk{
		OldStart: int(d.readInt()),
		OldLines: int(d.readInt()),
		NewStart: int(d.readInt()),
		NewLines: int(d.readInt()),
	}

	lineCount := d.readUint()
	hunk.Lines = makeDecodedSlice[string](d, lineCount)
	for i := range hunk.Lines {
		hunk.Lines[i] = d.readString()
	}
	return hunk
}

func (d *blobDecoder) readPlan() plan {
	filePath := d.readString()
	content := d.readString()
	timestamp := d.readInt()
	var ts time.Time
	if timestamp != 0 {
		ts = unixTime(timestamp)
	}
	return plan{
		FilePath:  filePath,
		Content:   content,
		Timestamp: ts,
	}
}

func (d *blobDecoder) readTokenUsage() tokenUsage {
	return tokenUsage{
		InputTokens:              d.readIntCount(d.readUint()),
		CacheCreationInputTokens: d.readIntCount(d.readUint()),
		CacheReadInputTokens:     d.readIntCount(d.readUint()),
		OutputTokens:             d.readIntCount(d.readUint()),
	}
}

func (d *blobDecoder) readStringIntMap() map[string]int {
	count := d.readUint()
	if count == 0 {
		return nil
	}

	values := make(map[string]int, d.readIntCount(count))
	for range count {
		values[d.readString()] = d.readIntCount(d.readUint())
	}
	return values
}

func (d *blobDecoder) readUint() uint64 {
	if d.err != nil {
		return 0
	}
	value, size := binary.Uvarint(d.data[d.pos:])
	if size <= 0 {
		d.err = errBlobDecoderInvalidVarint
		return 0
	}
	d.pos += size
	return value
}

func (d *blobDecoder) readInt() int64 {
	if d.err != nil {
		return 0
	}
	value, size := binary.Varint(d.data[d.pos:])
	if size <= 0 {
		d.err = errBlobDecoderInvalidVarint
		return 0
	}
	d.pos += size
	return value
}

func (d *blobDecoder) readBool() bool {
	if d.err != nil {
		return false
	}
	if d.pos >= len(d.data) {
		d.err = errBlobDecoderTruncated
		return false
	}
	value := d.data[d.pos]
	d.pos++
	return value == 1
}

func (d *blobDecoder) readString() string {
	length := d.readUint()
	if d.err != nil || length == 0 {
		return ""
	}

	size := d.readIntCount(length)
	if d.err != nil {
		return ""
	}
	if size > len(d.data)-d.pos {
		d.err = errBlobDecoderTruncated
		return ""
	}

	value := bytesToString(d.data[d.pos : d.pos+size])
	d.pos += size
	return value
}

func (d *blobDecoder) readIntCount(value uint64) int {
	if d.err != nil {
		return 0
	}
	maxInt := int(^uint(0) >> 1)
	if value > uint64(maxInt) {
		d.err = errBlobDecoderCountOverflow
		return 0
	}
	return int(value)
}

func makeDecodedSlice[T any](d *blobDecoder, count uint64) []T {
	size := d.readIntCount(count)
	if d.err != nil {
		return nil
	}
	if size == 0 {
		return make([]T, 0)
	}
	return make([]T, size)
}

func bytesToString(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	return unsafe.String(unsafe.SliceData(raw), len(raw))
}
