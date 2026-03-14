package canonical

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"sort"
	"time"
)

func writeStringIntMap(w *bufio.Writer, values map[string]int) error {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if err := writeUint(w, uint64(len(keys))); err != nil {
		return fmt.Errorf("writeUint: %w", err)
	}
	for _, key := range keys {
		if err := writeString(w, key); err != nil {
			return fmt.Errorf("writeString_key: %w", err)
		}
		if err := writeUint(w, uint64(values[key])); err != nil {
			return fmt.Errorf("writeUint_value: %w", err)
		}
	}
	return nil
}

func readStringIntMap(r *bufio.Reader) (map[string]int, error) {
	count, err := readUint(r)
	if err != nil {
		return nil, fmt.Errorf("readUint: %w", err)
	}
	if count == 0 {
		return nil, nil
	}

	values := make(map[string]int, count)
	for range count {
		key, err := readString(r)
		if err != nil {
			return nil, fmt.Errorf("readString_key: %w", err)
		}
		value, err := readUint(r)
		if err != nil {
			return nil, fmt.Errorf("readUint_value: %w", err)
		}
		values[key] = int(value)
	}
	return values, nil
}

func writeString(w *bufio.Writer, value string) error {
	if err := writeUint(w, uint64(len(value))); err != nil {
		return fmt.Errorf("writeUint: %w", err)
	}
	if _, err := w.WriteString(value); err != nil {
		return fmt.Errorf("WriteString: %w", err)
	}
	return nil
}

func readString(r *bufio.Reader) (string, error) {
	length, err := readUint(r)
	if err != nil {
		return "", fmt.Errorf("readUint: %w", err)
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", fmt.Errorf("io.ReadFull: %w", err)
	}
	return string(buf), nil
}

func writeUint(w *bufio.Writer, value uint64) error {
	var buf [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(buf[:], value)
	if _, err := w.Write(buf[:n]); err != nil {
		return fmt.Errorf("w.Write: %w", err)
	}
	return nil
}

func readUint(r *bufio.Reader) (uint64, error) {
	value, err := binary.ReadUvarint(r)
	if err != nil {
		return 0, fmt.Errorf("binary.ReadUvarint: %w", err)
	}
	return value, nil
}

func writeInt(w *bufio.Writer, value int64) error {
	var buf [binary.MaxVarintLen64]byte
	n := binary.PutVarint(buf[:], value)
	if _, err := w.Write(buf[:n]); err != nil {
		return fmt.Errorf("w.Write: %w", err)
	}
	return nil
}

func readInt(r *bufio.Reader) (int64, error) {
	value, err := binary.ReadVarint(r)
	if err != nil {
		return 0, fmt.Errorf("binary.ReadVarint: %w", err)
	}
	return value, nil
}

func writeBool(w *bufio.Writer, value bool) error {
	byteValue := byte(0)
	if value {
		byteValue = 1
	}
	if err := w.WriteByte(byteValue); err != nil {
		return fmt.Errorf("WriteByte: %w", err)
	}
	return nil
}

func readBool(r *bufio.Reader) (bool, error) {
	value, err := r.ReadByte()
	if err != nil {
		return false, fmt.Errorf("ReadByte: %w", err)
	}
	return value == 1, nil
}

type binReader struct {
	r   *bufio.Reader
	err error
}

func (br *binReader) readString() string {
	if br.err != nil {
		return ""
	}
	value, err := readString(br.r)
	if err != nil {
		br.err = err
	}
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

func unixTime(value int64) time.Time {
	return time.Unix(0, value).UTC()
}
