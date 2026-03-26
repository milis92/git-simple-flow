package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Step status constants.
const (
	StepPending = iota
	StepActive
	StepDone
	StepFailed
)

// StepDef defines a step in a multi-step workflow.
type StepDef struct {
	Label string
}

// step is the internal representation of a workflow step.
type step struct {
	label     string
	status    int
	startedAt time.Time
	elapsed   time.Duration
	errMsg    string
}

// Messages sent to the progress model from workflow goroutines.
type (
	StepStartMsg  struct{}
	StepDoneMsg   struct{}
	StepFailedMsg struct{ Err string }
	WorkflowDone  struct{}
)

// ProgressModel is the Bubble Tea model for multi-step workflow progress.
type ProgressModel struct {
	title    string
	subtitle string
	steps    []step
	current  int
	done     bool
	spinner  spinner.Model
	theme    Theme
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

// Update implements tea.Model. Returns (tea.Model, tea.Cmd) to satisfy the
// tea.Model interface. Callers who need to inspect ProgressModel fields
// should type-assert the result: updated.(ProgressModel).
func (m ProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case StepStartMsg:
		m.current++
		if m.current < len(m.steps) {
			m.steps[m.current].status = StepActive
			m.steps[m.current].startedAt = time.Now()
		}
		return m, nil

	case StepDoneMsg:
		if m.current >= 0 && m.current < len(m.steps) {
			m.steps[m.current].status = StepDone
			m.steps[m.current].elapsed = time.Since(m.steps[m.current].startedAt)
		}
		return m, nil

	case StepFailedMsg:
		if m.current >= 0 && m.current < len(m.steps) {
			m.steps[m.current].status = StepFailed
			m.steps[m.current].elapsed = time.Since(m.steps[m.current].startedAt)
			m.steps[m.current].errMsg = msg.Err
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

		b.WriteString(fmt.Sprintf("  %s %s", icon, label))
		if elapsed != "" {
			b.WriteString("  " + elapsed)
		}
		b.WriteString("\n")

		if s.status == StepFailed && s.errMsg != "" {
			b.WriteString("    " + m.theme.Error.Render(s.errMsg) + "\n")
		}
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

// TotalElapsed returns the sum of elapsed time across all completed steps.
func (m ProgressModel) TotalElapsed() time.Duration {
	var total time.Duration
	for _, s := range m.steps {
		total += s.elapsed
	}
	return total
}

// StepCallbacks provides functions for a workflow to signal step transitions.
type StepCallbacks struct {
	Start func()
	Done  func()
	Fail  func(err string)
}

// RunProgress runs a Bubble Tea progress view while executing the workflow function.
// The workflow function receives StepCallbacks to signal step transitions.
// Returns an error if the workflow fails.
func RunProgress(title, subtitle string, defs []StepDef, workflow func(StepCallbacks) error) error {
	model := NewProgressModel(title, subtitle, defs)

	p := tea.NewProgram(model)

	var workflowErr error

	go func() {
		cb := StepCallbacks{
			Start: func() { p.Send(StepStartMsg{}) },
			Done:  func() { p.Send(StepDoneMsg{}) },
			Fail:  func(err string) { p.Send(StepFailedMsg{Err: err}) },
		}
		workflowErr = workflow(cb)
		p.Send(WorkflowDone{})
	}()

	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	// Print summary after the TUI exits
	m := finalModel.(ProgressModel)
	_ = m // suppress unused warning if needed

	return workflowErr
}

// RunProgressPrint runs a workflow with print-style step output instead of
// a Bubble Tea TUI. Used when interactive mode is disabled.
func RunProgressPrint(u *UI, defs []StepDef, workflow func(StepCallbacks) error) error {
	current := -1
	cb := StepCallbacks{
		Start: func() {
			current++
		},
		Done: func() {
			if current >= 0 && current < len(defs) {
				u.Success(defs[current].Label)
			}
		},
		Fail: func(err string) {
			if current >= 0 && current < len(defs) {
				u.Error(defs[current].Label + " — " + err)
			}
		},
	}
	return workflow(cb)
}
