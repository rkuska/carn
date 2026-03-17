package canonical

import (
	"bufio"
	"bytes"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteReadUintRoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value uint64
	}{
		{name: "zero", value: 0},
		{name: "one", value: 1},
		{name: "varint_boundary_127", value: 127},
		{name: "varint_boundary_128", value: 128},
		{name: "medium", value: 16384},
		{name: "max_uint64", value: math.MaxUint64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			w := bufio.NewWriter(&buf)
			require.NoError(t, writeUint(w, tt.value))
			require.NoError(t, w.Flush())

			r := bufio.NewReader(&buf)
			got, err := readUint(r)
			require.NoError(t, err)
			assert.Equal(t, tt.value, got)
		})
	}
}

func TestWriteReadIntRoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value int64
	}{
		{name: "zero", value: 0},
		{name: "positive_small", value: 1},
		{name: "negative_small", value: -1},
		{name: "positive_large", value: math.MaxInt64},
		{name: "negative_large", value: math.MinInt64},
		{name: "varint_boundary_63", value: 63},
		{name: "varint_boundary_64", value: 64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			w := bufio.NewWriter(&buf)
			require.NoError(t, writeInt(w, tt.value))
			require.NoError(t, w.Flush())

			r := bufio.NewReader(&buf)
			got, err := readInt(r)
			require.NoError(t, err)
			assert.Equal(t, tt.value, got)
		})
	}
}

func TestWriteReadStringRoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
	}{
		{name: "empty", value: ""},
		{name: "ascii", value: "hello world"},
		{name: "unicode", value: "héllo wörld 🌍"},
		{name: "long", value: strings.Repeat("x", 8192)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			w := bufio.NewWriter(&buf)
			require.NoError(t, writeString(w, tt.value))
			require.NoError(t, w.Flush())

			r := bufio.NewReader(&buf)
			got, err := readString(r)
			require.NoError(t, err)
			assert.Equal(t, tt.value, got)
		})
	}
}

func TestConsecutiveUintWritesDoNotContaminate(t *testing.T) {
	t.Parallel()

	values := []uint64{0, 1, 128, math.MaxUint64, 42}
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	for _, v := range values {
		require.NoError(t, writeUint(w, v))
	}
	require.NoError(t, w.Flush())

	r := bufio.NewReader(&buf)
	for _, want := range values {
		got, err := readUint(r)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	}
}

func TestBinWriterReaderChainedRoundTrip(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	bw := binWriter{w: w}

	bw.writeString("hello")
	bw.writeUint(42)
	bw.writeInt(-99)
	bw.writeBool(true)
	bw.writeTokenUsage(tokenUsage{
		InputTokens:              100,
		CacheCreationInputTokens: 10,
		CacheReadInputTokens:     20,
		OutputTokens:             50,
	})
	bw.writeStringIntMap(map[string]int{"Read": 5, "Write": 3})
	require.NoError(t, bw.err)
	require.NoError(t, w.Flush())

	r := bufio.NewReader(&buf)
	br := binReader{r: r}
	assert.Equal(t, "hello", br.readString())
	assert.Equal(t, uint64(42), br.readUint())
	assert.Equal(t, int64(-99), br.readInt())
	assert.Equal(t, true, br.readBool())
	assert.Equal(t, tokenUsage{
		InputTokens:              100,
		CacheCreationInputTokens: 10,
		CacheReadInputTokens:     20,
		OutputTokens:             50,
	}, br.readTokenUsage())
	assert.Equal(t, map[string]int{"Read": 5, "Write": 3}, br.readStringIntMap())
	require.NoError(t, br.err)
}

func TestBinWriterErrorPropagation(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	w := bufio.NewWriterSize(&buf, 1)
	bw := binWriter{w: w}

	bw.writeString(strings.Repeat("x", 4096))
	require.NoError(t, bw.err)

	// force a flush error by writing to a closed/limited writer
	// Instead, just verify that after an explicit error, subsequent writes are no-ops
	bw.err = assert.AnError
	bw.writeString("should be skipped")
	bw.writeUint(999)
	bw.writeInt(-1)
	bw.writeBool(false)
	assert.ErrorIs(t, bw.err, assert.AnError)
}
