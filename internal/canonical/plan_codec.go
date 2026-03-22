package canonical

import (
	"bufio"
	"fmt"
)

func writePlan(w *bufio.Writer, p plan) error {
	bw := binWriter{w: w}
	bw.writeString(p.FilePath)
	bw.writeString(p.Content)

	timestamp := int64(0)
	if !p.Timestamp.IsZero() {
		timestamp = p.Timestamp.UnixNano()
	}
	bw.writeInt(timestamp)
	if bw.err != nil {
		return fmt.Errorf("writePlan: %w", bw.err)
	}
	return nil
}
