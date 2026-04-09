package canonical

import (
	"bufio"
	"fmt"

	conv "github.com/rkuska/carn/internal/conversation"
)

func writeNormalizedAction(w *bufio.Writer, action conv.NormalizedAction) error {
	if err := writeString(w, string(action.Type)); err != nil {
		return fmt.Errorf("writeString_type: %w", err)
	}
	if err := writeUint(w, uint64(len(action.Targets))); err != nil {
		return fmt.Errorf("writeUint_targets: %w", err)
	}
	for _, target := range action.Targets {
		if err := writeActionTarget(w, target); err != nil {
			return fmt.Errorf("writeActionTarget: %w", err)
		}
	}
	return nil
}

func writeActionTarget(w *bufio.Writer, target conv.ActionTarget) error {
	if err := writeString(w, string(target.Type)); err != nil {
		return fmt.Errorf("writeString_type: %w", err)
	}
	if err := writeString(w, target.Value); err != nil {
		return fmt.Errorf("writeString_value: %w", err)
	}
	return nil
}

func writeMessagePerformanceMeta(w *bufio.Writer, meta conv.MessagePerformanceMeta) error {
	for _, value := range []int{
		meta.ReasoningBlockCount,
		meta.ReasoningRedactionCount,
	} {
		if err := writeUint(w, uint64(value)); err != nil {
			return fmt.Errorf("writeUint: %w", err)
		}
	}
	for _, value := range []string{
		meta.StopReason,
		meta.Phase,
		meta.Effort,
	} {
		if err := writeString(w, value); err != nil {
			return fmt.Errorf("writeString: %w", err)
		}
	}
	return nil
}

func writeSessionPerformanceMeta(w *bufio.Writer, meta conv.SessionPerformanceMeta) error {
	for _, value := range []int{
		meta.ReasoningBlockCount,
		meta.ReasoningRedactionCount,
		meta.MaxThinkingTokens,
		meta.ModelContextWindow,
		meta.DurationMS,
		meta.RetryAttemptCount,
		meta.RetryDelayMS,
		meta.MaxRetries,
		meta.AbortCount,
		meta.CompactionCount,
		meta.MicroCompactionCount,
		meta.TaskStartedCount,
		meta.TaskCompleteCount,
		meta.ContextCompactedCount,
		meta.RateLimitSnapshotCount,
	} {
		if err := writeUint(w, uint64(value)); err != nil {
			return fmt.Errorf("writeUint: %w", err)
		}
	}
	for _, counts := range []map[string]int{
		meta.APIErrorCounts,
		meta.StopReasonCounts,
		meta.PhaseCounts,
		meta.EffortCounts,
		meta.ServerToolUseCounts,
		meta.ServiceTierCounts,
		meta.SpeedCounts,
	} {
		if err := writeStringIntMap(w, counts); err != nil {
			return fmt.Errorf("writeStringIntMap: %w", err)
		}
	}
	return nil
}

func isZeroSessionPerformanceMeta(meta conv.SessionPerformanceMeta) bool {
	return !hasSessionPerformanceScalars(meta) && !hasSessionPerformanceCounts(meta)
}

func hasSessionPerformanceScalars(meta conv.SessionPerformanceMeta) bool {
	for _, value := range []int{
		meta.ReasoningBlockCount,
		meta.ReasoningRedactionCount,
		meta.MaxThinkingTokens,
		meta.ModelContextWindow,
		meta.DurationMS,
		meta.RetryAttemptCount,
		meta.RetryDelayMS,
		meta.MaxRetries,
		meta.AbortCount,
		meta.CompactionCount,
		meta.MicroCompactionCount,
		meta.TaskStartedCount,
		meta.TaskCompleteCount,
		meta.ContextCompactedCount,
		meta.RateLimitSnapshotCount,
	} {
		if value != 0 {
			return true
		}
	}
	return false
}

func hasSessionPerformanceCounts(meta conv.SessionPerformanceMeta) bool {
	for _, counts := range []map[string]int{
		meta.APIErrorCounts,
		meta.StopReasonCounts,
		meta.PhaseCounts,
		meta.EffortCounts,
		meta.ServerToolUseCounts,
		meta.ServiceTierCounts,
		meta.SpeedCounts,
	} {
		if len(counts) > 0 {
			return true
		}
	}
	return false
}
