package canonical

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
)

func storeTranscriptFileKey(conversationID string) string {
	hash := sha1.New()
	_, _ = io.WriteString(hash, conversationID)
	return hex.EncodeToString(hash.Sum(nil))
}
