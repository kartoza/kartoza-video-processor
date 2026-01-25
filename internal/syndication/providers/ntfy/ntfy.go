package ntfy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/kartoza/kartoza-screencaster/internal/syndication"
)

const (
	defaultServerURL = "https://ntfy.sh"
	maxTitleLength   = 250
)

// Provider implements syndication.Provider for ntfy.sh
type Provider struct{}

func init() {
	syndication.RegisterProvider(&Provider{})
}

// Platform returns the platform type
func (p *Provider) Platform() syndication.PlatformType {
	return syndication.PlatformNtfy
}

// Name returns the provider display name
func (p *Provider) Name() string {
	return "ntfy.sh"
}

// Description returns a brief description of the platform
func (p *Provider) Description() string {
	return "Push notifications via ntfy.sh"
}

// IsConfigured returns true if the account has required credentials
func (p *Provider) IsConfigured(account *syndication.Account) bool {
	return account.Topic != ""
}

// IsAuthenticated returns true if the account has a valid token/session
// ntfy doesn't require authentication for public topics
func (p *Provider) IsAuthenticated(ctx context.Context, account *syndication.Account, configDir string) bool {
	// ntfy doesn't require OAuth - just having a topic is enough
	return p.IsConfigured(account)
}

// Authenticate performs the authentication flow
// For ntfy, there's no OAuth - authentication is optional via access token
func (p *Provider) Authenticate(ctx context.Context, account *syndication.Account, configDir string, urlCallback func(string)) error {
	// ntfy doesn't require OAuth authentication
	// Access tokens are optional and set directly in config
	return nil
}

// Post creates a new notification on ntfy.sh
func (p *Provider) Post(ctx context.Context, account *syndication.Account, configDir string, content *syndication.PostContent) (*syndication.PostResult, error) {
	result := &syndication.PostResult{
		AccountID:   account.ID,
		AccountName: account.GetDisplayName(),
		Platform:    syndication.PlatformNtfy,
	}

	if !p.IsConfigured(account) {
		result.Error = errors.New("ntfy topic not configured")
		result.Message = "Topic not configured"
		return result, nil
	}

	// Build message
	builder := syndication.NewPostBuilder(content)
	title, message := builder.BuildNtfyMessage()

	// Truncate title if needed
	if len(title) > maxTitleLength {
		title = title[:maxTitleLength-3] + "..."
	}

	// Determine server URL
	serverURL := account.ServerURL
	if serverURL == "" {
		serverURL = defaultServerURL
	}
	serverURL = strings.TrimSuffix(serverURL, "/")

	// Build URL for the topic
	topicURL := fmt.Sprintf("%s/%s", serverURL, account.Topic)

	// Create request body
	reqBody := map[string]interface{}{
		"title":   title,
		"message": message,
	}

	// Add click action if video URL is available
	if content.VideoURL != "" {
		reqBody["click"] = content.VideoURL
		reqBody["actions"] = []map[string]string{
			{
				"action": "view",
				"label":  "Watch Video",
				"url":    content.VideoURL,
			},
		}
	}

	// Add tags
	if len(content.Tags) > 0 {
		reqBody["tags"] = content.Tags
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		result.Error = fmt.Errorf("failed to marshal request: %w", err)
		result.Message = "Failed to create request"
		return result, nil
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", topicURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		result.Error = fmt.Errorf("failed to create request: %w", err)
		result.Message = "Failed to create request"
		return result, nil
	}

	req.Header.Set("Content-Type", "application/json")

	// Add access token if configured
	if account.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+account.AccessToken)
	}

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		result.Error = fmt.Errorf("failed to send notification: %w", err)
		result.Message = "Failed to send notification"
		return result, nil
	}
	defer resp.Body.Close()

	// Read response body
	body, _ := io.ReadAll(resp.Body)

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		result.Error = fmt.Errorf("ntfy API error: %s - %s", resp.Status, string(body))
		result.Message = fmt.Sprintf("API error: %s", resp.Status)
		return result, nil
	}

	// Parse response for message ID
	var respData struct {
		ID    string `json:"id"`
		Event string `json:"event"`
	}
	if err := json.Unmarshal(body, &respData); err == nil && respData.ID != "" {
		result.PostID = respData.ID
	}

	result.Success = true
	result.Message = fmt.Sprintf("Notification sent to %s", account.Topic)
	return result, nil
}

// ValidateCredentials checks if the provided credentials have valid format
func (p *Provider) ValidateCredentials(account *syndication.Account) error {
	if account.Topic == "" {
		return errors.New("topic is required")
	}

	// Topic should be alphanumeric with underscores/hyphens
	for _, c := range account.Topic {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-') {
			return errors.New("topic can only contain letters, numbers, underscores, and hyphens")
		}
	}

	// Validate server URL if provided
	if account.ServerURL != "" {
		if !strings.HasPrefix(account.ServerURL, "http://") && !strings.HasPrefix(account.ServerURL, "https://") {
			return errors.New("server URL must start with http:// or https://")
		}
	}

	return nil
}

// GetAccountInfo fetches and returns display info
func (p *Provider) GetAccountInfo(ctx context.Context, account *syndication.Account, configDir string) (string, error) {
	// ntfy doesn't have account info - return topic name
	serverURL := account.ServerURL
	if serverURL == "" {
		serverURL = defaultServerURL
	}
	return fmt.Sprintf("%s/%s", serverURL, account.Topic), nil
}

// SupportsImages returns true if the platform supports image attachments
func (p *Provider) SupportsImages() bool {
	return true // ntfy supports attachments
}

// MaxMessageLength returns the max character limit (0 for unlimited)
func (p *Provider) MaxMessageLength() int {
	return 0 // ntfy has no practical message limit
}

// RequiresAuth returns true if platform requires OAuth or similar auth flow
func (p *Provider) RequiresAuth() bool {
	return false // ntfy authentication is optional
}

// GetRequiredFields returns the field names needed for this platform
func (p *Provider) GetRequiredFields() []string {
	return []string{"topic"}
}
