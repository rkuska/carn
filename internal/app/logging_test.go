package app

import (
	"bytes"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestConsoleLogWriterUsesRFC3339TimestampWithOffset(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(newConsoleLogWriter(&buf)).Level(zerolog.InfoLevel)
	timestamp := time.Date(2026, time.April, 13, 10, 11, 12, 0, time.FixedZone("CEST", 2*60*60))

	logger.Info().
		Time(zerolog.TimestampFieldName, timestamp).
		Msg("carn started")

	logged := buf.String()
	// ConsoleWriter re-formats in local time, so assert the local
	// representation rather than a fixed offset.
	localTS := timestamp.Local().Format(time.RFC3339)
	assert.Contains(t, logged, localTS)
	assert.Contains(t, logged, "INF")
	assert.Contains(t, logged, "carn started")
}
