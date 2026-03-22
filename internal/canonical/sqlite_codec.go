package canonical

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"
)

type blobEncoderState struct {
	buf    bytes.Buffer
	writer *bufio.Writer
}

var blobEncoderPool = sync.Pool{
	New: func() any {
		s := &blobEncoderState{}
		s.buf.Grow(32768)
		s.writer = bufio.NewWriter(&s.buf)
		return s
	},
}

var blobReaderPool = sync.Pool{
	New: func() any { return bufio.NewReader(nil) },
}

func withEncodedSessionBlob(session sessionFull, use func([]byte) error) error {
	state, ok := blobEncoderPool.Get().(*blobEncoderState)
	if !ok {
		state = &blobEncoderState{}
		state.buf.Grow(32768)
		state.writer = bufio.NewWriter(&state.buf)
	}
	state.buf.Reset()
	state.writer.Reset(&state.buf)
	defer blobEncoderPool.Put(state)

	if err := writeSessionFull(state.writer, session); err != nil {
		return fmt.Errorf("writeSessionFull: %w", err)
	}
	if err := state.writer.Flush(); err != nil {
		return fmt.Errorf("writer.Flush: %w", err)
	}
	if err := use(state.buf.Bytes()); err != nil {
		return fmt.Errorf("use: %w", err)
	}
	return nil
}

func decodeSessionBlob(blob []byte) (sessionFull, error) {
	br, ok := blobReaderPool.Get().(*bufio.Reader)
	if !ok {
		br = bufio.NewReader(nil)
	}
	br.Reset(bytes.NewReader(blob))
	defer blobReaderPool.Put(br)
	return readSessionFull(br)
}

func marshalToolCountsCached(counts map[string]int) string {
	if len(counts) == 0 {
		return ""
	}

	keys := make([]string, 0, len(counts))
	size := 2
	for key, value := range counts {
		keys = append(keys, key)
		size += len(key) + 4 + digits(value)
	}
	sort.Strings(keys)

	raw := make([]byte, 0, size)
	raw = append(raw, '{')
	for i, key := range keys {
		if i > 0 {
			raw = append(raw, ',')
		}
		raw = strconv.AppendQuote(raw, key)
		raw = append(raw, ':')
		raw = strconv.AppendInt(raw, int64(counts[key]), 10)
	}
	raw = append(raw, '}')
	return string(raw)
}

func unmarshalToolCounts(raw string) (map[string]int, error) {
	if isEmptyJSONObject(raw) {
		return nil, nil
	}
	var counts map[string]int
	if err := json.Unmarshal([]byte(raw), &counts); err != nil {
		return nil, fmt.Errorf("json.Unmarshal: %w", err)
	}
	if len(counts) == 0 {
		return nil, nil
	}
	return counts, nil
}

func isEmptyJSONObject(raw string) bool {
	if raw == "" {
		return true
	}
	trimmed := bytes.TrimSpace([]byte(raw))
	return bytes.Equal(trimmed, []byte("{}"))
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

func digits(value int) int {
	if value == 0 {
		return 1
	}
	if value < 0 {
		value = -value
	}
	count := 0
	for value > 0 {
		count++
		value /= 10
	}
	return count
}
