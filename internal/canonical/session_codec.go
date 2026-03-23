package canonical

import (
	"bufio"
	"fmt"
)

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
	bw.writeUint(uint64(meta.UserMessageCount))
	bw.writeUint(uint64(meta.AssistantMessageCount))
	bw.writeString(meta.FilePath)
	bw.writeTokenUsage(meta.TotalUsage)
	bw.writeStringIntMap(meta.ToolCounts)
	bw.writeStringIntMap(meta.ToolErrorCounts)
	bw.writeStringIntMap(meta.ToolRejectCounts)
	bw.writeBool(meta.IsSubagent)
	if bw.err != nil {
		return fmt.Errorf("writeSessionMeta: %w", bw.err)
	}
	return nil
}
