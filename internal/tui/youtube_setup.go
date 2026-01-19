package tui

import (
	"context"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kartoza/kartoza-video-processor/internal/config"
	"github.com/kartoza/kartoza-video-processor/internal/youtube"
)

// YouTubeSetupStep represents the current step in the setup wizard
type YouTubeSetupStep int

const (
	YouTubeStepWelcome YouTubeSetupStep = iota
	YouTubeStepInstructions
	YouTubeStepCredentials
	YouTubeStepAuthenticating
	YouTubeStepConnected
	YouTubeStepError
)

// YouTubeSetupModel handles YouTube account setup
type YouTubeSetupModel struct {
	width  int
	height int

	step         YouTubeSetupStep
	clientID     textinput.Model
	clientSecret textinput.Model
	focusedInput int // 0 = client ID, 1 = client secret

	// Status
	authStatus       youtube.AuthStatus
	channelName      string
	errorMessage     string
	isAuthenticating bool
	authURL          string // URL for manual browser opening

	// Config
	cfg *config.Config
}

// NewYouTubeSetupModel creates a new YouTube setup model
func NewYouTubeSetupModel() *YouTubeSetupModel {
	clientIDInput := textinput.New()
	clientIDInput.Placeholder = "xxxxx.apps.googleusercontent.com"
	clientIDInput.CharLimit = 200
	clientIDInput.Width = 50

	clientSecretInput := textinput.New()
	clientSecretInput.Placeholder = "GOCSPX-xxxxx"
	clientSecretInput.CharLimit = 100
	clientSecretInput.Width = 50
	clientSecretInput.EchoMode = textinput.EchoPassword

	// Load existing config
	cfg, _ := config.Load()

	// Pre-fill if credentials exist
	if cfg.YouTube.ClientID != "" {
		clientIDInput.SetValue(cfg.YouTube.ClientID)
	}
	if cfg.YouTube.ClientSecret != "" {
		clientSecretInput.SetValue(cfg.YouTube.ClientSecret)
	}

	m := &YouTubeSetupModel{
		clientID:     clientIDInput,
		clientSecret: clientSecretInput,
		cfg:          cfg,
		authStatus:   cfg.GetYouTubeAuthStatus(),
	}

	// Start at appropriate step based on current status
	switch m.authStatus {
	case youtube.AuthStatusAuthenticated:
		m.step = YouTubeStepConnected
		m.channelName = cfg.YouTube.ChannelName
	case youtube.AuthStatusConfigured:
		m.step = YouTubeStepCredentials
	default:
		m.step = YouTubeStepWelcome
	}

	return m
}

// Init initializes the YouTube setup model
func (m *YouTubeSetupModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages
func (m *YouTubeSetupModel) Update(msg tea.Msg) (*YouTubeSetupModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case youtubeAuthStartedMsg:
		m.step = YouTubeStepAuthenticating
		m.isAuthenticating = true
		m.authURL = msg.authURL
		// Start waiting for the auth result
		return m, m.waitForAuthResult()

	case youtubeAuthCompleteMsg:
		m.isAuthenticating = false
		if msg.err != nil {
			m.step = YouTubeStepError
			m.errorMessage = msg.err.Error()
		} else {
			m.step = YouTubeStepConnected
			m.channelName = msg.channelName
			// Save channel name to config
			m.cfg.YouTube.ChannelName = msg.channelName
			config.Save(m.cfg)
		}
		return m, nil

	case youtubeDisconnectMsg:
		m.authStatus = youtube.AuthStatusConfigured
		m.channelName = ""
		m.cfg.YouTube.ChannelName = ""
		config.Save(m.cfg)
		m.step = YouTubeStepCredentials
		return m, nil
	}

	// Update text inputs
	if m.step == YouTubeStepCredentials {
		if m.focusedInput == 0 {
			m.clientID, cmd = m.clientID.Update(msg)
		} else {
			m.clientSecret, cmd = m.clientSecret.Update(msg)
		}
	}

	return m, cmd
}

// handleKeyMsg handles keyboard input
func (m *YouTubeSetupModel) handleKeyMsg(msg tea.KeyMsg) (*YouTubeSetupModel, tea.Cmd) {
	// Handle global keys first
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

	case "esc":
		if m.step == YouTubeStepAuthenticating {
			// Can't cancel during authentication
			return m, nil
		}
		return m, func() tea.Msg { return backToMenuMsg{} }
	}

	// Handle step-specific keys
	switch m.step {
	case YouTubeStepCredentials:
		switch msg.String() {
		case "tab", "shift+tab":
			m.focusedInput = (m.focusedInput + 1) % 2
			if m.focusedInput == 0 {
				m.clientID.Focus()
				m.clientSecret.Blur()
			} else {
				m.clientID.Blur()
				m.clientSecret.Focus()
			}
			return m, textinput.Blink

		case "enter":
			return m.handleEnter()

		default:
			// Forward all other keys to the focused text input
			var cmd tea.Cmd
			if m.focusedInput == 0 {
				m.clientID, cmd = m.clientID.Update(msg)
			} else {
				m.clientSecret, cmd = m.clientSecret.Update(msg)
			}
			return m, cmd
		}

	case YouTubeStepWelcome:
		switch msg.String() {
		case "enter", "n":
			m.step = YouTubeStepInstructions
			return m, nil
		}

	case YouTubeStepInstructions:
		switch msg.String() {
		case "enter", "c":
			m.step = YouTubeStepCredentials
			m.focusedInput = 0
			m.clientID.Focus()
			m.clientSecret.Blur()
			return m, textinput.Blink
		}

	case YouTubeStepConnected:
		switch msg.String() {
		case "enter":
			return m, func() tea.Msg { return backToMenuMsg{} }
		case "d":
			return m, m.disconnect()
		}

	case YouTubeStepError:
		switch msg.String() {
		case "enter", "r":
			m.step = YouTubeStepCredentials
			m.errorMessage = ""
			m.focusedInput = 0
			m.clientID.Focus()
			m.clientSecret.Blur()
			return m, textinput.Blink
		}
	}

	return m, nil
}

// handleEnter handles the enter key based on current step
func (m *YouTubeSetupModel) handleEnter() (*YouTubeSetupModel, tea.Cmd) {
	switch m.step {
	case YouTubeStepWelcome:
		m.step = YouTubeStepInstructions
		return m, nil

	case YouTubeStepInstructions:
		m.step = YouTubeStepCredentials
		m.clientID.Focus()
		return m, textinput.Blink

	case YouTubeStepCredentials:
		// Validate and save credentials
		clientID := strings.TrimSpace(m.clientID.Value())
		clientSecret := strings.TrimSpace(m.clientSecret.Value())

		if err := youtube.ValidateCredentials(context.Background(), clientID, clientSecret); err != nil {
			m.errorMessage = err.Error()
			return m, nil
		}

		// Save credentials
		m.cfg.YouTube.ClientID = clientID
		m.cfg.YouTube.ClientSecret = clientSecret
		if err := config.Save(m.cfg); err != nil {
			m.errorMessage = "Failed to save config: " + err.Error()
			return m, nil
		}

		// Start authentication
		return m, m.startAuth()

	case YouTubeStepConnected:
		// Return to menu
		return m, func() tea.Msg { return backToMenuMsg{} }

	case YouTubeStepError:
		m.step = YouTubeStepCredentials
		m.errorMessage = ""
		return m, nil
	}

	return m, nil
}

// authState holds the state for async authentication
type authState struct {
	urlChan    chan string
	resultChan chan tea.Msg
}

var currentAuthState *authState

// startAuth starts the OAuth authentication process
func (m *YouTubeSetupModel) startAuth() tea.Cmd {
	clientID := m.cfg.YouTube.ClientID
	clientSecret := m.cfg.YouTube.ClientSecret
	configDir := config.GetConfigDir()

	// Create channels for communication
	currentAuthState = &authState{
		urlChan:    make(chan string, 1),
		resultChan: make(chan tea.Msg, 1),
	}

	// Start authentication in background goroutine
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		auth := youtube.NewAuth(clientID, clientSecret, configDir)

		// Use the callback to capture and send the URL
		err := auth.AuthenticateWithCallback(ctx, func(url string) {
			// Send the URL through the channel
			select {
			case currentAuthState.urlChan <- url:
			default:
			}
		})

		if err != nil {
			currentAuthState.resultChan <- youtubeAuthCompleteMsg{err: err}
			return
		}

		// Get channel name
		channelName, err := auth.GetChannelName(ctx)
		if err != nil {
			channelName = "Unknown Channel"
		}

		currentAuthState.resultChan <- youtubeAuthCompleteMsg{channelName: channelName}
	}()

	// Return a command that waits for the URL and signals auth started
	return func() tea.Msg {
		// Wait for the URL from the auth flow (with timeout)
		select {
		case url := <-currentAuthState.urlChan:
			return youtubeAuthStartedMsg{authURL: url}
		case <-time.After(5 * time.Second):
			// If we don't get a URL in 5 seconds, start anyway without it
			return youtubeAuthStartedMsg{authURL: ""}
		}
	}
}

// waitForAuthResult returns a command that waits for auth to complete
func (m *YouTubeSetupModel) waitForAuthResult() tea.Cmd {
	return func() tea.Msg {
		if currentAuthState == nil {
			return youtubeAuthCompleteMsg{err: nil}
		}
		// Wait for result with timeout
		select {
		case msg := <-currentAuthState.resultChan:
			return msg
		case <-time.After(5 * time.Minute):
			return youtubeAuthCompleteMsg{err: context.DeadlineExceeded}
		}
	}
}

// disconnect disconnects the YouTube account
func (m *YouTubeSetupModel) disconnect() tea.Cmd {
	configDir := config.GetConfigDir()

	return func() tea.Msg {
		// Delete token
		youtube.DeleteToken(configDir)
		return youtubeDisconnectMsg{}
	}
}

// View renders the YouTube setup screen
func (m *YouTubeSetupModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var content string

	switch m.step {
	case YouTubeStepWelcome:
		content = m.renderWelcome()
	case YouTubeStepInstructions:
		content = m.renderInstructions()
	case YouTubeStepCredentials:
		content = m.renderCredentials()
	case YouTubeStepAuthenticating:
		content = m.renderAuthenticating()
	case YouTubeStepConnected:
		content = m.renderConnected()
	case YouTubeStepError:
		content = m.renderError()
	}

	return content
}

// renderWelcome renders the welcome screen
func (m *YouTubeSetupModel) renderWelcome() string {
	header := RenderHeader("YouTube Integration")

	// YouTube logo in ASCII
	logo := lipgloss.NewStyle().Foreground(ColorRed).Bold(true).Render(`
    ╔══════════════════════════╗
    ║                          ║
    ║      ▶  YouTube          ║
    ║                          ║
    ╚══════════════════════════╝
`)

	titleStyle := lipgloss.NewStyle().
		Foreground(ColorWhite).
		Bold(true).
		Align(lipgloss.Center)

	descStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Align(lipgloss.Center).
		Width(60)

	title := titleStyle.Render("Upload Videos Directly to YouTube")

	desc := descStyle.Render(`
Connect your YouTube account to upload processed recordings
directly from Kartoza Video Processor.

Features:
• Automatic thumbnail generation
• Set title, description, and tags
• Choose privacy settings
• Add to playlists
• Track upload progress
`)

	buttonStyle := lipgloss.NewStyle().
		Padding(1, 4).
		Background(ColorRed).
		Foreground(ColorWhite).
		Bold(true)

	button := buttonStyle.Render("Get Started →")

	helpStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Italic(true)

	helpText := helpStyle.Render("Press Enter to continue • Esc to go back")

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		header,
		logo,
		"",
		title,
		desc,
		"",
		button,
		"",
		helpText,
	)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

// renderInstructions renders the setup instructions
func (m *YouTubeSetupModel) renderInstructions() string {
	header := RenderHeader("YouTube Setup - Instructions")

	instructionStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorOrange).
		Padding(1, 2).
		Width(70)

	stepStyle := lipgloss.NewStyle().
		Foreground(ColorOrange).
		Bold(true)

	textStyle := lipgloss.NewStyle().
		Foreground(ColorWhite)

	linkStyle := lipgloss.NewStyle().
		Foreground(ColorBlue).
		Underline(true)

	instructions := lipgloss.JoinVertical(lipgloss.Left,
		stepStyle.Render("Step 1: Go to Google Cloud Console"),
		textStyle.Render("  Visit: ")+linkStyle.Render("https://console.cloud.google.com/"),
		"",
		stepStyle.Render("Step 2: Create or select a project"),
		textStyle.Render("  Create a new project or use an existing one"),
		"",
		stepStyle.Render("Step 3: Enable YouTube Data API v3"),
		textStyle.Render("  Go to APIs & Services → Library"),
		textStyle.Render("  Search for \"YouTube Data API v3\" and enable it"),
		"",
		stepStyle.Render("Step 4: Create OAuth credentials"),
		textStyle.Render("  Go to APIs & Services → Credentials"),
		textStyle.Render("  Click \"Create Credentials\" → \"OAuth client ID\""),
		textStyle.Render("  Select \"Desktop app\" as the application type"),
		"",
		stepStyle.Render("Step 5: Configure OAuth consent screen"),
		textStyle.Render("  Go to OAuth consent screen"),
		textStyle.Render("  Set user type to \"External\""),
		textStyle.Render("  Add your email as a test user"),
		"",
		stepStyle.Render("Step 6: Copy your credentials"),
		textStyle.Render("  Copy the Client ID and Client Secret"),
	)

	content := instructionStyle.Render(instructions)

	noteStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Italic(true).
		Width(70).
		Align(lipgloss.Center)

	note := noteStyle.Render("Note: While in testing mode, only users added as test users can authenticate.")

	helpStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Italic(true)

	helpText := helpStyle.Render("Press C to continue to credentials • Esc to go back")

	fullContent := lipgloss.JoinVertical(
		lipgloss.Center,
		header,
		"",
		content,
		"",
		note,
		"",
		helpText,
	)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, fullContent)
}

// renderCredentials renders the credentials input screen
func (m *YouTubeSetupModel) renderCredentials() string {
	header := RenderHeader("YouTube Setup - Credentials")

	titleStyle := lipgloss.NewStyle().
		Foreground(ColorOrange).
		Bold(true)

	labelStyle := lipgloss.NewStyle().
		Foreground(ColorGray)

	focusedLabelStyle := lipgloss.NewStyle().
		Foreground(ColorOrange).
		Bold(true)

	hintStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Italic(true)

	var rows []string

	rows = append(rows, titleStyle.Render("Enter your Google OAuth credentials:"))
	rows = append(rows, "")

	// Client ID
	if m.focusedInput == 0 {
		rows = append(rows, focusedLabelStyle.Render("▶ Client ID:"))
	} else {
		rows = append(rows, labelStyle.Render("  Client ID:"))
	}
	rows = append(rows, "  "+m.clientID.View())
	rows = append(rows, hintStyle.Render("  (ends with .apps.googleusercontent.com)"))
	rows = append(rows, "")

	// Client Secret
	if m.focusedInput == 1 {
		rows = append(rows, focusedLabelStyle.Render("▶ Client Secret:"))
	} else {
		rows = append(rows, labelStyle.Render("  Client Secret:"))
	}
	rows = append(rows, "  "+m.clientSecret.View())
	rows = append(rows, hintStyle.Render("  (starts with GOCSPX-)"))

	// Error message
	if m.errorMessage != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(ColorRed).
			Bold(true)
		rows = append(rows, "")
		rows = append(rows, errorStyle.Render("Error: "+m.errorMessage))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, rows...)

	helpStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Italic(true)

	helpText := helpStyle.Render("Tab: switch field • Enter: connect • Esc: cancel")

	footer := RenderHelpFooter(helpText, m.width)

	return LayoutWithHeaderFooter(header, content, footer, m.width, m.height)
}

// renderAuthenticating renders the authenticating screen
func (m *YouTubeSetupModel) renderAuthenticating() string {
	header := RenderHeader("YouTube Setup - Authenticating")

	spinnerFrames := []string{"◐", "◓", "◑", "◒"}
	frame := spinnerFrames[int(time.Now().UnixMilli()/200)%len(spinnerFrames)]

	spinnerStyle := lipgloss.NewStyle().
		Foreground(ColorOrange).
		Bold(true)

	messageStyle := lipgloss.NewStyle().
		Foreground(ColorWhite).
		Bold(true)

	subMessageStyle := lipgloss.NewStyle().
		Foreground(ColorGray)

	linkStyle := lipgloss.NewStyle().
		Foreground(ColorBlue)

	labelStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Bold(true)

	spinner := spinnerStyle.Render(frame)
	message := messageStyle.Render("Waiting for authorization...")
	subMessage := subMessageStyle.Render("A browser window should have opened.\nPlease sign in to your Google account and grant access.")

	var rows []string
	rows = append(rows, spinner+" "+message)
	rows = append(rows, "")
	rows = append(rows, subMessage)
	rows = append(rows, "")

	// Show the auth URL for manual opening
	if m.authURL != "" {
		rows = append(rows, labelStyle.Render("If browser didn't open, visit this URL:"))
		rows = append(rows, "")
		// Wrap the URL if it's too long
		url := m.authURL
		if len(url) > 70 {
			// Show truncated URL with indication it's truncated
			rows = append(rows, linkStyle.Render(url[:70]+"..."))
		} else {
			rows = append(rows, linkStyle.Render(url))
		}
		rows = append(rows, "")
	}

	rows = append(rows, subMessageStyle.Render("This may take a moment..."))

	content := lipgloss.JoinVertical(lipgloss.Center, rows...)

	helpText := "Waiting for browser authentication..."
	footer := RenderHelpFooter(helpText, m.width)

	return LayoutWithHeaderFooter(header, content, footer, m.width, m.height)
}

// renderConnected renders the connected screen
func (m *YouTubeSetupModel) renderConnected() string {
	header := RenderHeader("YouTube Connected")

	checkStyle := lipgloss.NewStyle().
		Foreground(ColorGreen).
		Bold(true)

	check := checkStyle.Render("✓")

	titleStyle := lipgloss.NewStyle().
		Foreground(ColorGreen).
		Bold(true)

	title := titleStyle.Render("Successfully Connected!")

	containerStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorGreen).
		Padding(1, 3).
		Width(50)

	labelStyle := lipgloss.NewStyle().
		Foreground(ColorGray)

	valueStyle := lipgloss.NewStyle().
		Foreground(ColorWhite).
		Bold(true)

	channelName := m.channelName
	if channelName == "" {
		channelName = "Connected"
	}

	info := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Top,
			labelStyle.Render("Channel: "),
			valueStyle.Render(channelName),
		),
		"",
		lipgloss.JoinHorizontal(lipgloss.Top,
			labelStyle.Render("Status: "),
			checkStyle.Render("Ready to upload"),
		),
	)

	content := containerStyle.Render(info)

	buttonStyle := lipgloss.NewStyle().
		Padding(0, 3).
		Background(ColorGreen).
		Foreground(ColorWhite).
		Bold(true)

	button := buttonStyle.Render("Done")

	disconnectStyle := lipgloss.NewStyle().
		Foreground(ColorGray)

	disconnectText := disconnectStyle.Render("Press D to disconnect account")

	helpStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Italic(true)

	helpText := helpStyle.Render("Enter: Return to menu • D: Disconnect • Esc: Back")

	fullContent := lipgloss.JoinVertical(
		lipgloss.Center,
		header,
		"",
		check + " " + title,
		"",
		content,
		"",
		button,
		"",
		disconnectText,
		"",
		helpText,
	)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, fullContent)
}

// renderError renders the error screen
func (m *YouTubeSetupModel) renderError() string {
	header := RenderHeader("YouTube Setup - Error")

	errorStyle := lipgloss.NewStyle().
		Foreground(ColorRed).
		Bold(true)

	errorIcon := errorStyle.Render("✗")

	titleStyle := lipgloss.NewStyle().
		Foreground(ColorRed).
		Bold(true)

	title := titleStyle.Render("Authentication Failed")

	containerStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorRed).
		Padding(1, 3).
		Width(60)

	messageStyle := lipgloss.NewStyle().
		Foreground(ColorWhite).
		Width(52)

	content := containerStyle.Render(messageStyle.Render(m.errorMessage))

	buttonStyle := lipgloss.NewStyle().
		Padding(0, 3).
		Background(ColorOrange).
		Foreground(ColorWhite).
		Bold(true)

	button := buttonStyle.Render("Try Again")

	helpStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Italic(true)

	helpText := helpStyle.Render("Press R or Enter to retry • Esc to go back")

	fullContent := lipgloss.JoinVertical(
		lipgloss.Center,
		header,
		"",
		errorIcon + " " + title,
		"",
		content,
		"",
		button,
		"",
		helpText,
	)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, fullContent)
}

// Message types for YouTube setup
type youtubeAuthStartedMsg struct {
	authURL string
}
type youtubeAuthCompleteMsg struct {
	err         error
	channelName string
}
type youtubeDisconnectMsg struct{}
