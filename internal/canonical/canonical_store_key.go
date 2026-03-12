package canonical

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
	"path/filepath"
	"strings"
)

func buildConversationStoreKey(
	rawDir string,
	provider conversationProvider,
	conv conversation,
) string {
	hash := sha1.New()
	_, _ = io.WriteString(hash, string(provider))
	_, _ = io.WriteString(hash, "\x00")

	if conv.IsSubagent() || conv.Name == "" {
		for _, path := range conv.FilePaths() {
			rel, err := filepath.Rel(rawDir, path)
			if err != nil {
				rel = path
			}
			_, _ = io.WriteString(hash, filepath.ToSlash(rel))
			_, _ = io.WriteString(hash, "\x00")
		}
		return hex.EncodeToString(hash.Sum(nil))
	}

	projectDir := conversationProjectDir(rawDir, conv)
	_, _ = io.WriteString(hash, projectDir)
	_, _ = io.WriteString(hash, "\x00")
	_, _ = io.WriteString(hash, conv.Name)
	return hex.EncodeToString(hash.Sum(nil))
}

func conversationProjectDir(rawDir string, conv conversation) string {
	if len(conv.Sessions) == 0 {
		return conv.Project.DisplayName
	}
	rel, err := filepath.Rel(rawDir, conv.Sessions[0].FilePath)
	if err != nil {
		return conv.Project.DisplayName
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) == 0 {
		return conv.Project.DisplayName
	}
	return parts[0]
}
