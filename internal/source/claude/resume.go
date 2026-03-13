package claude

import (
	"fmt"
	"os/exec"

	conv "github.com/rkuska/carn/internal/conversation"
	src "github.com/rkuska/carn/internal/source"
)

func (Source) ResumeCommand(target conv.ResumeTarget) (*exec.Cmd, error) {
	if err := src.ValidateResumeTarget(target); err != nil {
		return nil, fmt.Errorf("resumeCommand_validateResumeTarget: %w", err)
	}

	cmd := exec.Command("claude", "--resume", target.ID)
	cmd.Dir = target.CWD
	return cmd, nil
}
