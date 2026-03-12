package app

import (
	"context"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	arch "github.com/rkuska/carn/internal/archive"
)

type importPhase int

const (
	phaseAnalyzing importPhase = iota
	phaseReady
	phaseSyncing
	phaseDone
)

type importAnalysisStartedMsg struct {
	events <-chan tea.Msg
}

type analysisProgressMsg struct {
	progress arch.ImportProgress
}

type analysisFinishedMsg struct {
	analysis arch.ImportAnalysis
}

type importSyncStartedMsg struct {
	events <-chan tea.Msg
}

type importSyncProgressMsg struct {
	progress arch.SyncProgress
}

type importSyncFinishedMsg struct {
	result arch.SyncResult
	err    error
}

type importOverviewModel struct {
	ctx      context.Context
	cfg      arch.Config
	pipeline importPipeline
	phase    importPhase
	spinner  spinner.Model
	progress progress.Model

	analysisEvents   <-chan tea.Msg
	analysisProgress arch.ImportProgress
	analysis         arch.ImportAnalysis

	files        []string
	current      int
	total        int
	currentFile  string
	currentStage string
	result       arch.SyncResult
	syncEvents   <-chan tea.Msg

	done     bool
	width    int
	height   int
	helpOpen bool
}

func newImportOverviewModelWithPipeline(
	ctx context.Context,
	cfg arch.Config,
	pipeline importPipeline,
) importOverviewModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(colorPrimary)

	p := progress.New(
		progress.WithDefaultBlend(),
		progress.WithoutPercentage(),
	)

	return importOverviewModel{
		ctx:      ctx,
		cfg:      cfg,
		pipeline: pipeline,
		phase:    phaseAnalyzing,
		spinner:  s,
		progress: p,
	}
}

func newImportOverviewModel(ctx context.Context, cfg arch.Config) importOverviewModel {
	return newImportOverviewModelWithPipeline(
		ctx,
		cfg,
		newDefaultImportPipeline(cfg),
	)
}

func (m importOverviewModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, startImportAnalysisCmd(m.ctx, m.pipeline))
}

func (m importOverviewModel) Update(msg tea.Msg) (importOverviewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.progress.SetWidth(msg.Width / 3)
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	case importAnalysisStartedMsg:
		m.analysisEvents = msg.events
		return m, waitForAsyncImportMsg(m.analysisEvents)
	case analysisProgressMsg:
		return m.handleAnalysisProgress(msg)
	case analysisFinishedMsg:
		return m.handleAnalysisFinished(msg)
	case importSyncStartedMsg, importSyncProgressMsg, importSyncFinishedMsg:
		return m.handleSyncMsg(msg)
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case progress.FrameMsg:
		var cmd tea.Cmd
		m.progress, cmd = m.progress.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m importOverviewModel) handleSyncMsg(msg tea.Msg) (importOverviewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case importSyncStartedMsg:
		m.syncEvents = msg.events
		return m, waitForAsyncImportMsg(m.syncEvents)
	case importSyncProgressMsg:
		return m.handleSyncProgress(msg)
	case importSyncFinishedMsg:
		return m.handleSyncFinished(msg)
	}
	return m, nil
}

func (m importOverviewModel) handleKey(msg tea.KeyPressMsg) (importOverviewModel, tea.Cmd) {
	if m.helpOpen {
		switch {
		case key.Matches(msg, importOverviewKeys.Help), msg.Code == tea.KeyEscape, msg.Text == "q":
			m.helpOpen = false
		}
		return m, nil
	}

	switch {
	case key.Matches(msg, importOverviewKeys.Help):
		m.helpOpen = true
		return m, nil
	case key.Matches(msg, importOverviewKeys.Quit):
		return m, tea.Quit
	case key.Matches(msg, importOverviewKeys.Enter):
		return m.handleEnterKey()
	}

	return m, nil
}

func (m importOverviewModel) handleEnterKey() (importOverviewModel, tea.Cmd) {
	switch m.phase {
	case phaseAnalyzing, phaseSyncing:
		return m, nil
	case phaseReady:
		if m.analysis.Err != nil {
			return m, nil
		}
		if !m.analysis.NeedsSync() {
			m.done = true
			return m, nil
		}
		m.phase = phaseSyncing
		m.files = append([]string(nil), m.analysis.QueuedFiles...)
		m.current = 0
		m.total = len(m.files)
		m.currentFile = ""
		m.currentStage = ""
		m.result = arch.SyncResult{}
		return m, startImportSyncCmd(m.ctx, m.pipeline)
	case phaseDone:
		m.done = true
		return m, nil
	}

	return m, nil
}

func (m importOverviewModel) handleAnalysisProgress(msg analysisProgressMsg) (importOverviewModel, tea.Cmd) {
	m.analysisProgress = msg.progress
	return m, waitForAsyncImportMsg(m.analysisEvents)
}

func (m importOverviewModel) handleAnalysisFinished(msg analysisFinishedMsg) (importOverviewModel, tea.Cmd) {
	m.phase = phaseReady
	m.analysis = msg.analysis
	m.analysisEvents = nil
	return m, nil
}

func (m importOverviewModel) handleSyncProgress(msg importSyncProgressMsg) (importOverviewModel, tea.Cmd) {
	m.current = msg.progress.Current
	m.total = msg.progress.Total
	m.currentFile = msg.progress.File
	m.currentStage = msg.progress.Stage
	m.result.Copied = msg.progress.Copied
	m.result.Failed = msg.progress.Failed
	return m, waitForAsyncImportMsg(m.syncEvents)
}

func (m importOverviewModel) handleSyncFinished(msg importSyncFinishedMsg) (importOverviewModel, tea.Cmd) {
	m.phase = phaseDone
	m.result = msg.result
	m.current = m.total
	m.currentStage = ""
	m.syncEvents = nil
	return m, nil
}

func (m importOverviewModel) View() string {
	if m.width == 0 {
		return ""
	}

	var body string
	switch {
	case m.helpOpen:
		body = renderHelpOverlay(m.width, m.height, "Import Help", m.helpSections())
	default:
		body = m.viewDashboard()
	}

	return lipgloss.JoinVertical(lipgloss.Left, body, m.footerView())
}

func (m importOverviewModel) footerView() string {
	if m.helpOpen {
		return renderHelpFooter(
			m.width,
			[]helpItem{
				{key: "?", desc: "close help"},
				{key: "q/esc", desc: "close help"},
			},
			[]string{m.phaseLabel()},
			notification{},
		)
	}

	return renderHelpFooter(m.width, m.footerItems(), []string{m.phaseLabel()}, notification{})
}

func (m importOverviewModel) footerItems() []helpItem {
	switch m.phase {
	case phaseAnalyzing:
		return []helpItem{
			{key: "?", desc: "help"},
			{key: "q", desc: "quit"},
		}
	case phaseReady:
		if m.analysis.Err != nil {
			return []helpItem{
				{key: "?", desc: "help"},
				{key: "q", desc: "quit"},
			}
		}
		action := "continue"
		if m.analysis.NeedsSync() {
			action = "import"
		}
		return []helpItem{
			{key: "enter", desc: action},
			{key: "?", desc: "help"},
			{key: "q", desc: "quit"},
		}
	case phaseSyncing:
		return []helpItem{
			{key: "?", desc: "help"},
			{key: "q", desc: "quit"},
		}
	case phaseDone:
		return []helpItem{
			{key: "enter", desc: "continue"},
			{key: "?", desc: "help"},
			{key: "q", desc: "quit"},
		}
	default:
		return nil
	}
}

func (m importOverviewModel) helpSections() []helpSection {
	return []helpSection{
		{
			title: "Actions",
			items: m.footerItems(),
		},
	}
}

func (m importOverviewModel) phaseLabel() string {
	switch m.phase {
	case phaseAnalyzing:
		return "analyzing"
	case phaseReady:
		return "ready"
	case phaseSyncing:
		return "syncing"
	case phaseDone:
		return "done"
	default:
		return ""
	}
}

func (m importOverviewModel) renderBox(title string, boxWidth int, content string) string {
	box := renderFramedBox(title, boxWidth, colorPrimary, content)
	return lipgloss.Place(m.width, max(m.height-framedFooterRows, 1), lipgloss.Center, lipgloss.Center, box)
}
