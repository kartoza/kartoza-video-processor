package googlechat

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

// Provider implements syndication.Provider for Google Chat
type Provider struct{}

func init() {
	syndication.RegisterProvider(&Provider{})
}

// Platform returns the platform type
func (p *Provider) Platform() syndication.PlatformType {
	return syndication.PlatformGoogleChat
}

// Name returns the provider display name
func (p *Provider) Name() string {
	return "Google Chat"
}

// Description returns a brief description of the platform
func (p *Provider) Description() string {
	return "Send messages to Google Chat spaces via webhooks"
}

// IsConfigured returns true if the account has required credentials
func (p *Provider) IsConfigured(account *syndication.Account) bool {
	return account.WebhookURL != ""
}

// IsAuthenticated returns true if the account has a valid token/session
// Google Chat webhooks don't require separate authentication
func (p *Provider) IsAuthenticated(ctx context.Context, account *syndication.Account, configDir string) bool {
	return p.IsConfigured(account)
}

// Authenticate performs the authentication flow
// For Google Chat webhooks, there's no OAuth - the webhook URL contains the auth
func (p *Provider) Authenticate(ctx context.Context, account *syndication.Account, configDir string, urlCallback func(string)) error {
	// Google Chat webhooks don't require OAuth
	return nil
}

// Post creates a message in Google Chat
func (p *Provider) Post(ctx context.Context, account *syndication.Account, configDir string, content *syndication.PostContent) (*syndication.PostResult, error) {
	result := &syndication.PostResult{
		AccountID:   account.ID,
		AccountName: account.GetDisplayName(),
		Platform:    syndication.PlatformGoogleChat,
	}

	if !p.IsConfigured(account) {
		result.Error = errors.New("google Chat webhook not configured")
		result.Message = "Webhook URL not configured"
		return result, nil
	}

	// Build card message
	builder := syndication.NewPostBuilder(content)
	cardMessage := builder.BuildGoogleChatCard()

	jsonBody, err := json.Marshal(cardMessage)
	if err != nil {
		result.Error = fmt.Errorf("failed to marshal message: %w", err)
		result.Message = "Failed to create message"
		return result, nil
	}

	// Send request
	req, err := http.NewRequestWithContext(ctx, "POST", account.WebhookURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		result.Error = fmt.Errorf("failed to create request: %w", err)
		result.Message = "Failed to create request"
		return result, nil
	}
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		result.Error = fmt.Errorf("failed to send message: %w", err)
		result.Message = "Failed to send message"
		return result, nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// Check response
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		result.Error = fmt.Errorf("google Chat API error: %s - %s", resp.Status, string(body))
		result.Message = fmt.Sprintf("API error: %s", resp.Status)
		return result, nil
	}

	// Parse response for message name/ID
	var respData struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(body, &respData); err == nil && respData.Name != "" {
		result.PostID = respData.Name
	}

	result.Success = true
	result.Message = "Message sent to Google Chat"
	return result, nil
}

// ValidateCredentials checks if the provided credentials have valid format
func (p *Provider) ValidateCredentials(account *syndication.Account) error {
	if account.WebhookURL == "" {
		return errors.New("webhook URL is required")
	}

	// Validate URL format
	if !strings.HasPrefix(account.WebhookURL, "https://chat.googleapis.com/") {
		return errors.New("webhook URL must start with https://chat.googleapis.com/")
	}

	// Check for required URL parameters
	if !strings.Contains(account.WebhookURL, "key=") || !strings.Contains(account.WebhookURL, "token=") {
		return errors.New("webhook URL must contain key and token parameters")
	}

	return nil
}

// GetAccountInfo fetches and returns display info
func (p *Provider) GetAccountInfo(ctx context.Context, account *syndication.Account, configDir string) (string, error) {
	// Google Chat webhooks don't provide account info
	// Extract space ID from URL if possible
	if strings.Contains(account.WebhookURL, "/spaces/") {
		parts := strings.Split(account.WebhookURL, "/spaces/")
		if len(parts) > 1 {
			spaceID := strings.Split(parts[1], "/")[0]
			return fmt.Sprintf("Space: %s", spaceID), nil
		}
	}
	return "Google Chat Webhook", nil
}

// SupportsImages returns true if the platform supports image attachments
func (p *Provider) SupportsImages() bool {
	return true // Cards can include images via URL
}

// MaxMessageLength returns the max character limit (0 for unlimited)
func (p *Provider) MaxMessageLength() int {
	return 4096 // Google Chat has a 4096 character limit for text
}

// RequiresAuth returns true if platform requires OAuth or similar auth flow
func (p *Provider) RequiresAuth() bool {
	return false // Webhook URL contains the auth
}

// GetRequiredFields returns the field names needed for this platform
func (p *Provider) GetRequiredFields() []string {
	return []string{"webhook_url"}
}
