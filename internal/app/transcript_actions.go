package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"

	conv "github.com/rkuska/carn/internal/conversation"
)

func renderVisibleConversation(
	session conv.Session,
	opts transcriptOptions,
	planExpanded bool,
) string {
	var sb strings.Builder
	if planExpanded {
		for i, plan := range conv.AllPlans(session.Messages) {
			if i > 0 {
				sb.WriteString("\n\n---\n\n")
			}
			sb.WriteString(conv.FormatPlan(plan))
		}
		if sb.Len() > 0 {
			sb.WriteString("\n\n---\n\n")
		}
	}
	sb.WriteString(flattenSegments(renderTranscriptSegmented(session, opts)))
	text := strings.TrimSpace(sb.String())
	if text == "" {
		return ""
	}
	return text + "\n"
}

func readRawConversationText(conversation conv.Conversation, session conv.Session) (string, error) {
	paths := rawConversationPaths(conversation, session)
	if len(paths) == 0 {
		return "", nil
	}

	var sb strings.Builder
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("readRawConversationText_os.ReadFile: %w", err)
		}
		sb.Write(content)
		if len(content) > 0 && content[len(content)-1] != '\n' {
			sb.WriteByte('\n')
		}
	}
	return sb.String(), nil
}

func rawConversationPaths(conversation conv.Conversation, session conv.Session) []string {
	if paths := conversation.FilePaths(); len(paths) > 0 {
		return paths
	}
	if session.Meta.FilePath == "" {
		return nil
	}
	return []string{session.Meta.FilePath}
}

func copyRawConversationCmd(conversation conv.Conversation, session conv.Session) tea.Cmd {
	return func() tea.Msg {
		text, err := readRawConversationText(conversation, session)
		if err != nil {
			return errorNotification(fmt.Sprintf("copy raw failed: %v", err))
		}
		return copyTextCmd(text, "raw transcript copied to clipboard")()
	}
}

func openRawConversationCmd(conversation conv.Conversation, session conv.Session) tea.Cmd {
	return func() tea.Msg {
		text, err := readRawConversationText(conversation, session)
		if err != nil {
			return errorNotification(fmt.Sprintf("open raw failed: %v", err))
		}
		return openTextInEditorCmd(text, rawExportFileName(conversation, session))()
	}
}

func openTextInEditorCmd(text string, fileName string) tea.Cmd {
	return func() tea.Msg {
		path, err := writeTempEditorFile(text, fileName)
		if err != nil {
			return errorNotification(fmt.Sprintf("open in editor failed: %v", err))
		}
		return runEditorCmd(newEditorCmd(path))()
	}
}

func writeTempEditorFile(text string, fileName string) (string, error) {
	ext := filepath.Ext(fileName)
	if ext == "" {
		ext = ".txt"
	}
	pattern := "carn-*"
	if ext != "" {
		pattern += ext
	}
	f, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", fmt.Errorf("writeTempEditorFile_os.CreateTemp: %w", err)
	}

	name := f.Name()
	if _, writeErr := f.WriteString(text); writeErr != nil {
		_ = f.Close() //nolint:errcheck // best-effort cleanup; write error takes precedence
		return "", fmt.Errorf("writeTempEditorFile_WriteString: %w", writeErr)
	}
	if err := f.Close(); err != nil {
		return "", fmt.Errorf("writeTempEditorFile_f.Close: %w", err)
	}
	return name, nil
}

func planFileName(plan conv.Plan) string {
	if name := filepath.Base(plan.FilePath); name != "" {
		return name
	}
	return "plan.md"
}
