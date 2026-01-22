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
	"github.com/kartoza/kartoza-video-processor/internal/syndication"

	// Import providers to register them
	_ "github.com/kartoza/kartoza-video-processor/internal/syndication/providers/bluesky"
	_ "github.com/kartoza/kartoza-video-processor/internal/syndication/providers/googlechat"
	_ "github.com/kartoza/kartoza-video-processor/internal/syndication/providers/linkedin"
	_ "github.com/kartoza/kartoza-video-processor/internal/syndication/providers/mastodon"
	_ "github.com/kartoza/kartoza-video-processor/internal/syndication/providers/ntfy"
	_ "github.com/kartoza/kartoza-video-processor/internal/syndication/providers/signal"
	_ "github.com/kartoza/kartoza-video-processor/internal/syndication/providers/telegram"
	_ "github.com/kartoza/kartoza-video-processor/internal/syndication/providers/wordpress"
)

// SyndicationSetupStep represents the current step in the setup
type SyndicationSetupStep int

const (
	SyndicationStepPlatformList SyndicationSetupStep = iota
	SyndicationStepAccountList
	SyndicationStepAccountAdd
	SyndicationStepAccountEdit
	SyndicationStepAccountDelete
	SyndicationStepAuthenticating
	SyndicationStepAuthCode
	SyndicationStepError
)

// SyndicationSetupModel handles syndication account setup
type SyndicationSetupModel struct {
	width  int
	height int

	step SyndicationSetupStep

	// Platform selection
	platforms           []syndication.PlatformType
	selectedPlatformIdx int

	// Account management
	accounts           []syndication.Account
	selectedAccountIdx int
	editingAccount     *syndication.Account

	// Form inputs (used for various platforms)
	accountName   textinput.Model
	instanceURL   textinput.Model
	clientID      textinput.Model
	clientSecret  textinput.Model
	handle        textinput.Model
	appPassword   textinput.Model
	botToken      textinput.Model
	chatIDs       textinput.Model // Comma-separated
	signalNumber  textinput.Model
	recipients    textinput.Model // Comma-separated
	topic         textinput.Model
	serverURL     textinput.Model
	webhookURL    textinput.Model
	siteURL       textinput.Model
	username      textinput.Model
	postStatus    textinput.Model
	accessToken   textinput.Model
	authCodeInput textinput.Model

	formFocusIdx int

	// Status
	errorMessage     string
	isAuthenticating bool
	authURL          string

	// Config
	cfg *config.Config
}

// NewSyndicationSetupModel creates a new syndication setup model
func NewSyndicationSetupModel() *SyndicationSetupModel {
	cfg, _ := config.Load()

	m := &SyndicationSetupModel{
		platforms: syndication.AllPlatforms(),
		cfg:       cfg,
		accounts:  cfg.Syndication.GetAccounts(),
	}

	// Initialize all text inputs
	m.initTextInputs()

	return m
}

func (m *SyndicationSetupModel) initTextInputs() {
	m.accountName = createSyndInput("Account Name", 100)
	m.instanceURL = createSyndInput("mastodon.social", 200)
	m.clientID = createSyndInput("Client ID", 200)
	m.clientSecret = createSyndInputPassword("Client Secret", 200)
	m.handle = createSyndInput("user.bsky.social", 100)
	m.appPassword = createSyndInputPassword("App Password", 100)
	m.botToken = createSyndInputPassword("Bot Token", 200)
	m.chatIDs = createSyndInput("Chat IDs (comma-separated)", 500)
	m.signalNumber = createSyndInput("+1234567890", 20)
	m.recipients = createSyndInput("Recipients (comma-separated)", 500)
	m.topic = createSyndInput("my-topic", 100)
	m.serverURL = createSyndInput("https://ntfy.sh", 200)
	m.webhookURL = createSyndInput("https://chat.googleapis.com/...", 500)
	m.siteURL = createSyndInput("https://example.com", 200)
	m.username = createSyndInput("admin", 100)
	m.postStatus = createSyndInput("draft", 20)
	m.accessToken = createSyndInputPassword("Access Token (optional)", 200)
	m.authCodeInput = createSyndInput("Paste authorization code here", 200)
}

func createSyndInput(placeholder string, charLimit int) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = charLimit
	ti.Width = 50
	return ti
}

func createSyndInputPassword(placeholder string, charLimit int) textinput.Model {
	ti := createSyndInput(placeholder, charLimit)
	ti.EchoMode = textinput.EchoPassword
	return ti
}

// Init initializes the model
func (m *SyndicationSetupModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages
func (m *SyndicationSetupModel) Update(msg tea.Msg) (*SyndicationSetupModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case syndicationAuthStartedMsg:
		m.isAuthenticating = true
		m.authURL = msg.authURL
		if strings.Contains(msg.authURL, "OAUTH_PENDING") {
			// Need to show auth code input
			m.step = SyndicationStepAuthCode
			m.authCodeInput.Focus()
		} else {
			m.step = SyndicationStepAuthenticating
		}
		return m, nil

	case syndicationAuthCompleteMsg:
		m.isAuthenticating = false
		if msg.err != nil {
			if strings.HasPrefix(msg.err.Error(), "OAUTH_PENDING") {
				// Show auth code entry screen
				m.step = SyndicationStepAuthCode
				m.authURL = strings.TrimPrefix(msg.err.Error(), "OAUTH_PENDING:")
				m.authCodeInput.Focus()
				return m, textinput.Blink
			}
			m.step = SyndicationStepError
			m.errorMessage = msg.err.Error()
		} else {
			m.step = SyndicationStepAccountList
			// Refresh accounts
			m.filterAccountsByPlatform()
		}
		return m, nil
	}

	return m, cmd
}

func (m *SyndicationSetupModel) handleKeyMsg(msg tea.KeyMsg) (*SyndicationSetupModel, tea.Cmd) {
	switch m.step {
	case SyndicationStepPlatformList:
		return m.handlePlatformListKeys(msg)
	case SyndicationStepAccountList:
		return m.handleAccountListKeys(msg)
	case SyndicationStepAccountAdd, SyndicationStepAccountEdit:
		return m.handleAccountFormKeys(msg)
	case SyndicationStepAccountDelete:
		return m.handleDeleteConfirmKeys(msg)
	case SyndicationStepAuthCode:
		return m.handleAuthCodeKeys(msg)
	case SyndicationStepError:
		if msg.String() == "enter" || msg.String() == "esc" {
			m.step = SyndicationStepAccountList
		}
	}
	return m, nil
}

func (m *SyndicationSetupModel) handlePlatformListKeys(msg tea.KeyMsg) (*SyndicationSetupModel, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.selectedPlatformIdx > 0 {
			m.selectedPlatformIdx--
		}
	case "down", "j":
		if m.selectedPlatformIdx < len(m.platforms)-1 {
			m.selectedPlatformIdx++
		}
	case "enter":
		// Show accounts for selected platform
		m.filterAccountsByPlatform()
		m.step = SyndicationStepAccountList
		m.selectedAccountIdx = 0
	case "esc", "q":
		return m, func() tea.Msg { return backToMenuMsg{} }
	}
	return m, nil
}

func (m *SyndicationSetupModel) handleAccountListKeys(msg tea.KeyMsg) (*SyndicationSetupModel, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.selectedAccountIdx > 0 {
			m.selectedAccountIdx--
		}
	case "down", "j":
		if m.selectedAccountIdx < len(m.accounts)-1 {
			m.selectedAccountIdx++
		}
	case "n", "a":
		// Add new account
		m.prepareNewAccount()
		m.step = SyndicationStepAccountAdd
		return m, textinput.Blink
	case "e":
		// Edit account
		if len(m.accounts) > 0 {
			m.prepareEditAccount()
			m.step = SyndicationStepAccountEdit
			return m, textinput.Blink
		}
	case "d", "delete":
		// Delete account
		if len(m.accounts) > 0 {
			m.step = SyndicationStepAccountDelete
		}
	case "c":
		// Connect/authenticate account
		if len(m.accounts) > 0 {
			return m, m.authenticateAccount()
		}
	case "t":
		// Toggle enabled
		if len(m.accounts) > 0 {
			m.toggleAccountEnabled()
		}
	case "esc", "backspace":
		m.step = SyndicationStepPlatformList
		m.refreshAllAccounts()
	case "q":
		return m, func() tea.Msg { return backToMenuMsg{} }
	}
	return m, nil
}

func (m *SyndicationSetupModel) handleAccountFormKeys(msg tea.KeyMsg) (*SyndicationSetupModel, tea.Cmd) {
	switch msg.String() {
	case "tab":
		m.nextFormField()
		return m, textinput.Blink
	case "shift+tab":
		m.prevFormField()
		return m, textinput.Blink
	case "enter":
		if m.saveAccount() {
			m.step = SyndicationStepAccountList
			m.filterAccountsByPlatform()
		}
		return m, nil
	case "esc":
		m.step = SyndicationStepAccountList
		return m, nil
	default:
		// Forward all other keys to the focused text input
		platform := m.platforms[m.selectedPlatformIdx]
		inputs := m.getFormInputsForPlatform(platform)
		if m.formFocusIdx < len(inputs) {
			var cmd tea.Cmd
			*inputs[m.formFocusIdx], cmd = (*inputs[m.formFocusIdx]).Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m *SyndicationSetupModel) handleDeleteConfirmKeys(msg tea.KeyMsg) (*SyndicationSetupModel, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.deleteSelectedAccount()
		m.step = SyndicationStepAccountList
	case "n", "N", "esc":
		m.step = SyndicationStepAccountList
	}
	return m, nil
}

func (m *SyndicationSetupModel) handleAuthCodeKeys(msg tea.KeyMsg) (*SyndicationSetupModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		code := strings.TrimSpace(m.authCodeInput.Value())
		if code != "" {
			return m, m.completeOAuth(code)
		}
	case "esc":
		m.step = SyndicationStepAccountList
		return m, nil
	default:
		// Forward to auth code input
		var cmd tea.Cmd
		m.authCodeInput, cmd = m.authCodeInput.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *SyndicationSetupModel) getFormInputsForPlatform(platform syndication.PlatformType) []*textinput.Model {
	switch platform {
	case syndication.PlatformMastodon:
		return []*textinput.Model{&m.accountName, &m.instanceURL, &m.clientID, &m.clientSecret}
	case syndication.PlatformBluesky:
		return []*textinput.Model{&m.accountName, &m.handle, &m.appPassword}
	case syndication.PlatformLinkedIn:
		return []*textinput.Model{&m.accountName, &m.clientID, &m.clientSecret}
	case syndication.PlatformTelegram:
		return []*textinput.Model{&m.accountName, &m.botToken, &m.chatIDs}
	case syndication.PlatformSignal:
		return []*textinput.Model{&m.accountName, &m.signalNumber, &m.recipients}
	case syndication.PlatformNtfy:
		return []*textinput.Model{&m.accountName, &m.topic, &m.serverURL, &m.accessToken}
	case syndication.PlatformGoogleChat:
		return []*textinput.Model{&m.accountName, &m.webhookURL}
	case syndication.PlatformWordPress:
		return []*textinput.Model{&m.accountName, &m.siteURL, &m.username, &m.appPassword, &m.postStatus}
	default:
		return []*textinput.Model{&m.accountName}
	}
}

func (m *SyndicationSetupModel) nextFormField() {
	platform := m.platforms[m.selectedPlatformIdx]
	inputs := m.getFormInputsForPlatform(platform)

	inputs[m.formFocusIdx].Blur()
	m.formFocusIdx = (m.formFocusIdx + 1) % len(inputs)
	inputs[m.formFocusIdx].Focus()
}

func (m *SyndicationSetupModel) prevFormField() {
	platform := m.platforms[m.selectedPlatformIdx]
	inputs := m.getFormInputsForPlatform(platform)

	inputs[m.formFocusIdx].Blur()
	m.formFocusIdx = (m.formFocusIdx - 1 + len(inputs)) % len(inputs)
	inputs[m.formFocusIdx].Focus()
}

func (m *SyndicationSetupModel) prepareNewAccount() {
	m.editingAccount = nil
	m.formFocusIdx = 0

	// Clear all inputs
	m.accountName.SetValue("")
	m.instanceURL.SetValue("")
	m.clientID.SetValue("")
	m.clientSecret.SetValue("")
	m.handle.SetValue("")
	m.appPassword.SetValue("")
	m.botToken.SetValue("")
	m.chatIDs.SetValue("")
	m.signalNumber.SetValue("")
	m.recipients.SetValue("")
	m.topic.SetValue("")
	m.serverURL.SetValue("https://ntfy.sh")
	m.webhookURL.SetValue("")
	m.siteURL.SetValue("")
	m.username.SetValue("")
	m.postStatus.SetValue("draft")
	m.accessToken.SetValue("")

	// Blur all inputs first
	platform := m.platforms[m.selectedPlatformIdx]
	inputs := m.getFormInputsForPlatform(platform)
	for _, input := range inputs {
		input.Blur()
	}
	// Focus the first one
	inputs[0].Focus()
}

func (m *SyndicationSetupModel) prepareEditAccount() {
	if m.selectedAccountIdx >= len(m.accounts) {
		return
	}
	acc := m.accounts[m.selectedAccountIdx]
	m.editingAccount = &acc
	m.formFocusIdx = 0

	// Populate inputs from account
	m.accountName.SetValue(acc.Name)
	m.instanceURL.SetValue(acc.InstanceURL)
	m.clientID.SetValue(acc.ClientID)
	m.clientSecret.SetValue(acc.ClientSecret)
	m.handle.SetValue(acc.Handle)
	m.appPassword.SetValue(acc.AppPassword)
	m.botToken.SetValue(acc.BotToken)
	m.chatIDs.SetValue(strings.Join(acc.ChatIDs, ", "))
	m.signalNumber.SetValue(acc.SignalNumber)
	m.recipients.SetValue(strings.Join(acc.Recipients, ", "))
	m.topic.SetValue(acc.Topic)
	m.serverURL.SetValue(acc.ServerURL)
	m.webhookURL.SetValue(acc.WebhookURL)
	m.siteURL.SetValue(acc.SiteURL)
	m.username.SetValue(acc.Username)
	m.postStatus.SetValue(acc.PostStatus)
	m.accessToken.SetValue(acc.AccessToken)

	// Blur all inputs first
	platform := m.platforms[m.selectedPlatformIdx]
	inputs := m.getFormInputsForPlatform(platform)
	for _, input := range inputs {
		input.Blur()
	}
	// Focus the first one
	inputs[0].Focus()
}

func (m *SyndicationSetupModel) saveAccount() bool {
	platform := m.platforms[m.selectedPlatformIdx]

	acc := syndication.Account{
		Name:         strings.TrimSpace(m.accountName.Value()),
		Platform:     platform,
		Enabled:      true,
		InstanceURL:  strings.TrimSpace(m.instanceURL.Value()),
		ClientID:     strings.TrimSpace(m.clientID.Value()),
		ClientSecret: strings.TrimSpace(m.clientSecret.Value()),
		Handle:       strings.TrimSpace(m.handle.Value()),
		AppPassword:  strings.TrimSpace(m.appPassword.Value()),
		BotToken:     strings.TrimSpace(m.botToken.Value()),
		SignalNumber: strings.TrimSpace(m.signalNumber.Value()),
		Topic:        strings.TrimSpace(m.topic.Value()),
		ServerURL:    strings.TrimSpace(m.serverURL.Value()),
		WebhookURL:   strings.TrimSpace(m.webhookURL.Value()),
		SiteURL:      strings.TrimSpace(m.siteURL.Value()),
		Username:     strings.TrimSpace(m.username.Value()),
		PostStatus:   strings.TrimSpace(m.postStatus.Value()),
		AccessToken:  strings.TrimSpace(m.accessToken.Value()),
	}

	// Parse comma-separated fields
	if chatIDsStr := strings.TrimSpace(m.chatIDs.Value()); chatIDsStr != "" {
		acc.ChatIDs = splitAndTrim(chatIDsStr)
	}
	if recipientsStr := strings.TrimSpace(m.recipients.Value()); recipientsStr != "" {
		acc.Recipients = splitAndTrim(recipientsStr)
	}

	if m.editingAccount != nil {
		acc.ID = m.editingAccount.ID
		m.cfg.Syndication.UpdateAccount(acc)
	} else {
		m.cfg.Syndication.AddAccount(acc)
	}

	return config.Save(m.cfg) == nil
}

func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func (m *SyndicationSetupModel) deleteSelectedAccount() {
	if m.selectedAccountIdx >= len(m.accounts) {
		return
	}
	acc := m.accounts[m.selectedAccountIdx]
	m.cfg.Syndication.RemoveAccount(acc.ID)
	_ = config.Save(m.cfg)
	m.filterAccountsByPlatform()
	if m.selectedAccountIdx >= len(m.accounts) && m.selectedAccountIdx > 0 {
		m.selectedAccountIdx--
	}
}

func (m *SyndicationSetupModel) toggleAccountEnabled() {
	if m.selectedAccountIdx >= len(m.accounts) {
		return
	}
	acc := m.accounts[m.selectedAccountIdx]
	acc.Enabled = !acc.Enabled
	m.cfg.Syndication.UpdateAccount(acc)
	_ = config.Save(m.cfg)
	m.accounts[m.selectedAccountIdx] = acc
}

func (m *SyndicationSetupModel) filterAccountsByPlatform() {
	platform := m.platforms[m.selectedPlatformIdx]
	m.accounts = m.cfg.Syndication.GetAccountsByPlatform(platform)
}

func (m *SyndicationSetupModel) refreshAllAccounts() {
	m.accounts = m.cfg.Syndication.GetAccounts()
}

// Message types for syndication auth
type syndicationAuthStartedMsg struct {
	authURL string
}

type syndicationAuthCompleteMsg struct {
	err error
}

func (m *SyndicationSetupModel) authenticateAccount() tea.Cmd {
	if m.selectedAccountIdx >= len(m.accounts) {
		return nil
	}

	acc := m.accounts[m.selectedAccountIdx]
	provider, ok := syndication.GetRegistry().Get(acc.Platform)
	if !ok {
		return nil
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		var authURL string
		err := provider.Authenticate(ctx, &acc, config.GetConfigDir(), func(url string) {
			authURL = url
		})

		if authURL != "" {
			return syndicationAuthStartedMsg{authURL: authURL}
		}

		return syndicationAuthCompleteMsg{err: err}
	}
}

func (m *SyndicationSetupModel) completeOAuth(code string) tea.Cmd {
	if m.selectedAccountIdx >= len(m.accounts) {
		return nil
	}

	acc := m.accounts[m.selectedAccountIdx]

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// For Mastodon, use the CompleteAuth method
		if acc.Platform == syndication.PlatformMastodon {
			provider, ok := syndication.GetRegistry().Get(acc.Platform)
			if !ok {
				return syndicationAuthCompleteMsg{err: fmt.Errorf("provider not found")}
			}
			// Type assert to access CompleteAuth
			if mp, ok := provider.(interface {
				CompleteAuth(ctx context.Context, account *syndication.Account, configDir, authCode string) error
			}); ok {
				err := mp.CompleteAuth(ctx, &acc, config.GetConfigDir(), code)
				return syndicationAuthCompleteMsg{err: err}
			}
		}

		return syndicationAuthCompleteMsg{err: fmt.Errorf("OAuth completion not supported for this platform")}
	}
}

// View renders the syndication setup screen
func (m *SyndicationSetupModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var content string
	var helpText string

	switch m.step {
	case SyndicationStepPlatformList:
		content = m.renderPlatformList()
		helpText = "up/down: select • enter: manage accounts • q: back"
	case SyndicationStepAccountList:
		content = m.renderAccountList()
		helpText = "n: add • e: edit • d: delete • c: connect • t: toggle • esc: back"
	case SyndicationStepAccountAdd:
		content = m.renderAccountForm("Add Account")
		helpText = "tab: next field • enter: save • esc: cancel"
	case SyndicationStepAccountEdit:
		content = m.renderAccountForm("Edit Account")
		helpText = "tab: next field • enter: save • esc: cancel"
	case SyndicationStepAccountDelete:
		content = m.renderDeleteConfirm()
		helpText = "y: yes, delete • n: no, cancel"
	case SyndicationStepAuthenticating:
		content = m.renderAuthenticating()
		helpText = "Waiting for authentication..."
	case SyndicationStepAuthCode:
		content = m.renderAuthCodeEntry()
		helpText = "enter: submit • esc: cancel"
	case SyndicationStepError:
		content = m.renderError()
		helpText = "enter: continue"
	}

	header := RenderHeader("Syndication Setup")

	// Center the content
	centeredContent := lipgloss.NewStyle().
		Width(HeaderWidth).
		Align(lipgloss.Center).
		Render(content)

	footer := RenderHelpFooter(helpText, m.width)

	return LayoutWithHeaderFooter(header, centeredContent, footer, m.width, m.height)
}

func (m *SyndicationSetupModel) renderPlatformList() string {
	var rows []string

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorBlue)
	selectedStyle := lipgloss.NewStyle().Foreground(ColorGreen).Bold(true)
	normalStyle := lipgloss.NewStyle().Foreground(ColorWhite)
	dimStyle := lipgloss.NewStyle().Foreground(ColorGray)

	rows = append(rows, titleStyle.Render("Select Platform"))
	rows = append(rows, dimStyle.Render("Configure accounts to announce your videos"))
	rows = append(rows, "")

	for i, platform := range m.platforms {
		icon := syndication.PlatformIcon(platform)
		name := syndication.PlatformDisplayName(platform)

		// Count accounts for this platform
		accounts := m.cfg.Syndication.GetAccountsByPlatform(platform)
		countStr := ""
		if len(accounts) > 0 {
			enabled := 0
			for _, a := range accounts {
				if a.Enabled {
					enabled++
				}
			}
			countStr = fmt.Sprintf(" (%d accounts, %d enabled)", len(accounts), enabled)
		}

		line := fmt.Sprintf("%s %s%s", icon, name, dimStyle.Render(countStr))

		if i == m.selectedPlatformIdx {
			rows = append(rows, selectedStyle.Render("> "+line))
		} else {
			rows = append(rows, normalStyle.Render("  "+line))
		}
	}

	return strings.Join(rows, "\n")
}

func (m *SyndicationSetupModel) renderAccountList() string {
	var rows []string

	platform := m.platforms[m.selectedPlatformIdx]
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorBlue)
	selectedStyle := lipgloss.NewStyle().Foreground(ColorGreen).Bold(true)
	normalStyle := lipgloss.NewStyle().Foreground(ColorWhite)
	dimStyle := lipgloss.NewStyle().Foreground(ColorGray)
	enabledStyle := lipgloss.NewStyle().Foreground(ColorGreen)
	disabledStyle := lipgloss.NewStyle().Foreground(ColorGray)

	icon := syndication.PlatformIcon(platform)
	name := syndication.PlatformDisplayName(platform)

	rows = append(rows, titleStyle.Render(fmt.Sprintf("%s %s Accounts", icon, name)))
	rows = append(rows, "")

	if len(m.accounts) == 0 {
		rows = append(rows, dimStyle.Render("No accounts configured"))
		rows = append(rows, "")
		rows = append(rows, dimStyle.Render("Press 'n' to add a new account"))
	} else {
		for i, acc := range m.accounts {
			status := enabledStyle.Render("[enabled]")
			if !acc.Enabled {
				status = disabledStyle.Render("[disabled]")
			}

			displayName := acc.GetDisplayName()
			line := fmt.Sprintf("%s %s", displayName, status)

			if i == m.selectedAccountIdx {
				rows = append(rows, selectedStyle.Render("> "+line))
			} else {
				rows = append(rows, normalStyle.Render("  "+line))
			}
		}
	}

	return strings.Join(rows, "\n")
}

func (m *SyndicationSetupModel) renderAccountForm(title string) string {
	var rows []string

	platform := m.platforms[m.selectedPlatformIdx]
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorBlue)
	labelStyle := lipgloss.NewStyle().Foreground(ColorBlue)
	labelActiveStyle := lipgloss.NewStyle().Foreground(ColorOrange).Bold(true)

	icon := syndication.PlatformIcon(platform)
	name := syndication.PlatformDisplayName(platform)

	rows = append(rows, titleStyle.Render(fmt.Sprintf("%s %s - %s", icon, name, title)))
	rows = append(rows, "")

	inputs := m.getFormInputsForPlatform(platform)
	labels := m.getFormLabelsForPlatform(platform)

	for i, input := range inputs {
		if i == m.formFocusIdx {
			rows = append(rows, labelActiveStyle.Render(labels[i]+":"))
		} else {
			rows = append(rows, labelStyle.Render(labels[i]+":"))
		}
		rows = append(rows, "  "+input.View())
		rows = append(rows, "")
	}

	return strings.Join(rows, "\n")
}

func (m *SyndicationSetupModel) getFormLabelsForPlatform(platform syndication.PlatformType) []string {
	switch platform {
	case syndication.PlatformMastodon:
		return []string{"Account Name", "Instance URL", "Client ID", "Client Secret"}
	case syndication.PlatformBluesky:
		return []string{"Account Name", "Handle", "App Password"}
	case syndication.PlatformLinkedIn:
		return []string{"Account Name", "Client ID", "Client Secret"}
	case syndication.PlatformTelegram:
		return []string{"Account Name", "Bot Token", "Chat IDs (comma-separated)"}
	case syndication.PlatformSignal:
		return []string{"Account Name", "Signal Number", "Recipients (comma-separated)"}
	case syndication.PlatformNtfy:
		return []string{"Account Name", "Topic", "Server URL", "Access Token (optional)"}
	case syndication.PlatformGoogleChat:
		return []string{"Account Name", "Webhook URL"}
	case syndication.PlatformWordPress:
		return []string{"Account Name", "Site URL", "Username", "App Password", "Post Status (draft/publish)"}
	default:
		return []string{"Account Name"}
	}
}

func (m *SyndicationSetupModel) renderDeleteConfirm() string {
	var rows []string

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorRed)
	normalStyle := lipgloss.NewStyle().Foreground(ColorWhite)
	warnStyle := lipgloss.NewStyle().Foreground(ColorOrange)

	if m.selectedAccountIdx < len(m.accounts) {
		acc := m.accounts[m.selectedAccountIdx]
		rows = append(rows, titleStyle.Render("Delete Account?"))
		rows = append(rows, "")
		rows = append(rows, normalStyle.Render(fmt.Sprintf("Are you sure you want to delete '%s'?", acc.GetDisplayName())))
		rows = append(rows, "")
		rows = append(rows, warnStyle.Render("This action cannot be undone."))
	}

	return strings.Join(rows, "\n")
}

func (m *SyndicationSetupModel) renderAuthenticating() string {
	var rows []string

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorOrange)
	urlStyle := lipgloss.NewStyle().Foreground(ColorBlue)
	dimStyle := lipgloss.NewStyle().Foreground(ColorGray)

	rows = append(rows, titleStyle.Render("Authenticating..."))
	rows = append(rows, "")

	if m.authURL != "" {
		rows = append(rows, "Please open this URL in your browser:")
		rows = append(rows, "")
		rows = append(rows, urlStyle.Render(m.authURL))
		rows = append(rows, "")
	}

	rows = append(rows, dimStyle.Render("Waiting for authentication..."))

	return strings.Join(rows, "\n")
}

func (m *SyndicationSetupModel) renderAuthCodeEntry() string {
	var rows []string

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorOrange)
	urlStyle := lipgloss.NewStyle().Foreground(ColorBlue)
	labelStyle := lipgloss.NewStyle().Foreground(ColorBlue)

	rows = append(rows, titleStyle.Render("Enter Authorization Code"))
	rows = append(rows, "")

	if m.authURL != "" {
		rows = append(rows, "Please visit this URL and authorize the app:")
		rows = append(rows, "")
		rows = append(rows, urlStyle.Render(m.authURL))
		rows = append(rows, "")
	}

	rows = append(rows, labelStyle.Render("Authorization Code:"))
	rows = append(rows, "  "+m.authCodeInput.View())

	return strings.Join(rows, "\n")
}

func (m *SyndicationSetupModel) renderError() string {
	var rows []string

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorRed)
	errorStyle := lipgloss.NewStyle().Foreground(ColorRed)

	rows = append(rows, titleStyle.Render("Error"))
	rows = append(rows, "")
	rows = append(rows, errorStyle.Render(m.errorMessage))

	return strings.Join(rows, "\n")
}
