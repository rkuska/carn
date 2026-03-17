package codex

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
)

func visitRolloutRecords(
	ctx context.Context,
	path string,
	visit func(recordType string, payload []byte, timestamp string),
) error {
	file, br, err := openReader(path)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	defer readerPool.Put(br)

	var overflow []byte
	for {
		line, nextOverflow, done, err := readNextRolloutLine(ctx, br, overflow)
		overflow = nextOverflow
		if err != nil {
			return err
		}
		if len(line) == 0 {
			if done {
				return nil
			}
			continue
		}

		recordType, ok := extractEnvelopeJSONStringFieldByMarker(line, typeFieldMarker)
		if ok {
			timestamp, _ := extractEnvelopeJSONStringFieldByMarker(line, timestampFieldMarker)
			if payload, ok := extractPayload(line); ok {
				visit(recordType, payload, timestamp)
			}
		}
		if done {
			return nil
		}
	}
}

func readNextRolloutLine(
	ctx context.Context,
	br *bufio.Reader,
	overflow []byte,
) ([]byte, []byte, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, overflow, false, fmt.Errorf("readNextRolloutLine_ctx: %w", err)
	}

	line, nextOverflow, err := readScanLine(br, overflow)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, nextOverflow, false, fmt.Errorf("readNextRolloutLine_readScanLine: %w", err)
	}
	return line, nextOverflow, errors.Is(err, io.EOF), nil
}
