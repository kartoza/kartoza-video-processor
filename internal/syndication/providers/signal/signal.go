package signal

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/kartoza/kartoza-video-processor/internal/syndication"
)

// Provider implements syndication.Provider for Signal via signal-cli
type Provider struct{}

func init() {
	syndication.RegisterProvider(&Provider{})
}

// Platform returns the platform type
func (p *Provider) Platform() syndication.PlatformType {
	return syndication.PlatformSignal
}

// Name returns the provider display name
func (p *Provider) Name() string {
	return "Signal"
}

// Description returns a brief description of the platform
func (p *Provider) Description() string {
	return "Send messages via Signal using signal-cli"
}

// IsConfigured returns true if the account has required credentials
func (p *Provider) IsConfigured(account *syndication.Account) bool {
	return account.SignalNumber != "" && len(account.Recipients) > 0
}

// IsAuthenticated returns true if signal-cli is available and registered
func (p *Provider) IsAuthenticated(ctx context.Context, account *syndication.Account, configDir string) bool {
	if !p.IsConfigured(account) {
		return false
	}

	// Check if signal-cli is available
	if !p.isSignalCLIAvailable() {
		return false
	}

	// Check if number is registered
	return p.isNumberRegistered(ctx, account.SignalNumber)
}

// Authenticate for Signal - signal-cli registration is handled externally
func (p *Provider) Authenticate(ctx context.Context, account *syndication.Account, configDir string, urlCallback func(string)) error {
	if account.SignalNumber == "" {
		return errors.New("signal number is required")
	}

	if !p.isSignalCLIAvailable() {
		return errors.New("signal-cli is not installed. Please install it from https://github.com/AsamK/signal-cli")
	}

	if !p.isNumberRegistered(ctx, account.SignalNumber) {
		return errors.New("signal number is not registered. Please register using: signal-cli -a " + account.SignalNumber + " register")
	}

	return nil
}

// Post sends a message via Signal to all configured recipients
func (p *Provider) Post(ctx context.Context, account *syndication.Account, configDir string, content *syndication.PostContent) (*syndication.PostResult, error) {
	result := &syndication.PostResult{
		AccountID:   account.ID,
		AccountName: account.GetDisplayName(),
		Platform:    syndication.PlatformSignal,
	}

	if !p.IsConfigured(account) {
		result.Error = errors.New("signal not configured")
		result.Message = "Signal number or recipients not configured"
		return result, nil
	}

	if !p.isSignalCLIAvailable() {
		result.Error = errors.New("signal-cli not installed")
		result.Message = "signal-cli is not installed"
		return result, nil
	}

	// Build message
	builder := syndication.NewPostBuilder(content)
	message, err := builder.Build()
	if err != nil {
		result.Error = fmt.Errorf("failed to build message: %w", err)
		result.Message = "Failed to create message"
		return result, nil
	}

	// Send to all recipients
	var successCount, failCount int
	var lastError error

	for _, recipient := range account.Recipients {
		err := p.sendMessage(ctx, account.SignalNumber, recipient, message, content.ThumbnailPath)
		if err != nil {
			failCount++
			lastError = err
		} else {
			successCount++
		}
	}

	// Determine overall result
	if successCount == 0 {
		result.Error = lastError
		result.Message = fmt.Sprintf("Failed to send to all %d recipients", len(account.Recipients))
		return result, nil
	}

	result.Success = true
	if failCount > 0 {
		result.Message = fmt.Sprintf("Sent to %d/%d recipients", successCount, len(account.Recipients))
	} else {
		result.Message = fmt.Sprintf("Sent to %d recipient(s)", successCount)
	}

	return result, nil
}

// sendMessage sends a message to a single recipient using signal-cli
func (p *Provider) sendMessage(ctx context.Context, senderNumber, recipient, message, attachmentPath string) error {
	args := []string{"-a", senderNumber, "send", "-m", message}

	// Determine if recipient is a group or individual
	if strings.HasPrefix(recipient, "group.") {
		args = append(args, "-g", recipient)
	} else {
		args = append(args, recipient)
	}

	// Add attachment if available
	if attachmentPath != "" && fileExists(attachmentPath) {
		args = append(args, "-a", attachmentPath)
	}

	cmd := exec.CommandContext(ctx, "signal-cli", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("signal-cli error: %v - %s", err, stderr.String())
	}

	return nil
}

// isSignalCLIAvailable checks if signal-cli is installed
func (p *Provider) isSignalCLIAvailable() bool {
	_, err := exec.LookPath("signal-cli")
	return err == nil
}

// isNumberRegistered checks if a number is registered with signal-cli
func (p *Provider) isNumberRegistered(ctx context.Context, number string) bool {
	cmd := exec.CommandContext(ctx, "signal-cli", "-a", number, "listIdentities")
	return cmd.Run() == nil
}

// ValidateCredentials checks if the provided credentials have valid format
func (p *Provider) ValidateCredentials(account *syndication.Account) error {
	if account.SignalNumber == "" {
		return errors.New("signal number is required")
	}

	// Validate phone number format (basic check)
	number := account.SignalNumber
	if !strings.HasPrefix(number, "+") {
		return errors.New("signal number must start with + (e.g., +1234567890)")
	}

	// Remove + and check remaining are digits
	digits := strings.TrimPrefix(number, "+")
	for _, c := range digits {
		if c < '0' || c > '9' {
			return errors.New("signal number must contain only digits after +")
		}
	}

	if len(digits) < 7 || len(digits) > 15 {
		return errors.New("signal number appears invalid (should be 7-15 digits)")
	}

	if len(account.Recipients) == 0 {
		return errors.New("at least one recipient is required")
	}

	// Validate each recipient
	for _, recipient := range account.Recipients {
		if strings.HasPrefix(recipient, "group.") {
			// Group ID format
			continue
		}
		// Phone number
		if !strings.HasPrefix(recipient, "+") {
			return fmt.Errorf("recipient %q must start with + or be a group ID", recipient)
		}
	}

	return nil
}

// GetAccountInfo fetches and returns display info
func (p *Provider) GetAccountInfo(ctx context.Context, account *syndication.Account, configDir string) (string, error) {
	if account.SignalNumber == "" {
		return "", errors.New("signal number not configured")
	}

	recipientCount := len(account.Recipients)
	return fmt.Sprintf("%s (%d recipients)", account.SignalNumber, recipientCount), nil
}

// SupportsImages returns true if the platform supports image attachments
func (p *Provider) SupportsImages() bool {
	return true
}

// MaxMessageLength returns the max character limit (0 for unlimited)
func (p *Provider) MaxMessageLength() int {
	return 0 // Signal has no practical limit
}

// RequiresAuth returns true if platform requires authentication
func (p *Provider) RequiresAuth() bool {
	return true // signal-cli registration required
}

// GetRequiredFields returns the field names needed for this platform
func (p *Provider) GetRequiredFields() []string {
	return []string{"signal_number", "recipients"}
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// GetGroups retrieves the list of groups for a registered number
func GetGroups(ctx context.Context, number string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "signal-cli", "-a", number, "listGroups")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to list groups: %v - %s", err, stderr.String())
	}

	var groups []string
	lines := strings.Split(stdout.String(), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Id: ") {
			groupID := strings.TrimPrefix(line, "Id: ")
			groups = append(groups, groupID)
		}
	}

	return groups, nil
}
