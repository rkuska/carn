package codex

import (
	"context"
	"fmt"

	conv "github.com/rkuska/carn/internal/conversation"
)

func loadLinkedTranscripts(ctx context.Context, sessions []conv.SessionMeta) ([]rolloutTranscript, error) {
	linked := make([]rolloutTranscript, 0, len(sessions))
	for _, meta := range sessions {
		if !meta.IsSubagent {
			continue
		}

		transcript, err := loadRollout(ctx, meta)
		if err != nil {
			return nil, fmt.Errorf("loadRollout_%s: %w", meta.ID, err)
		}
		if len(transcript.messages) == 0 {
			continue
		}
		linked = append(linked, transcript)
	}
	return linked, nil
}
