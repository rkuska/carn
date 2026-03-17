package codex

import "bufio"

func trimScanLine(line []byte) []byte {
	for len(line) > 0 {
		switch line[len(line)-1] {
		case '\n', '\r':
			line = line[:len(line)-1]
		default:
			return line
		}
	}
	return line
}

func readScanLine(br *bufio.Reader, overflow []byte) ([]byte, []byte, error) {
	line, err := br.ReadSlice('\n')
	if err == bufio.ErrBufferFull {
		overflow = append(overflow[:0], line...)
		for err == bufio.ErrBufferFull {
			var more []byte
			more, err = br.ReadSlice('\n')
			overflow = append(overflow, more...)
		}
		line = overflow
	}
	return trimScanLine(line), overflow, err
}
