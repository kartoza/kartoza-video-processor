package tui

import (
	"errors"
	"testing"
	"time"
)

func TestNewProcessingState(t *testing.T) {
	p := NewProcessingState()

	if p == nil {
		t.Fatal("NewProcessingState returned nil")
	}

	if len(p.Steps) != 5 {
		t.Errorf("expected 5 steps, got %d", len(p.Steps))
	}

	if p.CurrentStep != -1 {
		t.Errorf("expected CurrentStep to be -1, got %d", p.CurrentStep)
	}

	if p.IsProcessing {
		t.Error("expected IsProcessing to be false")
	}

	// Check all steps are pending
	for i, step := range p.Steps {
		if step.Status != StepPending {
			t.Errorf("expected step %d to be StepPending, got %d", i, step.Status)
		}
	}
}

func TestProcessingState_Start(t *testing.T) {
	p := NewProcessingState()

	p.Start()

	if !p.IsProcessing {
		t.Error("expected IsProcessing to be true after Start")
	}

	if p.CurrentStep != 0 {
		t.Errorf("expected CurrentStep to be 0, got %d", p.CurrentStep)
	}

	if p.Steps[0].Status != StepRunning {
		t.Errorf("expected first step to be StepRunning, got %d", p.Steps[0].Status)
	}

	if p.StartTime.IsZero() {
		t.Error("expected StartTime to be set")
	}

	if p.Steps[0].StartTime.IsZero() {
		t.Error("expected first step StartTime to be set")
	}
}

func TestProcessingState_NextStep(t *testing.T) {
	p := NewProcessingState()
	p.Start()

	// Move to next step
	p.NextStep()

	if p.CurrentStep != 1 {
		t.Errorf("expected CurrentStep to be 1, got %d", p.CurrentStep)
	}

	if p.Steps[0].Status != StepComplete {
		t.Errorf("expected first step to be StepComplete, got %d", p.Steps[0].Status)
	}

	if p.Steps[1].Status != StepRunning {
		t.Errorf("expected second step to be StepRunning, got %d", p.Steps[1].Status)
	}

	if p.Steps[0].EndTime.IsZero() {
		t.Error("expected first step EndTime to be set")
	}
}

func TestProcessingState_SkipStep(t *testing.T) {
	p := NewProcessingState()
	p.Start()

	// Skip first step
	p.SkipStep()

	if p.CurrentStep != 1 {
		t.Errorf("expected CurrentStep to be 1, got %d", p.CurrentStep)
	}

	if p.Steps[0].Status != StepSkipped {
		t.Errorf("expected first step to be StepSkipped, got %d", p.Steps[0].Status)
	}

	if p.Steps[1].Status != StepRunning {
		t.Errorf("expected second step to be StepRunning, got %d", p.Steps[1].Status)
	}
}

func TestProcessingState_FailStep(t *testing.T) {
	p := NewProcessingState()
	p.Start()

	testErr := errors.New("test error")
	p.FailStep(testErr)

	if p.Steps[0].Status != StepFailed {
		t.Errorf("expected first step to be StepFailed, got %d", p.Steps[0].Status)
	}

	if p.Error != testErr {
		t.Errorf("expected Error to be set to test error")
	}

	if p.Steps[0].EndTime.IsZero() {
		t.Error("expected first step EndTime to be set")
	}
}

func TestProcessingState_Complete(t *testing.T) {
	p := NewProcessingState()
	p.Start()

	// Move through all steps (5 steps now: stop, analyze, normalize, merge, vertical)
	for i := 0; i < 4; i++ {
		p.NextStep()
	}

	// Complete
	p.Complete()

	if p.IsProcessing {
		t.Error("expected IsProcessing to be false after Complete")
	}

	if p.Steps[4].Status != StepComplete {
		t.Errorf("expected last step to be StepComplete, got %d", p.Steps[4].Status)
	}
}

func TestProcessingState_Reset(t *testing.T) {
	p := NewProcessingState()
	p.Start()
	p.NextStep()
	p.NextStep()
	p.FailStep(errors.New("test"))

	p.Reset()

	if p.IsProcessing {
		t.Error("expected IsProcessing to be false after Reset")
	}

	if p.CurrentStep != -1 {
		t.Errorf("expected CurrentStep to be -1, got %d", p.CurrentStep)
	}

	if p.Error != nil {
		t.Error("expected Error to be nil after Reset")
	}

	// Check all steps are pending
	for i, step := range p.Steps {
		if step.Status != StepPending {
			t.Errorf("expected step %d to be StepPending after Reset, got %d", i, step.Status)
		}
		if !step.StartTime.IsZero() {
			t.Errorf("expected step %d StartTime to be zero after Reset", i)
		}
		if !step.EndTime.IsZero() {
			t.Errorf("expected step %d EndTime to be zero after Reset", i)
		}
	}
}

func TestRenderProcessingView_Nil(t *testing.T) {
	result := RenderProcessingView(nil, 80, 24, 0, ProcessingButtonMenu, false)
	if result != "" {
		t.Errorf("expected empty string for nil state, got %q", result)
	}
}

func TestRenderProcessingView_Basic(t *testing.T) {
	p := NewProcessingState()
	p.Start()

	result := RenderProcessingView(p, 80, 24, 0, ProcessingButtonMenu, false)

	if result == "" {
		t.Error("expected non-empty view")
	}

	// Check for key content
	if !containsString(result, "Processing") {
		t.Error("expected view to contain 'Processing'")
	}

	if !containsString(result, "Stopping recorders") {
		t.Error("expected view to contain 'Stopping recorders'")
	}
}

func TestRenderProcessingView_Animation(t *testing.T) {
	p := NewProcessingState()
	p.Start()

	// Render at different frames should produce different output
	// (due to spinning animation)
	result0 := RenderProcessingView(p, 80, 24, 0, ProcessingButtonMenu, false)
	result1 := RenderProcessingView(p, 80, 24, 1, ProcessingButtonMenu, false)

	// They might be different due to animation, but both should be non-empty
	if result0 == "" || result1 == "" {
		t.Error("expected non-empty views")
	}
}

func TestRenderProcessingView_Complete(t *testing.T) {
	p := NewProcessingState()
	p.Start()
	for i := 0; i < 5; i++ {
		p.NextStep()
	}
	p.Complete()

	result := RenderProcessingView(p, 80, 24, 0, ProcessingButtonMenu, false)

	if !containsString(result, "complete") {
		t.Error("expected view to contain 'complete'")
	}
}

func TestRenderProcessingView_CompleteWithYouTube(t *testing.T) {
	p := NewProcessingState()
	p.Start()
	for i := 0; i < 5; i++ {
		p.NextStep()
	}
	p.Complete()

	result := RenderProcessingView(p, 80, 24, 0, ProcessingButtonUpload, true)

	if !containsString(result, "Upload to YouTube") {
		t.Error("expected view to contain 'Upload to YouTube' button")
	}
	if !containsString(result, "Return to Menu") {
		t.Error("expected view to contain 'Return to Menu' button")
	}
}

func TestRenderProcessingView_Error(t *testing.T) {
	p := NewProcessingState()
	p.Start()
	p.FailStep(errors.New("test error"))

	result := RenderProcessingView(p, 80, 24, 0, ProcessingButtonMenu, false)

	if !containsString(result, "test error") {
		t.Error("expected view to contain error message")
	}
}

func TestStepStatus_Values(t *testing.T) {
	// Ensure status values are as expected
	if StepPending != 0 {
		t.Error("StepPending should be 0")
	}
	if StepRunning != 1 {
		t.Error("StepRunning should be 1")
	}
	if StepComplete != 2 {
		t.Error("StepComplete should be 2")
	}
	if StepFailed != 3 {
		t.Error("StepFailed should be 3")
	}
	if StepSkipped != 4 {
		t.Error("StepSkipped should be 4")
	}
}

func TestProcessingStep_Duration(t *testing.T) {
	p := NewProcessingState()
	p.Start()

	// Wait a bit
	time.Sleep(10 * time.Millisecond)

	// Complete the step
	p.NextStep()

	// Check that duration is non-zero
	step := p.Steps[0]
	duration := step.EndTime.Sub(step.StartTime)

	if duration <= 0 {
		t.Error("expected positive duration for completed step")
	}
}
