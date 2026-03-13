package app

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"

	conv "github.com/rkuska/carn/internal/conversation"
)

func exportBaseName(name string) string {
	sanitized := sanitizeFileStem(name)
	if sanitized == "" {
		return "conversation"
	}
	return sanitized
}

func conversationExportFileName(conversation conv.Conversation, meta conv.SessionMeta) string {
	name := exportConversationName(conversation, meta)
	if provider := sanitizeFileStem(string(conversation.Ref.Provider)); provider != "" {
		return fmt.Sprintf("conversation-%s-%s.md", provider, name)
	}
	return fmt.Sprintf("conversation-%s.md", name)
}

func rawExportFileName(conversation conv.Conversation, session conv.Session) string {
	name := exportConversationName(conversation, session.Meta)
	if provider := sanitizeFileStem(string(conversation.Ref.Provider)); provider != "" {
		return fmt.Sprintf("conversation-%s-%s.raw.jsonl", provider, name)
	}
	return fmt.Sprintf("conversation-%s.raw.jsonl", name)
}

func exportConversationName(conversation conv.Conversation, meta conv.SessionMeta) string {
	switch {
	case conversation.Name != "":
		return exportBaseName(conversation.Name)
	case meta.Slug != "":
		return exportBaseName(meta.Slug)
	case meta.FirstMessage != "":
		return exportBaseName(conversation.DisplayName())
	case meta.ID != "":
		return exportBaseName(shortID(meta.ID))
	default:
		return "conversation"
	}
}

func sanitizeFileStem(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	var b strings.Builder
	for _, r := range value {
		if normalized, ok := normalizeFileStemRune(r); ok {
			b.WriteRune(normalized)
			continue
		}
		appendStemHyphen(&b)
	}

	sanitized := strings.Trim(b.String(), "-")
	if sanitized == "" {
		return ""
	}
	return filepath.Base(sanitized)
}

func normalizeFileStemRune(r rune) (rune, bool) {
	if unicode.IsLetter(r) || unicode.IsDigit(r) {
		return unicode.ToLower(r), true
	}
	if r == '.' || r == '-' || r == '_' || r == '/' || unicode.IsSpace(r) {
		return '-', false
	}
	return 0, false
}

func appendStemHyphen(b *strings.Builder) {
	if b.Len() == 0 || strings.HasSuffix(b.String(), "-") {
		return
	}
	b.WriteByte('-')
}
