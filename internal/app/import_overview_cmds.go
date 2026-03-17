package app

import (
	"context"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	arch "github.com/rkuska/carn/internal/archive"
)

// renderKeyHint renders a hint line with the key name highlighted white when
// active, or entirely grey when disabled. The surrounding text is always grey.
func renderKeyHint(parts ...string) string {
	grey := lipgloss.NewStyle().Foreground(colorSecondary)
	white := lipgloss.NewStyle().Foreground(colorStatusFg)
	var b strings.Builder
	for i, p := range parts {
		if i%2 == 0 {
			b.WriteString(grey.Render(p))
		} else {
			b.WriteString(white.Render(p))
		}
	}
	return b.String()
}

// formatElapsed formats a duration showing milliseconds for sub-second durations.
func formatElapsed(d time.Duration) string {
	if d < time.Second {
		return (time.Duration(d.Milliseconds()) * time.Millisecond).String()
	}
	return d.Round(100 * time.Millisecond).String()
}

func startImportAnalysisCmd(ctx context.Context, pipeline importPipeline) tea.Cmd {
	return func() tea.Msg {
		events := make(chan tea.Msg)
		go func() {
			analysis, err := pipeline.Analyze(ctx, func(progress arch.ImportProgress) {
				select {
				case events <- analysisProgressMsg{progress: progress}:
				case <-ctx.Done():
				}
			})
			if err != nil {
				analysis = arch.ImportAnalysis{Err: err}
			}
			select {
			case events <- analysisFinishedMsg{analysis: analysis}:
			case <-ctx.Done():
			}
			close(events)
		}()
		return importAnalysisStartedMsg{events: events}
	}
}

func startImportSyncCmd(ctx context.Context, pipeline importPipeline) tea.Cmd {
	return func() tea.Msg {
		events := make(chan tea.Msg)
		go func() {
			result, err := pipeline.Run(ctx, func(progress arch.SyncProgress) {
				select {
				case events <- importSyncProgressMsg{progress: progress}:
				case <-ctx.Done():
				}
			})
			select {
			case events <- importSyncFinishedMsg{result: result, err: err}:
			case <-ctx.Done():
			}
			close(events)
		}()
		return importSyncStartedMsg{events: events}
	}
}

func waitForAsyncImportMsg(events <-chan tea.Msg) tea.Cmd {
	if events == nil {
		return nil
	}

	return func() tea.Msg {
		msg, ok := <-events
		if !ok {
			return nil
		}
		return msg
	}
}
