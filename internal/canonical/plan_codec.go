package canonical

import (
	"bufio"
	"fmt"
	"time"
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

func readPlan(r *bufio.Reader) (plan, error) {
	br := binReader{r: r}
	filePath := br.readString()
	content := br.readString()
	timestamp := br.readInt()
	if br.err != nil {
		return plan{}, fmt.Errorf("readPlan: %w", br.err)
	}

	var ts time.Time
	if timestamp != 0 {
		ts = unixTime(timestamp)
	}
	return plan{
		FilePath:  filePath,
		Content:   content,
		Timestamp: ts,
	}, nil
}
