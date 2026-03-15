package canonical

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

func encodeSessionBlob(session sessionFull) ([]byte, error) {
	estimatedSize := len(session.Messages) * 256
	buf := bytes.NewBuffer(make([]byte, 0, estimatedSize))
	writer := bufio.NewWriter(buf)
	if err := writeSessionFull(writer, session); err != nil {
		return nil, fmt.Errorf("writeSessionFull: %w", err)
	}
	if err := writer.Flush(); err != nil {
		return nil, fmt.Errorf("writer.Flush: %w", err)
	}
	return buf.Bytes(), nil
}

func decodeSessionBlob(blob []byte) (sessionFull, error) {
	return readSessionFull(bufio.NewReader(bytes.NewReader(blob)))
}

func marshalToolCounts(counts map[string]int) (string, error) {
	if len(counts) == 0 {
		return "", nil
	}
	raw, err := json.Marshal(counts)
	if err != nil {
		return "", fmt.Errorf("json.Marshal: %w", err)
	}
	return string(raw), nil
}

func unmarshalToolCounts(raw string) (map[string]int, error) {
	if raw == "" {
		return nil, nil
	}
	var counts map[string]int
	if err := json.Unmarshal([]byte(raw), &counts); err != nil {
		return nil, fmt.Errorf("json.Unmarshal: %w", err)
	}
	return counts, nil
}

func timeToUnixNano(ts time.Time) int64 {
	if ts.IsZero() {
		return 0
	}
	return ts.UnixNano()
}

func conversationLastTimestamp(conv conversation) time.Time {
	var last time.Time
	for _, session := range conv.Sessions {
		if session.LastTimestamp.After(last) {
			last = session.LastTimestamp
		}
		if session.Timestamp.After(last) {
			last = session.Timestamp
		}
	}
	return last
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
