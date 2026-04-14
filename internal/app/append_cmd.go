package app

import tea "charm.land/bubbletea/v2"

func appendCmd(cmds *[]tea.Cmd, cmd tea.Cmd) {
	if cmd != nil {
		*cmds = append(*cmds, cmd)
	}
}
