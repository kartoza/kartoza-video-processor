package youtube

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PrivacyStatus represents YouTube video privacy settings
type PrivacyStatus string

const (
	PrivacyPublic   PrivacyStatus = "public"
	PrivacyUnlisted PrivacyStatus = "unlisted"
	PrivacyPrivate  PrivacyStatus = "private"
)

// Account represents a single YouTube account with its credentials
type Account struct {
	ID                 string        `json:"id"`                              // Unique identifier (generated)
	Name               string        `json:"name"`                            // User-friendly name for the account
	ClientID           string        `json:"client_id,omitempty"`
	ClientSecret       string        `json:"client_secret,omitempty"`
	DefaultPlaylistID  string        `json:"default_playlist_id,omitempty"`
	DefaultPlaylistName string       `json:"default_playlist_name,omitempty"` // For display
	ChannelName        string        `json:"channel_name,omitempty"`          // Cached channel name
	ChannelID          string        `json:"channel_id,omitempty"`            // Cached channel ID
}

// IsConfigured returns true if OAuth credentials are set for this account
func (a *Account) IsConfigured() bool {
	return a.ClientID != "" && a.ClientSecret != ""
}

// Config holds YouTube integration settings
type Config struct {
	// Legacy single-account fields (for backwards compatibility)
	ClientID           string        `json:"client_id,omitempty"`
	ClientSecret       string        `json:"client_secret,omitempty"`
	DefaultPlaylistID  string        `json:"default_playlist_id,omitempty"`
	DefaultPlaylistName string       `json:"default_playlist_name,omitempty"`
	ChannelName        string        `json:"channel_name,omitempty"`
	ChannelID          string        `json:"channel_id,omitempty"`

	// Multi-account support
	Accounts          []Account     `json:"accounts,omitempty"`
	LastUsedAccountID string        `json:"last_used_account_id,omitempty"`

	// Global settings
	DefaultPrivacy     PrivacyStatus `json:"default_privacy,omitempty"`
	AutoPromptUpload   bool          `json:"auto_prompt_upload,omitempty"`
}

// Token represents stored OAuth2 tokens
type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	Expiry       string `json:"expiry"` // RFC3339 format
}

// Playlist represents a YouTube playlist
type Playlist struct {
	ID          string
	Title       string
	Description string
	ItemCount   int64
	Thumbnails  string // URL to thumbnail
}

// UploadOptions contains all options for uploading a video
type UploadOptions struct {
	VideoPath         string
	Title             string
	Description       string
	Tags              []string
	CategoryID        string // YouTube category (e.g., "27" for Education, "28" for Science & Technology)
	PrivacyStatus     PrivacyStatus
	PlaylistID        string // Optional: add to playlist after upload
	ThumbnailPath     string // Optional: custom thumbnail
	NotifySubscribers bool
}

// UploadResult contains the result of a successful upload
type UploadResult struct {
	VideoID        string
	VideoURL       string
	PlaylistItemID string // If added to playlist
}

// UploadProgress reports upload progress
type UploadProgress struct {
	BytesUploaded int64
	TotalBytes    int64
	Percentage    float64
}

// DefaultConfig returns default YouTube configuration
func DefaultConfig() Config {
	return Config{
		DefaultPrivacy:   PrivacyUnlisted,
		AutoPromptUpload: true,
		Accounts:         []Account{},
	}
}

// IsConfigured returns true if at least one account has OAuth credentials set
func (c *Config) IsConfigured() bool {
	// Check legacy single-account config
	if c.ClientID != "" && c.ClientSecret != "" {
		return true
	}
	// Check multi-account config
	for _, acc := range c.Accounts {
		if acc.IsConfigured() {
			return true
		}
	}
	return false
}

// GetAccounts returns all configured accounts, including migrated legacy account
func (c *Config) GetAccounts() []Account {
	accounts := make([]Account, 0, len(c.Accounts)+1)

	// If there's a legacy account, include it first (with migration)
	if c.ClientID != "" && c.ClientSecret != "" {
		// Check if it's already in accounts list
		found := false
		for _, acc := range c.Accounts {
			if acc.ClientID == c.ClientID {
				found = true
				break
			}
		}
		if !found {
			legacyAccount := Account{
				ID:                  "legacy",
				Name:                c.ChannelName,
				ClientID:            c.ClientID,
				ClientSecret:        c.ClientSecret,
				DefaultPlaylistID:   c.DefaultPlaylistID,
				DefaultPlaylistName: c.DefaultPlaylistName,
				ChannelName:         c.ChannelName,
				ChannelID:           c.ChannelID,
			}
			if legacyAccount.Name == "" {
				legacyAccount.Name = "Default Account"
			}
			accounts = append(accounts, legacyAccount)
		}
	}

	accounts = append(accounts, c.Accounts...)
	return accounts
}

// GetAccount returns an account by ID
func (c *Config) GetAccount(id string) *Account {
	// Check legacy account
	if id == "legacy" && c.ClientID != "" && c.ClientSecret != "" {
		return &Account{
			ID:                  "legacy",
			Name:                c.ChannelName,
			ClientID:            c.ClientID,
			ClientSecret:        c.ClientSecret,
			DefaultPlaylistID:   c.DefaultPlaylistID,
			DefaultPlaylistName: c.DefaultPlaylistName,
			ChannelName:         c.ChannelName,
			ChannelID:           c.ChannelID,
		}
	}

	for i := range c.Accounts {
		if c.Accounts[i].ID == id {
			return &c.Accounts[i]
		}
	}
	return nil
}

// GetLastUsedAccount returns the last used account, or the first available account
func (c *Config) GetLastUsedAccount() *Account {
	accounts := c.GetAccounts()
	if len(accounts) == 0 {
		return nil
	}

	// Try to find the last used account
	if c.LastUsedAccountID != "" {
		for i := range accounts {
			if accounts[i].ID == c.LastUsedAccountID {
				return &accounts[i]
			}
		}
	}

	// Return the first account
	return &accounts[0]
}

// GetAccountByChannelID finds an account by its channel ID
func (c *Config) GetAccountByChannelID(channelID string) *Account {
	if channelID == "" {
		return nil
	}
	accounts := c.GetAccounts()
	for i := range accounts {
		if accounts[i].ChannelID == channelID {
			return &accounts[i]
		}
	}
	return nil
}

// AddAccount adds a new account to the config
func (c *Config) AddAccount(account Account) {
	// Generate ID if not set
	if account.ID == "" {
		account.ID = generateAccountID()
	}
	c.Accounts = append(c.Accounts, account)
}

// UpdateAccount updates an existing account
func (c *Config) UpdateAccount(account Account) bool {
	// Handle legacy account update
	if account.ID == "legacy" {
		c.ClientID = account.ClientID
		c.ClientSecret = account.ClientSecret
		c.DefaultPlaylistID = account.DefaultPlaylistID
		c.DefaultPlaylistName = account.DefaultPlaylistName
		c.ChannelName = account.ChannelName
		c.ChannelID = account.ChannelID
		return true
	}

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
	// Handle legacy account removal
	if id == "legacy" {
		c.ClientID = ""
		c.ClientSecret = ""
		c.DefaultPlaylistID = ""
		c.DefaultPlaylistName = ""
		c.ChannelName = ""
		c.ChannelID = ""
		return true
	}

	for i := range c.Accounts {
		if c.Accounts[i].ID == id {
			c.Accounts = append(c.Accounts[:i], c.Accounts[i+1:]...)
			return true
		}
	}
	return false
}

// generateAccountID generates a unique account ID
func generateAccountID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("acc_%x", b)
}

// GetTokenPath returns the path to the token file (legacy, for backwards compatibility)
func GetTokenPath(configDir string) string {
	return filepath.Join(configDir, "youtube_token.json")
}

// GetTokenPathForAccount returns the path to the token file for a specific account
func GetTokenPathForAccount(configDir, accountID string) string {
	if accountID == "" || accountID == "legacy" {
		return GetTokenPath(configDir)
	}
	return filepath.Join(configDir, fmt.Sprintf("youtube_token_%s.json", accountID))
}

// LoadToken loads the OAuth token from disk (legacy)
func LoadToken(configDir string) (*Token, error) {
	return LoadTokenForAccount(configDir, "legacy")
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

// SaveToken saves the OAuth token to disk (legacy)
func SaveToken(configDir string, token *Token) error {
	return SaveTokenForAccount(configDir, "legacy", token)
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

// DeleteToken removes the stored OAuth token (legacy)
func DeleteToken(configDir string) error {
	return DeleteTokenForAccount(configDir, "legacy")
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

// HasToken returns true if a token file exists (legacy)
func HasToken(configDir string) bool {
	return HasTokenForAccount(configDir, "legacy")
}

// HasTokenForAccount returns true if a token file exists for a specific account
func HasTokenForAccount(configDir, accountID string) bool {
	tokenPath := GetTokenPathForAccount(configDir, accountID)
	_, err := os.Stat(tokenPath)
	return err == nil
}

// YouTube video categories (common ones)
var VideoCategories = map[string]string{
	"1":  "Film & Animation",
	"2":  "Autos & Vehicles",
	"10": "Music",
	"15": "Pets & Animals",
	"17": "Sports",
	"19": "Travel & Events",
	"20": "Gaming",
	"22": "People & Blogs",
	"23": "Comedy",
	"24": "Entertainment",
	"25": "News & Politics",
	"26": "Howto & Style",
	"27": "Education",
	"28": "Science & Technology",
	"29": "Nonprofits & Activism",
}

// DefaultCategoryID is the default category for uploads (Science & Technology)
const DefaultCategoryID = "28"

// ParseTags parses a comma-separated string of tags into a slice
func ParseTags(tagsStr string) []string {
	if tagsStr == "" {
		return nil
	}

	parts := strings.Split(tagsStr, ",")
	var tags []string
	for _, part := range parts {
		tag := strings.TrimSpace(part)
		if tag != "" {
			tags = append(tags, tag)
		}
	}
	return tags
}
