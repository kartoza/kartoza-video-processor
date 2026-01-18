package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// ========================================
// Brand Colors - Kartoza standard palette
// ========================================

var (
	ColorOrange   = lipgloss.Color("#DDA036") // Primary/Active
	ColorBlue     = lipgloss.Color("#569FC6") // Secondary/Links
	ColorGray     = lipgloss.Color("#9A9EA0") // Inactive/Subtle
	ColorWhite    = lipgloss.Color("#FFFFFF") // Text
	ColorDarkGray = lipgloss.Color("#3A3A3A") // Background
	ColorRed      = lipgloss.Color("#E95420") // Error/Recording
	ColorGreen    = lipgloss.Color("#4CAF50") // Success
)

// HeaderWidth is the standard width for the header
const HeaderWidth = 60

// ========================================
// Header State for dynamic updates
// ========================================

// HeaderState contains the dynamic state for the header
type HeaderState struct {
	IsRecording   bool
	Monitor       string
	Duration      string
	BlinkOn       bool // For blinking status indicator
}

// ========================================
// Header Rendering
// ========================================

// RenderHeader renders the standard application header
// screenTitle should be the name of the current screen (e.g., "Main", "Recording")
func RenderHeader(screenTitle string, state *HeaderState) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorOrange).
		Align(lipgloss.Center).
		Width(HeaderWidth)

	mottoStyle := lipgloss.NewStyle().
		Italic(true).
		Foreground(ColorGray).
		Align(lipgloss.Center).
		Width(HeaderWidth)

	dividerStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Width(HeaderWidth)

	statusStyle := lipgloss.NewStyle().
		Foreground(ColorWhite).
		Align(lipgloss.Center).
		Width(HeaderWidth)

	title := titleStyle.Render("Kartoza Video Processor - " + screenTitle)
	motto := mottoStyle.Render("capture your screen")
	divider := dividerStyle.Render("────────────────────────────────────────────────────────────")

	// Build status line if state is provided
	var status string
	if state != nil {
		recorderState := "Ready"
		stateColor := ColorGray
		if state.IsRecording {
			// Blink the dot when recording is active
			if state.BlinkOn {
				recorderState = "● REC"
			} else {
				recorderState = "○ REC"
			}
			stateColor = ColorRed
		}

		recorderStateStyled := lipgloss.NewStyle().
			Foreground(stateColor).
			Bold(true).
			Render(recorderState)

		monitorInfo := state.Monitor
		if monitorInfo == "" {
			monitorInfo = "Auto"
		}

		durationInfo := state.Duration
		if durationInfo == "" {
			durationInfo = "00:00:00"
		}

		statusLine := fmt.Sprintf("Status: %s  |  Monitor: %s  |  Duration: %s",
			recorderStateStyled,
			monitorInfo,
			durationInfo,
		)
		status = statusStyle.Render(statusLine)
	}

	if state != nil {
		return lipgloss.JoinVertical(
			lipgloss.Center,
			title,
			motto,
			divider,
			status,
			divider,
		)
	}

	return lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		motto,
		divider,
	)
}

// RenderSimpleHeader renders a header without the full status bar
func RenderSimpleHeader(screenTitle string) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorOrange).
		Align(lipgloss.Center).
		Width(HeaderWidth)

	mottoStyle := lipgloss.NewStyle().
		Italic(true).
		Foreground(ColorGray).
		Align(lipgloss.Center).
		Width(HeaderWidth)

	dividerStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Width(HeaderWidth)

	title := titleStyle.Render("Kartoza Video Processor - " + screenTitle)
	motto := mottoStyle.Render("capture your screen")
	divider := dividerStyle.Render("────────────────────────────────────────────────────────────")

	return lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		motto,
		divider,
	)
}

// ========================================
// Footer Rendering
// ========================================

// RenderHelpFooter renders the standard help footer at the bottom of the screen
func RenderHelpFooter(helpText string, width int) string {
	helpStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Italic(true)

	footerStyle := lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Center)

	return footerStyle.Render(helpStyle.Render(helpText))
}

// ========================================
// Layout Helpers
// ========================================

// LayoutWithHeaderFooter creates a standard layout with header at top and footer at bottom
func LayoutWithHeaderFooter(header, content, footer string, width, height int) string {
	// Main section with header and content
	mainSection := lipgloss.JoinVertical(
		lipgloss.Center,
		header,
		"",
		content,
	)

	// Center main content at top (leave room for footer)
	centeredMain := lipgloss.Place(
		width,
		height-2,
		lipgloss.Center,
		lipgloss.Top,
		mainSection,
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		centeredMain,
		footer,
	)
}

// CenterContent centers content both horizontally and vertically
func CenterContent(content string, width, height int) string {
	return lipgloss.Place(
		width,
		height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

// ========================================
// Common Styles
// ========================================

// Box style for content areas
var BoxStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(ColorOrange).
	Padding(1, 2)

// Title style for section headings
var TitleStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(ColorOrange)

// Subtitle style
var SubtitleStyle = lipgloss.NewStyle().
	Foreground(ColorBlue)

// Label style for form labels
var LabelStyle = lipgloss.NewStyle().
	Foreground(ColorGray)

// Value style for displaying values
var ValueStyle = lipgloss.NewStyle().
	Foreground(ColorWhite)

// Active style for active/selected items
var ActiveStyle = lipgloss.NewStyle().
	Foreground(ColorOrange).
	Bold(true)

// Inactive style for inactive items
var InactiveStyle = lipgloss.NewStyle().
	Foreground(ColorGray)

// Error style for error messages
var ErrorStyle = lipgloss.NewStyle().
	Foreground(ColorRed).
	Bold(true)

// Success style for success messages
var SuccessStyle = lipgloss.NewStyle().
	Foreground(ColorGreen).
	Bold(true)

// Recording style for recording indicator (blinking red)
var RecordingStyle = lipgloss.NewStyle().
	Foreground(ColorRed).
	Bold(true).
	Blink(true)
