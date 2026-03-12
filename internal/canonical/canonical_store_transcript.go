package canonical

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
)

func writeTranscriptFile(path string, session sessionFull) error {
	return writeBinaryFile(path, func(w *bufio.Writer) error {
		if _, err := w.WriteString(transcriptMagic); err != nil {
			return fmt.Errorf("WriteString: %w", err)
		}
		if err := writeSessionFull(w, session); err != nil {
			return fmt.Errorf("writeSessionFull: %w", err)
		}
		return nil
	})
}

func readTranscriptFile(path string) (sessionFull, error) {
	return readBinaryFile(path, func(r *bufio.Reader) (sessionFull, error) {
		magic, err := readFixedString(r, len(transcriptMagic))
		if err != nil {
			return sessionFull{}, fmt.Errorf("readFixedString: %w", err)
		}
		if magic != transcriptMagic {
			return sessionFull{}, fmt.Errorf("readTranscriptFile: %w", errInvalidMagic("transcript"))
		}
		session, err := readSessionFull(r)
		if err != nil {
			return sessionFull{}, fmt.Errorf("readSessionFull: %w", err)
		}
		return session, nil
	})
}

func writeBinaryFile(path string, writeFn func(*bufio.Writer) error) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("os.MkdirAll: %w", err)
	}
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("os.Create: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			zerolog.Ctx(context.Background()).Warn().Err(closeErr).Msgf("failed to close %s", path)
		}
	}()

	writer := bufio.NewWriter(file)
	if err := writeFn(writer); err != nil {
		return err
	}
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("writer.Flush: %w", err)
	}
	return nil
}

func readBinaryFile[T any](path string, readFn func(*bufio.Reader) (T, error)) (T, error) {
	file, err := os.Open(path)
	if err != nil {
		var zero T
		return zero, fmt.Errorf("os.Open: %w", err)
	}
	defer func() { _ = file.Close() }()
	return readFn(bufio.NewReader(file))
}

func writeSessionFull(w *bufio.Writer, session sessionFull) error {
	if err := writeSessionMeta(w, session.Meta); err != nil {
		return fmt.Errorf("writeSessionMeta: %w", err)
	}
	if err := writeUint(w, uint64(len(session.Messages))); err != nil {
		return fmt.Errorf("writeUint: %w", err)
	}
	for _, msg := range session.Messages {
		if err := writeMessage(w, msg); err != nil {
			return fmt.Errorf("writeMessage: %w", err)
		}
	}
	return nil
}

func readSessionFull(r *bufio.Reader) (sessionFull, error) {
	meta, err := readSessionMeta(r)
	if err != nil {
		return sessionFull{}, fmt.Errorf("readSessionMeta: %w", err)
	}
	messageCount, err := readUint(r)
	if err != nil {
		return sessionFull{}, fmt.Errorf("readUint: %w", err)
	}

	messages := make([]message, 0, messageCount)
	for range messageCount {
		msg, err := readMessage(r)
		if err != nil {
			return sessionFull{}, fmt.Errorf("readMessage: %w", err)
		}
		messages = append(messages, msg)
	}
	return sessionFull{Meta: meta, Messages: messages}, nil
}

func writeSessionMeta(w *bufio.Writer, meta sessionMeta) error {
	bw := binWriter{w: w}
	bw.writeString(meta.ID)
	bw.writeString(meta.Project.DisplayName)
	bw.writeString(meta.Slug)
	bw.writeInt(meta.Timestamp.UnixNano())
	bw.writeInt(meta.LastTimestamp.UnixNano())
	bw.writeString(meta.CWD)
	bw.writeString(meta.GitBranch)
	bw.writeString(meta.Version)
	bw.writeString(meta.Model)
	bw.writeString(meta.FirstMessage)
	bw.writeUint(uint64(meta.MessageCount))
	bw.writeUint(uint64(meta.MainMessageCount))
	bw.writeString(meta.FilePath)
	bw.writeTokenUsage(meta.TotalUsage)
	bw.writeStringIntMap(meta.ToolCounts)
	bw.writeBool(meta.IsSubagent)
	if bw.err != nil {
		return fmt.Errorf("writeSessionMeta: %w", bw.err)
	}
	return nil
}

func readSessionMeta(r *bufio.Reader) (sessionMeta, error) {
	br := binReader{r: r}
	id := br.readString()
	projectName := br.readString()
	slug := br.readString()
	timestampValue := br.readInt()
	lastTimestampValue := br.readInt()
	cwd := br.readString()
	gitBranch := br.readString()
	version := br.readString()
	model := br.readString()
	firstMessage := br.readString()
	messageCount := br.readUint()
	mainMessageCount := br.readUint()
	filePath := br.readString()
	usage := br.readTokenUsage()
	toolCounts := br.readStringIntMap()
	isSubagent := br.readBool()
	if br.err != nil {
		return sessionMeta{}, fmt.Errorf("readSessionMeta: %w", br.err)
	}

	meta := sessionMeta{
		ID:               id,
		Project:          project{DisplayName: projectName},
		Slug:             slug,
		CWD:              cwd,
		GitBranch:        gitBranch,
		Version:          version,
		Model:            model,
		FirstMessage:     firstMessage,
		MessageCount:     int(messageCount),
		MainMessageCount: int(mainMessageCount),
		FilePath:         filePath,
		TotalUsage:       usage,
		ToolCounts:       toolCounts,
		IsSubagent:       isSubagent,
	}
	if timestampValue != 0 {
		meta.Timestamp = unixTime(timestampValue)
	}
	if lastTimestampValue != 0 {
		meta.LastTimestamp = unixTime(lastTimestampValue)
	}
	return meta, nil
}
