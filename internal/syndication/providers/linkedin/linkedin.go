package linkedin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kartoza/kartoza-screencaster/internal/syndication"
)

const (
	authURL     = "https://www.linkedin.com/oauth/v2/authorization"
	tokenURL    = "https://www.linkedin.com/oauth/v2/accessToken"
	apiBaseURL  = "https://api.linkedin.com/v2"
	redirectURI = "http://localhost:8089/callback"
	scopes      = "w_member_social r_liteprofile"
)

// Provider implements syndication.Provider for LinkedIn
type Provider struct{}

func init() {
	syndication.RegisterProvider(&Provider{})
}

// Platform returns the platform type
func (p *Provider) Platform() syndication.PlatformType {
	return syndication.PlatformLinkedIn
}

// Name returns the provider display name
func (p *Provider) Name() string {
	return "LinkedIn"
}

// Description returns a brief description of the platform
func (p *Provider) Description() string {
	return "Share posts on LinkedIn"
}

// IsConfigured returns true if the account has required credentials
func (p *Provider) IsConfigured(account *syndication.Account) bool {
	return account.ClientID != "" && account.ClientSecret != ""
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

	// Check if token is expired
	if token.Expiry != "" {
		expiry, err := time.Parse(time.RFC3339, token.Expiry)
		if err == nil && time.Now().After(expiry) {
			return false
		}
	}

	// Verify token by calling userinfo endpoint
	url := apiBaseURL + "/userinfo"
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
	if account.ClientID == "" || account.ClientSecret == "" {
		return errors.New("client ID and client secret are required")
	}

	// Build authorization URL
	authURLWithParams := fmt.Sprintf("%s?response_type=code&client_id=%s&redirect_uri=%s&scope=%s&state=%s",
		authURL,
		url.QueryEscape(account.ClientID),
		url.QueryEscape(redirectURI),
		url.QueryEscape(scopes),
		url.QueryEscape(account.ID),
	)

	// Call the URL callback
	if urlCallback != nil {
		urlCallback(authURLWithParams)
	}

	return errors.New("OAUTH_PENDING:Please complete the OAuth flow in your browser")
}

// CompleteAuth completes the OAuth flow with the authorization code
func (p *Provider) CompleteAuth(ctx context.Context, account *syndication.Account, configDir, authCode string) error {
	// Exchange code for token
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {authCode},
		"redirect_uri":  {redirectURI},
		"client_id":     {account.ClientID},
		"client_secret": {account.ClientSecret},
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
		ExpiresIn   int    `json:"expires_in"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return fmt.Errorf("failed to parse token response: %w", err)
	}

	// Calculate expiry time
	expiry := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	// Save token
	token := &syndication.Token{
		AccessToken: tokenResp.AccessToken,
		TokenType:   "Bearer",
		Expiry:      expiry.Format(time.RFC3339),
	}

	return syndication.SaveTokenForAccount(configDir, account.ID, token)
}

// Post creates a new share/post on LinkedIn
func (p *Provider) Post(ctx context.Context, account *syndication.Account, configDir string, content *syndication.PostContent) (*syndication.PostResult, error) {
	result := &syndication.PostResult{
		AccountID:   account.ID,
		AccountName: account.GetDisplayName(),
		Platform:    syndication.PlatformLinkedIn,
	}

	if !p.IsConfigured(account) {
		result.Error = errors.New("linkedin not configured")
		result.Message = "Client ID or client secret not configured"
		return result, nil
	}

	token, err := syndication.LoadTokenForAccount(configDir, account.ID)
	if err != nil || token.AccessToken == "" {
		result.Error = errors.New("not authenticated")
		result.Message = "Please authenticate with LinkedIn first"
		return result, nil
	}

	// Get user's LinkedIn URN
	personURN, err := p.getPersonURN(ctx, token.AccessToken)
	if err != nil {
		result.Error = fmt.Errorf("failed to get user profile: %w", err)
		result.Message = "Failed to get user profile"
		return result, nil
	}

	// Build post text
	builder := syndication.NewPostBuilder(content)
	postText, err := builder.Build()
	if err != nil {
		result.Error = fmt.Errorf("failed to build post: %w", err)
		result.Message = "Failed to create post"
		return result, nil
	}

	// Create UGC Post
	shareContent := map[string]interface{}{
		"author":         personURN,
		"lifecycleState": "PUBLISHED",
		"specificContent": map[string]interface{}{
			"com.linkedin.ugc.ShareContent": map[string]interface{}{
				"shareCommentary": map[string]interface{}{
					"text": postText,
				},
				"shareMediaCategory": "ARTICLE",
				"media": []map[string]interface{}{
					{
						"status": "READY",
						"originalUrl": content.VideoURL,
						"title": map[string]interface{}{
							"text": content.Title,
						},
						"description": map[string]interface{}{
							"text": truncate(content.Description, 256),
						},
					},
				},
			},
		},
		"visibility": map[string]interface{}{
			"com.linkedin.ugc.MemberNetworkVisibility": "PUBLIC",
		},
	}

	jsonBody, err := json.Marshal(shareContent)
	if err != nil {
		result.Error = fmt.Errorf("failed to marshal post: %w", err)
		result.Message = "Failed to create post"
		return result, nil
	}

	postURL := apiBaseURL + "/ugcPosts"
	req, err := http.NewRequestWithContext(ctx, "POST", postURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		result.Error = fmt.Errorf("failed to create request: %w", err)
		result.Message = "Failed to create request"
		return result, nil
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Restli-Protocol-Version", "2.0.0")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		result.Error = fmt.Errorf("failed to post: %w", err)
		result.Message = "Failed to post"
		return result, nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		result.Error = fmt.Errorf("linkedin API error: %s - %s", resp.Status, string(body))
		result.Message = fmt.Sprintf("API error: %s", resp.Status)
		return result, nil
	}

	// Get post ID from header
	postID := resp.Header.Get("X-RestLi-Id")
	if postID == "" {
		// Try to parse from response
		var respData struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(body, &respData)
		postID = respData.ID
	}

	result.PostID = postID
	if postID != "" {
		// LinkedIn post URL format
		result.PostURL = fmt.Sprintf("https://www.linkedin.com/feed/update/%s", postID)
	}

	result.Success = true
	result.Message = "Posted to LinkedIn"
	return result, nil
}

// getPersonURN gets the user's LinkedIn URN
func (p *Provider) getPersonURN(ctx context.Context, accessToken string) (string, error) {
	url := apiBaseURL + "/userinfo"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("failed to get profile: %s", string(body))
	}

	var respData struct {
		Sub string `json:"sub"`
	}

	if err := json.Unmarshal(body, &respData); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return "urn:li:person:" + respData.Sub, nil
}

// ValidateCredentials checks if the provided credentials have valid format
func (p *Provider) ValidateCredentials(account *syndication.Account) error {
	if account.ClientID == "" {
		return errors.New("client ID is required")
	}

	if account.ClientSecret == "" {
		return errors.New("client secret is required")
	}

	return nil
}

// GetAccountInfo fetches and returns display info (name)
func (p *Provider) GetAccountInfo(ctx context.Context, account *syndication.Account, configDir string) (string, error) {
	if !p.IsConfigured(account) {
		return "", errors.New("linkedin not configured")
	}

	token, err := syndication.LoadTokenForAccount(configDir, account.ID)
	if err != nil || token.AccessToken == "" {
		return "", errors.New("not authenticated")
	}

	url := apiBaseURL + "/userinfo"
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
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	if err := json.Unmarshal(body, &respData); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if respData.Name != "" {
		return respData.Name, nil
	}
	if respData.Email != "" {
		return respData.Email, nil
	}
	return "LinkedIn User", nil
}

// SupportsImages returns true if the platform supports image attachments
func (p *Provider) SupportsImages() bool {
	return true // Via rich media shares
}

// MaxMessageLength returns the max character limit
func (p *Provider) MaxMessageLength() int {
	return 3000 // LinkedIn has a 3000 char limit for posts
}

// RequiresAuth returns true if platform requires OAuth
func (p *Provider) RequiresAuth() bool {
	return true
}

// GetRequiredFields returns the field names needed for this platform
func (p *Provider) GetRequiredFields() []string {
	return []string{"client_id", "client_secret"}
}

// GetRedirectURI returns the OAuth redirect URI used by this provider
func (p *Provider) GetRedirectURI() string {
	return redirectURI
}

// truncate truncates a string to maxLen
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
