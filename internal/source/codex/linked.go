package codex

import (
	"context"
	"fmt"

	conv "github.com/rkuska/carn/internal/conversation"
	"golang.org/x/sync/errgroup"
)

func loadLinkedTranscripts(ctx context.Context, sessions []conv.SessionMeta) ([]rolloutTranscript, error) {
	subagents := make([]conv.SessionMeta, 0, len(sessions))
	for _, meta := range sessions {
		if meta.IsSubagent {
			subagents = append(subagents, meta)
		}
	}
	if len(subagents) == 0 {
		return nil, nil
	}

	results := make([]rolloutTranscript, len(subagents))
	valid := make([]bool, len(subagents))
	group, groupCtx := errgroup.WithContext(ctx)

	for i := range subagents {
		index := i
		meta := subagents[i]
		group.Go(func() error {
			transcript, err := loadRollout(groupCtx, meta)
			if err != nil {
				return fmt.Errorf("loadRollout_%s: %w", meta.ID, err)
			}
			if len(transcript.messages) > 0 {
				results[index] = transcript
				valid[index] = true
			}
			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return nil, err
	}

	linked := make([]rolloutTranscript, 0, len(subagents))
	for i, ok := range valid {
		if ok {
			linked = append(linked, results[i])
		}
	}
	return linked, nil
}
