package models

// Monitor represents a display/monitor attached to the system
type Monitor struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	X           int    `json:"x"`
	Y           int    `json:"y"`
	Focused     bool   `json:"focused"`
	Scale       float64 `json:"scale"`
	Transform   int    `json:"transform"`
	ActiveWorkspace struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"activeWorkspace,omitempty"`
}

// CursorPosition represents the current cursor position
type CursorPosition struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// ContainsCursor checks if the monitor contains the given cursor position
func (m *Monitor) ContainsCursor(pos CursorPosition) bool {
	return pos.X >= m.X &&
		pos.X < m.X+m.Width &&
		pos.Y >= m.Y &&
		pos.Y < m.Y+m.Height
}
