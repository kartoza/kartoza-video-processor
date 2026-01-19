package tui

import (
	"context"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kartoza/kartoza-video-processor/internal/config"
	"github.com/kartoza/kartoza-video-processor/internal/youtube"
)

// YouTubeUploadStep represents the current step in the upload process
type YouTubeUploadStep int

const (
	YouTubeUploadStepPrompt YouTubeUploadStep = iota
	YouTubeUploadStepMetadata
	YouTubeUploadStepThumbnail
	YouTubeUploadStepUploading
	YouTubeUploadStepComplete
	YouTubeUploadStepError
	YouTubeUploadStepSkipped
)

// YouTubeUploadField represents which field is focused
type YouTubeUploadField int

const (
	YouTubeUploadFieldTitle YouTubeUploadField = iota
	YouTubeUploadFieldDescription
	YouTubeUploadFieldTags
	YouTubeUploadFieldPrivacy
	YouTubeUploadFieldUpload
	YouTubeUploadFieldCancel
)

// YouTubeUploadModel handles YouTube upload UI
type YouTubeUploadModel struct {
	width  int
	height int

	step         YouTubeUploadStep
	focusedField YouTubeUploadField

	// Video info
	videoPath   string
	outputDir   string
	title       string
	description string
	topic       string

	// Editable fields
	titleInput       textinput.Model
	descriptionInput textinput.Model
	tagsInput        textinput.Model

	// Privacy selection
	privacyOptions []youtube.PrivacyStatus
	selectedPrivacy int

	// Upload progress
	progress     progress.Model
	uploadPct    float64
	isUploading  bool
	uploadResult *youtube.UploadResult

	// Status
	errorMessage string
	statusMessage string

	// Config
	cfg *config.Config
}

// NewYouTubeUploadModel creates a new YouTube upload model
func NewYouTubeUploadModel(videoPath, outputDir, title, description, topic string) *YouTubeUploadModel {
	titleInput := textinput.New()
	titleInput.Placeholder = "Video title"
	titleInput.CharLimit = 100
	titleInput.Width = 50
	titleInput.SetValue(title)
	titleInput.Focus()

	descInput := textinput.New()
	descInput.Placeholder = "Video description"
	descInput.CharLimit = 5000
	descInput.Width = 50
	descInput.SetValue(description)

	tagsInput := textinput.New()
	tagsInput.Placeholder = "Tags (comma separated)"
	tagsInput.CharLimit = 500
	tagsInput.Width = 50
	if topic != "" {
		tagsInput.SetValue(topic)
	}

	cfg, _ := config.Load()

	prog := progress.New(progress.WithDefaultGradient())

	return &YouTubeUploadModel{
		step:             YouTubeUploadStepPrompt,
		focusedField:     YouTubeUploadFieldTitle,
		videoPath:        videoPath,
		outputDir:        outputDir,
		title:            title,
		description:      description,
		topic:            topic,
		titleInput:       titleInput,
		descriptionInput: descInput,
		tagsInput:        tagsInput,
		privacyOptions:   []youtube.PrivacyStatus{youtube.PrivacyUnlisted, youtube.PrivacyPrivate, youtube.PrivacyPublic},
		selectedPrivacy:  0, // Unlisted by default
		progress:         prog,
		cfg:              cfg,
	}
}

// Init initializes the upload model
func (m *YouTubeUploadModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages
func (m *YouTubeUploadModel) Update(msg tea.Msg) (*YouTubeUploadModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.progress.Width = m.width - 20

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case uploadProgressMsg:
		m.uploadPct = msg.percent
		return m, nil

	case uploadCompleteMsg:
		m.isUploading = false
		if msg.err != nil {
			m.step = YouTubeUploadStepError
			m.errorMessage = msg.err.Error()
		} else {
			m.step = YouTubeUploadStepComplete
			m.uploadResult = msg.result
		}
		// Refresh YouTube status
		updateGlobalAppState(GlobalAppState.IsRecording, GlobalAppState.BlinkOn, GlobalAppState.Status)
		return m, nil
	}

	// Update focused input
	switch m.focusedField {
	case YouTubeUploadFieldTitle:
		m.titleInput, cmd = m.titleInput.Update(msg)
	case YouTubeUploadFieldDescription:
		m.descriptionInput, cmd = m.descriptionInput.Update(msg)
	case YouTubeUploadFieldTags:
		m.tagsInput, cmd = m.tagsInput.Update(msg)
	}

	return m, cmd
}

// handleKeyMsg handles keyboard input
func (m *YouTubeUploadModel) handleKeyMsg(msg tea.KeyMsg) (*YouTubeUploadModel, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

	case "esc":
		if m.step == YouTubeUploadStepUploading {
			// Can't cancel during upload
			return m, nil
		}
		if m.step == YouTubeUploadStepPrompt {
			m.step = YouTubeUploadStepSkipped
			return m, func() tea.Msg { return youtubeUploadSkippedMsg{} }
		}
		// Go back to prompt
		m.step = YouTubeUploadStepPrompt
		return m, nil

	case "y", "Y":
		if m.step == YouTubeUploadStepPrompt {
			// Check if YouTube is connected
			if !m.cfg.IsYouTubeConnected() {
				m.errorMessage = "YouTube not connected. Go to Options > YouTube to set up."
				return m, nil
			}
			m.step = YouTubeUploadStepMetadata
			m.titleInput.Focus()
			return m, textinput.Blink
		}

	case "n", "N":
		if m.step == YouTubeUploadStepPrompt {
			m.step = YouTubeUploadStepSkipped
			return m, func() tea.Msg { return youtubeUploadSkippedMsg{} }
		}

	case "tab", "down":
		if m.step == YouTubeUploadStepMetadata {
			m.nextField()
			return m, textinput.Blink
		}

	case "shift+tab", "up":
		if m.step == YouTubeUploadStepMetadata {
			m.prevField()
			return m, textinput.Blink
		}

	case "left", "right":
		if m.step == YouTubeUploadStepMetadata && m.focusedField == YouTubeUploadFieldPrivacy {
			if msg.String() == "left" {
				m.selectedPrivacy--
				if m.selectedPrivacy < 0 {
					m.selectedPrivacy = len(m.privacyOptions) - 1
				}
			} else {
				m.selectedPrivacy++
				if m.selectedPrivacy >= len(m.privacyOptions) {
					m.selectedPrivacy = 0
				}
			}
			return m, nil
		}

	case "enter":
		return m.handleEnter()
	}

	return m, nil
}

// handleEnter handles the enter key
func (m *YouTubeUploadModel) handleEnter() (*YouTubeUploadModel, tea.Cmd) {
	switch m.step {
	case YouTubeUploadStepPrompt:
		// Same as pressing 'y'
		if !m.cfg.IsYouTubeConnected() {
			m.errorMessage = "YouTube not connected. Go to Options > YouTube to set up."
			return m, nil
		}
		m.step = YouTubeUploadStepMetadata
		m.titleInput.Focus()
		return m, textinput.Blink

	case YouTubeUploadStepMetadata:
		switch m.focusedField {
		case YouTubeUploadFieldUpload:
			// Validate and start upload
			if m.titleInput.Value() == "" {
				m.errorMessage = "Title is required"
				return m, nil
			}
			return m, m.startUpload()
		case YouTubeUploadFieldCancel:
			m.step = YouTubeUploadStepPrompt
			return m, nil
		default:
			m.nextField()
			return m, textinput.Blink
		}

	case YouTubeUploadStepComplete, YouTubeUploadStepError:
		return m, func() tea.Msg { return youtubeUploadDoneMsg{} }
	}

	return m, nil
}

// nextField moves to the next field
func (m *YouTubeUploadModel) nextField() {
	m.unfocusAll()
	m.focusedField++
	if m.focusedField > YouTubeUploadFieldCancel {
		m.focusedField = YouTubeUploadFieldTitle
	}
	m.focusCurrent()
}

// prevField moves to the previous field
func (m *YouTubeUploadModel) prevField() {
	m.unfocusAll()
	m.focusedField--
	if m.focusedField < YouTubeUploadFieldTitle {
		m.focusedField = YouTubeUploadFieldCancel
	}
	m.focusCurrent()
}

// unfocusAll removes focus from all inputs
func (m *YouTubeUploadModel) unfocusAll() {
	m.titleInput.Blur()
	m.descriptionInput.Blur()
	m.tagsInput.Blur()
}

// focusCurrent focuses the current field
func (m *YouTubeUploadModel) focusCurrent() {
	switch m.focusedField {
	case YouTubeUploadFieldTitle:
		m.titleInput.Focus()
	case YouTubeUploadFieldDescription:
		m.descriptionInput.Focus()
	case YouTubeUploadFieldTags:
		m.tagsInput.Focus()
	}
}

// startUpload begins the YouTube upload
func (m *YouTubeUploadModel) startUpload() tea.Cmd {
	m.step = YouTubeUploadStepUploading
	m.isUploading = true
	m.uploadPct = 0
	m.errorMessage = ""

	return func() tea.Msg {
		ctx := context.Background()

		// Create auth
		auth := youtube.NewAuth(m.cfg.YouTube.ClientID, m.cfg.YouTube.ClientSecret, config.GetConfigDir())

		// Create uploader
		uploader, err := youtube.NewUploader(ctx, auth)
		if err != nil {
			return uploadCompleteMsg{err: err}
		}

		// Parse tags
		tags := youtube.ParseTags(m.tagsInput.Value())

		// Build upload options
		opts := youtube.BuildUploadOptions(
			m.videoPath,
			m.titleInput.Value(),
			m.descriptionInput.Value(),
			m.topic,
			tags,
			m.privacyOptions[m.selectedPrivacy],
		)

		// First extract thumbnail if it doesn't exist
		thumbnailPath := youtube.GetThumbnailPath(m.videoPath)
		if err := youtube.ExtractThumbnailForYouTube(m.videoPath, thumbnailPath); err == nil {
			opts.ThumbnailPath = thumbnailPath
		}

		// Create a progress channel
		progressChan := make(chan float64, 100)
		go func() {
			for pct := range progressChan {
				// Send progress to the TUI via program.Send
				// Since we can't access the program directly, we'll use a different approach
				_ = pct
			}
		}()

		// Upload with progress callback
		result, err := uploader.Upload(ctx, opts, func(read, total int64) {
			if total > 0 {
				pct := float64(read) / float64(total)
				// We can't directly send messages here, so we'll poll in a goroutine
				progressChan <- pct
			}
		})
		close(progressChan)

		if err != nil {
			return uploadCompleteMsg{err: err}
		}

		return uploadCompleteMsg{result: result}
	}
}

// View renders the upload UI
func (m *YouTubeUploadModel) View() string {
	header := RenderHeader("YouTube Upload")

	var content string
	switch m.step {
	case YouTubeUploadStepPrompt:
		content = m.renderPrompt()
	case YouTubeUploadStepMetadata:
		content = m.renderMetadata()
	case YouTubeUploadStepUploading:
		content = m.renderUploading()
	case YouTubeUploadStepComplete:
		content = m.renderComplete()
	case YouTubeUploadStepError:
		content = m.renderError()
	case YouTubeUploadStepSkipped:
		content = m.renderSkipped()
	}

	helpText := m.getHelpText()
	footer := RenderHelpFooter(helpText, m.width)

	return LayoutWithHeaderFooter(header, content, footer, m.width, m.height)
}

// renderPrompt renders the initial prompt
func (m *YouTubeUploadModel) renderPrompt() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorOrange)

	textStyle := lipgloss.NewStyle().
		Foreground(ColorWhite)

	videoName := filepath.Base(m.videoPath)

	var errorLine string
	if m.errorMessage != "" {
		errorLine = lipgloss.NewStyle().
			Foreground(ColorRed).
			Render(m.errorMessage)
	}

	content := lipgloss.JoinVertical(lipgloss.Center,
		titleStyle.Render("Upload to YouTube?"),
		"",
		textStyle.Render("Video: "+videoName),
		textStyle.Render("Title: "+m.title),
		"",
		lipgloss.NewStyle().Foreground(ColorGray).Render("Press Y to upload, N to skip"),
		"",
		errorLine,
	)

	return content
}

// renderMetadata renders the metadata editing form
func (m *YouTubeUploadModel) renderMetadata() string {
	labelStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Width(15).
		Align(lipgloss.Right)

	labelActiveStyle := lipgloss.NewStyle().
		Foreground(ColorOrange).
		Bold(true).
		Width(15).
		Align(lipgloss.Right)

	buttonStyle := lipgloss.NewStyle().
		Padding(0, 2).
		Bold(true)

	activeButtonStyle := buttonStyle.
		Background(ColorOrange).
		Foreground(lipgloss.Color("#000000"))

	inactiveButtonStyle := buttonStyle.
		Background(ColorGray).
		Foreground(ColorWhite)

	// Title row
	titleLabel := labelStyle.Render("Title: ")
	if m.focusedField == YouTubeUploadFieldTitle {
		titleLabel = labelActiveStyle.Render("Title: ")
	}
	titleRow := lipgloss.JoinHorizontal(lipgloss.Center, titleLabel, m.titleInput.View())

	// Description row
	descLabel := labelStyle.Render("Description: ")
	if m.focusedField == YouTubeUploadFieldDescription {
		descLabel = labelActiveStyle.Render("Description: ")
	}
	descRow := lipgloss.JoinHorizontal(lipgloss.Center, descLabel, m.descriptionInput.View())

	// Tags row
	tagsLabel := labelStyle.Render("Tags: ")
	if m.focusedField == YouTubeUploadFieldTags {
		tagsLabel = labelActiveStyle.Render("Tags: ")
	}
	tagsRow := lipgloss.JoinHorizontal(lipgloss.Center, tagsLabel, m.tagsInput.View())

	// Privacy row
	privacyLabel := labelStyle.Render("Privacy: ")
	if m.focusedField == YouTubeUploadFieldPrivacy {
		privacyLabel = labelActiveStyle.Render("Privacy: ")
	}
	var privacyOptions []string
	for i, opt := range m.privacyOptions {
		style := lipgloss.NewStyle().Foreground(ColorGray)
		if i == m.selectedPrivacy {
			if m.focusedField == YouTubeUploadFieldPrivacy {
				style = lipgloss.NewStyle().Background(ColorOrange).Foreground(lipgloss.Color("#000000"))
			} else {
				style = lipgloss.NewStyle().Foreground(ColorWhite).Bold(true)
			}
		}
		privacyOptions = append(privacyOptions, style.Render(" "+string(opt)+" "))
	}
	privacyValue := lipgloss.JoinHorizontal(lipgloss.Center, privacyOptions...)
	privacyRow := lipgloss.JoinHorizontal(lipgloss.Center, privacyLabel, privacyValue)

	// Buttons
	uploadBtn := inactiveButtonStyle.Render("Upload")
	if m.focusedField == YouTubeUploadFieldUpload {
		uploadBtn = activeButtonStyle.Render("Upload")
	}
	cancelBtn := inactiveButtonStyle.Render("Cancel")
	if m.focusedField == YouTubeUploadFieldCancel {
		cancelBtn = activeButtonStyle.Render("Cancel")
	}
	buttonRow := lipgloss.JoinHorizontal(lipgloss.Center, uploadBtn, "  ", cancelBtn)

	var errorLine string
	if m.errorMessage != "" {
		errorLine = lipgloss.NewStyle().
			Foreground(ColorRed).
			Render(m.errorMessage)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		titleRow,
		descRow,
		tagsRow,
		privacyRow,
		"",
		buttonRow,
		"",
		errorLine,
	)
}

// renderUploading renders the upload progress
func (m *YouTubeUploadModel) renderUploading() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorOrange)

	spinnerFrames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	frame := spinnerFrames[int(time.Now().UnixMilli()/100)%len(spinnerFrames)]

	pctText := lipgloss.NewStyle().
		Foreground(ColorWhite).
		Render(frame + " Uploading to YouTube...")

	return lipgloss.JoinVertical(lipgloss.Center,
		titleStyle.Render("Uploading"),
		"",
		m.progress.ViewAs(m.uploadPct),
		"",
		pctText,
	)
}

// renderComplete renders the success message
func (m *YouTubeUploadModel) renderComplete() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorGreen)

	textStyle := lipgloss.NewStyle().
		Foreground(ColorWhite)

	linkStyle := lipgloss.NewStyle().
		Foreground(ColorBlue).
		Underline(true)

	var url string
	if m.uploadResult != nil {
		url = m.uploadResult.VideoURL
	}

	return lipgloss.JoinVertical(lipgloss.Center,
		titleStyle.Render("Upload Complete!"),
		"",
		textStyle.Render("Your video has been uploaded to YouTube."),
		"",
		linkStyle.Render(url),
		"",
		lipgloss.NewStyle().Foreground(ColorGray).Render("Press Enter to continue"),
	)
}

// renderError renders the error message
func (m *YouTubeUploadModel) renderError() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorRed)

	return lipgloss.JoinVertical(lipgloss.Center,
		titleStyle.Render("Upload Failed"),
		"",
		lipgloss.NewStyle().Foreground(ColorWhite).Render(m.errorMessage),
		"",
		lipgloss.NewStyle().Foreground(ColorGray).Render("Press Enter to continue, or R to retry"),
	)
}

// renderSkipped renders the skipped message
func (m *YouTubeUploadModel) renderSkipped() string {
	return lipgloss.JoinVertical(lipgloss.Center,
		lipgloss.NewStyle().Foreground(ColorGray).Render("Upload skipped"),
	)
}

// getHelpText returns the help text for the current step
func (m *YouTubeUploadModel) getHelpText() string {
	switch m.step {
	case YouTubeUploadStepPrompt:
		return "y: upload • n: skip • esc: skip"
	case YouTubeUploadStepMetadata:
		return "tab: next field • enter: select • ←/→: change privacy • esc: back"
	case YouTubeUploadStepUploading:
		return "uploading..."
	case YouTubeUploadStepComplete:
		return "enter: continue"
	case YouTubeUploadStepError:
		return "enter: continue • r: retry"
	default:
		return ""
	}
}

// Messages for YouTube upload

type uploadProgressMsg struct {
	percent float64
}

type uploadCompleteMsg struct {
	result *youtube.UploadResult
	err    error
}

type youtubeUploadSkippedMsg struct{}

type youtubeUploadDoneMsg struct{}
