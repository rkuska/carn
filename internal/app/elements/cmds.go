package elements

import tea "charm.land/bubbletea/v2"

func AppendCmd(cmds *[]tea.Cmd, cmd tea.Cmd) {
	if cmd != nil {
		*cmds = append(*cmds, cmd)
	}
}
