package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kartoza/kartoza-screencaster/internal/config"
	"github.com/kartoza/kartoza-screencaster/internal/models"
	"github.com/kartoza/kartoza-screencaster/internal/spellcheck"
	"github.com/kartoza/kartoza-screencaster/internal/youtube"
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
	YouTubeUploadFieldAccount YouTubeUploadField = iota
	YouTubeUploadFieldVideoSource
	YouTubeUploadFieldTitle
	YouTubeUploadFieldDescription
	YouTubeUploadFieldTags
	YouTubeUploadFieldPlaylist
	YouTubeUploadFieldPrivacy
	YouTubeUploadFieldUpload
	YouTubeUploadFieldCancel
)

// VideoSourceOption represents which video file to upload
type VideoSourceOption int

const (
	VideoSourceVertical VideoSourceOption = iota
	VideoSourceMerged
)

func (v VideoSourceOption) String() string {
	switch v {
	case VideoSourceVertical:
		return "Vertical (9:16)"
	case VideoSourceMerged:
		return "Landscape (16:9)"
	default:
		return "Unknown"
	}
}

// YouTubeUploadModel handles YouTube upload UI
type YouTubeUploadModel struct {
	width  int
	height int

	step         YouTubeUploadStep
	focusedField YouTubeUploadField

	// Account selection
	accounts        []youtube.Account
	selectedAccount int

	// Video source selection
	videoSourceOptions   []VideoSourceOption
	selectedVideoSource  int
	verticalVideoPath    string
	mergedVideoPath      string
	hasVerticalVideo     bool
	hasMergedVideo       bool

	// Video info
	videoPath     string
	outputDir     string
	title         string
	description   string
	topic         string
	recordingInfo *models.RecordingInfo

	// Editable fields
	titleInput       textinput.Model
	descriptionInput textinput.Model
	tagsInput        textinput.Model

	// Playlist selection
	playlists        []youtube.Playlist
	selectedPlaylist int // -1 means no playlist, 0+ is index into playlists
	loadingPlaylists bool
	playlistError    string

	// Privacy selection
	privacyOptions  []youtube.PrivacyStatus
	selectedPrivacy int

	// Upload progress
	progress         progress.Model
	uploadPct        float64
	isUploading      bool
	uploadResult     *youtube.UploadResult
	uploadProgressCh chan uploadUpdate

	// Status
	errorMessage string

	// Spell checking
	spellChecker   *spellcheck.SpellChecker
	titleIssues    []spellcheck.Issue
	descIssues     []spellcheck.Issue

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

	// Determine default privacy from config
	defaultPrivacyIdx := 0 // Default to unlisted
	switch cfg.YouTube.DefaultPrivacy {
	case youtube.PrivacyPrivate:
		defaultPrivacyIdx = 1
	case youtube.PrivacyPublic:
		defaultPrivacyIdx = 2
	}

	sc := spellcheck.NewSpellChecker()

	// Get available YouTube accounts
	accounts := cfg.YouTube.GetAccounts()
	selectedAccountIdx := 0

	// Find the last used account
	if cfg.YouTube.LastUsedAccountID != "" {
		for i, acc := range accounts {
			if acc.ID == cfg.YouTube.LastUsedAccountID {
				selectedAccountIdx = i
				break
			}
		}
	}

	m := &YouTubeUploadModel{
		step:             YouTubeUploadStepPrompt,
		focusedField:     YouTubeUploadFieldTitle,
		accounts:         accounts,
		selectedAccount:  selectedAccountIdx,
		videoPath:        videoPath,
		outputDir:        outputDir,
		title:            title,
		description:      description,
		topic:            topic,
		titleInput:       titleInput,
		descriptionInput: descInput,
		tagsInput:        tagsInput,
		privacyOptions:   []youtube.PrivacyStatus{youtube.PrivacyUnlisted, youtube.PrivacyPrivate, youtube.PrivacyPublic},
		selectedPrivacy:  defaultPrivacyIdx,
		selectedPlaylist: -1, // No playlist by default
		progress:         prog,
		spellChecker:     sc,
		cfg:              cfg,
	}

	// Initial spell check
	m.updateSpellCheck()

	return m
}

// updateSpellCheck updates the spell check issues for title and description
func (m *YouTubeUploadModel) updateSpellCheck() {
	if m.spellChecker == nil {
		return
	}
	m.titleIssues = m.spellChecker.Check(m.titleInput.Value())
	m.descIssues = m.spellChecker.Check(m.descriptionInput.Value())
}

// NewYouTubeUploadModelWithRecording creates a new YouTube upload model with recording info
func NewYouTubeUploadModelWithRecording(videoPath string, recordingInfo *models.RecordingInfo) *YouTubeUploadModel {
	m := NewYouTubeUploadModel(
		videoPath,
		recordingInfo.Files.FolderPath,
		recordingInfo.Metadata.Title,
		recordingInfo.Metadata.Description,
		recordingInfo.Metadata.Topic,
	)
	m.recordingInfo = recordingInfo

	// Set up video source options based on available files
	m.verticalVideoPath = recordingInfo.Files.VerticalFile
	m.mergedVideoPath = recordingInfo.Files.MergedFile

	// Check which video files actually exist
	if m.verticalVideoPath != "" {
		if _, err := os.Stat(m.verticalVideoPath); err == nil {
			m.hasVerticalVideo = true
		}
	}
	if m.mergedVideoPath != "" {
		if _, err := os.Stat(m.mergedVideoPath); err == nil {
			m.hasMergedVideo = true
		}
	}

	// Build available options list
	m.videoSourceOptions = []VideoSourceOption{}
	if m.hasVerticalVideo {
		m.videoSourceOptions = append(m.videoSourceOptions, VideoSourceVertical)
	}
	if m.hasMergedVideo {
		m.videoSourceOptions = append(m.videoSourceOptions, VideoSourceMerged)
	}

	// Default to vertical if available, otherwise merged
	m.selectedVideoSource = 0
	if m.hasVerticalVideo {
		m.videoPath = m.verticalVideoPath
	} else if m.hasMergedVideo {
		m.videoPath = m.mergedVideoPath
	}

	return m
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

	case playlistsLoadedMsg:
		m.loadingPlaylists = false
		if msg.err != nil {
			m.playlistError = msg.err.Error()
		} else {
			m.playlists = msg.playlists
			// Select default playlist if configured
			if m.cfg.YouTube.DefaultPlaylistID != "" {
				for i, pl := range m.playlists {
					if pl.ID == m.cfg.YouTube.DefaultPlaylistID {
						m.selectedPlaylist = i
						break
					}
				}
			}
		}
		return m, nil

	case uploadProgressMsg:
		m.uploadPct = msg.percent
		// Continue waiting for more progress updates
		return m, waitForUploadProgress(m.uploadProgressCh)

	case uploadCompleteMsg:
		m.isUploading = false
		if msg.err != nil {
			m.step = YouTubeUploadStepError
			m.errorMessage = msg.err.Error()
		} else {
			m.step = YouTubeUploadStepComplete
			m.uploadResult = msg.result

			// Save YouTube metadata to recording
			if m.recordingInfo != nil && msg.result != nil {
				m.saveYouTubeMetadata(msg.result)
			}

			// Update config with last used playlist
			if m.selectedPlaylist >= 0 && m.selectedPlaylist < len(m.playlists) {
				m.cfg.YouTube.DefaultPlaylistID = m.playlists[m.selectedPlaylist].ID
				m.cfg.YouTube.DefaultPlaylistName = m.playlists[m.selectedPlaylist].Title
				_ = config.Save(m.cfg)
			}
		}
		// Refresh YouTube status
		updateGlobalAppState(GlobalAppState.IsRecording, GlobalAppState.BlinkOn, GlobalAppState.Status)
		return m, nil
	}

	// Update focused input and re-run spell check
	switch m.focusedField {
	case YouTubeUploadFieldTitle:
		oldValue := m.titleInput.Value()
		m.titleInput, cmd = m.titleInput.Update(msg)
		if m.titleInput.Value() != oldValue {
			m.titleIssues = m.spellChecker.Check(m.titleInput.Value())
		}
	case YouTubeUploadFieldDescription:
		oldValue := m.descriptionInput.Value()
		m.descriptionInput, cmd = m.descriptionInput.Update(msg)
		if m.descriptionInput.Value() != oldValue {
			m.descIssues = m.spellChecker.Check(m.descriptionInput.Value())
		}
	case YouTubeUploadFieldTags:
		m.tagsInput, cmd = m.tagsInput.Update(msg)
	}

	return m, cmd
}

// saveYouTubeMetadata saves YouTube upload details to the recording metadata
func (m *YouTubeUploadModel) saveYouTubeMetadata(result *youtube.UploadResult) {
	if m.recordingInfo == nil {
		return
	}

	ytMeta := &models.YouTubeMetadata{
		VideoID:    result.VideoID,
		VideoURL:   result.VideoURL,
		Privacy:    string(m.privacyOptions[m.selectedPrivacy]),
		UploadedAt: time.Now().Format(time.RFC3339),
	}

	// Add playlist info if selected
	if m.selectedPlaylist >= 0 && m.selectedPlaylist < len(m.playlists) {
		ytMeta.PlaylistID = m.playlists[m.selectedPlaylist].ID
		ytMeta.PlaylistName = m.playlists[m.selectedPlaylist].Title
	}

	// Add channel info from selected account
	if len(m.accounts) > 0 && m.selectedAccount < len(m.accounts) {
		acc := m.accounts[m.selectedAccount]
		if acc.ChannelName != "" {
			ytMeta.ChannelName = acc.ChannelName
		}
		if acc.ChannelID != "" {
			ytMeta.ChannelID = acc.ChannelID
		}
	} else {
		// Fallback to legacy config
		if m.cfg.YouTube.ChannelName != "" {
			ytMeta.ChannelName = m.cfg.YouTube.ChannelName
		}
		if m.cfg.YouTube.ChannelID != "" {
			ytMeta.ChannelID = m.cfg.YouTube.ChannelID
		}
	}

	m.recordingInfo.Metadata.YouTube = ytMeta
	_ = m.recordingInfo.Save()

	// Save last used account ID
	if len(m.accounts) > 0 && m.selectedAccount < len(m.accounts) {
		m.cfg.YouTube.LastUsedAccountID = m.accounts[m.selectedAccount].ID
		_ = config.Save(m.cfg)
	}
}

// handleKeyMsg handles keyboard input
func (m *YouTubeUploadModel) handleKeyMsg(msg tea.KeyMsg) (*YouTubeUploadModel, tea.Cmd) {
	// Handle global keys first
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
	}

	// Handle step-specific keys
	switch m.step {
	case YouTubeUploadStepPrompt:
		switch msg.String() {
		case "y", "Y", "enter":
			// Check if any YouTube account is configured
			if len(m.accounts) == 0 {
				m.errorMessage = "No YouTube accounts configured. Go to Options > YouTube to set up."
				return m, nil
			}
			// Check if selected account is authenticated
			selectedAcc := m.accounts[m.selectedAccount]
			if !youtube.IsAccountAuthenticated(&m.cfg.YouTube, config.GetConfigDir(), selectedAcc.ID) {
				m.errorMessage = "Selected account not connected. Go to Options > YouTube to authenticate."
				return m, nil
			}
			m.step = YouTubeUploadStepMetadata
			// Set initial focus based on available fields
			m.focusedField = m.getFirstField()
			if m.focusedField == YouTubeUploadFieldTitle {
				m.titleInput.Focus()
			}
			m.loadingPlaylists = true
			return m, tea.Batch(textinput.Blink, m.loadPlaylists())

		case "n", "N":
			m.step = YouTubeUploadStepSkipped
			return m, func() tea.Msg { return youtubeUploadSkippedMsg{} }
		}

	case YouTubeUploadStepMetadata:
		switch msg.String() {
		case "tab", "down":
			m.nextField()
			return m, textinput.Blink

		case "shift+tab", "up":
			m.prevField()
			return m, textinput.Blink

		case "left", "right":
			if m.focusedField == YouTubeUploadFieldAccount && len(m.accounts) > 1 {
				if msg.String() == "left" {
					m.selectedAccount--
					if m.selectedAccount < 0 {
						m.selectedAccount = len(m.accounts) - 1
					}
				} else {
					m.selectedAccount++
					if m.selectedAccount >= len(m.accounts) {
						m.selectedAccount = 0
					}
				}
				// Reload playlists for new account
				m.playlists = nil
				m.selectedPlaylist = -1
				m.loadingPlaylists = true
				return m, m.loadPlaylists()
			}
			if m.focusedField == YouTubeUploadFieldVideoSource && len(m.videoSourceOptions) > 1 {
				if msg.String() == "left" {
					m.selectedVideoSource--
					if m.selectedVideoSource < 0 {
						m.selectedVideoSource = len(m.videoSourceOptions) - 1
					}
				} else {
					m.selectedVideoSource++
					if m.selectedVideoSource >= len(m.videoSourceOptions) {
						m.selectedVideoSource = 0
					}
				}
				// Update the video path based on selection
				m.updateVideoPath()
				return m, nil
			}
			if m.focusedField == YouTubeUploadFieldPrivacy {
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
			if m.focusedField == YouTubeUploadFieldPlaylist {
				// Navigate through playlists: -1 (none), 0, 1, 2, ...
				totalOptions := len(m.playlists) + 1 // +1 for "None"
				if msg.String() == "left" {
					m.selectedPlaylist--
					if m.selectedPlaylist < -1 {
						m.selectedPlaylist = len(m.playlists) - 1
					}
				} else {
					m.selectedPlaylist++
					if m.selectedPlaylist >= totalOptions-1 {
						m.selectedPlaylist = -1
					}
				}
				return m, nil
			}

		case "enter":
			return m.handleEnter()

		default:
			// Forward all other keys to the focused text input
			var cmd tea.Cmd
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

	case YouTubeUploadStepComplete, YouTubeUploadStepError:
		if msg.String() == "enter" {
			return m, func() tea.Msg { return youtubeUploadDoneMsg{} }
		}
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
		// Set initial focus to video source if multiple options, otherwise title
		if len(m.videoSourceOptions) > 1 {
			m.focusedField = YouTubeUploadFieldVideoSource
		} else {
			m.focusedField = YouTubeUploadFieldTitle
			m.titleInput.Focus()
		}
		m.loadingPlaylists = true
		return m, tea.Batch(textinput.Blink, m.loadPlaylists())

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
	// Skip account if only one account available
	if m.focusedField == YouTubeUploadFieldAccount && len(m.accounts) <= 1 {
		m.focusedField++
	}
	// Skip video source if only one option available
	if m.focusedField == YouTubeUploadFieldVideoSource && len(m.videoSourceOptions) <= 1 {
		m.focusedField++
	}
	if m.focusedField > YouTubeUploadFieldCancel {
		m.focusedField = m.getFirstField()
	}
	m.focusCurrent()
}

// prevField moves to the previous field
func (m *YouTubeUploadModel) prevField() {
	m.unfocusAll()
	m.focusedField--
	// Skip video source if only one option available
	if m.focusedField == YouTubeUploadFieldVideoSource && len(m.videoSourceOptions) <= 1 {
		m.focusedField--
	}
	// Skip account if only one account available
	if m.focusedField == YouTubeUploadFieldAccount && len(m.accounts) <= 1 {
		m.focusedField--
	}
	if m.focusedField < YouTubeUploadFieldAccount {
		m.focusedField = YouTubeUploadFieldCancel
	}
	m.focusCurrent()
}

// updateVideoPath updates the video path based on the selected video source
func (m *YouTubeUploadModel) updateVideoPath() {
	if m.selectedVideoSource < 0 || m.selectedVideoSource >= len(m.videoSourceOptions) {
		return
	}
	switch m.videoSourceOptions[m.selectedVideoSource] {
	case VideoSourceVertical:
		m.videoPath = m.verticalVideoPath
	case VideoSourceMerged:
		m.videoPath = m.mergedVideoPath
	}
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

// getFirstField returns the first field to focus based on available options
func (m *YouTubeUploadModel) getFirstField() YouTubeUploadField {
	// Start with account if multiple accounts
	if len(m.accounts) > 1 {
		return YouTubeUploadFieldAccount
	}
	// Then video source if multiple options
	if len(m.videoSourceOptions) > 1 {
		return YouTubeUploadFieldVideoSource
	}
	// Otherwise title
	return YouTubeUploadFieldTitle
}

// loadPlaylists fetches playlists from YouTube
func (m *YouTubeUploadModel) loadPlaylists() tea.Cmd {
	// Capture the selected account for the goroutine
	var clientID, clientSecret, accountID string
	if len(m.accounts) > 0 && m.selectedAccount < len(m.accounts) {
		acc := m.accounts[m.selectedAccount]
		clientID = acc.ClientID
		clientSecret = acc.ClientSecret
		accountID = acc.ID
	} else {
		// Fallback to legacy config
		clientID = m.cfg.YouTube.ClientID
		clientSecret = m.cfg.YouTube.ClientSecret
		accountID = "legacy"
	}

	return func() tea.Msg {
		ctx := context.Background()
		auth := youtube.NewAuthForAccount(clientID, clientSecret, config.GetConfigDir(), accountID)

		uploader, err := youtube.NewUploader(ctx, auth)
		if err != nil {
			return playlistsLoadedMsg{err: err}
		}

		playlists, err := uploader.ListPlaylists(ctx)
		if err != nil {
			return playlistsLoadedMsg{err: err}
		}

		return playlistsLoadedMsg{playlists: playlists}
	}
}

// uploadUpdate carries progress or completion info from the upload goroutine
type uploadUpdate struct {
	percent  float64
	done     bool
	err      error
	result   *youtube.UploadResult
}

// startUpload begins the YouTube upload
func (m *YouTubeUploadModel) startUpload() tea.Cmd {
	m.step = YouTubeUploadStepUploading
	m.isUploading = true
	m.uploadPct = 0
	m.errorMessage = ""

	// Create progress channel that will be used to send updates
	m.uploadProgressCh = make(chan uploadUpdate, 100)

	// Capture values needed by the goroutine
	progressCh := m.uploadProgressCh
	videoPath := m.videoPath
	title := m.titleInput.Value()
	description := m.descriptionInput.Value()
	topic := m.topic
	tags := youtube.ParseTags(m.tagsInput.Value())
	privacy := m.privacyOptions[m.selectedPrivacy]
	var playlistID string
	if m.selectedPlaylist >= 0 && m.selectedPlaylist < len(m.playlists) {
		playlistID = m.playlists[m.selectedPlaylist].ID
	}

	// Get selected account credentials
	var clientID, clientSecret, accountID string
	if len(m.accounts) > 0 && m.selectedAccount < len(m.accounts) {
		acc := m.accounts[m.selectedAccount]
		clientID = acc.ClientID
		clientSecret = acc.ClientSecret
		accountID = acc.ID
	} else {
		// Fallback to legacy config
		clientID = m.cfg.YouTube.ClientID
		clientSecret = m.cfg.YouTube.ClientSecret
		accountID = "legacy"
	}

	// Start the upload in a goroutine
	go func() {
		ctx := context.Background()

		// Create auth for selected account
		auth := youtube.NewAuthForAccount(clientID, clientSecret, config.GetConfigDir(), accountID)

		// Create uploader
		uploader, err := youtube.NewUploader(ctx, auth)
		if err != nil {
			progressCh <- uploadUpdate{done: true, err: err}
			close(progressCh)
			return
		}

		// Build upload options
		opts := youtube.BuildUploadOptions(
			videoPath,
			title,
			description,
			topic,
			tags,
			privacy,
		)

		// Add playlist if selected
		if playlistID != "" {
			opts.PlaylistID = playlistID
		}

		// First extract thumbnail if it doesn't exist
		thumbnailPath := youtube.GetThumbnailPath(videoPath)
		if err := youtube.ExtractThumbnailForYouTube(videoPath, thumbnailPath); err == nil {
			opts.ThumbnailPath = thumbnailPath
		}

		// Upload with progress callback
		result, err := uploader.Upload(ctx, opts, func(read, total int64) {
			if total > 0 {
				pct := float64(read) / float64(total)
				select {
				case progressCh <- uploadUpdate{percent: pct}:
				default:
					// Channel full, skip this update
				}
			}
		})

		// Send completion
		progressCh <- uploadUpdate{done: true, err: err, result: result}
		close(progressCh)
	}()

	// Return command to wait for first progress update
	return waitForUploadProgress(m.uploadProgressCh)
}

// waitForUploadProgress waits for the next upload progress update
func waitForUploadProgress(ch chan uploadUpdate) tea.Cmd {
	if ch == nil {
		return nil
	}
	return func() tea.Msg {
		update, ok := <-ch
		if !ok {
			// Channel closed unexpectedly
			return uploadCompleteMsg{err: nil}
		}
		if update.done {
			return uploadCompleteMsg{err: update.err, result: update.result}
		}
		return uploadProgressMsg{percent: update.percent}
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
		lipgloss.NewStyle().Foreground(ColorGray).Render("y: upload • n: skip"),
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

	// Account row (only show if multiple accounts available)
	var accountRow string
	if len(m.accounts) > 1 {
		accountLabel := labelStyle.Render("Account: ")
		if m.focusedField == YouTubeUploadFieldAccount {
			accountLabel = labelActiveStyle.Render("Account: ")
		}
		var accountValues []string
		for i, acc := range m.accounts {
			displayName := acc.Name
			if displayName == "" {
				displayName = acc.ChannelName
			}
			if displayName == "" {
				displayName = "Account " + acc.ID[:8]
			}
			style := lipgloss.NewStyle().Foreground(ColorGray)
			if i == m.selectedAccount {
				if m.focusedField == YouTubeUploadFieldAccount {
					style = lipgloss.NewStyle().Background(ColorOrange).Foreground(lipgloss.Color("#000000"))
				} else {
					style = lipgloss.NewStyle().Foreground(ColorWhite).Bold(true)
				}
			}
			accountValues = append(accountValues, style.Render(" "+displayName+" "))
		}
		accountValue := lipgloss.JoinHorizontal(lipgloss.Center, accountValues...)
		accountRow = lipgloss.JoinHorizontal(lipgloss.Center, accountLabel, accountValue)
	}

	// Video source row (only show if multiple options available)
	var videoSourceRow string
	if len(m.videoSourceOptions) > 1 {
		videoSourceLabel := labelStyle.Render("Video: ")
		if m.focusedField == YouTubeUploadFieldVideoSource {
			videoSourceLabel = labelActiveStyle.Render("Video: ")
		}
		var videoSourceValues []string
		for i, opt := range m.videoSourceOptions {
			style := lipgloss.NewStyle().Foreground(ColorGray)
			if i == m.selectedVideoSource {
				if m.focusedField == YouTubeUploadFieldVideoSource {
					style = lipgloss.NewStyle().Background(ColorOrange).Foreground(lipgloss.Color("#000000"))
				} else {
					style = lipgloss.NewStyle().Foreground(ColorWhite).Bold(true)
				}
			}
			videoSourceValues = append(videoSourceValues, style.Render(" "+opt.String()+" "))
		}
		videoSourceValue := lipgloss.JoinHorizontal(lipgloss.Center, videoSourceValues...)
		videoSourceRow = lipgloss.JoinHorizontal(lipgloss.Center, videoSourceLabel, videoSourceValue)
	}

	// Spell check warning style
	warningStyle := lipgloss.NewStyle().
		Foreground(ColorOrange).
		Italic(true).
		PaddingLeft(16)

	// Title row
	titleLabel := labelStyle.Render("Title: ")
	if m.focusedField == YouTubeUploadFieldTitle {
		titleLabel = labelActiveStyle.Render("Title: ")
	}
	titleRow := lipgloss.JoinHorizontal(lipgloss.Center, titleLabel, m.titleInput.View())

	// Title spell check warnings
	var titleWarnings string
	if len(m.titleIssues) > 0 {
		var warnings []string
		for _, issue := range m.titleIssues {
			warning := "⚠ " + issue.Word + ": " + issue.Message
			if len(issue.Suggestions) > 0 && issue.Suggestions[0] != "" {
				warning += " → " + issue.Suggestions[0]
			}
			warnings = append(warnings, warningStyle.Render(warning))
		}
		titleWarnings = lipgloss.JoinVertical(lipgloss.Left, warnings...)
	}

	// Description row
	descLabel := labelStyle.Render("Description: ")
	if m.focusedField == YouTubeUploadFieldDescription {
		descLabel = labelActiveStyle.Render("Description: ")
	}
	descRow := lipgloss.JoinHorizontal(lipgloss.Center, descLabel, m.descriptionInput.View())

	// Description spell check warnings
	var descWarnings string
	if len(m.descIssues) > 0 {
		var warnings []string
		maxWarnings := 3 // Limit to avoid UI clutter
		for i, issue := range m.descIssues {
			if i >= maxWarnings {
				remaining := len(m.descIssues) - maxWarnings
				warnings = append(warnings, warningStyle.Render(fmt.Sprintf("  ... and %d more issues", remaining)))
				break
			}
			warning := "⚠ " + issue.Word + ": " + issue.Message
			if len(issue.Suggestions) > 0 && issue.Suggestions[0] != "" {
				warning += " → " + issue.Suggestions[0]
			}
			warnings = append(warnings, warningStyle.Render(warning))
		}
		descWarnings = lipgloss.JoinVertical(lipgloss.Left, warnings...)
	}

	// Tags row
	tagsLabel := labelStyle.Render("Tags: ")
	if m.focusedField == YouTubeUploadFieldTags {
		tagsLabel = labelActiveStyle.Render("Tags: ")
	}
	tagsRow := lipgloss.JoinHorizontal(lipgloss.Center, tagsLabel, m.tagsInput.View())

	// Playlist row
	playlistLabel := labelStyle.Render("Playlist: ")
	if m.focusedField == YouTubeUploadFieldPlaylist {
		playlistLabel = labelActiveStyle.Render("Playlist: ")
	}
	var playlistValue string
	if m.loadingPlaylists {
		playlistValue = lipgloss.NewStyle().Foreground(ColorGray).Italic(true).Render("Loading playlists...")
	} else if m.playlistError != "" {
		playlistValue = lipgloss.NewStyle().Foreground(ColorRed).Render("Error: " + m.playlistError)
	} else {
		// Build playlist selection
		var playlistName string
		if m.selectedPlaylist < 0 {
			playlistName = "None"
		} else if m.selectedPlaylist < len(m.playlists) {
			playlistName = m.playlists[m.selectedPlaylist].Title
		}

		style := lipgloss.NewStyle().Foreground(ColorGray)
		if m.focusedField == YouTubeUploadFieldPlaylist {
			style = lipgloss.NewStyle().Background(ColorOrange).Foreground(lipgloss.Color("#000000"))
		} else if m.selectedPlaylist >= 0 {
			style = lipgloss.NewStyle().Foreground(ColorWhite).Bold(true)
		}
		playlistValue = style.Render(" " + playlistName + " ")

		if m.focusedField == YouTubeUploadFieldPlaylist && len(m.playlists) > 0 {
			playlistValue += lipgloss.NewStyle().Foreground(ColorGray).Render(" (←/→ to change)")
		}
	}
	playlistRow := lipgloss.JoinHorizontal(lipgloss.Center, playlistLabel, playlistValue)

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

	// Build the rows to display
	rows := []string{}
	if accountRow != "" {
		rows = append(rows, accountRow)
	}
	if videoSourceRow != "" {
		rows = append(rows, videoSourceRow)
	}
	rows = append(rows, titleRow)
	if titleWarnings != "" {
		rows = append(rows, titleWarnings)
	}
	rows = append(rows, descRow)
	if descWarnings != "" {
		rows = append(rows, descWarnings)
	}
	rows = append(rows, tagsRow, playlistRow, privacyRow, "", buttonRow, "", errorLine)

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
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

	var playlistInfo string
	if m.selectedPlaylist >= 0 && m.selectedPlaylist < len(m.playlists) {
		playlistInfo = lipgloss.NewStyle().
			Foreground(ColorGray).
			Render("Added to playlist: " + m.playlists[m.selectedPlaylist].Title)
	}

	return lipgloss.JoinVertical(lipgloss.Center,
		titleStyle.Render("Upload Complete!"),
		"",
		textStyle.Render("Your video has been uploaded to YouTube."),
		"",
		linkStyle.Render(url),
		"",
		playlistInfo,
		"",
		lipgloss.NewStyle().Foreground(ColorGray).Render("enter: continue"),
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
		lipgloss.NewStyle().Foreground(ColorGray).Render("enter: continue • r: retry"),
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
		return "tab: next field • enter: select • ←/→: change playlist/privacy • esc: back"
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

type playlistsLoadedMsg struct {
	playlists []youtube.Playlist
	err       error
}

type uploadProgressMsg struct {
	percent float64
}

type uploadCompleteMsg struct {
	result *youtube.UploadResult
	err    error
}

type youtubeUploadSkippedMsg struct{}

type youtubeUploadDoneMsg struct{}
