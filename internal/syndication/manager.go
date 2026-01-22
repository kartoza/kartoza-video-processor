package syndication

import (
	"context"
	"sync"
	"time"

	"github.com/kartoza/kartoza-video-processor/internal/models"
)

// Manager orchestrates syndication across multiple platforms
type Manager struct {
	config    *Config
	configDir string
	registry  *Registry
}

// NewManager creates a new syndication manager
func NewManager(config *Config, configDir string) *Manager {
	return &Manager{
		config:    config,
		configDir: configDir,
		registry:  GetRegistry(),
	}
}

// PostToAccounts posts content to the specified accounts
// Returns results for each account (success or failure)
func (m *Manager) PostToAccounts(ctx context.Context, accountIDs []string, content *PostContent) []PostResult {
	var results []PostResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, accountID := range accountIDs {
		account := m.config.GetAccount(accountID)
		if account == nil {
			results = append(results, PostResult{
				AccountID: accountID,
				Platform:  "",
				Success:   false,
				Message:   "Account not found",
			})
			continue
		}

		provider, ok := m.registry.Get(account.Platform)
		if !ok {
			results = append(results, PostResult{
				AccountID:   accountID,
				AccountName: account.GetDisplayName(),
				Platform:    account.Platform,
				Success:     false,
				Message:     "Provider not available",
			})
			continue
		}

		wg.Add(1)
		go func(acc *Account, prov Provider) {
			defer wg.Done()

			result, err := prov.Post(ctx, acc, m.configDir, content)
			if err != nil {
				result = &PostResult{
					AccountID:   acc.ID,
					AccountName: acc.GetDisplayName(),
					Platform:    acc.Platform,
					Success:     false,
					Error:       err,
					Message:     err.Error(),
				}
			}

			mu.Lock()
			results = append(results, *result)
			mu.Unlock()
		}(account, provider)
	}

	wg.Wait()
	return results
}

// PostToAllEnabled posts content to all enabled accounts
func (m *Manager) PostToAllEnabled(ctx context.Context, content *PostContent) []PostResult {
	enabled := m.config.GetEnabledAccounts()
	accountIDs := make([]string, len(enabled))
	for i, acc := range enabled {
		accountIDs[i] = acc.ID
	}
	return m.PostToAccounts(ctx, accountIDs, content)
}

// PostToDefaults posts content to the default accounts
func (m *Manager) PostToDefaults(ctx context.Context, content *PostContent) []PostResult {
	defaults := m.config.GetDefaultAccounts()
	accountIDs := make([]string, len(defaults))
	for i, acc := range defaults {
		accountIDs[i] = acc.ID
	}
	return m.PostToAccounts(ctx, accountIDs, content)
}

// RecordResults saves syndication results to recording metadata
func (m *Manager) RecordResults(metadata *models.RecordingMetadata, results []PostResult) {
	for _, result := range results {
		post := models.SyndicationPost{
			AccountID:   result.AccountID,
			Platform:    string(result.Platform),
			AccountName: result.AccountName,
			PostID:      result.PostID,
			PostURL:     result.PostURL,
			PostedAt:    time.Now().Format(time.RFC3339),
			Success:     result.Success,
		}
		if result.Error != nil {
			post.Error = result.Error.Error()
		}
		metadata.AddSyndicationPost(post)
	}
}

// GetAvailableAccounts returns all configured and enabled accounts grouped by platform
func (m *Manager) GetAvailableAccounts() map[PlatformType][]Account {
	result := make(map[PlatformType][]Account)
	for _, acc := range m.config.GetEnabledAccounts() {
		result[acc.Platform] = append(result[acc.Platform], acc)
	}
	return result
}

// GetAccountStatus returns the authentication status of an account
func (m *Manager) GetAccountStatus(ctx context.Context, accountID string) (configured, authenticated bool) {
	account := m.config.GetAccount(accountID)
	if account == nil {
		return false, false
	}

	provider, ok := m.registry.Get(account.Platform)
	if !ok {
		return false, false
	}

	configured = provider.IsConfigured(account)
	if configured {
		authenticated = provider.IsAuthenticated(ctx, account, m.configDir)
	}
	return
}

// AuthenticateAccount initiates authentication for an account
func (m *Manager) AuthenticateAccount(ctx context.Context, accountID string, urlCallback func(string)) error {
	account := m.config.GetAccount(accountID)
	if account == nil {
		return nil
	}

	provider, ok := m.registry.Get(account.Platform)
	if !ok {
		return nil
	}

	return provider.Authenticate(ctx, account, m.configDir, urlCallback)
}

// GetAccountInfo fetches display info for an account
func (m *Manager) GetAccountInfo(ctx context.Context, accountID string) (string, error) {
	account := m.config.GetAccount(accountID)
	if account == nil {
		return "", nil
	}

	provider, ok := m.registry.Get(account.Platform)
	if !ok {
		return "", nil
	}

	return provider.GetAccountInfo(ctx, account, m.configDir)
}

// TestAccount tests if an account can be used for posting
func (m *Manager) TestAccount(ctx context.Context, accountID string) error {
	account := m.config.GetAccount(accountID)
	if account == nil {
		return nil
	}

	provider, ok := m.registry.Get(account.Platform)
	if !ok {
		return nil
	}

	// First validate credentials format
	if err := provider.ValidateCredentials(account); err != nil {
		return err
	}

	// Then check authentication
	if provider.RequiresAuth() && !provider.IsAuthenticated(ctx, account, m.configDir) {
		return nil
	}

	return nil
}

// CreateContentFromMetadata creates PostContent from recording metadata
func CreateContentFromMetadata(metadata *models.RecordingMetadata, customMessage string) *PostContent {
	content := &PostContent{
		Title:         metadata.Title,
		Description:   metadata.Description,
		CustomMessage: customMessage,
	}

	if metadata.YouTube != nil {
		content.VideoURL = metadata.YouTube.VideoURL
		content.ThumbnailPath = metadata.YouTube.ThumbnailURL
	}

	// Convert topic to tag
	if metadata.Topic != "" {
		content.Tags = append(content.Tags, metadata.Topic)
	}

	return content
}

// BuildPreview generates a preview of what the post will look like
func BuildPreview(content *PostContent, platform PlatformType) (string, error) {
	provider, ok := GetRegistry().Get(platform)
	if !ok {
		// Use default template
		builder := NewPostBuilder(content)
		return builder.Build()
	}

	maxLen := provider.MaxMessageLength()
	builder := NewPostBuilder(content)
	return builder.BuildForPlatform(platform, maxLen)
}
