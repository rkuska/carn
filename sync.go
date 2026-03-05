package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type syncModel struct {
	cfg         archiveConfig
	progress    progress.Model
	spinner     spinner.Model
	current     int
	total       int
	currentFile string
	done        bool
	result      syncResult
	width       int
	height      int

	// Phase tracking
	files   []string // files to sync, populated after scan
	scanned bool
}

// Messages

type syncFilesScannedMsg struct {
	files []string
}

type syncFileCopiedMsg struct {
	file string
	err  error
}

func newSyncModel(cfg archiveConfig) syncModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(colorPrimary)

	p := progress.New(
		progress.WithDefaultBlend(),
		progress.WithoutPercentage(),
	)

	return syncModel{
		cfg:      cfg,
		spinner:  s,
		progress: p,
	}
}

func (m syncModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, scanFilesCmd(m.cfg))
}

func (m syncModel) Update(msg tea.Msg) (syncModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.progress.SetWidth(msg.Width / 3)

	case syncFilesScannedMsg:
		m.scanned = true
		m.files = msg.files
		m.total = len(msg.files)
		if m.total == 0 {
			m.done = true
			m.result = syncResult{}
			return m, nil
		}
		return m, copyNextFileCmd(m.cfg, m.files[0])

	case syncFileCopiedMsg:
		m.current++
		if msg.err != nil {
			m.result.failed++
		} else {
			m.result.copied++
		}

		if m.current >= m.total {
			m.done = true
			return m, nil
		}

		m.currentFile = filepath.Base(m.files[m.current])
		return m, copyNextFileCmd(m.cfg, m.files[m.current])

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

func (m syncModel) View() string {
	if m.width == 0 {
		return ""
	}

	var b strings.Builder

	// Title with spinner
	title := fmt.Sprintf("%s Syncing sessions...", m.spinner.View())

	// Progress section
	var progressLine string
	if m.total > 0 {
		pct := float64(m.current) / float64(m.total)
		progressLine = fmt.Sprintf("%s %d/%d",
			m.progress.ViewAs(pct),
			m.current, m.total)
	} else if !m.scanned {
		progressLine = "Scanning files..."
	}

	// Current file
	fileName := m.currentFile
	if fileName == "" && len(m.files) > 0 {
		fileName = filepath.Base(m.files[0])
	}

	// Archive destination
	dest := fmt.Sprintf("→ %s", m.cfg.archiveDir)

	// Build content
	content := lipgloss.JoinVertical(lipgloss.Left,
		"",
		"   "+title,
		"",
		"   "+progressLine,
		"   "+styleSubtitle.Render(fileName),
		"",
		"   "+styleSubtitle.Render(dest),
		"",
	)

	// Wrap in a border box
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary).
		Width(m.width / 2).
		Padding(0)

	box := boxStyle.Render(content)

	// Center the box
	b.WriteString(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box))

	return b.String()
}

// Commands

func scanFilesCmd(cfg archiveConfig) tea.Cmd {
	return func() tea.Msg {
		files, _ := collectFilesToSync(cfg)
		return syncFilesScannedMsg{files: files}
	}
}

func copyNextFileCmd(cfg archiveConfig, srcPath string) tea.Cmd {
	return func() tea.Msg {
		rel, _ := filepath.Rel(cfg.sourceDir, srcPath)
		dst := filepath.Join(cfg.archiveDir, rel)
		err := copyFile(srcPath, dst)
		return syncFileCopiedMsg{file: srcPath, err: err}
	}
}

// collectFilesToSync walks the source dir and returns paths of .jsonl files needing sync.
func collectFilesToSync(cfg archiveConfig) ([]string, error) {
	if _, err := statDir(cfg.sourceDir); err != nil {
		return nil, nil
	}

	var files []string
	err := filepath.WalkDir(cfg.sourceDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".jsonl") {
			return nil
		}

		rel, err := filepath.Rel(cfg.sourceDir, path)
		if err != nil {
			return nil
		}
		dstPath := filepath.Join(cfg.archiveDir, rel)

		info, err := d.Info()
		if err != nil {
			return nil
		}

		if fileNeedsSync(info, dstPath) {
			files = append(files, path)
		}
		return nil
	})

	return files, err
}

func statDir(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}
