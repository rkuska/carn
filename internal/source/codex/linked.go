package codex

import (
	"context"
	"fmt"
	"runtime"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"

	conv "github.com/rkuska/carn/internal/conversation"
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
	sem := semaphore.NewWeighted(int64(runtime.NumCPU()))

	for i := range subagents {
		index := i
		meta := subagents[i]
		group.Go(func() error {
			return loadSingleLinkedTranscript(groupCtx, sem, meta, &results[index], &valid[index])
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

func loadSingleLinkedTranscript(
	ctx context.Context,
	sem *semaphore.Weighted,
	meta conv.SessionMeta,
	result *rolloutTranscript,
	ok *bool,
) error {
	if err := sem.Acquire(ctx, 1); err != nil {
		return err
	}
	defer sem.Release(1)

	transcript, err := loadRollout(ctx, meta)
	if err != nil {
		return fmt.Errorf("loadRollout_%s: %w", meta.ID, err)
	}
	if len(transcript.messages) > 0 {
		*result = transcript
		*ok = true
	}
	return nil
}
