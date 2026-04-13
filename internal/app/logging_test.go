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
	assert.Contains(t, logged, "2026-04-13T10:11:12+02:00")
	assert.Contains(t, logged, "INF")
	assert.Contains(t, logged, "carn started")
}
