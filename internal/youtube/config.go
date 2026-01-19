package youtube

import (
	"encoding/json"
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

// Config holds YouTube integration settings
type Config struct {
	ClientID          string        `json:"client_id,omitempty"`
	ClientSecret      string        `json:"client_secret,omitempty"`
	DefaultPlaylistID string        `json:"default_playlist_id,omitempty"`
	DefaultPrivacy    PrivacyStatus `json:"default_privacy,omitempty"`
	AutoPromptUpload  bool          `json:"auto_prompt_upload,omitempty"`
	ChannelName       string        `json:"channel_name,omitempty"` // Cached channel name
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
	}
}

// IsConfigured returns true if OAuth credentials are set
func (c *Config) IsConfigured() bool {
	return c.ClientID != "" && c.ClientSecret != ""
}

// GetTokenPath returns the path to the token file
func GetTokenPath(configDir string) string {
	return filepath.Join(configDir, "youtube_token.json")
}

// LoadToken loads the OAuth token from disk
func LoadToken(configDir string) (*Token, error) {
	tokenPath := GetTokenPath(configDir)
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

// SaveToken saves the OAuth token to disk
func SaveToken(configDir string, token *Token) error {
	tokenPath := GetTokenPath(configDir)

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

// DeleteToken removes the stored OAuth token
func DeleteToken(configDir string) error {
	tokenPath := GetTokenPath(configDir)
	err := os.Remove(tokenPath)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// HasToken returns true if a token file exists
func HasToken(configDir string) bool {
	tokenPath := GetTokenPath(configDir)
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
