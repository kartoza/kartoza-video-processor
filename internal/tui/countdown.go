package tui

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Big segment-style digit patterns (7-segment style)
// Each digit is 7 lines tall
var bigDigits = map[rune][]string{
	'5': {
		" ███████ ",
		" █       ",
		" █       ",
		" ███████ ",
		"       █ ",
		"       █ ",
		" ███████ ",
	},
	'4': {
		" █     █ ",
		" █     █ ",
		" █     █ ",
		" ███████ ",
		"       █ ",
		"       █ ",
		"       █ ",
	},
	'3': {
		" ███████ ",
		"       █ ",
		"       █ ",
		" ███████ ",
		"       █ ",
		"       █ ",
		" ███████ ",
	},
	'2': {
		" ███████ ",
		"       █ ",
		"       █ ",
		" ███████ ",
		" █       ",
		" █       ",
		" ███████ ",
	},
	'1': {
		"    █    ",
		"   ██    ",
		"    █    ",
		"    █    ",
		"    █    ",
		"    █    ",
		"   ███   ",
	},
	'0': {
		" ███████ ",
		" █     █ ",
		" █     █ ",
		" █     █ ",
		" █     █ ",
		" █     █ ",
		" ███████ ",
	},
}

// getBigDigit returns the big digit pattern for a count number (1-5)
func getBigDigit(count int) []string {
	if count < 0 || count > 9 {
		return nil
	}
	digit := rune('0' + count)
	return bigDigits[digit]
}

// "GO!" in big letters
var bigGO = []string{
	"  ██████   ██████  ██ ",
	" ██       ██    ██ ██ ",
	" ██   ███ ██    ██ ██ ",
	" ██    ██ ██    ██ ██ ",
	" ██    ██ ██    ██    ",
	"  ██████   ██████  ██ ",
}

// Descending frequencies for countdown beeps (Hz)
// 5=880Hz, 4=784Hz, 3=698Hz, 2=622Hz, 1=554Hz (descending A5 to C#5)
var beepFrequencies = map[int]int{
	5: 880,
	4: 784,
	3: 698,
	2: 622,
	1: 554,
}

// CountdownModel represents the countdown screen state
type CountdownModel struct {
	width     int
	height    int
	count     int
	done      bool
	cancelled bool
}

// countdownDoneMsg signals countdown is complete
type countdownDoneMsg struct{}

// NewCountdownModel creates a new countdown model
func NewCountdownModel() *CountdownModel {
	return &CountdownModel{
		count: 5,
	}
}

// Init initializes the countdown
func (c *CountdownModel) Init() tea.Cmd {
	// Play initial beep and start countdown
	go playBeep(c.count)
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return countdownTickMsg{}
	})
}

// Update handles messages for the countdown
func (c *CountdownModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		c.width = msg.Width
		c.height = msg.Height
		return c, nil

	case tea.KeyMsg:
		// Allow cancelling countdown with Escape or q
		if msg.String() == "esc" || msg.String() == "q" {
			c.cancelled = true
			c.done = true
			return c, tea.Quit
		}
		return c, nil

	case countdownTickMsg:
		c.count--

		if c.count < 0 {
			c.done = true
			return c, tea.Quit
		}

		// Play beep for counts 5-1 (not for 0/GO)
		if c.count > 0 {
			go playBeep(c.count)
		}

		return c, tea.Tick(time.Second, func(t time.Time) tea.Msg {
			return countdownTickMsg{}
		})
	}

	return c, nil
}

// View renders the countdown display
func (c *CountdownModel) View() string {
	if c.width == 0 || c.height == 0 {
		return ""
	}

	var bigText []string
	var color lipgloss.Color

	if c.count > 0 {
		// Show digit
		digit := rune('0' + c.count)
		bigText = bigDigits[digit]
		// Color changes as countdown progresses (orange -> red)
		switch c.count {
		case 5, 4:
			color = ColorOrange
		case 3, 2:
			color = lipgloss.Color("#FF8C00") // Dark orange
		case 1:
			color = ColorRed
		}
	} else {
		// Show GO!
		bigText = bigGO
		color = ColorGreen
	}

	// Style the big text
	digitStyle := lipgloss.NewStyle().
		Foreground(color).
		Bold(true)

	// Build the display
	var lines []string
	for _, line := range bigText {
		lines = append(lines, digitStyle.Render(line))
	}

	bigDisplay := strings.Join(lines, "\n")

	// Add subtitle
	subtitleStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Italic(true)

	var subtitle string
	if c.count > 0 {
		subtitle = subtitleStyle.Render("Get ready... Recording starts soon!")
	} else {
		subtitle = subtitleStyle.Render("Recording!")
	}

	// Add cancel hint
	hintStyle := lipgloss.NewStyle().
		Foreground(ColorGray)
	hint := hintStyle.Render("Press ESC to cancel")

	// Combine content
	content := lipgloss.JoinVertical(
		lipgloss.Center,
		"",
		bigDisplay,
		"",
		subtitle,
		"",
		hint,
	)

	// Center on screen
	return lipgloss.Place(
		c.width,
		c.height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

// IsDone returns whether the countdown is complete
func (c *CountdownModel) IsDone() bool {
	return c.done
}

// IsCancelled returns whether the countdown was cancelled
func (c *CountdownModel) IsCancelled() bool {
	return c.cancelled
}

// playBeep plays a beep at the specified frequency for the countdown number
func playBeep(count int) {
	freq, ok := beepFrequencies[count]
	if !ok {
		return
	}

	// Try different methods to play a beep

	// Method 1: Use pw-play with a generated tone (PipeWire)
	// Generate a short sine wave using ffmpeg and play it
	if tryFFmpegBeep(freq) {
		return
	}

	// Method 2: Use speaker-test (ALSA)
	if trySpeakerTest(freq) {
		return
	}

	// Method 3: Use paplay with a system sound (PulseAudio)
	if tryPaplay() {
		return
	}

	// Method 4: Console beep (may not work on all systems)
	fmt.Print("\a")
}

// tryFFmpegBeep generates and plays a tone using ffmpeg and pw-play/aplay
func tryFFmpegBeep(freq int) bool {
	// Generate a 100ms sine wave tone and pipe to audio player
	// Using pw-cat (PipeWire) or aplay (ALSA)
	duration := "0.1"

	// Try pw-cat first (PipeWire)
	cmd := exec.Command("bash", "-c",
		fmt.Sprintf("ffmpeg -f lavfi -i 'sine=frequency=%d:duration=%s' -f wav - 2>/dev/null | pw-cat --playback - 2>/dev/null",
			freq, duration))
	if err := cmd.Run(); err == nil {
		return true
	}

	// Try aplay (ALSA)
	cmd = exec.Command("bash", "-c",
		fmt.Sprintf("ffmpeg -f lavfi -i 'sine=frequency=%d:duration=%s' -f wav - 2>/dev/null | aplay -q - 2>/dev/null",
			freq, duration))
	if err := cmd.Run(); err == nil {
		return true
	}

	return false
}

// trySpeakerTest uses speaker-test to generate a tone
func trySpeakerTest(freq int) bool {
	cmd := exec.Command("speaker-test", "-t", "sine", "-f", fmt.Sprintf("%d", freq), "-l", "1", "-p", "1")
	cmd.Stdout = nil
	cmd.Stderr = nil
	err := cmd.Start()
	if err != nil {
		return false
	}

	// Kill after 100ms
	go func() {
		time.Sleep(100 * time.Millisecond)
		cmd.Process.Kill()
	}()

	return true
}

// tryPaplay plays a system sound using paplay
func tryPaplay() bool {
	// Try common system sound locations
	sounds := []string{
		"/usr/share/sounds/freedesktop/stereo/message.oga",
		"/usr/share/sounds/freedesktop/stereo/bell.oga",
		"/usr/share/sounds/sound-icons/bell.wav",
	}

	for _, sound := range sounds {
		cmd := exec.Command("paplay", sound)
		if err := cmd.Run(); err == nil {
			return true
		}
	}

	return false
}

// ShowCountdown displays the countdown and returns true if completed (not cancelled)
func ShowCountdown() (bool, error) {
	countdown := NewCountdownModel()
	p := tea.NewProgram(countdown, tea.WithAltScreen())

	model, err := p.Run()
	if err != nil {
		return false, err
	}

	if m, ok := model.(*CountdownModel); ok {
		return !m.IsCancelled(), nil
	}

	return true, nil
}
