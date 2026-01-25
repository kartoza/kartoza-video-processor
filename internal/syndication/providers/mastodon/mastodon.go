package mastodon

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kartoza/kartoza-screencaster/internal/syndication"
)

const (
	maxStatusLength = 500
	oauthScopes     = "read write:statuses write:media"
	redirectURI     = "urn:ietf:wg:oauth:2.0:oob"
)

// Provider implements syndication.Provider for Mastodon
type Provider struct{}

func init() {
	syndication.RegisterProvider(&Provider{})
}

// Platform returns the platform type
func (p *Provider) Platform() syndication.PlatformType {
	return syndication.PlatformMastodon
}

// Name returns the provider display name
func (p *Provider) Name() string {
	return "Mastodon"
}

// Description returns a brief description of the platform
func (p *Provider) Description() string {
	return "Post to Mastodon/Fediverse instances"
}

// IsConfigured returns true if the account has required credentials
func (p *Provider) IsConfigured(account *syndication.Account) bool {
	return account.InstanceURL != "" && account.ClientID != "" && account.ClientSecret != ""
}

// IsAuthenticated returns true if the account has a valid token
func (p *Provider) IsAuthenticated(ctx context.Context, account *syndication.Account, configDir string) bool {
	if !p.IsConfigured(account) {
		return false
	}

	token, err := syndication.LoadTokenForAccount(configDir, account.ID)
	if err != nil || token.AccessToken == "" {
		return false
	}

	// Verify token by calling verify_credentials
	instanceURL := normalizeInstanceURL(account.InstanceURL)
	url := instanceURL + "/api/v1/accounts/verify_credentials"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200
}

// Authenticate performs the OAuth2 authentication flow
func (p *Provider) Authenticate(ctx context.Context, account *syndication.Account, configDir string, urlCallback func(string)) error {
	if account.InstanceURL == "" || account.ClientID == "" || account.ClientSecret == "" {
		return errors.New("instance URL, client ID, and client secret are required")
	}

	instanceURL := normalizeInstanceURL(account.InstanceURL)

	// Build authorization URL
	authURL := fmt.Sprintf("%s/oauth/authorize?client_id=%s&scope=%s&redirect_uri=%s&response_type=code",
		instanceURL,
		url.QueryEscape(account.ClientID),
		url.QueryEscape(oauthScopes),
		url.QueryEscape(redirectURI),
	)

	// Call the URL callback to let the user know they need to visit this URL
	if urlCallback != nil {
		urlCallback(authURL)
	}

	// For out-of-band OAuth, the user will need to paste the authorization code
	// This is handled by the TUI layer - we return here to indicate manual step needed
	return errors.New("OAUTH_PENDING:Please visit the authorization URL and paste the code")
}

// CompleteAuth completes the OAuth flow with the authorization code
func (p *Provider) CompleteAuth(ctx context.Context, account *syndication.Account, configDir, authCode string) error {
	instanceURL := normalizeInstanceURL(account.InstanceURL)

	// Exchange code for token
	tokenURL := instanceURL + "/oauth/token"

	data := url.Values{
		"client_id":     {account.ClientID},
		"client_secret": {account.ClientSecret},
		"redirect_uri":  {redirectURI},
		"grant_type":    {"authorization_code"},
		"code":          {authCode},
		"scope":         {oauthScopes},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to exchange code for token: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return fmt.Errorf("token exchange failed: %s - %s", resp.Status, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`
		CreatedAt   int64  `json:"created_at"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return fmt.Errorf("failed to parse token response: %w", err)
	}

	// Save token
	token := &syndication.Token{
		AccessToken: tokenResp.AccessToken,
		TokenType:   tokenResp.TokenType,
	}

	return syndication.SaveTokenForAccount(configDir, account.ID, token)
}

// Post creates a new status on Mastodon
func (p *Provider) Post(ctx context.Context, account *syndication.Account, configDir string, content *syndication.PostContent) (*syndication.PostResult, error) {
	result := &syndication.PostResult{
		AccountID:   account.ID,
		AccountName: account.GetDisplayName(),
		Platform:    syndication.PlatformMastodon,
	}

	if !p.IsConfigured(account) {
		result.Error = errors.New("mastodon not configured")
		result.Message = "Instance URL or credentials not configured"
		return result, nil
	}

	token, err := syndication.LoadTokenForAccount(configDir, account.ID)
	if err != nil || token.AccessToken == "" {
		result.Error = errors.New("not authenticated")
		result.Message = "Please authenticate with Mastodon first"
		return result, nil
	}

	instanceURL := normalizeInstanceURL(account.InstanceURL)

	// Upload media if thumbnail is available
	var mediaIDs []string
	if content.ThumbnailPath != "" && fileExists(content.ThumbnailPath) {
		mediaID, err := p.uploadMedia(ctx, instanceURL, token.AccessToken, content.ThumbnailPath, content.Title)
		if err == nil {
			mediaIDs = append(mediaIDs, mediaID)
		}
		// Don't fail if media upload fails
	}

	// Build status text
	builder := syndication.NewPostBuilder(content).WithShortTemplate()
	statusText, err := builder.BuildForPlatform(syndication.PlatformMastodon, maxStatusLength)
	if err != nil {
		result.Error = fmt.Errorf("failed to build status: %w", err)
		result.Message = "Failed to create status"
		return result, nil
	}

	// Create status
	statusData := map[string]interface{}{
		"status": statusText,
	}
	if len(mediaIDs) > 0 {
		statusData["media_ids"] = mediaIDs
	}

	jsonBody, err := json.Marshal(statusData)
	if err != nil {
		result.Error = fmt.Errorf("failed to marshal status: %w", err)
		result.Message = "Failed to create status"
		return result, nil
	}

	statusURL := instanceURL + "/api/v1/statuses"
	req, err := http.NewRequestWithContext(ctx, "POST", statusURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		result.Error = fmt.Errorf("failed to create request: %w", err)
		result.Message = "Failed to create request"
		return result, nil
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		result.Error = fmt.Errorf("failed to post status: %w", err)
		result.Message = "Failed to post status"
		return result, nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		result.Error = fmt.Errorf("mastodon API error: %s - %s", resp.Status, string(body))
		result.Message = fmt.Sprintf("API error: %s", resp.Status)
		return result, nil
	}

	// Parse response
	var respData struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	}
	if err := json.Unmarshal(body, &respData); err == nil {
		result.PostID = respData.ID
		result.PostURL = respData.URL
	}

	result.Success = true
	result.Message = "Posted to Mastodon"
	return result, nil
}

// uploadMedia uploads an image to Mastodon
func (p *Provider) uploadMedia(ctx context.Context, instanceURL, accessToken, filePath, description string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add file
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return "", fmt.Errorf("failed to copy file: %w", err)
	}

	// Add description (alt text)
	if description != "" {
		_ = writer.WriteField("description", description)
	}

	writer.Close()

	url := instanceURL + "/api/v2/media"
	req, err := http.NewRequestWithContext(ctx, "POST", url, &buf)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to upload media: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("media upload error: %s - %s", resp.Status, string(body))
	}

	var respData struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &respData); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return respData.ID, nil
}

// ValidateCredentials checks if the provided credentials have valid format
func (p *Provider) ValidateCredentials(account *syndication.Account) error {
	if account.InstanceURL == "" {
		return errors.New("instance URL is required")
	}

	// Validate instance URL format
	instanceURL := normalizeInstanceURL(account.InstanceURL)
	if !strings.HasPrefix(instanceURL, "https://") {
		return errors.New("instance URL must use HTTPS")
	}

	if account.ClientID == "" {
		return errors.New("client ID is required")
	}

	if account.ClientSecret == "" {
		return errors.New("client secret is required")
	}

	return nil
}

// GetAccountInfo fetches and returns display info (username)
func (p *Provider) GetAccountInfo(ctx context.Context, account *syndication.Account, configDir string) (string, error) {
	if !p.IsConfigured(account) {
		return "", errors.New("mastodon not configured")
	}

	token, err := syndication.LoadTokenForAccount(configDir, account.ID)
	if err != nil || token.AccessToken == "" {
		return "", errors.New("not authenticated")
	}

	instanceURL := normalizeInstanceURL(account.InstanceURL)
	url := instanceURL + "/api/v1/accounts/verify_credentials"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var respData struct {
		Username    string `json:"username"`
		Acct        string `json:"acct"`
		DisplayName string `json:"display_name"`
	}

	if err := json.Unmarshal(body, &respData); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if respData.DisplayName != "" {
		return respData.DisplayName + " (@" + respData.Acct + ")", nil
	}
	return "@" + respData.Acct, nil
}

// SupportsImages returns true if the platform supports image attachments
func (p *Provider) SupportsImages() bool {
	return true
}

// MaxMessageLength returns the max character limit
func (p *Provider) MaxMessageLength() int {
	return maxStatusLength
}

// RequiresAuth returns true if platform requires OAuth
func (p *Provider) RequiresAuth() bool {
	return true
}

// GetRequiredFields returns the field names needed for this platform
func (p *Provider) GetRequiredFields() []string {
	return []string{"instance_url", "client_id", "client_secret"}
}

// normalizeInstanceURL ensures the instance URL has proper format
func normalizeInstanceURL(instanceURL string) string {
	instanceURL = strings.TrimSpace(instanceURL)
	instanceURL = strings.TrimSuffix(instanceURL, "/")

	// Add https:// if no protocol
	if !strings.HasPrefix(instanceURL, "http://") && !strings.HasPrefix(instanceURL, "https://") {
		instanceURL = "https://" + instanceURL
	}

	return instanceURL
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// RegisterApp registers an OAuth application with a Mastodon instance
// This is a helper function that can be used to create client credentials
func RegisterApp(ctx context.Context, instanceURL, appName, website string) (clientID, clientSecret string, err error) {
	instanceURL = normalizeInstanceURL(instanceURL)

	data := url.Values{
		"client_name":   {appName},
		"redirect_uris": {redirectURI},
		"scopes":        {oauthScopes},
	}
	if website != "" {
		data.Set("website", website)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", instanceURL+"/api/v1/apps", strings.NewReader(data.Encode()))
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to register app: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return "", "", fmt.Errorf("registration failed: %s - %s", resp.Status, string(body))
	}

	var respData struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	}

	if err := json.Unmarshal(body, &respData); err != nil {
		return "", "", fmt.Errorf("failed to parse response: %w", err)
	}

	return respData.ClientID, respData.ClientSecret, nil
}
