package canonical

import (
	"bufio"
	"time"

	conv "github.com/rkuska/carn/internal/conversation"
)

type binReader struct {
	r      *bufio.Reader
	err    error
	strBuf []byte
}

func (br *binReader) readString() string {
	if br.err != nil {
		return ""
	}
	var value string
	value, br.strBuf, br.err = readStringBuf(br.r, br.strBuf)
	return value
}

func (br *binReader) readInt() int64 {
	if br.err != nil {
		return 0
	}
	value, err := readInt(br.r)
	if err != nil {
		br.err = err
	}
	return value
}

func (br *binReader) readUint() uint64 {
	if br.err != nil {
		return 0
	}
	value, err := readUint(br.r)
	if err != nil {
		br.err = err
	}
	return value
}

func (br *binReader) readBool() bool {
	if br.err != nil {
		return false
	}
	value, err := readBool(br.r)
	if err != nil {
		br.err = err
	}
	return value
}

func (br *binReader) readTokenUsage() tokenUsage {
	if br.err != nil {
		return tokenUsage{}
	}
	value, err := readTokenUsage(br.r)
	if err != nil {
		br.err = err
	}
	return value
}

func (br *binReader) readStringIntMap() map[string]int {
	if br.err != nil {
		return nil
	}
	value, err := readStringIntMap(br.r)
	if err != nil {
		br.err = err
	}
	return value
}

type binWriter struct {
	w   *bufio.Writer
	err error
}

func (bw *binWriter) writeString(value string) {
	if bw.err != nil {
		return
	}
	bw.err = writeString(bw.w, value)
}

func (bw *binWriter) writeInt(value int64) {
	if bw.err != nil {
		return
	}
	bw.err = writeInt(bw.w, value)
}

func (bw *binWriter) writeUint(value uint64) {
	if bw.err != nil {
		return
	}
	bw.err = writeUint(bw.w, value)
}

func (bw *binWriter) writeBool(value bool) {
	if bw.err != nil {
		return
	}
	bw.err = writeBool(bw.w, value)
}

func (bw *binWriter) writeTokenUsage(value tokenUsage) {
	if bw.err != nil {
		return
	}
	bw.err = writeTokenUsage(bw.w, value)
}

func (bw *binWriter) writeStringIntMap(value map[string]int) {
	if bw.err != nil {
		return
	}
	bw.err = writeStringIntMap(bw.w, value)
}

func (bw *binWriter) writeNormalizedAction(value conv.NormalizedAction) {
	if bw.err != nil {
		return
	}
	bw.err = writeNormalizedAction(bw.w, value)
}

func (bw *binWriter) writeMessagePerformanceMeta(value conv.MessagePerformanceMeta) {
	if bw.err != nil {
		return
	}
	bw.err = writeMessagePerformanceMeta(bw.w, value)
}

func (bw *binWriter) writeSessionPerformanceMeta(value conv.SessionPerformanceMeta) {
	if bw.err != nil {
		return
	}
	bw.err = writeSessionPerformanceMeta(bw.w, value)
}

func unixTime(value int64) time.Time {
	return time.Unix(0, value).In(canonicalTimeLocation())
}
