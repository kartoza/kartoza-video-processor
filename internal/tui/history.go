package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kartoza/kartoza-video-processor/internal/config"
	"github.com/kartoza/kartoza-video-processor/internal/models"
	"github.com/kartoza/kartoza-video-processor/internal/youtube"
)

// HistoryViewMode represents the current mode of the history view
type HistoryViewMode int

const (
	HistoryListMode HistoryViewMode = iota
	HistoryDetailMode
	HistoryEditMode
	HistoryDeleteConfirmMode
	HistoryYouTubePrivacyMode
	HistoryYouTubeDeleteConfirmMode
	HistoryYouTubeUploadMode
)

// HistoryModel displays recording history with navigation
type HistoryModel struct {
	width  int
	height int

	// Data
	recordings []models.RecordingInfo

	// Scrolling - cursor is absolute position in recordings
	cursor int

	// View mode
	mode HistoryViewMode

	// Detail/Edit view state
	selectedRecording *models.RecordingInfo
	editFields        struct {
		title       textinput.Model
		description textarea.Model
		presenter   textinput.Model
	}
	editTopicIndex int
	topics         []models.Topic
	editFocusField int
	editError      string
	editSuccess    string
	isSaving       bool

	// State
	err     error
	loading bool

	// Delete confirmation state
	deleteConfirmRecording *models.RecordingInfo
	deleteError            string

	// YouTube action state
	youtubePrivacyOptions  []string
	youtubeSelectedPrivacy int
	youtubeActionError     string
	youtubeActionSuccess   string
	youtubeActionLoading   bool
}

// NewHistoryModel creates a new history model
func NewHistoryModel() *HistoryModel {
	// Initialize edit fields
	titleInput := textinput.New()
	titleInput.Placeholder = "Recording title..."
	titleInput.CharLimit = 100
	titleInput.Width = 40

	descInput := textarea.New()
	descInput.Placeholder = "Description..."
	descInput.CharLimit = 2000
	descInput.SetWidth(40)
	descInput.SetHeight(3)
	descInput.ShowLineNumbers = false

	presenterInput := textinput.New()
	presenterInput.Placeholder = "Presenter name..."
	presenterInput.CharLimit = 100
	presenterInput.Width = 40

	// Load topics
	cfg, _ := config.Load()
	topics := cfg.Topics
	if len(topics) == 0 {
		topics = models.DefaultTopics()
	}

	h := &HistoryModel{
		cursor:                0,
		loading:               true,
		mode:                  HistoryListMode,
		topics:                topics,
		youtubePrivacyOptions: []string{"unlisted", "private", "public"},
	}

	h.editFields.title = titleInput
	h.editFields.description = descInput
	h.editFields.presenter = presenterInput

	return h
}

// Init initializes the history view
func (h *HistoryModel) Init() tea.Cmd {
	return h.loadRecordings()
}

// getVisibleCount returns how many entries can fit on screen
func (h *HistoryModel) getVisibleCount() int {
	availableHeight := h.height - 12
	if availableHeight < 3 {
		return 1
	}
	count := availableHeight / 3
	if count < 1 {
		count = 1
	}
	if count > 12 {
		count = 12
	}
	return count
}

// Update handles messages
func (h *HistoryModel) Update(msg tea.Msg) (*HistoryModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h.width = msg.Width
		h.height = msg.Height

	case tea.KeyMsg:
		switch h.mode {
		case HistoryListMode:
			return h.updateListMode(msg)
		case HistoryDetailMode:
			return h.updateDetailMode(msg)
		case HistoryEditMode:
			return h.updateEditMode(msg)
		case HistoryDeleteConfirmMode:
			return h.updateDeleteConfirmMode(msg)
		case HistoryYouTubePrivacyMode:
			return h.updateYouTubePrivacyMode(msg)
		case HistoryYouTubeDeleteConfirmMode:
			return h.updateYouTubeDeleteConfirmMode(msg)
		}

	case recordingsLoadedMsg:
		h.loading = false
		h.recordings = msg.recordings
		h.err = msg.err

	case recordingSavedMsg:
		h.isSaving = false
		if msg.err != nil {
			h.editError = msg.err.Error()
		} else {
			h.editSuccess = "Recording saved successfully!"
			// Update local copy
			if h.selectedRecording != nil {
				for i := range h.recordings {
					if h.recordings[i].Files.FolderPath == h.selectedRecording.Files.FolderPath {
						h.recordings[i] = *h.selectedRecording
						break
					}
				}
			}
		}

	case youtubePrivacyChangedMsg:
		h.youtubeActionLoading = false
		if msg.err != nil {
			h.youtubeActionError = msg.err.Error()
		} else {
			h.youtubeActionSuccess = "Privacy updated to " + msg.newPrivacy
			// Update local metadata
			if h.selectedRecording != nil && h.selectedRecording.Metadata.YouTube != nil {
				h.selectedRecording.Metadata.YouTube.Privacy = msg.newPrivacy
				h.selectedRecording.Save()
				// Update in list
				for i := range h.recordings {
					if h.recordings[i].Files.FolderPath == h.selectedRecording.Files.FolderPath {
						h.recordings[i] = *h.selectedRecording
						break
					}
				}
			}
			h.mode = HistoryDetailMode
		}

	case youtubeVideoDeletedMsg:
		h.youtubeActionLoading = false
		if msg.err != nil {
			h.youtubeActionError = msg.err.Error()
		} else {
			h.youtubeActionSuccess = "Video deleted from YouTube"
			// Clear YouTube metadata
			if h.selectedRecording != nil {
				h.selectedRecording.Metadata.YouTube = nil
				h.selectedRecording.Save()
				// Update in list
				for i := range h.recordings {
					if h.recordings[i].Files.FolderPath == h.selectedRecording.Files.FolderPath {
						h.recordings[i] = *h.selectedRecording
						break
					}
				}
			}
			h.mode = HistoryDetailMode
		}

	case startYouTubeUploadMsg:
		// This is handled by the parent app model
		return h, func() tea.Msg { return msg }
	}

	return h, nil
}

// updateListMode handles input in list mode
func (h *HistoryModel) updateListMode(msg tea.KeyMsg) (*HistoryModel, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return h, tea.Quit

	case "esc", "q":
		return h, func() tea.Msg { return backToMenuMsg{} }

	case "up", "k":
		if h.cursor > 0 {
			h.cursor--
		}

	case "down", "j":
		if h.cursor < len(h.recordings)-1 {
			h.cursor++
		}

	case "home", "g":
		h.cursor = 0

	case "end", "G":
		if len(h.recordings) > 0 {
			h.cursor = len(h.recordings) - 1
		}

	case "pgup":
		h.cursor -= h.getVisibleCount()
		if h.cursor < 0 {
			h.cursor = 0
		}

	case "pgdown":
		h.cursor += h.getVisibleCount()
		if h.cursor >= len(h.recordings) {
			h.cursor = len(h.recordings) - 1
		}
		if h.cursor < 0 {
			h.cursor = 0
		}

	case "enter", " ":
		// Open detail view
		if len(h.recordings) > 0 && h.cursor < len(h.recordings) {
			rec := h.recordings[h.cursor]
			h.selectedRecording = &rec
			h.mode = HistoryDetailMode
			h.editError = ""
			h.editSuccess = ""
		}

	case "r":
		h.loading = true
		h.cursor = 0
		return h, h.loadRecordings()

	case "d":
		// Delete selected recording (with confirmation)
		if len(h.recordings) > 0 && h.cursor < len(h.recordings) {
			rec := h.recordings[h.cursor]
			h.deleteConfirmRecording = &rec
			h.deleteError = ""
			h.mode = HistoryDeleteConfirmMode
		}
	}

	return h, nil
}

// updateDeleteConfirmMode handles input in delete confirmation mode
func (h *HistoryModel) updateDeleteConfirmMode(msg tea.KeyMsg) (*HistoryModel, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return h, tea.Quit

	case "esc", "n", "N":
		// Cancel deletion
		h.mode = HistoryListMode
		h.deleteConfirmRecording = nil
		h.deleteError = ""

	case "y", "Y":
		// Confirm deletion
		if h.deleteConfirmRecording != nil {
			folderPath := h.deleteConfirmRecording.Files.FolderPath
			err := os.RemoveAll(folderPath)
			if err != nil {
				h.deleteError = fmt.Sprintf("Failed to delete: %v", err)
				return h, nil
			}

			// Remove from list
			for i := range h.recordings {
				if h.recordings[i].Files.FolderPath == folderPath {
					h.recordings = append(h.recordings[:i], h.recordings[i+1:]...)
					break
				}
			}

			// Adjust cursor if needed
			if h.cursor >= len(h.recordings) && h.cursor > 0 {
				h.cursor--
			}

			// Return to list mode
			h.mode = HistoryListMode
			h.deleteConfirmRecording = nil
			h.deleteError = ""

			// Update global recording count
			updateGlobalAppState(GlobalAppState.IsRecording, GlobalAppState.BlinkOn, GlobalAppState.Status)
		}
	}

	return h, nil
}

// updateDetailMode handles input in detail view mode
func (h *HistoryModel) updateDetailMode(msg tea.KeyMsg) (*HistoryModel, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return h, tea.Quit

	case "esc", "q":
		// Go back to list
		h.mode = HistoryListMode
		h.selectedRecording = nil
		h.editError = ""
		h.editSuccess = ""
		h.youtubeActionError = ""
		h.youtubeActionSuccess = ""

	case "e":
		// Enter edit mode
		if h.selectedRecording != nil {
			h.mode = HistoryEditMode
			h.initEditFields()
			h.editFocusField = 0
			h.editFields.title.Focus()
			return h, textinput.Blink
		}

	case "u":
		// Upload to YouTube (only if not already uploaded)
		if h.selectedRecording != nil && !h.selectedRecording.Metadata.IsPublishedToYouTube() {
			// Check if YouTube is connected
			cfg, _ := config.Load()
			if !cfg.IsYouTubeConnected() {
				h.youtubeActionError = "YouTube not connected. Go to Options > YouTube to set up."
				return h, nil
			}
			// Find video file to upload
			videoPath := h.selectedRecording.Files.MergedFile
			if videoPath == "" {
				videoPath = h.selectedRecording.Files.VideoFile
			}
			if videoPath == "" {
				h.youtubeActionError = "No video file found to upload"
				return h, nil
			}
			// Send message to parent to start upload
			return h, func() tea.Msg {
				return startYouTubeUploadMsg{
					recording: h.selectedRecording,
					videoPath: videoPath,
				}
			}
		}

	case "p":
		// Change privacy (only if already uploaded)
		if h.selectedRecording != nil && h.selectedRecording.Metadata.IsPublishedToYouTube() {
			h.mode = HistoryYouTubePrivacyMode
			h.youtubeActionError = ""
			h.youtubeActionSuccess = ""
			// Set current privacy as selected
			currentPrivacy := h.selectedRecording.Metadata.YouTube.Privacy
			for i, p := range h.youtubePrivacyOptions {
				if p == currentPrivacy {
					h.youtubeSelectedPrivacy = i
					break
				}
			}
		}

	case "x":
		// Delete from YouTube (only if already uploaded)
		if h.selectedRecording != nil && h.selectedRecording.Metadata.IsPublishedToYouTube() {
			h.mode = HistoryYouTubeDeleteConfirmMode
			h.youtubeActionError = ""
			h.youtubeActionSuccess = ""
		}
	}

	return h, nil
}

// updateEditMode handles input in edit mode
func (h *HistoryModel) updateEditMode(msg tea.KeyMsg) (*HistoryModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.String() {
	case "ctrl+c":
		return h, tea.Quit

	case "esc":
		// Go back to detail view
		h.mode = HistoryDetailMode
		h.editError = ""
		h.editSuccess = ""

	case "tab", "down":
		// Move to next field
		h.blurAllEditFields()
		h.editFocusField = (h.editFocusField + 1) % 4
		h.focusEditField()
		return h, textinput.Blink

	case "shift+tab", "up":
		// Move to previous field
		h.blurAllEditFields()
		h.editFocusField = (h.editFocusField + 3) % 4
		h.focusEditField()
		return h, textinput.Blink

	case "left", "h":
		if h.editFocusField == 2 { // Topic field
			h.editTopicIndex--
			if h.editTopicIndex < 0 {
				h.editTopicIndex = len(h.topics) - 1
			}
			return h, nil
		}
		// Fall through to let input handle it

	case "right", "l":
		if h.editFocusField == 2 { // Topic field
			h.editTopicIndex++
			if h.editTopicIndex >= len(h.topics) {
				h.editTopicIndex = 0
			}
			return h, nil
		}
		// Fall through to let input handle it

	case "ctrl+s":
		// Save changes
		if !h.isSaving {
			return h, h.saveRecording()
		}
		return h, nil

	case "enter":
		// In description, allow newlines; otherwise move to next field
		if h.editFocusField == 1 { // Description field
			h.editFields.description, cmd = h.editFields.description.Update(msg)
			return h, cmd
		}
		// Move to next field
		h.blurAllEditFields()
		h.editFocusField = (h.editFocusField + 1) % 4
		h.focusEditField()
		return h, textinput.Blink
	}

	// Update focused field
	switch h.editFocusField {
	case 0:
		h.editFields.title, cmd = h.editFields.title.Update(msg)
	case 1:
		h.editFields.description, cmd = h.editFields.description.Update(msg)
	case 3:
		h.editFields.presenter, cmd = h.editFields.presenter.Update(msg)
	}
	return h, cmd
}

// updateYouTubePrivacyMode handles input in YouTube privacy change mode
func (h *HistoryModel) updateYouTubePrivacyMode(msg tea.KeyMsg) (*HistoryModel, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return h, tea.Quit

	case "esc", "q":
		h.mode = HistoryDetailMode
		h.youtubeActionError = ""

	case "left", "h":
		h.youtubeSelectedPrivacy--
		if h.youtubeSelectedPrivacy < 0 {
			h.youtubeSelectedPrivacy = len(h.youtubePrivacyOptions) - 1
		}

	case "right", "l":
		h.youtubeSelectedPrivacy++
		if h.youtubeSelectedPrivacy >= len(h.youtubePrivacyOptions) {
			h.youtubeSelectedPrivacy = 0
		}

	case "enter":
		if h.selectedRecording != nil && h.selectedRecording.Metadata.YouTube != nil {
			newPrivacy := h.youtubePrivacyOptions[h.youtubeSelectedPrivacy]
			if newPrivacy != h.selectedRecording.Metadata.YouTube.Privacy {
				h.youtubeActionLoading = true
				return h, h.changeYouTubePrivacy(newPrivacy)
			}
			h.mode = HistoryDetailMode
		}
	}

	return h, nil
}

// updateYouTubeDeleteConfirmMode handles input in YouTube delete confirmation mode
func (h *HistoryModel) updateYouTubeDeleteConfirmMode(msg tea.KeyMsg) (*HistoryModel, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return h, tea.Quit

	case "esc", "n", "N":
		h.mode = HistoryDetailMode
		h.youtubeActionError = ""

	case "y", "Y":
		if h.selectedRecording != nil && h.selectedRecording.Metadata.YouTube != nil {
			h.youtubeActionLoading = true
			return h, h.deleteFromYouTube()
		}
	}

	return h, nil
}

// changeYouTubePrivacy changes the privacy setting of a YouTube video
func (h *HistoryModel) changeYouTubePrivacy(newPrivacy string) tea.Cmd {
	rec := h.selectedRecording
	return func() tea.Msg {
		ctx := context.Background()
		cfg, err := config.Load()
		if err != nil {
			return youtubePrivacyChangedMsg{err: err}
		}

		auth := youtube.NewAuth(cfg.YouTube.ClientID, cfg.YouTube.ClientSecret, config.GetConfigDir())
		uploader, err := youtube.NewUploader(ctx, auth)
		if err != nil {
			return youtubePrivacyChangedMsg{err: err}
		}

		err = uploader.UpdateVideoPrivacy(ctx, rec.Metadata.YouTube.VideoID, youtube.PrivacyStatus(newPrivacy))
		if err != nil {
			return youtubePrivacyChangedMsg{err: err}
		}

		return youtubePrivacyChangedMsg{newPrivacy: newPrivacy}
	}
}

// deleteFromYouTube deletes the video from YouTube
func (h *HistoryModel) deleteFromYouTube() tea.Cmd {
	rec := h.selectedRecording
	return func() tea.Msg {
		ctx := context.Background()
		cfg, err := config.Load()
		if err != nil {
			return youtubeVideoDeletedMsg{err: err}
		}

		auth := youtube.NewAuth(cfg.YouTube.ClientID, cfg.YouTube.ClientSecret, config.GetConfigDir())
		uploader, err := youtube.NewUploader(ctx, auth)
		if err != nil {
			return youtubeVideoDeletedMsg{err: err}
		}

		err = uploader.DeleteVideo(ctx, rec.Metadata.YouTube.VideoID)
		if err != nil {
			return youtubeVideoDeletedMsg{err: err}
		}

		return youtubeVideoDeletedMsg{}
	}
}

// initEditFields populates edit fields from selected recording
func (h *HistoryModel) initEditFields() {
	if h.selectedRecording == nil {
		return
	}

	h.editFields.title.SetValue(h.selectedRecording.Metadata.Title)
	h.editFields.description.SetValue(h.selectedRecording.Metadata.Description)
	h.editFields.presenter.SetValue(h.selectedRecording.Metadata.Presenter)

	// Find matching topic
	h.editTopicIndex = 0
	for i, topic := range h.topics {
		if topic.Name == h.selectedRecording.Metadata.Topic {
			h.editTopicIndex = i
			break
		}
	}
}

// blurAllEditFields removes focus from all edit fields
func (h *HistoryModel) blurAllEditFields() {
	h.editFields.title.Blur()
	h.editFields.description.Blur()
	h.editFields.presenter.Blur()
}

// focusEditField focuses the current edit field
func (h *HistoryModel) focusEditField() {
	switch h.editFocusField {
	case 0:
		h.editFields.title.Focus()
	case 1:
		h.editFields.description.Focus()
	case 3:
		h.editFields.presenter.Focus()
	}
}

// saveRecording saves the edited recording
func (h *HistoryModel) saveRecording() tea.Cmd {
	if h.selectedRecording == nil {
		return nil
	}

	h.isSaving = true
	h.editError = ""

	// Update metadata
	h.selectedRecording.Metadata.Title = strings.TrimSpace(h.editFields.title.Value())
	h.selectedRecording.Metadata.Description = strings.TrimSpace(h.editFields.description.Value())
	h.selectedRecording.Metadata.Presenter = strings.TrimSpace(h.editFields.presenter.Value())
	if h.editTopicIndex >= 0 && h.editTopicIndex < len(h.topics) {
		h.selectedRecording.Metadata.Topic = h.topics[h.editTopicIndex].Name
	}

	rec := h.selectedRecording
	return func() tea.Msg {
		err := rec.Save()
		return recordingSavedMsg{err: err}
	}
}

// View renders the history view
func (h *HistoryModel) View() string {
	if h.width == 0 {
		return "Loading..."
	}

	switch h.mode {
	case HistoryDetailMode:
		return h.renderDetailView()
	case HistoryEditMode:
		return h.renderEditView()
	case HistoryDeleteConfirmMode:
		return h.renderDeleteConfirmView()
	case HistoryYouTubePrivacyMode:
		return h.renderYouTubePrivacyView()
	case HistoryYouTubeDeleteConfirmMode:
		return h.renderYouTubeDeleteConfirmView()
	default:
		return h.renderListView()
	}
}

// renderListView renders the list mode view
func (h *HistoryModel) renderListView() string {
	header := RenderHeader("Recording History")

	if h.loading {
		loadingStyle := lipgloss.NewStyle().
			Foreground(ColorGray).
			Align(lipgloss.Center)

		mainContent := loadingStyle.Render("Loading recordings...")

		mainSection := lipgloss.JoinVertical(
			lipgloss.Center,
			header,
			"",
			mainContent,
		)

		centeredMain := lipgloss.Place(
			h.width,
			h.height-2,
			lipgloss.Center,
			lipgloss.Top,
			mainSection,
		)

		return centeredMain
	}

	if h.err != nil {
		errorStyle := lipgloss.NewStyle().
			Foreground(ColorRed).
			Align(lipgloss.Center)

		mainContent := lipgloss.JoinVertical(
			lipgloss.Center,
			errorStyle.Render("Error: "+h.err.Error()),
		)

		mainSection := lipgloss.JoinVertical(
			lipgloss.Center,
			header,
			"",
			mainContent,
		)

		helpStyle := lipgloss.NewStyle().
			Width(h.width).
			Align(lipgloss.Center).
			Foreground(ColorGray).
			Italic(true)

		centeredMain := lipgloss.Place(
			h.width,
			h.height-2,
			lipgloss.Center,
			lipgloss.Top,
			mainSection,
		)

		return lipgloss.JoinVertical(
			lipgloss.Left,
			centeredMain,
			helpStyle.Render("Press 'r' to retry, Esc to go back"),
		)
	}

	if len(h.recordings) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(ColorGray).
			Align(lipgloss.Center)

		mainContent := emptyStyle.Render("No recordings found")

		mainSection := lipgloss.JoinVertical(
			lipgloss.Center,
			header,
			"",
			mainContent,
		)

		helpStyle := lipgloss.NewStyle().
			Width(h.width).
			Align(lipgloss.Center).
			Foreground(ColorGray).
			Italic(true)

		centeredMain := lipgloss.Place(
			h.width,
			h.height-2,
			lipgloss.Center,
			lipgloss.Top,
			mainSection,
		)

		return lipgloss.JoinVertical(
			lipgloss.Left,
			centeredMain,
			helpStyle.Render("Press Esc to go back"),
		)
	}

	// Position info
	positionInfo := fmt.Sprintf("Recording %d of %d", h.cursor+1, len(h.recordings))
	posStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Align(lipgloss.Center)

	table := h.renderScrollableTable()
	scrollBar := h.renderScrollBar()

	helpStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Italic(true)

	tableWithScroll := lipgloss.JoinHorizontal(lipgloss.Top, table, " ", scrollBar)

	infoLine := posStyle.Render(positionInfo)

	mainSection := lipgloss.JoinVertical(
		lipgloss.Center,
		header,
		"",
		infoLine,
		"",
		tableWithScroll,
	)

	centeredMain := lipgloss.Place(
		h.width,
		h.height-2,
		lipgloss.Center,
		lipgloss.Top,
		mainSection,
	)

	helpFooter := lipgloss.NewStyle().
		Width(h.width).
		Align(lipgloss.Center)

	helpText := "↑/↓: Navigate • Enter: View Details • d: Delete • r: Refresh • Esc/q: Back"

	return lipgloss.JoinVertical(
		lipgloss.Left,
		centeredMain,
		helpFooter.Render(helpStyle.Render(helpText)),
	)
}

// renderDetailView renders the detail view for a selected recording
func (h *HistoryModel) renderDetailView() string {
	if h.selectedRecording == nil {
		return "No recording selected"
	}

	rec := h.selectedRecording
	header := RenderHeader("Recording Details")

	// Styles
	labelStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Width(14).
		Align(lipgloss.Right)

	valueStyle := lipgloss.NewStyle().
		Foreground(ColorWhite).
		Bold(true)

	highlightStyle := lipgloss.NewStyle().
		Foreground(ColorBlue).
		Bold(true)

	containerStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorOrange).
		Padding(1, 3).
		Width(70)

	dividerStyle := lipgloss.NewStyle().
		Foreground(ColorGray)

	// Build detail rows
	var rows []string

	// Folder badge
	folderBadge := lipgloss.NewStyle().
		Background(ColorBlue).
		Foreground(ColorWhite).
		Padding(0, 1).
		Bold(true).
		Render(rec.Metadata.FolderName)

	folderRow := lipgloss.NewStyle().Align(lipgloss.Center).Width(62).Render(folderBadge)
	rows = append(rows, folderRow)
	rows = append(rows, "")

	// Title
	rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top,
		labelStyle.Render("Title:"),
		"  ",
		highlightStyle.Render(rec.Metadata.Title),
	))

	// Topic
	rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top,
		labelStyle.Render("Topic:"),
		"  ",
		valueStyle.Render(rec.Metadata.Topic),
	))

	// Presenter
	if rec.Metadata.Presenter != "" {
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top,
			labelStyle.Render("Presenter:"),
			"  ",
			valueStyle.Render(rec.Metadata.Presenter),
		))
	}

	// Divider
	rows = append(rows, "")
	rows = append(rows, dividerStyle.Render(strings.Repeat("─", 62)))
	rows = append(rows, "")

	// Date
	rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top,
		labelStyle.Render("Date:"),
		"  ",
		valueStyle.Render(rec.StartTime.Format("Monday, January 2, 2006")),
	))

	// Duration
	durationStr := models.FormatDuration(rec.Duration)
	rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top,
		labelStyle.Render("Duration:"),
		"  ",
		highlightStyle.Render(durationStr),
	))

	// Total size
	totalSize := models.FormatFileSize(rec.Files.TotalSize)
	rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top,
		labelStyle.Render("Total Size:"),
		"  ",
		valueStyle.Render(totalSize),
	))

	// Divider
	rows = append(rows, "")
	rows = append(rows, dividerStyle.Render(strings.Repeat("─", 62)))
	rows = append(rows, "")

	// Files section
	fileStyle := lipgloss.NewStyle().Foreground(ColorGray)
	if rec.Files.MergedFile != "" {
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top,
			labelStyle.Render("Merged:"),
			"  ",
			fileStyle.Render(filepath.Base(rec.Files.MergedFile)+" ("+models.FormatFileSize(rec.Files.MergedSize)+")"),
		))
	}
	if rec.Files.VerticalFile != "" {
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top,
			labelStyle.Render("Vertical:"),
			"  ",
			fileStyle.Render(filepath.Base(rec.Files.VerticalFile)+" ("+models.FormatFileSize(rec.Files.VerticalSize)+")"),
		))
	}

	// Divider
	rows = append(rows, "")
	rows = append(rows, dividerStyle.Render(strings.Repeat("─", 62)))
	rows = append(rows, "")

	// Description
	rows = append(rows, labelStyle.Render("Description:"))
	desc := rec.Metadata.Description
	if desc == "" {
		desc = "(no description)"
	}
	descStyle := lipgloss.NewStyle().
		Foreground(ColorWhite).
		Width(60).
		MarginLeft(2)
	rows = append(rows, descStyle.Render(desc))

	// YouTube section
	rows = append(rows, "")
	rows = append(rows, dividerStyle.Render(strings.Repeat("─", 62)))
	rows = append(rows, "")

	ytLabelStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Width(14).
		Align(lipgloss.Right)

	if rec.Metadata.IsPublishedToYouTube() {
		// Show YouTube status
		ytStatusBadge := lipgloss.NewStyle().
			Background(ColorRed).
			Foreground(ColorWhite).
			Padding(0, 1).
			Bold(true).
			Render("▶ YouTube")
		ytStatusRow := lipgloss.NewStyle().Align(lipgloss.Center).Width(62).Render(ytStatusBadge)
		rows = append(rows, ytStatusRow)
		rows = append(rows, "")

		yt := rec.Metadata.YouTube

		// Video URL
		linkStyle := lipgloss.NewStyle().
			Foreground(ColorBlue).
			Underline(true)
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top,
			ytLabelStyle.Render("URL:"),
			"  ",
			linkStyle.Render(yt.VideoURL),
		))

		// Privacy
		privacyStyle := lipgloss.NewStyle().Foreground(ColorOrange).Bold(true)
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top,
			ytLabelStyle.Render("Privacy:"),
			"  ",
			privacyStyle.Render(yt.Privacy),
		))

		// Playlist
		if yt.PlaylistName != "" {
			rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top,
				ytLabelStyle.Render("Playlist:"),
				"  ",
				valueStyle.Render(yt.PlaylistName),
			))
		}

		// Upload date
		if yt.UploadedAt != "" {
			uploadTime, _ := time.Parse(time.RFC3339, yt.UploadedAt)
			rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top,
				ytLabelStyle.Render("Uploaded:"),
				"  ",
				valueStyle.Render(uploadTime.Format("Jan 2, 2006 15:04")),
			))
		}
	} else {
		// Not on YouTube
		ytStatusStyle := lipgloss.NewStyle().
			Foreground(ColorGray).
			Italic(true).
			Align(lipgloss.Center).
			Width(62)
		rows = append(rows, ytStatusStyle.Render("Not published to YouTube"))
	}

	// Success/Error messages
	if h.editSuccess != "" || h.youtubeActionSuccess != "" {
		successStyle := lipgloss.NewStyle().
			Foreground(ColorGreen).
			Bold(true).
			Align(lipgloss.Center).
			Width(62)
		rows = append(rows, "")
		msg := h.editSuccess
		if h.youtubeActionSuccess != "" {
			msg = h.youtubeActionSuccess
		}
		rows = append(rows, successStyle.Render(msg))
	}

	if h.youtubeActionError != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(ColorRed).
			Bold(true).
			Align(lipgloss.Center).
			Width(62)
		rows = append(rows, "")
		rows = append(rows, errorStyle.Render(h.youtubeActionError))
	}

	content := containerStyle.Render(lipgloss.JoinVertical(lipgloss.Left, rows...))

	// Help text - changes based on YouTube status
	helpStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Italic(true)

	var helpText string
	if rec.Metadata.IsPublishedToYouTube() {
		helpText = "e: Edit • p: Change Privacy • x: Delete from YouTube • Esc: Back"
	} else {
		helpText = "e: Edit • u: Upload to YouTube • Esc: Back"
	}

	mainSection := lipgloss.JoinVertical(
		lipgloss.Center,
		header,
		"",
		content,
	)

	centeredMain := lipgloss.Place(
		h.width,
		h.height-2,
		lipgloss.Center,
		lipgloss.Top,
		mainSection,
	)

	helpFooter := lipgloss.NewStyle().
		Width(h.width).
		Align(lipgloss.Center)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		centeredMain,
		helpFooter.Render(helpStyle.Render(helpText)),
	)
}

// renderDeleteConfirmView renders the delete confirmation dialog
func (h *HistoryModel) renderDeleteConfirmView() string {
	if h.deleteConfirmRecording == nil {
		return "No recording selected"
	}

	rec := h.deleteConfirmRecording
	header := RenderHeader("Delete Recording")

	// Styles
	warningStyle := lipgloss.NewStyle().
		Foreground(ColorRed).
		Bold(true).
		Align(lipgloss.Center)

	labelStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Width(14).
		Align(lipgloss.Right)

	valueStyle := lipgloss.NewStyle().
		Foreground(ColorWhite).
		Bold(true)

	containerStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorRed).
		Padding(1, 3).
		Width(70)

	// Build confirmation dialog
	var rows []string

	// Warning message
	rows = append(rows, warningStyle.Width(62).Render("⚠ DELETE RECORDING ⚠"))
	rows = append(rows, "")
	rows = append(rows, warningStyle.Width(62).Render("This action cannot be undone!"))
	rows = append(rows, "")

	// Recording details
	rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top,
		labelStyle.Render("Folder:"),
		"  ",
		valueStyle.Render(rec.Metadata.FolderName),
	))

	rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top,
		labelStyle.Render("Title:"),
		"  ",
		valueStyle.Render(rec.Metadata.Title),
	))

	rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top,
		labelStyle.Render("Date:"),
		"  ",
		valueStyle.Render(rec.StartTime.Format("2006-01-02")),
	))

	rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top,
		labelStyle.Render("Size:"),
		"  ",
		valueStyle.Render(models.FormatFileSize(rec.Files.TotalSize)),
	))

	rows = append(rows, "")

	// Error message if any
	if h.deleteError != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(ColorRed).
			Bold(true).
			Align(lipgloss.Center).
			Width(62)
		rows = append(rows, errorStyle.Render(h.deleteError))
		rows = append(rows, "")
	}

	// Confirmation prompt
	promptStyle := lipgloss.NewStyle().
		Foreground(ColorOrange).
		Bold(true).
		Align(lipgloss.Center).
		Width(62)
	rows = append(rows, promptStyle.Render("Are you sure you want to delete this recording?"))
	rows = append(rows, "")

	// Buttons
	yesStyle := lipgloss.NewStyle().
		Foreground(ColorRed).
		Bold(true).
		Padding(0, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorRed)

	noStyle := lipgloss.NewStyle().
		Foreground(ColorGreen).
		Bold(true).
		Padding(0, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorGreen)

	buttons := lipgloss.JoinHorizontal(lipgloss.Center,
		yesStyle.Render("Y - Yes, Delete"),
		"    ",
		noStyle.Render("N - No, Cancel"),
	)
	buttonRow := lipgloss.NewStyle().Width(62).Align(lipgloss.Center).Render(buttons)
	rows = append(rows, buttonRow)

	content := containerStyle.Render(lipgloss.JoinVertical(lipgloss.Left, rows...))

	// Help text
	helpStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Italic(true)

	mainSection := lipgloss.JoinVertical(
		lipgloss.Center,
		header,
		"",
		content,
	)

	centeredMain := lipgloss.Place(
		h.width,
		h.height-2,
		lipgloss.Center,
		lipgloss.Top,
		mainSection,
	)

	helpFooter := lipgloss.NewStyle().
		Width(h.width).
		Align(lipgloss.Center)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		centeredMain,
		helpFooter.Render(helpStyle.Render("Y: Confirm Delete • N/Esc: Cancel")),
	)
}

// renderEditView renders the edit view for a selected recording
func (h *HistoryModel) renderEditView() string {
	if h.selectedRecording == nil {
		return "No recording selected"
	}

	rec := h.selectedRecording
	header := RenderHeader("Edit Recording")

	// Styles
	labelStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Width(14).
		Align(lipgloss.Right)

	focusedLabelStyle := lipgloss.NewStyle().
		Foreground(ColorOrange).
		Bold(true).
		Width(14).
		Align(lipgloss.Right)

	infoStyle := lipgloss.NewStyle().
		Foreground(ColorBlue).
		Bold(true)

	containerStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorOrange).
		Padding(1, 3).
		Width(70)

	dividerStyle := lipgloss.NewStyle().
		Foreground(ColorGray)

	inputBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorGray).
		Padding(0, 1).
		Width(44)

	focusedInputBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorOrange).
		Padding(0, 1).
		Width(44)

	// Build edit form
	var rows []string

	// Folder (read-only)
	rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top,
		labelStyle.Render("Folder:"),
		"  ",
		infoStyle.Render(rec.Metadata.FolderName),
	))

	// Date (read-only)
	rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top,
		labelStyle.Render("Date:"),
		"  ",
		infoStyle.Render(rec.StartTime.Format("2006-01-02")),
	))

	// Duration (read-only)
	rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top,
		labelStyle.Render("Duration:"),
		"  ",
		infoStyle.Render(models.FormatDuration(rec.Duration)),
	))

	// Divider
	rows = append(rows, "")
	rows = append(rows, dividerStyle.Render(strings.Repeat("─", 62)))
	rows = append(rows, "")

	// Editable fields
	// Title
	titleLabel := labelStyle.Render("Title:")
	titleBoxStyle := inputBoxStyle
	if h.editFocusField == 0 {
		titleLabel = focusedLabelStyle.Render("Title:")
		titleBoxStyle = focusedInputBoxStyle
	}
	rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top,
		titleLabel,
		"  ",
		titleBoxStyle.Render(h.editFields.title.View()),
	))

	// Description
	descLabel := labelStyle.Render("Description:")
	descBoxStyle := inputBoxStyle.Height(5)
	if h.editFocusField == 1 {
		descLabel = focusedLabelStyle.Render("Description:")
		descBoxStyle = focusedInputBoxStyle.Height(5)
	}
	rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top,
		descLabel,
		"  ",
		descBoxStyle.Render(h.editFields.description.View()),
	))

	// Topic
	topicLabel := labelStyle.Render("Topic:")
	if h.editFocusField == 2 {
		topicLabel = focusedLabelStyle.Render("Topic:")
	}
	var topicOptions []string
	for i, topic := range h.topics {
		topicStyle := lipgloss.NewStyle().
			Padding(0, 1).
			Margin(0, 1)

		if i == h.editTopicIndex {
			if h.editFocusField == 2 {
				topicStyle = topicStyle.
					Background(ColorOrange).
					Foreground(lipgloss.Color("#000000")).
					Bold(true)
			} else {
				topicStyle = topicStyle.
					Background(ColorGray).
					Foreground(ColorWhite)
			}
		} else {
			topicStyle = topicStyle.
				Foreground(ColorGray)
		}
		topicOptions = append(topicOptions, topicStyle.Render(topic.Name))
	}
	rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top,
		topicLabel,
		"  ",
		lipgloss.JoinHorizontal(lipgloss.Center, topicOptions...),
	))

	// Presenter
	presenterLabel := labelStyle.Render("Presenter:")
	presenterBoxStyle := inputBoxStyle
	if h.editFocusField == 3 {
		presenterLabel = focusedLabelStyle.Render("Presenter:")
		presenterBoxStyle = focusedInputBoxStyle
	}
	rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top,
		presenterLabel,
		"  ",
		presenterBoxStyle.Render(h.editFields.presenter.View()),
	))

	// Error/Success messages
	if h.editError != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(ColorRed).
			Bold(true).
			Align(lipgloss.Center).
			Width(62)
		rows = append(rows, "")
		rows = append(rows, errorStyle.Render("Error: "+h.editError))
	}

	if h.editSuccess != "" {
		successStyle := lipgloss.NewStyle().
			Foreground(ColorGreen).
			Bold(true).
			Align(lipgloss.Center).
			Width(62)
		rows = append(rows, "")
		rows = append(rows, successStyle.Render(h.editSuccess))
	}

	if h.isSaving {
		savingStyle := lipgloss.NewStyle().
			Foreground(ColorOrange).
			Bold(true).
			Align(lipgloss.Center).
			Width(62)
		rows = append(rows, "")
		rows = append(rows, savingStyle.Render("Saving..."))
	}

	content := containerStyle.Render(lipgloss.JoinVertical(lipgloss.Left, rows...))

	// Help text
	helpStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Italic(true)

	mainSection := lipgloss.JoinVertical(
		lipgloss.Center,
		header,
		"",
		content,
	)

	centeredMain := lipgloss.Place(
		h.width,
		h.height-2,
		lipgloss.Center,
		lipgloss.Top,
		mainSection,
	)

	helpFooter := lipgloss.NewStyle().
		Width(h.width).
		Align(lipgloss.Center)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		centeredMain,
		helpFooter.Render(helpStyle.Render("Tab: Next • ←/→: Topic • Ctrl+S: Save • Esc: Cancel")),
	)
}

// renderScrollBar renders a visual scroll indicator
func (h *HistoryModel) renderScrollBar() string {
	if len(h.recordings) == 0 {
		return ""
	}

	visibleCount := h.getVisibleCount()
	totalEntries := len(h.recordings)

	barHeight := h.height - 16
	if barHeight < 5 {
		barHeight = 5
	}

	if totalEntries <= visibleCount {
		return ""
	}

	thumbSize := (visibleCount * barHeight) / totalEntries
	if thumbSize < 1 {
		thumbSize = 1
	}
	if thumbSize > barHeight {
		thumbSize = barHeight
	}

	thumbPos := (h.cursor * (barHeight - thumbSize)) / (totalEntries - 1)
	if thumbPos < 0 {
		thumbPos = 0
	}
	if thumbPos > barHeight-thumbSize {
		thumbPos = barHeight - thumbSize
	}

	var sb strings.Builder
	trackStyle := lipgloss.NewStyle().Foreground(ColorGray)
	thumbStyle := lipgloss.NewStyle().Foreground(ColorOrange)

	for i := 0; i < barHeight; i++ {
		if i >= thumbPos && i < thumbPos+thumbSize {
			sb.WriteString(thumbStyle.Render("┃"))
		} else {
			sb.WriteString(trackStyle.Render("│"))
		}
		if i < barHeight-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// renderScrollableTable renders the visible portion of the recordings table
func (h *HistoryModel) renderScrollableTable() string {
	visibleCount := h.getVisibleCount()
	totalEntries := len(h.recordings)

	startIdx := h.cursor - visibleCount/2
	if startIdx < 0 {
		startIdx = 0
	}
	endIdx := startIdx + visibleCount
	if endIdx > totalEntries {
		endIdx = totalEntries
		startIdx = endIdx - visibleCount
		if startIdx < 0 {
			startIdx = 0
		}
	}

	visibleRecordings := h.recordings[startIdx:endIdx]

	// Column headers
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorOrange).
		Width(12).
		Align(lipgloss.Left)

	cellStyle := lipgloss.NewStyle().
		Width(12).
		Align(lipgloss.Left)

	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#000000")).
		Background(ColorOrange).
		Width(12).
		Align(lipgloss.Left)

	descStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Width(56).
		Align(lipgloss.Left)

	selectedDescStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#000000")).
		Background(ColorOrange).
		Width(56).
		Align(lipgloss.Left)

	header := lipgloss.JoinHorizontal(lipgloss.Top,
		headerStyle.Width(12).Render("Topic"),
		headerStyle.Width(12).Render("Date"),
		headerStyle.Width(10).Render("Duration"),
		headerStyle.Width(10).Render("Size"),
	)

	var rows []string
	for i, rec := range visibleRecordings {
		absoluteIdx := startIdx + i
		isSelected := absoluteIdx == h.cursor

		topic := truncateStr(rec.Metadata.Topic, 10)
		dateStr := rec.StartTime.Format("2006-01-02")
		duration := models.FormatDuration(rec.Duration)
		size := models.FormatFileSize(rec.Files.TotalSize)
		folder := rec.Metadata.FolderName

		var row1 string
		if isSelected {
			row1 = lipgloss.JoinHorizontal(lipgloss.Top,
				selectedStyle.Width(12).Render(topic),
				selectedStyle.Width(12).Render(dateStr),
				selectedStyle.Width(10).Render(duration),
				selectedStyle.Width(10).Render(size),
			)
		} else {
			row1 = lipgloss.JoinHorizontal(lipgloss.Top,
				cellStyle.Width(12).Render(topic),
				cellStyle.Width(12).Render(dateStr),
				cellStyle.Width(10).Render(duration),
				cellStyle.Width(10).Render(size),
			)
		}

		var row2 string
		if isSelected {
			row2 = selectedDescStyle.Render("  📁 " + folder)
		} else {
			row2 = descStyle.Render("  📁 " + folder)
		}

		rows = append(rows, row1, row2)

		if i < len(visibleRecordings)-1 {
			sep := lipgloss.NewStyle().
				Foreground(ColorGray).
				Render(strings.Repeat("─", 56))
			rows = append(rows, sep)
		}
	}

	var topIndicator, bottomIndicator string
	indicatorStyle := lipgloss.NewStyle().
		Foreground(ColorOrange).
		Bold(true).
		Align(lipgloss.Center).
		Width(56)

	if startIdx > 0 {
		topIndicator = indicatorStyle.Render(fmt.Sprintf("↑ %d more recordings above", startIdx))
	}
	if endIdx < totalEntries {
		bottomIndicator = indicatorStyle.Render(fmt.Sprintf("↓ %d more recordings below", totalEntries-endIdx))
	}

	tableContent := lipgloss.JoinVertical(lipgloss.Left, append([]string{header, ""}, rows...)...)

	if topIndicator != "" {
		tableContent = lipgloss.JoinVertical(lipgloss.Left, topIndicator, "", tableContent)
	}
	if bottomIndicator != "" {
		tableContent = lipgloss.JoinVertical(lipgloss.Left, tableContent, "", bottomIndicator)
	}

	tableStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorOrange).
		Padding(1, 2)

	return tableStyle.Render(tableContent)
}

// loadRecordings loads all recordings from the screencasts folder
func (h *HistoryModel) loadRecordings() tea.Cmd {
	return func() tea.Msg {
		videosDir := config.GetDefaultVideosDir()

		// Check if directory exists
		if _, err := os.Stat(videosDir); os.IsNotExist(err) {
			return recordingsLoadedMsg{recordings: nil, err: nil}
		}

		entries, err := os.ReadDir(videosDir)
		if err != nil {
			return recordingsLoadedMsg{recordings: nil, err: err}
		}

		var recordings []models.RecordingInfo

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			folderPath := filepath.Join(videosDir, entry.Name())
			info, err := models.LoadRecordingInfo(folderPath)
			if err != nil {
				// Skip folders without valid recording.json
				continue
			}

			recordings = append(recordings, *info)
		}

		// Sort by date, newest first
		sort.Slice(recordings, func(i, j int) bool {
			return recordings[i].StartTime.After(recordings[j].StartTime)
		})

		return recordingsLoadedMsg{recordings: recordings, err: nil}
	}
}

// Helper function
func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// renderYouTubePrivacyView renders the YouTube privacy change view
func (h *HistoryModel) renderYouTubePrivacyView() string {
	if h.selectedRecording == nil || h.selectedRecording.Metadata.YouTube == nil {
		return "No recording selected"
	}

	rec := h.selectedRecording
	header := RenderHeader("Change YouTube Privacy")

	// Styles
	containerStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorOrange).
		Padding(1, 3).
		Width(60)

	labelStyle := lipgloss.NewStyle().
		Foreground(ColorGray)

	valueStyle := lipgloss.NewStyle().
		Foreground(ColorWhite).
		Bold(true)

	// Build rows
	var rows []string

	// Title
	titleBadge := lipgloss.NewStyle().
		Background(ColorBlue).
		Foreground(ColorWhite).
		Padding(0, 1).
		Bold(true).
		Render(rec.Metadata.Title)
	titleRow := lipgloss.NewStyle().Align(lipgloss.Center).Width(52).Render(titleBadge)
	rows = append(rows, titleRow)
	rows = append(rows, "")

	// Current privacy
	rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top,
		labelStyle.Render("Current Privacy: "),
		valueStyle.Render(rec.Metadata.YouTube.Privacy),
	))
	rows = append(rows, "")

	// Privacy options
	rows = append(rows, labelStyle.Render("Select new privacy:"))
	rows = append(rows, "")

	var privacyOptions []string
	for i, opt := range h.youtubePrivacyOptions {
		style := lipgloss.NewStyle().
			Padding(0, 2).
			Margin(0, 1)

		if i == h.youtubeSelectedPrivacy {
			style = style.
				Background(ColorOrange).
				Foreground(lipgloss.Color("#000000")).
				Bold(true)
		} else {
			style = style.
				Foreground(ColorGray)
		}
		privacyOptions = append(privacyOptions, style.Render(opt))
	}
	optionsRow := lipgloss.NewStyle().Width(52).Align(lipgloss.Center).Render(
		lipgloss.JoinHorizontal(lipgloss.Center, privacyOptions...),
	)
	rows = append(rows, optionsRow)

	// Error message
	if h.youtubeActionError != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(ColorRed).
			Bold(true).
			Align(lipgloss.Center).
			Width(52)
		rows = append(rows, "")
		rows = append(rows, errorStyle.Render(h.youtubeActionError))
	}

	// Loading
	if h.youtubeActionLoading {
		loadingStyle := lipgloss.NewStyle().
			Foreground(ColorOrange).
			Bold(true).
			Align(lipgloss.Center).
			Width(52)
		rows = append(rows, "")
		rows = append(rows, loadingStyle.Render("Updating privacy..."))
	}

	content := containerStyle.Render(lipgloss.JoinVertical(lipgloss.Left, rows...))

	// Help text
	helpStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Italic(true)

	mainSection := lipgloss.JoinVertical(
		lipgloss.Center,
		header,
		"",
		content,
	)

	centeredMain := lipgloss.Place(
		h.width,
		h.height-2,
		lipgloss.Center,
		lipgloss.Top,
		mainSection,
	)

	helpFooter := lipgloss.NewStyle().
		Width(h.width).
		Align(lipgloss.Center)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		centeredMain,
		helpFooter.Render(helpStyle.Render("←/→: Select • Enter: Confirm • Esc: Cancel")),
	)
}

// renderYouTubeDeleteConfirmView renders the YouTube delete confirmation view
func (h *HistoryModel) renderYouTubeDeleteConfirmView() string {
	if h.selectedRecording == nil || h.selectedRecording.Metadata.YouTube == nil {
		return "No recording selected"
	}

	rec := h.selectedRecording
	header := RenderHeader("Delete from YouTube")

	// Styles
	containerStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorRed).
		Padding(1, 3).
		Width(60)

	labelStyle := lipgloss.NewStyle().
		Foreground(ColorGray)

	valueStyle := lipgloss.NewStyle().
		Foreground(ColorWhite).
		Bold(true)

	// Build rows
	var rows []string

	// Warning icon
	warningBadge := lipgloss.NewStyle().
		Foreground(ColorRed).
		Bold(true).
		Render("⚠ DELETE VIDEO FROM YOUTUBE")
	warningRow := lipgloss.NewStyle().Align(lipgloss.Center).Width(52).Render(warningBadge)
	rows = append(rows, warningRow)
	rows = append(rows, "")

	// Title
	rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top,
		labelStyle.Render("Title: "),
		valueStyle.Render(rec.Metadata.Title),
	))

	// URL
	linkStyle := lipgloss.NewStyle().Foreground(ColorBlue).Underline(true)
	rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top,
		labelStyle.Render("URL: "),
		linkStyle.Render(rec.Metadata.YouTube.VideoURL),
	))
	rows = append(rows, "")

	// Warning message
	warningStyle := lipgloss.NewStyle().
		Foreground(ColorOrange).
		Bold(true).
		Align(lipgloss.Center).
		Width(52)
	rows = append(rows, warningStyle.Render("This action cannot be undone!"))
	rows = append(rows, warningStyle.Render("The video will be permanently deleted from YouTube."))
	rows = append(rows, "")

	// Error message
	if h.youtubeActionError != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(ColorRed).
			Bold(true).
			Align(lipgloss.Center).
			Width(52)
		rows = append(rows, errorStyle.Render(h.youtubeActionError))
		rows = append(rows, "")
	}

	// Loading
	if h.youtubeActionLoading {
		loadingStyle := lipgloss.NewStyle().
			Foreground(ColorOrange).
			Bold(true).
			Align(lipgloss.Center).
			Width(52)
		rows = append(rows, loadingStyle.Render("Deleting from YouTube..."))
		rows = append(rows, "")
	}

	// Buttons
	yesStyle := lipgloss.NewStyle().
		Foreground(ColorRed).
		Bold(true).
		Padding(0, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorRed)

	noStyle := lipgloss.NewStyle().
		Foreground(ColorGreen).
		Bold(true).
		Padding(0, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorGreen)

	buttons := lipgloss.JoinHorizontal(lipgloss.Center,
		yesStyle.Render("Y - Yes, Delete"),
		"    ",
		noStyle.Render("N - No, Cancel"),
	)
	buttonRow := lipgloss.NewStyle().Width(52).Align(lipgloss.Center).Render(buttons)
	rows = append(rows, buttonRow)

	content := containerStyle.Render(lipgloss.JoinVertical(lipgloss.Left, rows...))

	// Help text
	helpStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Italic(true)

	mainSection := lipgloss.JoinVertical(
		lipgloss.Center,
		header,
		"",
		content,
	)

	centeredMain := lipgloss.Place(
		h.width,
		h.height-2,
		lipgloss.Center,
		lipgloss.Top,
		mainSection,
	)

	helpFooter := lipgloss.NewStyle().
		Width(h.width).
		Align(lipgloss.Center)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		centeredMain,
		helpFooter.Render(helpStyle.Render("Y: Confirm Delete • N/Esc: Cancel")),
	)
}

// Message types
type recordingsLoadedMsg struct {
	recordings []models.RecordingInfo
	err        error
}

type recordingSavedMsg struct {
	err error
}

// backToMenuMsg signals returning to the main menu
type backToMenuMsg struct{}

// YouTube action messages
type youtubePrivacyChangedMsg struct {
	newPrivacy string
	err        error
}

type youtubeVideoDeletedMsg struct {
	err error
}

type startYouTubeUploadMsg struct {
	recording *models.RecordingInfo
	videoPath string
}
