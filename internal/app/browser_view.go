package app

import (
	"charm.land/lipgloss/v2"
)

func (m browserModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var body string
	if m.helpOpen {
		body = renderHelpOverlay(m.width, m.height, m.helpTitle(), m.helpSections())
	} else {
		switch m.transcriptMode {
		case transcriptClosed:
			body = m.renderListPane(m.width, true)
		case transcriptSplit:
			listWidth := m.listPaneWidth()
			transcriptWidth := max(m.width-listWidth-1, 1)
			listPane := m.renderListPane(listWidth, m.focus == focusList)
			transcriptPane := m.renderTranscriptPane(transcriptWidth, m.focus == focusTranscript)
			body = lipgloss.JoinHorizontal(lipgloss.Top, listPane, " ", transcriptPane)
		case transcriptFullscreen:
			body = m.renderTranscriptPane(m.width, true)
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, body, m.footerView())
}

func (m *browserModel) updateLayout() {
	listWidth := m.width
	if m.transcriptMode == transcriptSplit {
		listWidth = m.listPaneWidth()
	}

	m.list.SetSize(max(listWidth-2, 1), framedBodyHeight(m.height))
	if m.transcriptVisible() && m.viewer.session.Meta.ID != "" {
		m.viewer.SetSize(m.viewerWidth(), m.height)
	}
}

func (m browserModel) renderListPane(width int, active bool) string {
	borderColor := colorSecondary
	if active {
		borderColor = colorAccent
	}
	return renderFramedPane(
		"Claude Sessions",
		width,
		framedBodyHeight(m.height),
		borderColor,
		m.list.View(),
	)
}

func (m browserModel) renderTranscriptPane(width int, active bool) string {
	borderColor := colorSecondary
	if active {
		borderColor = colorAccent
	}

	if m.loadingConversationID != "" {
		title := "Transcript"
		if conv, ok := m.selectedConversation(); ok {
			title = conv.Title()
		}
		return renderFramedPane(
			title,
			width,
			framedBodyHeight(m.height),
			borderColor,
			"Loading transcript...",
		)
	}

	if m.viewer.session.Meta.ID == "" {
		return renderFramedPane(
			"Transcript",
			width,
			framedBodyHeight(m.height),
			borderColor,
			"No transcript selected",
		)
	}

	return m.viewer.paneView(borderColor)
}

func (m browserModel) listPaneWidth() int {
	if m.transcriptMode != transcriptSplit {
		return m.width
	}

	listWidth := max((m.width-1)/2, 32)
	return min(listWidth, max(m.width-1, 1))
}

func (m browserModel) viewerWidth() int {
	if m.transcriptMode == transcriptFullscreen {
		return m.width
	}
	if m.transcriptMode == transcriptSplit {
		return max(m.width-m.listPaneWidth()-1, 1)
	}
	return m.width
}
