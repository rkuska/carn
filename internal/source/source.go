package source

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"

	conv "github.com/rkuska/carn/internal/conversation"
)

var (
	ErrResumeTargetIDEmpty = errors.New("resume target id is empty")
	ErrResumeDirEmpty      = errors.New("resume directory is empty")
	ErrResumeDirNotDir     = errors.New("resume directory is not a directory")
)

// Analysis describes provider-local import state before syncing raw files.
type Analysis struct {
	UnitsTotal       int
	FilesInspected   int
	Conversations    int
	NewConversations int
	ToUpdate         int
	UpToDate         int
	SyncCandidates   []string
}

// SyncCandidate is a backend-owned raw-file sync plan item.
type SyncCandidate struct {
	SourcePath string
	DestPath   string
}

// Progress reports provider-local analysis progress using provider-neutral terms.
type Progress struct {
	Provider         conv.Provider
	UnitsCompleted   int
	UnitsTotal       int
	FilesInspected   int
	Conversations    int
	NewConversations int
	ToUpdate         int
	CurrentUnit      string
	Err              error
}

// Backend is the generic provider contract used by archive, canonical, and app.
type Backend interface {
	Provider() conv.Provider
	SourceEnvVars() []string
	DefaultSourceDir(home string) string
	Scan(ctx context.Context, rawDir string) ([]conv.Conversation, error)
	Load(ctx context.Context, conversation conv.Conversation) (conv.Session, error)
	Analyze(ctx context.Context, sourceDir, rawDir string, onProgress func(Progress)) (Analysis, error)
	SyncCandidates(ctx context.Context, sourceDir, rawDir string) ([]SyncCandidate, error)
	ResumeCommand(target conv.ResumeTarget) (*exec.Cmd, error)
}

// ValidateResumeTarget applies the shared strict resume policy.
func ValidateResumeTarget(target conv.ResumeTarget) error {
	if target.ID == "" {
		return fmt.Errorf("validateResumeTarget: %w", ErrResumeTargetIDEmpty)
	}
	if target.CWD == "" {
		return fmt.Errorf("validateResumeTarget: %w", ErrResumeDirEmpty)
	}

	info, err := os.Stat(target.CWD)
	if err != nil {
		return fmt.Errorf("validateResumeTarget_osStat: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("validateResumeTarget: %w", ErrResumeDirNotDir)
	}
	return nil
}
