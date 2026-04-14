package app

import (
	"os"
	"os/exec"
)

func newEditorCmd(filePath string) *exec.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}
	return exec.Command(editor, filePath)
}
