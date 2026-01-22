package wordpress

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/kartoza/kartoza-video-processor/internal/syndication"
)

// Provider implements syndication.Provider for WordPress
type Provider struct{}

func init() {
	syndication.RegisterProvider(&Provider{})
}

// Platform returns the platform type
func (p *Provider) Platform() syndication.PlatformType {
	return syndication.PlatformWordPress
}

// Name returns the provider display name
func (p *Provider) Name() string {
	return "WordPress"
}

// Description returns a brief description of the platform
func (p *Provider) Description() string {
	return "Publish posts to WordPress sites via REST API"
}

// IsConfigured returns true if the account has required credentials
func (p *Provider) IsConfigured(account *syndication.Account) bool {
	return account.SiteURL != "" && account.Username != "" && account.AppPassword != ""
}

// IsAuthenticated returns true if the account has valid credentials
func (p *Provider) IsAuthenticated(ctx context.Context, account *syndication.Account, configDir string) bool {
	if !p.IsConfigured(account) {
		return false
	}

	// Verify credentials by calling users/me endpoint
	url := p.buildAPIURL(account.SiteURL, "users/me")
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false
	}

	p.addAuthHeader(req, account)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200
}

// Authenticate performs the authentication flow
// WordPress uses application passwords - no OAuth flow needed
func (p *Provider) Authenticate(ctx context.Context, account *syndication.Account, configDir string, urlCallback func(string)) error {
	// WordPress doesn't require OAuth - just verify credentials work
	if !p.IsAuthenticated(ctx, account, configDir) {
		return errors.New("invalid WordPress credentials")
	}
	return nil
}

// Post creates a new post on WordPress
func (p *Provider) Post(ctx context.Context, account *syndication.Account, configDir string, content *syndication.PostContent) (*syndication.PostResult, error) {
	result := &syndication.PostResult{
		AccountID:   account.ID,
		AccountName: account.GetDisplayName(),
		Platform:    syndication.PlatformWordPress,
	}

	if !p.IsConfigured(account) {
		result.Error = errors.New("wordpress not configured")
		result.Message = "Site URL, username, or app password not configured"
		return result, nil
	}

	// Build HTML content
	builder := syndication.NewPostBuilder(content)
	htmlContent, err := builder.BuildHTML()
	if err != nil {
		result.Error = fmt.Errorf("failed to build HTML content: %w", err)
		result.Message = "Failed to create post content"
		return result, nil
	}

	// Upload featured image if available
	var featuredMediaID int
	if content.ThumbnailPath != "" && fileExists(content.ThumbnailPath) {
		mediaID, err := p.uploadMedia(ctx, account, content.ThumbnailPath)
		if err == nil {
			featuredMediaID = mediaID
		}
		// Don't fail if image upload fails
	}

	// Determine post status
	postStatus := account.PostStatus
	if postStatus == "" {
		postStatus = "draft" // Default to draft for safety
	}

	// Create post
	postData := map[string]interface{}{
		"title":   content.Title,
		"content": htmlContent,
		"status":  postStatus,
	}

	if featuredMediaID > 0 {
		postData["featured_media"] = featuredMediaID
	}

	if account.CategoryID > 0 {
		postData["categories"] = []int{account.CategoryID}
	}

	// Add tags if available
	if len(content.Tags) > 0 {
		postData["tags"] = content.Tags
	}

	jsonBody, err := json.Marshal(postData)
	if err != nil {
		result.Error = fmt.Errorf("failed to marshal post data: %w", err)
		result.Message = "Failed to create post"
		return result, nil
	}

	url := p.buildAPIURL(account.SiteURL, "posts")
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		result.Error = fmt.Errorf("failed to create request: %w", err)
		result.Message = "Failed to create request"
		return result, nil
	}

	req.Header.Set("Content-Type", "application/json")
	p.addAuthHeader(req, account)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		result.Error = fmt.Errorf("failed to create post: %w", err)
		result.Message = "Failed to create post"
		return result, nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		result.Error = fmt.Errorf("wordpress API error: %s - %s", resp.Status, string(body))
		result.Message = fmt.Sprintf("API error: %s", resp.Status)
		return result, nil
	}

	// Parse response
	var respData struct {
		ID   int    `json:"id"`
		Link string `json:"link"`
	}
	if err := json.Unmarshal(body, &respData); err == nil {
		result.PostID = fmt.Sprintf("%d", respData.ID)
		result.PostURL = respData.Link
	}

	result.Success = true
	result.Message = fmt.Sprintf("Post created (%s)", postStatus)
	return result, nil
}

// uploadMedia uploads an image to WordPress media library
func (p *Provider) uploadMedia(ctx context.Context, account *syndication.Account, filePath string) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	fileData, err := io.ReadAll(file)
	if err != nil {
		return 0, fmt.Errorf("failed to read file: %w", err)
	}

	url := p.buildAPIURL(account.SiteURL, "media")
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(fileData))
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	// Determine content type
	ext := filepath.Ext(filePath)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filepath.Base(filePath)))
	p.addAuthHeader(req, account)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to upload media: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("wordpress media upload error: %s - %s", resp.Status, string(body))
	}

	var respData struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal(body, &respData); err != nil {
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}

	return respData.ID, nil
}

// buildAPIURL constructs the WordPress REST API URL
func (p *Provider) buildAPIURL(siteURL, endpoint string) string {
	siteURL = strings.TrimSuffix(siteURL, "/")
	return fmt.Sprintf("%s/wp-json/wp/v2/%s", siteURL, endpoint)
}

// addAuthHeader adds Basic authentication header
func (p *Provider) addAuthHeader(req *http.Request, account *syndication.Account) {
	auth := base64.StdEncoding.EncodeToString([]byte(account.Username + ":" + account.AppPassword))
	req.Header.Set("Authorization", "Basic "+auth)
}

// ValidateCredentials checks if the provided credentials have valid format
func (p *Provider) ValidateCredentials(account *syndication.Account) error {
	if account.SiteURL == "" {
		return errors.New("site URL is required")
	}

	// Validate URL format
	if !strings.HasPrefix(account.SiteURL, "http://") && !strings.HasPrefix(account.SiteURL, "https://") {
		return errors.New("site URL must start with http:// or https://")
	}

	if account.Username == "" {
		return errors.New("username is required")
	}

	if account.AppPassword == "" {
		return errors.New("application password is required")
	}

	// App password format: typically xxxx xxxx xxxx xxxx xxxx xxxx (with spaces)
	// But we'll accept with or without spaces
	appPass := strings.ReplaceAll(account.AppPassword, " ", "")
	if len(appPass) < 20 {
		return errors.New("application password appears too short")
	}

	return nil
}

// GetAccountInfo fetches and returns display info (site name)
func (p *Provider) GetAccountInfo(ctx context.Context, account *syndication.Account, configDir string) (string, error) {
	if !p.IsConfigured(account) {
		return "", errors.New("wordpress not configured")
	}

	// Get site info
	url := strings.TrimSuffix(account.SiteURL, "/") + "/wp-json"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var respData struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(body, &respData); err == nil && respData.Name != "" {
		return respData.Name, nil
	}

	// Fallback to site URL
	return account.SiteURL, nil
}

// SupportsImages returns true if the platform supports image attachments
func (p *Provider) SupportsImages() bool {
	return true
}

// MaxMessageLength returns the max character limit (0 for unlimited)
func (p *Provider) MaxMessageLength() int {
	return 0 // WordPress has no practical limit
}

// RequiresAuth returns true if platform requires OAuth or similar auth flow
func (p *Provider) RequiresAuth() bool {
	return false // Uses application passwords
}

// GetRequiredFields returns the field names needed for this platform
func (p *Provider) GetRequiredFields() []string {
	return []string{"site_url", "username", "app_password"}
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
