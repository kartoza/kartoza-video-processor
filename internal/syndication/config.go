package syndication

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Account represents a single platform account with its credentials
type Account struct {
	ID          string       `json:"id"`       // Unique identifier (generated)
	Name        string       `json:"name"`     // User-friendly name
	Platform    PlatformType `json:"platform"` // Platform type
	Enabled     bool         `json:"enabled"`  // Whether to include in syndication

	// Platform-specific settings (only relevant fields populated)

	// Mastodon
	InstanceURL  string `json:"instance_url,omitempty"` // e.g., "mastodon.social"
	ClientID     string `json:"client_id,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`

	// Bluesky
	Handle      string `json:"handle,omitempty"`       // e.g., "user.bsky.social"
	AppPassword string `json:"app_password,omitempty"` // App password (also used by WordPress)

	// LinkedIn
	// Uses ClientID/ClientSecret for OAuth2

	// Telegram
	BotToken string   `json:"bot_token,omitempty"`
	ChatIDs  []string `json:"chat_ids,omitempty"` // Multiple channels/groups

	// Signal
	SignalNumber string   `json:"signal_number,omitempty"` // Sender's number
	Recipients   []string `json:"recipients,omitempty"`    // Phone numbers or group IDs

	// ntfy.sh
	ServerURL   string `json:"server_url,omitempty"`   // Default: "https://ntfy.sh"
	Topic       string `json:"topic,omitempty"`        // Topic name
	AccessToken string `json:"access_token,omitempty"` // Optional auth token

	// Google Chat
	WebhookURL string `json:"webhook_url,omitempty"`

	// WordPress
	SiteURL    string `json:"site_url,omitempty"`
	Username   string `json:"username,omitempty"`
	PostStatus string `json:"post_status,omitempty"` // draft, publish, private
	CategoryID int    `json:"category_id,omitempty"`

	// Cached info
	DisplayName string `json:"display_name,omitempty"` // Fetched account/channel name
}

// IsConfigured returns true if the account has minimum required credentials
func (a *Account) IsConfigured() bool {
	switch a.Platform {
	case PlatformMastodon:
		return a.InstanceURL != "" && a.ClientID != "" && a.ClientSecret != ""
	case PlatformBluesky:
		return a.Handle != "" && a.AppPassword != ""
	case PlatformLinkedIn:
		return a.ClientID != "" && a.ClientSecret != ""
	case PlatformTelegram:
		return a.BotToken != "" && len(a.ChatIDs) > 0
	case PlatformSignal:
		return a.SignalNumber != "" && len(a.Recipients) > 0
	case PlatformNtfy:
		return a.Topic != ""
	case PlatformGoogleChat:
		return a.WebhookURL != ""
	case PlatformWordPress:
		return a.SiteURL != "" && a.Username != "" && a.AppPassword != ""
	default:
		return false
	}
}

// GetDisplayName returns a display name for the account
func (a *Account) GetDisplayName() string {
	if a.DisplayName != "" {
		return a.DisplayName
	}
	if a.Name != "" {
		return a.Name
	}
	// Platform-specific fallbacks
	switch a.Platform {
	case PlatformMastodon:
		if a.InstanceURL != "" {
			return "@" + a.InstanceURL
		}
	case PlatformBluesky:
		if a.Handle != "" {
			return "@" + a.Handle
		}
	case PlatformTelegram:
		if len(a.ChatIDs) > 0 {
			return "Bot: " + a.ChatIDs[0]
		}
	case PlatformNtfy:
		if a.Topic != "" {
			return "Topic: " + a.Topic
		}
	case PlatformWordPress:
		if a.SiteURL != "" {
			return a.SiteURL
		}
	}
	return string(a.Platform) + " Account"
}

// Config holds syndication settings
type Config struct {
	Accounts          []Account `json:"accounts,omitempty"`
	AutoPromptAfterYT bool      `json:"auto_prompt_after_yt,omitempty"` // Prompt after YouTube upload
	DefaultAccounts   []string  `json:"default_accounts,omitempty"`    // Account IDs to select by default
	PostTemplate      string    `json:"post_template,omitempty"`       // Custom template for posts
}

// DefaultConfig returns default syndication configuration
func DefaultConfig() Config {
	return Config{
		AutoPromptAfterYT: true,
		Accounts:          []Account{},
		PostTemplate:      "", // Empty means use default
	}
}

// IsConfigured returns true if at least one account is configured
func (c *Config) IsConfigured() bool {
	for _, acc := range c.Accounts {
		if acc.Enabled && acc.IsConfigured() {
			return true
		}
	}
	return false
}

// GetAccounts returns all accounts
func (c *Config) GetAccounts() []Account {
	return c.Accounts
}

// GetAccount returns an account by ID
func (c *Config) GetAccount(id string) *Account {
	for i := range c.Accounts {
		if c.Accounts[i].ID == id {
			return &c.Accounts[i]
		}
	}
	return nil
}

// GetAccountsByPlatform returns all accounts for a platform type
func (c *Config) GetAccountsByPlatform(platform PlatformType) []Account {
	var accounts []Account
	for _, acc := range c.Accounts {
		if acc.Platform == platform {
			accounts = append(accounts, acc)
		}
	}
	return accounts
}

// GetEnabledAccounts returns all enabled and configured accounts
func (c *Config) GetEnabledAccounts() []Account {
	var accounts []Account
	for _, acc := range c.Accounts {
		if acc.Enabled && acc.IsConfigured() {
			accounts = append(accounts, acc)
		}
	}
	return accounts
}

// GetDefaultAccounts returns accounts that should be selected by default
func (c *Config) GetDefaultAccounts() []Account {
	if len(c.DefaultAccounts) == 0 {
		return c.GetEnabledAccounts()
	}

	var accounts []Account
	for _, id := range c.DefaultAccounts {
		if acc := c.GetAccount(id); acc != nil && acc.Enabled && acc.IsConfigured() {
			accounts = append(accounts, *acc)
		}
	}
	return accounts
}

// AddAccount adds a new account
func (c *Config) AddAccount(account Account) {
	if account.ID == "" {
		account.ID = generateAccountID()
	}
	if account.Enabled == false {
		account.Enabled = true // Enable by default
	}
	c.Accounts = append(c.Accounts, account)
}

// UpdateAccount updates an existing account
func (c *Config) UpdateAccount(account Account) bool {
	for i := range c.Accounts {
		if c.Accounts[i].ID == account.ID {
			c.Accounts[i] = account
			return true
		}
	}
	return false
}

// RemoveAccount removes an account by ID
func (c *Config) RemoveAccount(id string) bool {
	for i := range c.Accounts {
		if c.Accounts[i].ID == id {
			c.Accounts = append(c.Accounts[:i], c.Accounts[i+1:]...)
			// Also remove from default accounts
			c.removeFromDefaultAccounts(id)
			return true
		}
	}
	return false
}

func (c *Config) removeFromDefaultAccounts(id string) {
	var filtered []string
	for _, accID := range c.DefaultAccounts {
		if accID != id {
			filtered = append(filtered, accID)
		}
	}
	c.DefaultAccounts = filtered
}

// SetDefaultAccount adds an account to the default selection
func (c *Config) SetDefaultAccount(id string, isDefault bool) {
	if isDefault {
		// Add if not already present
		for _, accID := range c.DefaultAccounts {
			if accID == id {
				return
			}
		}
		c.DefaultAccounts = append(c.DefaultAccounts, id)
	} else {
		c.removeFromDefaultAccounts(id)
	}
}

// generateAccountID generates a unique account ID
func generateAccountID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("synd_%x", b)
}

// Token represents stored OAuth2 tokens
type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenType    string `json:"token_type"`
	Expiry       string `json:"expiry,omitempty"` // RFC3339 format
}

// GetTokenPathForAccount returns the path to the token file for a specific account
func GetTokenPathForAccount(configDir, accountID string) string {
	return filepath.Join(configDir, fmt.Sprintf("syndication_token_%s.json", accountID))
}

// LoadTokenForAccount loads the OAuth token for a specific account
func LoadTokenForAccount(configDir, accountID string) (*Token, error) {
	tokenPath := GetTokenPathForAccount(configDir, accountID)
	data, err := os.ReadFile(tokenPath)
	if err != nil {
		return nil, err
	}

	var token Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, err
	}

	return &token, nil
}

// SaveTokenForAccount saves the OAuth token for a specific account
func SaveTokenForAccount(configDir, accountID string, token *Token) error {
	tokenPath := GetTokenPathForAccount(configDir, accountID)

	// Ensure directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return err
	}

	// Write with restricted permissions (owner only)
	return os.WriteFile(tokenPath, data, 0600)
}

// HasTokenForAccount returns true if a token file exists for a specific account
func HasTokenForAccount(configDir, accountID string) bool {
	tokenPath := GetTokenPathForAccount(configDir, accountID)
	_, err := os.Stat(tokenPath)
	return err == nil
}

// DeleteTokenForAccount removes the stored OAuth token for a specific account
func DeleteTokenForAccount(configDir, accountID string) error {
	tokenPath := GetTokenPathForAccount(configDir, accountID)
	err := os.Remove(tokenPath)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// BlueskySession stores Bluesky AT Protocol session data
type BlueskySession struct {
	AccessJwt  string `json:"accessJwt"`
	RefreshJwt string `json:"refreshJwt"`
	Handle     string `json:"handle"`
	DID        string `json:"did"`
}

// GetSessionPathForAccount returns the path to the session file for a specific account
func GetSessionPathForAccount(configDir, accountID string) string {
	return filepath.Join(configDir, fmt.Sprintf("syndication_session_%s.json", accountID))
}

// LoadSessionForAccount loads a session for a specific account
func LoadSessionForAccount(configDir, accountID string, session interface{}) error {
	sessionPath := GetSessionPathForAccount(configDir, accountID)
	data, err := os.ReadFile(sessionPath)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, session)
}

// SaveSessionForAccount saves a session for a specific account
func SaveSessionForAccount(configDir, accountID string, session interface{}) error {
	sessionPath := GetSessionPathForAccount(configDir, accountID)

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(sessionPath, data, 0600)
}

// DeleteSessionForAccount removes stored session data for a specific account
func DeleteSessionForAccount(configDir, accountID string) error {
	sessionPath := GetSessionPathForAccount(configDir, accountID)
	err := os.Remove(sessionPath)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
