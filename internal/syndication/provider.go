package syndication

import (
	"context"
	"io"
)

// PlatformType identifies a syndication platform
type PlatformType string

const (
	PlatformMastodon   PlatformType = "mastodon"
	PlatformBluesky    PlatformType = "bluesky"
	PlatformLinkedIn   PlatformType = "linkedin"
	PlatformTelegram   PlatformType = "telegram"
	PlatformSignal     PlatformType = "signal"
	PlatformNtfy       PlatformType = "ntfy"
	PlatformGoogleChat PlatformType = "googlechat"
	PlatformWordPress  PlatformType = "wordpress"
)

// AllPlatforms returns all supported platform types
func AllPlatforms() []PlatformType {
	return []PlatformType{
		PlatformMastodon,
		PlatformBluesky,
		PlatformLinkedIn,
		PlatformTelegram,
		PlatformSignal,
		PlatformNtfy,
		PlatformGoogleChat,
		PlatformWordPress,
	}
}

// PlatformDisplayName returns a human-readable name for a platform
func PlatformDisplayName(p PlatformType) string {
	switch p {
	case PlatformMastodon:
		return "Mastodon"
	case PlatformBluesky:
		return "Bluesky"
	case PlatformLinkedIn:
		return "LinkedIn"
	case PlatformTelegram:
		return "Telegram"
	case PlatformSignal:
		return "Signal"
	case PlatformNtfy:
		return "ntfy.sh"
	case PlatformGoogleChat:
		return "Google Chat"
	case PlatformWordPress:
		return "WordPress"
	default:
		return string(p)
	}
}

// PlatformIcon returns an icon/emoji for a platform
func PlatformIcon(p PlatformType) string {
	switch p {
	case PlatformMastodon:
		return "🐘"
	case PlatformBluesky:
		return "🦋"
	case PlatformLinkedIn:
		return "💼"
	case PlatformTelegram:
		return "✈️"
	case PlatformSignal:
		return "📡"
	case PlatformNtfy:
		return "🔔"
	case PlatformGoogleChat:
		return "💬"
	case PlatformWordPress:
		return "📝"
	default:
		return "📢"
	}
}

// PostContent contains the content to be posted
type PostContent struct {
	Title         string   // Video title
	Description   string   // Video description
	VideoURL      string   // YouTube video URL
	ThumbnailPath string   // Path to thumbnail image file
	Tags          []string // Tags/hashtags
	CustomMessage string   // Optional user-provided custom message
}

// PostResult contains the result of a post attempt
type PostResult struct {
	AccountID   string // Which account was used
	AccountName string // Display name of account
	Platform    PlatformType
	Success     bool
	PostURL     string // URL to view the post (if available)
	PostID      string // Platform-specific post ID
	Error       error
	Message     string // Success/error message for display
}

// Provider defines the interface all platform providers must implement
type Provider interface {
	// Platform returns the platform type
	Platform() PlatformType

	// Name returns the provider display name
	Name() string

	// Description returns a brief description of the platform
	Description() string

	// IsConfigured returns true if the account has required credentials
	IsConfigured(account *Account) bool

	// IsAuthenticated returns true if the account has a valid token/session
	IsAuthenticated(ctx context.Context, account *Account, configDir string) bool

	// Authenticate performs the authentication flow (OAuth, app password, etc.)
	// For OAuth providers, urlCallback is called with the auth URL for user to visit
	Authenticate(ctx context.Context, account *Account, configDir string, urlCallback func(string)) error

	// Post creates a new post/announcement on the platform
	Post(ctx context.Context, account *Account, configDir string, content *PostContent) (*PostResult, error)

	// ValidateCredentials checks if the provided credentials have valid format
	ValidateCredentials(account *Account) error

	// GetAccountInfo fetches and returns display info (username, etc.)
	GetAccountInfo(ctx context.Context, account *Account, configDir string) (string, error)

	// SupportsImages returns true if the platform supports image attachments
	SupportsImages() bool

	// MaxMessageLength returns the max character limit (0 for unlimited)
	MaxMessageLength() int

	// RequiresAuth returns true if platform requires OAuth or similar auth flow
	RequiresAuth() bool

	// GetRequiredFields returns the field names needed for this platform
	GetRequiredFields() []string
}

// ProviderWithImageUpload extends Provider for platforms that support image uploads
type ProviderWithImageUpload interface {
	Provider

	// UploadImage uploads an image and returns a media ID or URL
	UploadImage(ctx context.Context, account *Account, configDir string, reader io.Reader, filename, mimeType string) (string, error)
}

// Registry holds all registered providers
type Registry struct {
	providers map[PlatformType]Provider
}

// NewRegistry creates a new provider registry
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[PlatformType]Provider),
	}
}

// Register adds a provider to the registry
func (r *Registry) Register(provider Provider) {
	r.providers[provider.Platform()] = provider
}

// Get returns a provider by platform type
func (r *Registry) Get(platform PlatformType) (Provider, bool) {
	p, ok := r.providers[platform]
	return p, ok
}

// All returns all registered providers
func (r *Registry) All() []Provider {
	providers := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		providers = append(providers, p)
	}
	return providers
}

// GetByPlatform returns providers by platform type
func (r *Registry) GetByPlatform(platform PlatformType) Provider {
	return r.providers[platform]
}

// global registry instance
var globalRegistry *Registry

// GetRegistry returns the global provider registry
func GetRegistry() *Registry {
	if globalRegistry == nil {
		globalRegistry = NewRegistry()
	}
	return globalRegistry
}

// RegisterProvider registers a provider in the global registry
func RegisterProvider(provider Provider) {
	GetRegistry().Register(provider)
}
