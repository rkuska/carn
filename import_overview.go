package main

import (
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type importPhase int

const (
	phaseAnalyzing importPhase = iota
	phaseReady
	phaseSyncing
	phaseDone
)

// Messages

type listProjectDirsMsg struct {
	dirs []string
	err  error
}

type analysisProgressMsg struct {
	progress       importProgress
	seen           map[groupKey]*conversationState // local results to merge
	syncCandidates []string                        // files needing sync from this project
}

type analysisFinishedMsg struct {
	analysis importAnalysis
}

type importSyncFileCopiedMsg struct {
	file string
	err  error
}

type importOverviewModel struct {
	cfg      archiveConfig
	phase    importPhase
	spinner  spinner.Model
	progress progress.Model

	// Live analysis state
	analysisProgress importProgress // latest progress snapshot
	analysis         importAnalysis // final result (valid when phase >= phaseReady)

	// Sync state (worker pool pattern)
	files       []string
	current     int
	total       int
	currentFile string
	nextIndex   int
	inFlight    int
	maxWorkers  int
	startTime   time.Time
	result      syncResult

	done     bool // signals app.go to transition to browser
	width    int
	height   int
	helpOpen bool

	// Analysis pipeline state
	projectDirs    []string                        // discovered project dirs
	projIndex      int                             // next project dir to analyze
	seen           map[groupKey]*conversationState // running aggregate
	syncCandidates []string                        // files needing sync
	totalInspected int                             // running file count
}

func newImportOverviewModel(cfg archiveConfig) importOverviewModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(colorPrimary)

	p := progress.New(
		progress.WithDefaultBlend(),
		progress.WithoutPercentage(),
	)

	return importOverviewModel{
		cfg:      cfg,
		phase:    phaseAnalyzing,
		spinner:  s,
		progress: p,
		seen:     make(map[groupKey]*conversationState),
	}
}

func (m importOverviewModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, listProjectDirsCmd(m.cfg.sourceDir))
}

func (m importOverviewModel) Update(msg tea.Msg) (importOverviewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.progress.SetWidth(msg.Width / 3)

	case tea.KeyPressMsg:
		return m.handleKey(msg)

	case listProjectDirsMsg:
		return m.handleListProjectDirs(msg)

	case analysisProgressMsg:
		return m.handleAnalysisProgress(msg)

	case analysisFinishedMsg:
		return m.handleAnalysisFinished(msg)

	case importSyncFileCopiedMsg:
		return m.handleSyncFileCopied(msg)

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
		switch m.phase {
		case phaseAnalyzing:
			// Enter is disabled during analysis
			return m, nil

		case phaseReady:
			if !m.analysis.needsSync() {
				m.done = true
				return m, nil
			}
			// Start sync
			m.phase = phaseSyncing
			m.files = m.analysis.filesToSync
			m.total = len(m.files)
			m.startTime = time.Now()
			return m, m.startCopyBatch()

		case phaseSyncing:
			// Disabled during sync
			return m, nil

		case phaseDone:
			m.done = true
			return m, nil
		}
	}

	return m, nil
}

func (m importOverviewModel) handleListProjectDirs(msg listProjectDirsMsg) (importOverviewModel, tea.Cmd) {
	if msg.err != nil {
		// Analysis failed — transition to ready with empty result
		m.phase = phaseReady
		m.analysis = importAnalysis{
			sourceDir:  m.cfg.sourceDir,
			archiveDir: m.cfg.archiveDir,
		}
		return m, nil
	}

	m.projectDirs = msg.dirs
	if len(m.projectDirs) == 0 {
		// No projects — transition to ready
		m.phase = phaseReady
		m.analysis = importAnalysis{
			sourceDir:  m.cfg.sourceDir,
			archiveDir: m.cfg.archiveDir,
		}
		return m, nil
	}

	m.projIndex = 0
	return m, analyzeProjectCmd(m.projectDirs[0], m.cfg)
}

func (m importOverviewModel) handleAnalysisProgress(msg analysisProgressMsg) (importOverviewModel, tea.Cmd) {
	if msg.progress.err != nil {
		m.projIndex++
	} else {
		m.totalInspected += msg.progress.filesInspected
		m.projIndex++

		// Merge local results into running aggregate
		for gk, state := range msg.seen {
			existing, exists := m.seen[gk]
			if !exists {
				m.seen[gk] = state
			} else {
				existing.hasUpToDate = existing.hasUpToDate || state.hasUpToDate
				existing.hasStale = existing.hasStale || state.hasStale
				existing.allNew = existing.allNew && state.allNew && !existing.hasUpToDate
			}
		}
		m.syncCandidates = append(m.syncCandidates, msg.syncCandidates...)
	}

	// Update live progress for display
	newConvs, toUpdate, _ := classifyConversations(m.seen)
	m.analysisProgress = importProgress{
		filesInspected:   m.totalInspected,
		conversations:    len(m.seen),
		newConversations: newConvs,
		toUpdate:         toUpdate,
	}

	// More projects to analyze?
	if m.projIndex < len(m.projectDirs) {
		m.analysisProgress.currentProject = filepath.Base(m.projectDirs[m.projIndex])
		return m, analyzeProjectCmd(m.projectDirs[m.projIndex], m.cfg)
	}

	// All projects analyzed — build final result
	_, _, upToDate := classifyConversations(m.seen)
	analysis := importAnalysis{
		sourceDir:        m.cfg.sourceDir,
		archiveDir:       m.cfg.archiveDir,
		filesInspected:   m.totalInspected,
		projects:         len(m.projectDirs),
		conversations:    len(m.seen),
		newConversations: newConvs,
		toUpdate:         toUpdate,
		upToDate:         upToDate,
		filesToSync:      m.syncCandidates,
	}

	return m, func() tea.Msg {
		return analysisFinishedMsg{analysis: analysis}
	}
}

func (m importOverviewModel) handleAnalysisFinished(msg analysisFinishedMsg) (importOverviewModel, tea.Cmd) {
	m.phase = phaseReady
	m.analysis = msg.analysis
	return m, nil
}

func (m importOverviewModel) handleSyncFileCopied(msg importSyncFileCopiedMsg) (importOverviewModel, tea.Cmd) {
	m.current++
	if msg.err != nil {
		m.result.failed++
	} else {
		m.result.copied++
	}
	m.currentFile = filepath.Base(msg.file)
	if m.inFlight > 0 {
		m.inFlight--
	}

	if m.current >= m.total && m.inFlight == 0 {
		m.phase = phaseDone
		m.result.elapsed = time.Since(m.startTime)
		return m, nil
	}

	if m.nextIndex < m.total {
		next := m.files[m.nextIndex]
		m.nextIndex++
		m.inFlight++
		return m, importCopyFileCmd(m.cfg, next)
	}
	return m, nil
}

func (m *importOverviewModel) startCopyBatch() tea.Cmd {
	if m.maxWorkers <= 0 {
		m.maxWorkers = min(max(runtime.NumCPU(), 1), 8)
	}

	startCount := min(m.maxWorkers, m.total)

	cmds := make([]tea.Cmd, 0, startCount)
	for range startCount {
		next := m.files[m.nextIndex]
		m.nextIndex++
		m.inFlight++
		cmds = append(cmds, importCopyFileCmd(m.cfg, next))
	}

	return tea.Batch(cmds...)
}

// View renders the import overview based on current phase.
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
		action := "continue"
		if m.analysis.needsSync() {
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

func (m importOverviewModel) renderBox(title string, boxWidth int, content string) string {
	box := renderFramedBox(title, boxWidth, colorPrimary, content)
	return lipgloss.Place(m.width, max(m.height-framedFooterRows, 1), lipgloss.Center, lipgloss.Center, box)
}

// Commands

func listProjectDirsCmd(sourceDir string) tea.Cmd {
	return func() tea.Msg {
		dirs, err := listProjectDirs(sourceDir)
		return listProjectDirsMsg{dirs: dirs, err: err}
	}
}

func analyzeProjectCmd(projDir string, cfg archiveConfig) tea.Cmd {
	return func() tea.Msg {
		localSeen := make(map[groupKey]*conversationState)
		var syncCandidates []string

		filesInspected, err := analyzeProjectDir(projDir, cfg, localSeen, &syncCandidates)
		if err != nil {
			return analysisProgressMsg{
				progress: importProgress{
					err:            err,
					currentProject: filepath.Base(projDir),
				},
			}
		}

		return analysisProgressMsg{
			progress: importProgress{
				filesInspected: filesInspected,
				currentProject: filepath.Base(projDir),
			},
			seen:           localSeen,
			syncCandidates: syncCandidates,
		}
	}
}

func importCopyFileCmd(cfg archiveConfig, srcPath string) tea.Cmd {
	return func() tea.Msg {
		rel, _ := filepath.Rel(cfg.sourceDir, srcPath)
		dst := filepath.Join(cfg.archiveDir, rel)
		err := copyFile(srcPath, dst)
		return importSyncFileCopiedMsg{file: srcPath, err: err}
	}
}
