package scenarios

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

type runResult struct {
	model tea.Model
	err   error
}

type trackedModel struct {
	mu          sync.Mutex
	model       tea.Model
	initialSize tea.WindowSizeMsg
}

func newTrackedModel(model tea.Model, width, height int) *trackedModel {
	return &trackedModel{
		model:       model,
		initialSize: tea.WindowSizeMsg{Width: width, Height: height},
	}
}

func (m *trackedModel) Init() tea.Cmd {
	m.mu.Lock()
	defer m.mu.Unlock()

	initCmd := m.model.Init()
	if m.initialSize.Width == 0 && m.initialSize.Height == 0 {
		return initCmd
	}

	sizeMsg := m.initialSize
	return tea.Batch(initCmd, func() tea.Msg {
		return sizeMsg
	})
}

func (m *trackedModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.mu.Lock()
	current := m.model
	m.mu.Unlock()

	next, cmd := current.Update(msg)

	m.mu.Lock()
	m.model = next
	m.mu.Unlock()

	return m, cmd
}

func (m *trackedModel) View() tea.View {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.model.View()
}

type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

type programHarness struct {
	program  *tea.Program
	tracked  *trackedModel
	output   *lockedBuffer
	cancel   context.CancelFunc
	resultCh chan runResult
	done     sync.Once
	mu       sync.Mutex
	finished bool
	result   runResult
}

func newProgramHarness(tb testing.TB, model tea.Model, width, height int) *programHarness {
	tb.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	output := &lockedBuffer{}
	tracked := newTrackedModel(model, width, height)

	h := &programHarness{
		program: tea.NewProgram(
			tracked,
			tea.WithContext(ctx),
			tea.WithInput(bytes.NewBuffer(nil)),
			tea.WithOutput(output),
			tea.WithoutSignals(),
		),
		tracked:  tracked,
		output:   output,
		cancel:   cancel,
		resultCh: make(chan runResult, 1),
	}

	go func() {
		model, err := h.program.Run()
		h.resultCh <- runResult{model: model, err: err}
	}()

	tb.Cleanup(func() {
		cancel()
		if !h.isFinished() {
			h.program.Kill()
			h.wait(tb, 2*time.Second)
		}
	})

	return h
}

func (h *programHarness) waitForText(tb testing.TB, text string) {
	tb.Helper()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(h.currentView(), text) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	tb.Fatalf("waitForText(%q): last view:\n%s", text, h.currentView())
}

func (h *programHarness) pressEnter() {
	h.program.Send(tea.KeyPressMsg{Code: tea.KeyEnter})
}

func (h *programHarness) quit(tb testing.TB) {
	tb.Helper()
	h.program.Quit()
	h.wait(tb, 2*time.Second)
	h.cancel()
}

func (h *programHarness) finalView(tb testing.TB) string {
	tb.Helper()
	h.wait(tb, 2*time.Second)

	if h.result.err != nil {
		tb.Fatalf("program.Run: %v", h.result.err)
	}
	if h.result.model == nil {
		tb.Fatal("program.Run returned a nil model")
	}

	return h.currentView()
}

func (h *programHarness) wait(tb testing.TB, timeout time.Duration) {
	tb.Helper()

	h.done.Do(func() {
		select {
		case h.result = <-h.resultCh:
			h.mu.Lock()
			h.finished = true
			h.mu.Unlock()
		case <-time.After(timeout):
			tb.Fatalf("wait: timeout after %s", timeout)
		}
	})
}

func (h *programHarness) isFinished() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.finished
}

func (h *programHarness) currentView() string {
	return normalizeOutput(h.tracked.View().Content)
}

func normalizeOutput(s string) string {
	s = ansi.Strip(s)
	s = strings.ReplaceAll(s, "\r", "")
	return s
}
