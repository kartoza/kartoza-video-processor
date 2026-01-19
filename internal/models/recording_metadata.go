package models

import (
	"fmt"
	"regexp"
	"strings"
)

// RecordingMetadata holds user-provided metadata for a recording
type RecordingMetadata struct {
	Number      int    `json:"number"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Topic       string `json:"topic"`
	Presenter   string `json:"presenter"`
	FolderName  string `json:"folder_name,omitempty"`

	// YouTube upload information
	YouTube *YouTubeMetadata `json:"youtube,omitempty"`
}

// YouTubeMetadata holds information about a video uploaded to YouTube
type YouTubeMetadata struct {
	VideoID      string `json:"video_id"`
	VideoURL     string `json:"video_url"`
	PlaylistID   string `json:"playlist_id,omitempty"`
	PlaylistName string `json:"playlist_name,omitempty"`
	Privacy      string `json:"privacy"` // public, unlisted, private
	UploadedAt   string `json:"uploaded_at"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
	ChannelID    string `json:"channel_id,omitempty"`
	ChannelName  string `json:"channel_name,omitempty"`
}

// IsPublishedToYouTube returns true if the recording has been uploaded to YouTube
func (m *RecordingMetadata) IsPublishedToYouTube() bool {
	return m.YouTube != nil && m.YouTube.VideoID != ""
}

// GenerateFolderName creates a folder name from the counter and title
// Format: NNN-sanitized-title
func (m *RecordingMetadata) GenerateFolderName() string {
	// Sanitize title for filesystem use
	sanitized := sanitizeForFilename(m.Title)
	if sanitized == "" {
		sanitized = "recording"
	}

	m.FolderName = fmt.Sprintf("%03d-%s", m.Number, sanitized)
	return m.FolderName
}

// sanitizeForFilename removes or replaces characters that are invalid in filenames
func sanitizeForFilename(s string) string {
	// Convert to lowercase
	s = strings.ToLower(s)

	// Replace spaces with hyphens
	s = strings.ReplaceAll(s, " ", "-")

	// Remove or replace invalid characters
	// Keep only alphanumeric, hyphens, and underscores
	reg := regexp.MustCompile(`[^a-z0-9\-_]`)
	s = reg.ReplaceAllString(s, "")

	// Remove multiple consecutive hyphens
	reg = regexp.MustCompile(`-+`)
	s = reg.ReplaceAllString(s, "-")

	// Trim hyphens from ends
	s = strings.Trim(s, "-")

	// Limit length
	if len(s) > 50 {
		s = s[:50]
	}

	return s
}

// Topic represents a recording topic/category
type Topic struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// DefaultTopics returns a list of default topics
func DefaultTopics() []Topic {
	return []Topic{
		{ID: "tutorial", Name: "Tutorial"},
		{ID: "demo", Name: "Demo"},
		{ID: "presentation", Name: "Presentation"},
		{ID: "meeting", Name: "Meeting"},
		{ID: "training", Name: "Training"},
		{ID: "review", Name: "Code Review"},
		{ID: "other", Name: "Other"},
	}
}
