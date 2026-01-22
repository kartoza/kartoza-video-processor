package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/kartoza/kartoza-video-processor/internal/syndication"
)

const (
	apiBaseURL       = "https://api.telegram.org/bot"
	maxMessageLength = 4096
	maxCaptionLength = 1024
)

// Provider implements syndication.Provider for Telegram
type Provider struct{}

func init() {
	syndication.RegisterProvider(&Provider{})
}

// Platform returns the platform type
func (p *Provider) Platform() syndication.PlatformType {
	return syndication.PlatformTelegram
}

// Name returns the provider display name
func (p *Provider) Name() string {
	return "Telegram"
}

// Description returns a brief description of the platform
func (p *Provider) Description() string {
	return "Send messages to Telegram channels and groups"
}

// IsConfigured returns true if the account has required credentials
func (p *Provider) IsConfigured(account *syndication.Account) bool {
	return account.BotToken != "" && len(account.ChatIDs) > 0
}

// IsAuthenticated returns true if the account has a valid token/session
func (p *Provider) IsAuthenticated(ctx context.Context, account *syndication.Account, configDir string) bool {
	if !p.IsConfigured(account) {
		return false
	}

	// Verify bot token by calling getMe
	url := fmt.Sprintf("%s%s/getMe", apiBaseURL, account.BotToken)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200
}

// Authenticate performs the authentication flow
// For Telegram, there's no OAuth - the bot token is provided directly
func (p *Provider) Authenticate(ctx context.Context, account *syndication.Account, configDir string, urlCallback func(string)) error {
	// Telegram doesn't require OAuth - just verify the bot token works
	if !p.IsAuthenticated(ctx, account, configDir) {
		return errors.New("invalid bot token")
	}
	return nil
}

// Post creates messages in all configured Telegram chats
func (p *Provider) Post(ctx context.Context, account *syndication.Account, configDir string, content *syndication.PostContent) (*syndication.PostResult, error) {
	result := &syndication.PostResult{
		AccountID:   account.ID,
		AccountName: account.GetDisplayName(),
		Platform:    syndication.PlatformTelegram,
	}

	if !p.IsConfigured(account) {
		result.Error = errors.New("telegram bot not configured")
		result.Message = "Bot token or chat IDs not configured"
		return result, nil
	}

	// Build message
	builder := syndication.NewPostBuilder(content)
	message := builder.BuildTelegramMessage()

	// Send to all chat IDs
	var successCount, failCount int
	var lastError error
	var messageIDs []string

	for _, chatID := range account.ChatIDs {
		var msgID string
		var err error

		// If thumbnail is available, send as photo with caption
		if content.ThumbnailPath != "" && fileExists(content.ThumbnailPath) {
			// Truncate message for caption
			caption := message
			if len(caption) > maxCaptionLength {
				caption = caption[:maxCaptionLength-3] + "..."
			}
			msgID, err = p.sendPhoto(ctx, account.BotToken, chatID, content.ThumbnailPath, caption)
		} else {
			// Send text message
			if len(message) > maxMessageLength {
				message = message[:maxMessageLength-3] + "..."
			}
			msgID, err = p.sendMessage(ctx, account.BotToken, chatID, message)
		}

		if err != nil {
			failCount++
			lastError = err
		} else {
			successCount++
			messageIDs = append(messageIDs, msgID)
		}
	}

	// Determine overall result
	if successCount == 0 {
		result.Error = lastError
		result.Message = fmt.Sprintf("Failed to send to all %d chats", len(account.ChatIDs))
		return result, nil
	}

	result.Success = true
	result.PostID = strings.Join(messageIDs, ",")
	if failCount > 0 {
		result.Message = fmt.Sprintf("Sent to %d/%d chats", successCount, len(account.ChatIDs))
	} else {
		result.Message = fmt.Sprintf("Sent to %d chat(s)", successCount)
	}

	return result, nil
}

// sendMessage sends a text message to a chat
func (p *Provider) sendMessage(ctx context.Context, botToken, chatID, text string) (string, error) {
	url := fmt.Sprintf("%s%s/sendMessage", apiBaseURL, botToken)

	reqBody := map[string]interface{}{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "MarkdownV2",
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		// If MarkdownV2 failed, try without parse mode
		return p.sendMessagePlain(ctx, botToken, chatID, text)
	}

	var respData struct {
		OK     bool `json:"ok"`
		Result struct {
			MessageID int `json:"message_id"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &respData); err == nil && respData.OK {
		return fmt.Sprintf("%d", respData.Result.MessageID), nil
	}

	return "", fmt.Errorf("telegram API error: %s", string(body))
}

// sendMessagePlain sends a message without markdown parsing
func (p *Provider) sendMessagePlain(ctx context.Context, botToken, chatID, text string) (string, error) {
	url := fmt.Sprintf("%s%s/sendMessage", apiBaseURL, botToken)

	// Strip markdown escaping for plain text
	text = stripMarkdownEscaping(text)

	reqBody := map[string]interface{}{
		"chat_id": chatID,
		"text":    text,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var respData struct {
		OK     bool `json:"ok"`
		Result struct {
			MessageID int `json:"message_id"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &respData); err == nil && respData.OK {
		return fmt.Sprintf("%d", respData.Result.MessageID), nil
	}

	return "", fmt.Errorf("telegram API error: %s", string(body))
}

// sendPhoto sends a photo with caption to a chat
func (p *Provider) sendPhoto(ctx context.Context, botToken, chatID, photoPath, caption string) (string, error) {
	url := fmt.Sprintf("%s%s/sendPhoto", apiBaseURL, botToken)

	// Open the photo file
	file, err := os.Open(photoPath)
	if err != nil {
		return "", fmt.Errorf("failed to open photo: %w", err)
	}
	defer file.Close()

	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add chat_id
	_ = writer.WriteField("chat_id", chatID)

	// Add caption
	if caption != "" {
		_ = writer.WriteField("caption", caption)
		_ = writer.WriteField("parse_mode", "MarkdownV2")
	}

	// Add photo file
	part, err := writer.CreateFormFile("photo", filepath.Base(photoPath))
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return "", fmt.Errorf("failed to copy file: %w", err)
	}

	writer.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", url, &buf)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send photo: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// If markdown parsing failed, try without
	if resp.StatusCode != 200 && caption != "" {
		return p.sendPhotoPlain(ctx, botToken, chatID, photoPath, caption)
	}

	var respData struct {
		OK     bool `json:"ok"`
		Result struct {
			MessageID int `json:"message_id"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &respData); err == nil && respData.OK {
		return fmt.Sprintf("%d", respData.Result.MessageID), nil
	}

	return "", fmt.Errorf("telegram API error: %s", string(body))
}

// sendPhotoPlain sends a photo without markdown parsing
func (p *Provider) sendPhotoPlain(ctx context.Context, botToken, chatID, photoPath, caption string) (string, error) {
	url := fmt.Sprintf("%s%s/sendPhoto", apiBaseURL, botToken)

	file, err := os.Open(photoPath)
	if err != nil {
		return "", fmt.Errorf("failed to open photo: %w", err)
	}
	defer file.Close()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	_ = writer.WriteField("chat_id", chatID)
	if caption != "" {
		_ = writer.WriteField("caption", stripMarkdownEscaping(caption))
	}

	part, err := writer.CreateFormFile("photo", filepath.Base(photoPath))
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return "", fmt.Errorf("failed to copy file: %w", err)
	}

	writer.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", url, &buf)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send photo: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var respData struct {
		OK     bool `json:"ok"`
		Result struct {
			MessageID int `json:"message_id"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &respData); err == nil && respData.OK {
		return fmt.Sprintf("%d", respData.Result.MessageID), nil
	}

	return "", fmt.Errorf("telegram API error: %s", string(body))
}

// stripMarkdownEscaping removes Telegram MarkdownV2 escape characters
func stripMarkdownEscaping(s string) string {
	// Remove backslash escapes for special characters
	specialChars := []string{"_", "[", "]", "(", ")", "~", "`", ">", "#", "+", "-", "=", "|", "{", "}", ".", "!"}
	for _, char := range specialChars {
		s = strings.ReplaceAll(s, "\\"+char, char)
	}
	// Remove bold markers
	s = strings.ReplaceAll(s, "*", "")
	return s
}

// ValidateCredentials checks if the provided credentials have valid format
func (p *Provider) ValidateCredentials(account *syndication.Account) error {
	if account.BotToken == "" {
		return errors.New("bot token is required")
	}

	// Bot token format: <bot_id>:<secret>
	if !strings.Contains(account.BotToken, ":") {
		return errors.New("invalid bot token format (should be bot_id:secret)")
	}

	if len(account.ChatIDs) == 0 {
		return errors.New("at least one chat ID is required")
	}

	return nil
}

// GetAccountInfo fetches and returns display info (bot username)
func (p *Provider) GetAccountInfo(ctx context.Context, account *syndication.Account, configDir string) (string, error) {
	if account.BotToken == "" {
		return "", errors.New("bot token not configured")
	}

	url := fmt.Sprintf("%s%s/getMe", apiBaseURL, account.BotToken)
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
		OK     bool `json:"ok"`
		Result struct {
			Username  string `json:"username"`
			FirstName string `json:"first_name"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &respData); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if !respData.OK {
		return "", errors.New("invalid bot token")
	}

	if respData.Result.Username != "" {
		return "@" + respData.Result.Username, nil
	}
	return respData.Result.FirstName, nil
}

// SupportsImages returns true if the platform supports image attachments
func (p *Provider) SupportsImages() bool {
	return true
}

// MaxMessageLength returns the max character limit
func (p *Provider) MaxMessageLength() int {
	return maxMessageLength
}

// RequiresAuth returns true if platform requires OAuth or similar auth flow
func (p *Provider) RequiresAuth() bool {
	return false // Bot token is provided directly
}

// GetRequiredFields returns the field names needed for this platform
func (p *Provider) GetRequiredFields() []string {
	return []string{"bot_token", "chat_ids"}
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
