package bluesky

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/kartoza/kartoza-video-processor/internal/syndication"
)

const (
	defaultPDS     = "https://bsky.social"
	maxPostLength  = 300
	maxImageSize   = 1000000 // 1MB
	sessionTimeout = 2 * time.Hour
)

// Provider implements syndication.Provider for Bluesky
type Provider struct{}

func init() {
	syndication.RegisterProvider(&Provider{})
}

// Platform returns the platform type
func (p *Provider) Platform() syndication.PlatformType {
	return syndication.PlatformBluesky
}

// Name returns the provider display name
func (p *Provider) Name() string {
	return "Bluesky"
}

// Description returns a brief description of the platform
func (p *Provider) Description() string {
	return "Post to Bluesky social network"
}

// IsConfigured returns true if the account has required credentials
func (p *Provider) IsConfigured(account *syndication.Account) bool {
	return account.Handle != "" && account.AppPassword != ""
}

// IsAuthenticated returns true if the account has a valid session
func (p *Provider) IsAuthenticated(ctx context.Context, account *syndication.Account, configDir string) bool {
	if !p.IsConfigured(account) {
		return false
	}

	session, err := p.loadSession(configDir, account.ID)
	if err != nil || session.AccessJwt == "" {
		return false
	}

	// Verify session by calling getSession
	pds := getPDS(account.InstanceURL)
	url := pds + "/xrpc/com.atproto.server.getSession"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false
	}
	req.Header.Set("Authorization", "Bearer "+session.AccessJwt)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200
}

// Authenticate performs authentication using app password
func (p *Provider) Authenticate(ctx context.Context, account *syndication.Account, configDir string, urlCallback func(string)) error {
	if account.Handle == "" || account.AppPassword == "" {
		return errors.New("handle and app password are required")
	}

	pds := getPDS(account.InstanceURL)

	// Create session using app password
	reqBody := map[string]string{
		"identifier": account.Handle,
		"password":   account.AppPassword,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := pds + "/xrpc/com.atproto.server.createSession"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return fmt.Errorf("authentication failed: %s - %s", resp.Status, string(body))
	}

	var session syndication.BlueskySession
	if err := json.Unmarshal(body, &session); err != nil {
		return fmt.Errorf("failed to parse session: %w", err)
	}

	// Save session
	return syndication.SaveSessionForAccount(configDir, account.ID, &session)
}

// Post creates a new post on Bluesky
func (p *Provider) Post(ctx context.Context, account *syndication.Account, configDir string, content *syndication.PostContent) (*syndication.PostResult, error) {
	result := &syndication.PostResult{
		AccountID:   account.ID,
		AccountName: account.GetDisplayName(),
		Platform:    syndication.PlatformBluesky,
	}

	if !p.IsConfigured(account) {
		result.Error = errors.New("bluesky not configured")
		result.Message = "Handle or app password not configured"
		return result, nil
	}

	session, err := p.loadSession(configDir, account.ID)
	if err != nil || session.AccessJwt == "" {
		// Try to authenticate
		if err := p.Authenticate(ctx, account, configDir, nil); err != nil {
			result.Error = errors.New("not authenticated")
			result.Message = "Please authenticate with Bluesky first"
			return result, nil
		}
		session, _ = p.loadSession(configDir, account.ID)
	}

	pds := getPDS(account.InstanceURL)

	// Build post text
	builder := syndication.NewPostBuilder(content).WithShortTemplate()
	postText, err := builder.BuildForPlatform(syndication.PlatformBluesky, maxPostLength)
	if err != nil {
		result.Error = fmt.Errorf("failed to build post: %w", err)
		result.Message = "Failed to create post"
		return result, nil
	}

	// Build the post record
	record := map[string]interface{}{
		"$type":     "app.bsky.feed.post",
		"text":      postText,
		"createdAt": time.Now().UTC().Format(time.RFC3339),
	}

	// Extract facets (links, mentions, hashtags)
	facets := p.extractFacets(postText, content.VideoURL)
	if len(facets) > 0 {
		record["facets"] = facets
	}

	// Upload image if available
	if content.ThumbnailPath != "" && fileExists(content.ThumbnailPath) {
		blob, err := p.uploadBlob(ctx, pds, session.AccessJwt, content.ThumbnailPath)
		if err == nil {
			record["embed"] = map[string]interface{}{
				"$type": "app.bsky.embed.images",
				"images": []map[string]interface{}{
					{
						"alt":   content.Title,
						"image": blob,
					},
				},
			}
		}
		// Don't fail if image upload fails
	} else if content.VideoURL != "" {
		// Add external link embed if no image
		record["embed"] = map[string]interface{}{
			"$type": "app.bsky.embed.external",
			"external": map[string]interface{}{
				"uri":         content.VideoURL,
				"title":       content.Title,
				"description": truncate(content.Description, 300),
			},
		}
	}

	// Create post
	createRecord := map[string]interface{}{
		"repo":       session.DID,
		"collection": "app.bsky.feed.post",
		"record":     record,
	}

	jsonBody, err := json.Marshal(createRecord)
	if err != nil {
		result.Error = fmt.Errorf("failed to marshal post: %w", err)
		result.Message = "Failed to create post"
		return result, nil
	}

	url := pds + "/xrpc/com.atproto.repo.createRecord"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		result.Error = fmt.Errorf("failed to create request: %w", err)
		result.Message = "Failed to create request"
		return result, nil
	}
	req.Header.Set("Authorization", "Bearer "+session.AccessJwt)
	req.Header.Set("Content-Type", "application/json")

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
		// Check if session expired
		if resp.StatusCode == 401 {
			// Try to refresh session and retry
			if err := p.Authenticate(ctx, account, configDir, nil); err == nil {
				return p.Post(ctx, account, configDir, content)
			}
		}
		result.Error = fmt.Errorf("bluesky API error: %s - %s", resp.Status, string(body))
		result.Message = fmt.Sprintf("API error: %s", resp.Status)
		return result, nil
	}

	// Parse response
	var respData struct {
		URI string `json:"uri"`
		CID string `json:"cid"`
	}
	if err := json.Unmarshal(body, &respData); err == nil {
		result.PostID = respData.CID
		// Convert AT URI to web URL
		result.PostURL = p.atURIToWebURL(respData.URI, session.Handle)
	}

	result.Success = true
	result.Message = "Posted to Bluesky"
	return result, nil
}

// uploadBlob uploads an image to Bluesky
func (p *Provider) uploadBlob(ctx context.Context, pds, accessJwt, filePath string) (map[string]interface{}, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Check size limit
	if len(data) > maxImageSize {
		return nil, errors.New("image too large (max 1MB)")
	}

	// Detect content type
	contentType := "image/jpeg"
	if strings.HasSuffix(strings.ToLower(filePath), ".png") {
		contentType = "image/png"
	} else if strings.HasSuffix(strings.ToLower(filePath), ".gif") {
		contentType = "image/gif"
	} else if strings.HasSuffix(strings.ToLower(filePath), ".webp") {
		contentType = "image/webp"
	}

	url := pds + "/xrpc/com.atproto.repo.uploadBlob"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessJwt)
	req.Header.Set("Content-Type", contentType)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to upload: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("upload failed: %s - %s", resp.Status, string(body))
	}

	var respData struct {
		Blob map[string]interface{} `json:"blob"`
	}
	if err := json.Unmarshal(body, &respData); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return respData.Blob, nil
}

// extractFacets extracts links, mentions, and hashtags from text
func (p *Provider) extractFacets(text, videoURL string) []map[string]interface{} {
	var facets []map[string]interface{}

	// Find video URL in text
	if videoURL != "" {
		if idx := strings.Index(text, videoURL); idx >= 0 {
			facets = append(facets, map[string]interface{}{
				"index": map[string]interface{}{
					"byteStart": idx,
					"byteEnd":   idx + len(videoURL),
				},
				"features": []map[string]interface{}{
					{
						"$type": "app.bsky.richtext.facet#link",
						"uri":   videoURL,
					},
				},
			})
		}
	}

	// Find hashtags
	words := strings.Fields(text)
	currentIdx := 0
	for _, word := range words {
		wordIdx := strings.Index(text[currentIdx:], word) + currentIdx
		if strings.HasPrefix(word, "#") && len(word) > 1 {
			tag := strings.TrimPrefix(word, "#")
			// Remove trailing punctuation
			tag = strings.TrimRight(tag, ".,!?;:")
			facets = append(facets, map[string]interface{}{
				"index": map[string]interface{}{
					"byteStart": wordIdx,
					"byteEnd":   wordIdx + len("#"+tag),
				},
				"features": []map[string]interface{}{
					{
						"$type": "app.bsky.richtext.facet#tag",
						"tag":   tag,
					},
				},
			})
		}
		currentIdx = wordIdx + len(word)
	}

	return facets
}

// atURIToWebURL converts an AT Protocol URI to a web URL
func (p *Provider) atURIToWebURL(atURI, handle string) string {
	// AT URI format: at://did:plc:xxx/app.bsky.feed.post/rkey
	parts := strings.Split(atURI, "/")
	if len(parts) < 5 {
		return ""
	}
	rkey := parts[len(parts)-1]
	return fmt.Sprintf("https://bsky.app/profile/%s/post/%s", handle, rkey)
}

// loadSession loads the Bluesky session from disk
func (p *Provider) loadSession(configDir, accountID string) (*syndication.BlueskySession, error) {
	var session syndication.BlueskySession
	err := syndication.LoadSessionForAccount(configDir, accountID, &session)
	return &session, err
}

// ValidateCredentials checks if the provided credentials have valid format
func (p *Provider) ValidateCredentials(account *syndication.Account) error {
	if account.Handle == "" {
		return errors.New("handle is required")
	}

	// Handle should be like user.bsky.social or a DID
	if !strings.Contains(account.Handle, ".") && !strings.HasPrefix(account.Handle, "did:") {
		return errors.New("handle should be like user.bsky.social")
	}

	if account.AppPassword == "" {
		return errors.New("app password is required")
	}

	// App password format: xxxx-xxxx-xxxx-xxxx
	appPass := strings.ReplaceAll(account.AppPassword, "-", "")
	if len(appPass) < 16 {
		return errors.New("app password appears too short")
	}

	return nil
}

// GetAccountInfo fetches and returns display info (display name)
func (p *Provider) GetAccountInfo(ctx context.Context, account *syndication.Account, configDir string) (string, error) {
	if !p.IsConfigured(account) {
		return "", errors.New("bluesky not configured")
	}

	session, err := p.loadSession(configDir, account.ID)
	if err != nil || session.AccessJwt == "" {
		return "", errors.New("not authenticated")
	}

	pds := getPDS(account.InstanceURL)
	url := pds + "/xrpc/app.bsky.actor.getProfile?actor=" + session.DID

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+session.AccessJwt)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var respData struct {
		Handle      string `json:"handle"`
		DisplayName string `json:"displayName"`
	}

	if err := json.Unmarshal(body, &respData); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if respData.DisplayName != "" {
		return respData.DisplayName + " (@" + respData.Handle + ")", nil
	}
	return "@" + respData.Handle, nil
}

// SupportsImages returns true if the platform supports image attachments
func (p *Provider) SupportsImages() bool {
	return true
}

// MaxMessageLength returns the max character limit
func (p *Provider) MaxMessageLength() int {
	return maxPostLength
}

// RequiresAuth returns true if platform requires authentication
func (p *Provider) RequiresAuth() bool {
	return true // App password auth, but no OAuth flow needed
}

// GetRequiredFields returns the field names needed for this platform
func (p *Provider) GetRequiredFields() []string {
	return []string{"handle", "app_password"}
}

// getPDS returns the PDS URL, defaulting to bsky.social
func getPDS(instanceURL string) string {
	if instanceURL != "" {
		url := strings.TrimSpace(instanceURL)
		url = strings.TrimSuffix(url, "/")
		if !strings.HasPrefix(url, "https://") {
			url = "https://" + url
		}
		return url
	}
	return defaultPDS
}

// truncate truncates a string to maxLen
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
