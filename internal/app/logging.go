package app

import (
	"io"
	"time"

	"github.com/rs/zerolog"
)

func newAppLogger(out io.Writer, level zerolog.Level) zerolog.Logger {
	return zerolog.New(newConsoleLogWriter(out)).
		With().Timestamp().Logger().Level(level)
}

func newConsoleLogWriter(out io.Writer) zerolog.ConsoleWriter {
	return zerolog.ConsoleWriter{
		Out:        out,
		NoColor:    true,
		TimeFormat: time.RFC3339,
	}
}
