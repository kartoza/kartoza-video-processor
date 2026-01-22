package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kartoza/kartoza-video-processor/internal/config"
	"github.com/kartoza/kartoza-video-processor/internal/models"
	"github.com/kartoza/kartoza-video-processor/internal/syndication"
)

// SyndicationPostStep represents the current step in the posting flow
type SyndicationPostStep int

const (
	SyndicationPostStepSelect SyndicationPostStep = iota
	SyndicationPostStepPreview
	SyndicationPostStepPosting
	SyndicationPostStepResults
)

// SyndicationPostModel handles posting to syndication platforms
type SyndicationPostModel struct {
	width  int
	height int

	step SyndicationPostStep

	// Recording being syndicated
	metadata     *models.RecordingMetadata
	recordingDir string

	// Account selection
	accounts        []syndication.Account
	selectedIndices map[int]bool

	// Custom message
	customMessage textinput.Model

	// Posting state
	isPosting bool
	results   []syndication.PostResult
	currentPostIdx int

	// Config
	cfg *config.Config
}

// NewSyndicationPostModel creates a new syndication post model
func NewSyndicationPostModel(metadata *models.RecordingMetadata, recordingDir string) *SyndicationPostModel {
	cfg, _ := config.Load()

	customMsg := textinput.New()
	customMsg.Placeholder = "Optional: Add a custom message..."
	customMsg.CharLimit = 280
	customMsg.Width = 60

	m := &SyndicationPostModel{
		metadata:        metadata,
		recordingDir:    recordingDir,
		cfg:             cfg,
		accounts:        cfg.Syndication.GetEnabledAccounts(),
		selectedIndices: make(map[int]bool),
		customMessage:   customMsg,
	}

	// Pre-select default accounts
	defaults := cfg.Syndication.GetDefaultAccounts()
	defaultIDs := make(map[string]bool)
	for _, acc := range defaults {
		defaultIDs[acc.ID] = true
	}

	for i, acc := range m.accounts {
		if defaultIDs[acc.ID] {
			m.selectedIndices[i] = true
		}
	}

	return m
}

// Init initializes the model
func (m *SyndicationPostModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages
func (m *SyndicationPostModel) Update(msg tea.Msg) (*SyndicationPostModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case syndicationPostProgressMsg:
		m.currentPostIdx = msg.index
		return m, nil

	case syndicationPostCompleteMsg:
		m.isPosting = false
		m.results = msg.results
		m.step = SyndicationPostStepResults

		// Save results to metadata
		manager := syndication.NewManager(&m.cfg.Syndication, config.GetConfigDir())
		manager.RecordResults(m.metadata, m.results)

		return m, nil
	}

	// Update custom message input
	if m.step == SyndicationPostStepSelect || m.step == SyndicationPostStepPreview {
		m.customMessage, cmd = m.customMessage.Update(msg)
	}

	return m, cmd
}

func (m *SyndicationPostModel) handleKeyMsg(msg tea.KeyMsg) (*SyndicationPostModel, tea.Cmd) {
	switch m.step {
	case SyndicationPostStepSelect:
		return m.handleSelectKeys(msg)
	case SyndicationPostStepPreview:
		return m.handlePreviewKeys(msg)
	case SyndicationPostStepResults:
		return m.handleResultsKeys(msg)
	}
	return m, nil
}

func (m *SyndicationPostModel) handleSelectKeys(msg tea.KeyMsg) (*SyndicationPostModel, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		// Focus previous account or custom message field
	case "down", "j":
		// Focus next account or custom message field
	case " ", "x":
		// Toggle selection at current index
		// Find focused account index
		for i := range m.accounts {
			// Simple toggle for now - could add cursor later
			if _, exists := m.selectedIndices[i]; exists {
				delete(m.selectedIndices, i)
			} else {
				m.selectedIndices[i] = true
			}
			break
		}
	case "a":
		// Select all
		for i := range m.accounts {
			m.selectedIndices[i] = true
		}
	case "n":
		// Select none
		m.selectedIndices = make(map[int]bool)
	case "tab":
		// Toggle focus between list and custom message
		if m.customMessage.Focused() {
			m.customMessage.Blur()
		} else {
			m.customMessage.Focus()
		}
	case "enter":
		if len(m.selectedIndices) > 0 {
			m.step = SyndicationPostStepPreview
		}
	case "esc", "q":
		return m, func() tea.Msg { return backToHistoryMsg{} }
	}
	return m, nil
}

func (m *SyndicationPostModel) handlePreviewKeys(msg tea.KeyMsg) (*SyndicationPostModel, tea.Cmd) {
	switch msg.String() {
	case "enter", "p":
		// Start posting
		m.step = SyndicationPostStepPosting
		m.isPosting = true
		return m, m.startPosting()
	case "esc", "backspace":
		m.step = SyndicationPostStepSelect
	case "e":
		// Edit custom message
		m.step = SyndicationPostStepSelect
		m.customMessage.Focus()
	}
	return m, nil
}

func (m *SyndicationPostModel) handleResultsKeys(msg tea.KeyMsg) (*SyndicationPostModel, tea.Cmd) {
	switch msg.String() {
	case "enter", "esc", "q":
		return m, func() tea.Msg { return backToHistoryMsg{} }
	case "r":
		// Retry failed posts
		m.retryFailed()
		return m, m.startPosting()
	}
	return m, nil
}

// Message types
type syndicationPostProgressMsg struct {
	index int
}

type syndicationPostCompleteMsg struct {
	results []syndication.PostResult
}

type backToHistoryMsg struct{}

func (m *SyndicationPostModel) startPosting() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		// Build content from metadata
		content := syndication.CreateContentFromMetadata(m.metadata, strings.TrimSpace(m.customMessage.Value()))

		// Get selected account IDs
		var accountIDs []string
		for i := range m.selectedIndices {
			if i < len(m.accounts) {
				accountIDs = append(accountIDs, m.accounts[i].ID)
			}
		}

		// Post to all selected accounts
		manager := syndication.NewManager(&m.cfg.Syndication, config.GetConfigDir())
		results := manager.PostToAccounts(ctx, accountIDs, content)

		return syndicationPostCompleteMsg{results: results}
	}
}

func (m *SyndicationPostModel) retryFailed() {
	// Clear current selections
	m.selectedIndices = make(map[int]bool)

	// Select only failed accounts
	for _, result := range m.results {
		if !result.Success {
			for i, acc := range m.accounts {
				if acc.ID == result.AccountID {
					m.selectedIndices[i] = true
					break
				}
			}
		}
	}
}

// View renders the syndication post screen
func (m *SyndicationPostModel) View() string {
	var content string

	switch m.step {
	case SyndicationPostStepSelect:
		content = m.renderSelect()
	case SyndicationPostStepPreview:
		content = m.renderPreview()
	case SyndicationPostStepPosting:
		content = m.renderPosting()
	case SyndicationPostStepResults:
		content = m.renderResults()
	}

	return content
}

func (m *SyndicationPostModel) renderSelect() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	subtitleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	unselectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	b.WriteString(titleStyle.Render("Syndicate: " + m.metadata.Title))
	b.WriteString("\n")
	if m.metadata.YouTube != nil {
		b.WriteString(subtitleStyle.Render(m.metadata.YouTube.VideoURL))
	}
	b.WriteString("\n\n")

	if len(m.accounts) == 0 {
		b.WriteString(dimStyle.Render("No syndication accounts configured."))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("Go to Options > Syndication Setup to add accounts."))
	} else {
		b.WriteString(subtitleStyle.Render("Select accounts to post to:"))
		b.WriteString("\n\n")

		// Group by platform
		grouped := make(map[syndication.PlatformType][]int)
		for i, acc := range m.accounts {
			grouped[acc.Platform] = append(grouped[acc.Platform], i)
		}

		for _, platform := range syndication.AllPlatforms() {
			indices, ok := grouped[platform]
			if !ok {
				continue
			}

			icon := syndication.PlatformIcon(platform)
			name := syndication.PlatformDisplayName(platform)
			b.WriteString(dimStyle.Render(fmt.Sprintf("%s %s:", icon, name)))
			b.WriteString("\n")

			for _, i := range indices {
				acc := m.accounts[i]
				checkbox := "[ ]"
				style := unselectedStyle
				if m.selectedIndices[i] {
					checkbox = "[x]"
					style = selectedStyle
				}

				b.WriteString(style.Render(fmt.Sprintf("  %s %s", checkbox, acc.GetDisplayName())))
				b.WriteString("\n")
			}
			b.WriteString("\n")
		}

		b.WriteString(subtitleStyle.Render("Custom message (optional):"))
		b.WriteString("\n")
		b.WriteString(m.customMessage.View())
		b.WriteString("\n\n")
	}

	selected := len(m.selectedIndices)
	b.WriteString(dimStyle.Render(fmt.Sprintf("Selected: %d accounts", selected)))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("space: toggle • a: all • n: none • tab: message • enter: preview • esc: cancel"))

	return b.String()
}

func (m *SyndicationPostModel) renderPreview() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	subtitleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	contentStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7")).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("8")).
		Padding(1)
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	b.WriteString(titleStyle.Render("Preview Post"))
	b.WriteString("\n\n")

	// Show preview content
	content := syndication.CreateContentFromMetadata(m.metadata, strings.TrimSpace(m.customMessage.Value()))
	preview, _ := syndication.BuildPreview(content, "")

	b.WriteString(contentStyle.Render(preview))
	b.WriteString("\n\n")

	// Show selected platforms
	b.WriteString(subtitleStyle.Render("Will post to:"))
	b.WriteString("\n")

	for i := range m.selectedIndices {
		if i < len(m.accounts) {
			acc := m.accounts[i]
			icon := syndication.PlatformIcon(acc.Platform)
			b.WriteString(fmt.Sprintf("  %s %s\n", icon, acc.GetDisplayName()))
		}
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("enter/p: post now • e: edit message • esc: back"))

	return b.String()
}

func (m *SyndicationPostModel) renderPosting() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11"))
	progressStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	b.WriteString(titleStyle.Render("Posting..."))
	b.WriteString("\n\n")

	total := len(m.selectedIndices)
	b.WriteString(progressStyle.Render(fmt.Sprintf("Posting to %d accounts...", total)))
	b.WriteString("\n\n")

	// Show which accounts we're posting to
	for i := range m.selectedIndices {
		if i < len(m.accounts) {
			acc := m.accounts[i]
			icon := syndication.PlatformIcon(acc.Platform)
			status := dimStyle.Render("pending")
			if i < m.currentPostIdx {
				status = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render("done")
			} else if i == m.currentPostIdx {
				status = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Render("posting...")
			}
			b.WriteString(fmt.Sprintf("  %s %s: %s\n", icon, acc.GetDisplayName(), status))
		}
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("Please wait..."))

	return b.String()
}

func (m *SyndicationPostModel) renderResults() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	failStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	urlStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	b.WriteString(titleStyle.Render("Syndication Results"))
	b.WriteString("\n\n")

	successCount := 0
	failCount := 0

	for _, result := range m.results {
		icon := syndication.PlatformIcon(result.Platform)

		if result.Success {
			successCount++
			b.WriteString(successStyle.Render(fmt.Sprintf("  %s %s: %s", icon, result.AccountName, result.Message)))
			b.WriteString("\n")
			if result.PostURL != "" {
				b.WriteString(urlStyle.Render(fmt.Sprintf("     %s", result.PostURL)))
				b.WriteString("\n")
			}
		} else {
			failCount++
			b.WriteString(failStyle.Render(fmt.Sprintf("  %s %s: %s", icon, result.AccountName, result.Message)))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	summary := fmt.Sprintf("Success: %d, Failed: %d", successCount, failCount)
	if failCount == 0 {
		b.WriteString(successStyle.Render(summary))
	} else {
		b.WriteString(failStyle.Render(summary))
	}
	b.WriteString("\n\n")

	if failCount > 0 {
		b.WriteString(dimStyle.Render("r: retry failed • enter: done"))
	} else {
		b.WriteString(dimStyle.Render("enter: done"))
	}

	return b.String()
}

// HasAccounts returns true if there are enabled accounts to post to
func (m *SyndicationPostModel) HasAccounts() bool {
	return len(m.accounts) > 0
}

// GetResults returns the posting results
func (m *SyndicationPostModel) GetResults() []syndication.PostResult {
	return m.results
}
