package ui

import (
	"testing"
	"time"
)

func TestProgressModelInitialState(t *testing.T) {
	steps := []StepDef{
		{Label: "Find PR"},
		{Label: "Check CI"},
		{Label: "Merge"},
	}
	m := NewProgressModel("Finishing feature", "feature/auth", steps)

	if m.title != "Finishing feature" {
		t.Errorf("title = %q, want %q", m.title, "Finishing feature")
	}
	if len(m.steps) != 3 {
		t.Fatalf("steps count = %d, want 3", len(m.steps))
	}
	if m.steps[0].status != StepPending {
		t.Errorf("step 0 status = %d, want StepPending", m.steps[0].status)
	}
	if m.current != -1 {
		t.Errorf("current = %d, want -1 (not started)", m.current)
	}
}

func TestProgressModelStepTransitions(t *testing.T) {
	steps := []StepDef{
		{Label: "Step A"},
		{Label: "Step B"},
	}
	m := NewProgressModel("Test", "branch", steps)

	// Start step 0
	updated, _ := m.Update(StepStartMsg{})
	m = updated.(ProgressModel)
	if m.current != 0 {
		t.Errorf("after StepStart: current = %d, want 0", m.current)
	}
	if m.steps[0].status != StepActive {
		t.Errorf("step 0 should be Active")
	}

	// Complete step 0
	updated, _ = m.Update(StepDoneMsg{})
	m = updated.(ProgressModel)
	if m.steps[0].status != StepDone {
		t.Errorf("step 0 should be Done")
	}

	// Start step 1
	updated, _ = m.Update(StepStartMsg{})
	m = updated.(ProgressModel)
	if m.current != 1 {
		t.Errorf("after second StepStart: current = %d, want 1", m.current)
	}

	// Fail step 1
	updated, _ = m.Update(StepFailedMsg{Err: "CI failed"})
	m = updated.(ProgressModel)
	if m.steps[1].status != StepFailed {
		t.Errorf("step 1 should be Failed")
	}
	if m.steps[1].errMsg != "CI failed" {
		t.Errorf("step 1 errMsg = %q, want %q", m.steps[1].errMsg, "CI failed")
	}
}

func TestProgressModelElapsedTime(t *testing.T) {
	steps := []StepDef{{Label: "Slow step"}}
	m := NewProgressModel("Test", "branch", steps)

	updated, _ := m.Update(StepStartMsg{})
	m = updated.(ProgressModel)
	// Simulate time passing
	m.steps[0].startedAt = time.Now().Add(-2 * time.Second)
	updated, _ = m.Update(StepDoneMsg{})
	m = updated.(ProgressModel)

	if m.steps[0].elapsed < time.Second {
		t.Errorf("elapsed = %v, expected >= 1s", m.steps[0].elapsed)
	}
}

func TestStepCallbacksType(t *testing.T) {
	var started, completed int
	var failedMsg string

	sc := StepCallbacks{
		Start: func() { started++ },
		Done:  func() { completed++ },
		Fail:  func(err string) { failedMsg = err },
	}

	sc.Start()
	sc.Done()
	sc.Start()
	sc.Done()
	sc.Start()
	sc.Fail("something broke")

	if started != 3 {
		t.Errorf("started = %d, want 3", started)
	}
	if completed != 2 {
		t.Errorf("completed = %d, want 2", completed)
	}
	if failedMsg != "something broke" {
		t.Errorf("failedMsg = %q, want %q", failedMsg, "something broke")
	}
}

func TestProgressModelStepOverflow(t *testing.T) {
	steps := []StepDef{{Label: "Only step"}}
	m := NewProgressModel("Test", "branch", steps)

	// Start and complete the only step
	updated, _ := m.Update(StepStartMsg{})
	m = updated.(ProgressModel)
	updated, _ = m.Update(StepDoneMsg{})
	m = updated.(ProgressModel)

	// One more StepStart than steps defined — should set overflow error
	updated, _ = m.Update(StepStartMsg{})
	m = updated.(ProgressModel)

	if m.overflowErr == "" {
		t.Error("expected overflowErr to be set when steps exceed definitions")
	}
}
