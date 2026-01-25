//go:build !cgo

package systray

import (
	"fmt"
	"time"
)

// Stub implementation for non-CGO builds
// System tray functionality requires CGO and is not available in this build.

type TrayState int

const (
	StateIdle TrayState = iota
	StateRecording
	StatePaused
	StateProcessing
)

type RecordingInfo struct {
	Monitor   string
	StartTime time.Time
	IsPaused  bool
}

type Manager struct{}

func New() *Manager {
	return &Manager{}
}

func (m *Manager) StartChan() <-chan struct{}        { return make(chan struct{}) }
func (m *Manager) StopChan() <-chan struct{}         { return make(chan struct{}) }
func (m *Manager) PauseChan() <-chan struct{}        { return make(chan struct{}) }
func (m *Manager) TUIChan() <-chan struct{}          { return make(chan struct{}) }
func (m *Manager) QuitChan() <-chan struct{}         { return make(chan struct{}) }
func (m *Manager) OnReady()                          {}
func (m *Manager) OnExit()                           {}
func (m *Manager) SetRecordingActive(string, time.Time) {}
func (m *Manager) SetRecordingPaused()               {}
func (m *Manager) SetIdle()                          {}
func (m *Manager) SetProcessing()                    {}
func (m *Manager) StartRecording() error             { return fmt.Errorf("systray not available: built without CGO") }
func (m *Manager) StopRecording() error              { return fmt.Errorf("systray not available: built without CGO") }
func (m *Manager) PauseRecording() error             { return fmt.Errorf("systray not available: built without CGO") }
func (m *Manager) OpenTUI() error                    { return fmt.Errorf("systray not available: built without CGO") }

func Run() {
	fmt.Println("System tray not available: this build was compiled without CGO support.")
	fmt.Println("Use 'kartoza-screencaster' (TUI mode) instead.")
}

func RunWithHandler() {
	Run()
}
