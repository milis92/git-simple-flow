package ui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	"strings"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// StepStatus represents the current state of a workflow step.
type StepStatus int

// Step status constants.
const (
	StepPending StepStatus = iota
	StepActive
	StepDone
	StepFailed
	StepSkipped
)

// StepDef defines a step in a multi-step workflow.
type StepDef struct {
	Label string
}

// step is the internal representation of a workflow step.
type step struct {
	label     string
	status    StepStatus
	startedAt time.Time
	elapsed   time.Duration
	errMsg    string
}

// Messages sent to the progress model from workflow goroutines.
type (
	StepStartMsg   struct{}
	StepDoneMsg    struct{}
	StepFailedMsg  struct{ Err string }
	StepSkippedMsg struct{ Reason string }
	WorkflowDone   struct{}
)

// ProgressModel is the Bubble Tea model for multi-step workflow progress.
type ProgressModel struct {
	title       string
	subtitle    string
	steps       []step
	current     int
	done        bool
	overflowErr string
	spinner     spinner.Model
	theme       Theme
}

// NewProgressModel creates a new progress model for the given workflow.
func NewProgressModel(title, subtitle string, defs []StepDef) ProgressModel {
	steps := make([]step, len(defs))
	for i, d := range defs {
		steps[i] = step{label: d.Label, status: StepPending}
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))

	return ProgressModel{
		title:    title,
		subtitle: subtitle,
		steps:    steps,
		current:  -1,
		spinner:  s,
		theme:    DefaultTheme(),
	}
}

// Init implements tea.Model.
func (m ProgressModel) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update implements tea.Model.
func (m ProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case StepStartMsg:
		m.current++
		if m.current < len(m.steps) {
			m.steps[m.current].status = StepActive
			m.steps[m.current].startedAt = time.Now()
		} else {
			m.overflowErr = fmt.Sprintf("workflow sent more steps (%d) than defined (%d)", m.current+1, len(m.steps))
		}
		return m, nil

	case StepDoneMsg:
		if m.current >= 0 && m.current < len(m.steps) && m.steps[m.current].status == StepActive {
			m.steps[m.current].status = StepDone
			m.steps[m.current].elapsed = time.Since(m.steps[m.current].startedAt)
		}
		return m, nil

	case StepFailedMsg:
		if m.current >= 0 && m.current < len(m.steps) && m.steps[m.current].status == StepActive {
			m.steps[m.current].status = StepFailed
			m.steps[m.current].elapsed = time.Since(m.steps[m.current].startedAt)
			m.steps[m.current].errMsg = msg.Err
		}
		return m, nil

	case StepSkippedMsg:
		if m.current >= 0 && m.current < len(m.steps) && m.steps[m.current].status == StepActive {
			m.steps[m.current].status = StepSkipped
			m.steps[m.current].elapsed = time.Since(m.steps[m.current].startedAt)
			m.steps[m.current].errMsg = msg.Reason
		}
		return m, nil

	case WorkflowDone:
		m.done = true
		return m, tea.Quit

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	default:
		return m, nil
	}
}

// View implements tea.Model.
func (m ProgressModel) View() string {
	var b strings.Builder

	// Header
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Bold(true)
	subtitleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	if m.done && !m.hasFailed() {
		titleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)
	}

	b.WriteString(titleStyle.Render(m.title))
	b.WriteString("  ")
	b.WriteString(subtitleStyle.Render(m.subtitle))
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(strings.Repeat("─", 40)))
	b.WriteString("\n")

	// Steps
	for _, s := range m.steps {
		var icon string
		switch s.status {
		case StepDone:
			icon = m.theme.Success.Render(m.theme.IconDone)
		case StepActive:
			icon = m.spinner.View()
		case StepSkipped:
			icon = m.theme.Warning.Render(m.theme.IconWarning)
		case StepFailed:
			icon = m.theme.Error.Render(m.theme.IconFail)
		default:
			icon = m.theme.Muted.Render(m.theme.IconPending)
		}

		label := s.label
		if s.status == StepPending {
			label = m.theme.Muted.Render(label)
		}

		elapsed := ""
		if s.elapsed > 0 {
			elapsed = subtitleStyle.Render(fmt.Sprintf("%.1fs", s.elapsed.Seconds()))
		}

		fmt.Fprintf(&b, "  %s %s", icon, label)
		if elapsed != "" {
			b.WriteString("  " + elapsed)
		}
		b.WriteString("\n")

		if s.errMsg != "" {
			switch s.status {
			case StepFailed:
				b.WriteString("    " + m.theme.Error.Render(s.errMsg) + "\n")
			case StepSkipped:
				b.WriteString("    " + m.theme.Muted.Render(s.errMsg) + "\n")
			}
		}
	}

	if m.overflowErr != "" {
		b.WriteString("\n" + m.theme.Error.Render("BUG: "+m.overflowErr) + "\n")
	}

	// Wrap in border
	content := b.String()
	bordered := m.theme.Border.Render(content)
	return "\n" + bordered + "\n"
}

func (m ProgressModel) hasFailed() bool {
	for _, s := range m.steps {
		if s.status == StepFailed {
			return true
		}
	}
	return false
}

// TotalElapsed returns the sum of elapsed time across all finished steps (both successful and failed).
func (m ProgressModel) TotalElapsed() time.Duration {
	var total time.Duration
	for _, s := range m.steps {
		total += s.elapsed
	}
	return total
}

// StepCallbacks provides functions for a workflow to signal step transitions.
// The workflow must call Start before each step, followed by exactly one of
// Done, Fail, or Skip (via SkipStep). The number of Start calls must not
// exceed the number of step definitions passed to NewProgressModel.
type StepCallbacks struct {
	Start func()
	Done  func()
	Fail  func(err string)
	Skip  func(reason string)
}

// Run executes fn as the current step. It calls Start before fn and
// Done or Fail after, depending on whether fn returns an error.
func (cb StepCallbacks) Run(fn func() error) error {
	cb.Start()
	if err := fn(); err != nil {
		cb.Fail(err.Error())
		return err
	}
	cb.Done()
	return nil
}

// RunSoftFail executes fn as the current step. On failure it marks the step
// as skipped (not failed) and returns nil, allowing the workflow to continue.
// Context cancellation errors are not swallowed — they are propagated so the
// caller can detect user interruption.
func (cb StepCallbacks) RunSoftFail(fn func() error) error {
	cb.Start()
	if err := fn(); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			cb.Fail(err.Error())
			return err
		}
		cb.SkipStep(err.Error())
	} else {
		cb.Done()
	}
	return nil
}

// SkipStep marks the current step as skipped when the workflow can continue.
// If Skip is not provided, it falls back to Done so soft-fail paths do not
// escalate to a hard failure.
func (cb StepCallbacks) SkipStep(reason string) {
	if cb.Skip != nil {
		cb.Skip(reason)
		return
	}
	cb.Done()
}

// RunProgress runs a Bubble Tea progress view while executing the workflow function.
// The workflow function receives a context and StepCallbacks to signal step transitions.
// The context is cancelled when Bubble Tea exits, allowing workflows to bail out early.
// Returns an error if the workflow fails, if Bubble Tea encounters a runtime error, or if the user interrupts execution.
func RunProgress(title, subtitle string, defs []StepDef, workflow func(context.Context, StepCallbacks) error) error {
	model := NewProgressModel(title, subtitle, defs)

	p := tea.NewProgram(model)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	var unexpectedPanic atomic.Value // stores string if safeSend catches a non-"send on closed" panic

	// safeSend sends a message to the Bubble Tea program, suppressing the
	// expected "send on closed channel" panic from a program that has already
	// quit (e.g. user pressed Ctrl+C). Unexpected panics are stored and
	// surfaced by RunProgress after the program exits.
	safeSend := func(msg tea.Msg) {
		defer func() {
			if r := recover(); r != nil {
				s := fmt.Sprintf("%v", r)
				if !strings.Contains(s, "send on closed") {
					unexpectedPanic.Store(s)
					_, _ = fmt.Fprintf(os.Stderr, "unexpected panic in progress view: %v\n", r)
				}
			}
		}()
		p.Send(msg)
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				safeSend(StepFailedMsg{Err: fmt.Sprintf("panic: %v", r)})
				errCh <- fmt.Errorf("workflow panicked: %v\n%s", r, debug.Stack())
				safeSend(WorkflowDone{})
			}
		}()
		cb := StepCallbacks{
			Start: func() { safeSend(StepStartMsg{}) },
			Done:  func() { safeSend(StepDoneMsg{}) },
			Fail:  func(err string) { safeSend(StepFailedMsg{Err: err}) },
			Skip:  func(reason string) { safeSend(StepSkippedMsg{Reason: reason}) },
		}
		errCh <- workflow(ctx, cb)
		safeSend(WorkflowDone{})
	}()

	finalModel, err := p.Run()
	if err != nil {
		cancel()
		return err
	}

	cancel() // Signal goroutine to stop between steps
	wfErr := <-errCh
	if errors.Is(wfErr, context.Canceled) {
		return fmt.Errorf("interrupted: %w", context.Canceled)
	}
	if wfErr != nil {
		return wfErr
	}

	if m, ok := finalModel.(ProgressModel); ok && m.overflowErr != "" {
		return fmt.Errorf("BUG: %s", m.overflowErr)
	}

	if v := unexpectedPanic.Load(); v != nil {
		return fmt.Errorf("unexpected panic in progress view: %s", v.(string))
	}

	return nil
}
