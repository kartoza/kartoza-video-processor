# YouTube Upload Feature - Implementation Plan

## Overview

Add the ability to upload processed videos to YouTube after processing completes, using the YouTube Data API v3 with OAuth2 authentication.

## Architecture

### New Package: `internal/youtube`

```
internal/youtube/
├── auth.go          # OAuth2 authentication flow
├── upload.go        # Video upload functionality
├── playlist.go      # Playlist management
├── thumbnail.go     # Thumbnail extraction
└── config.go        # YouTube-specific configuration
```

### Dependencies

```go
import (
    "golang.org/x/oauth2"
    "golang.org/x/oauth2/google"
    "google.golang.org/api/youtube/v3"
    "google.golang.org/api/option"
)
```

## Implementation Details

### 1. OAuth2 Authentication Flow (`auth.go`)

**Flow for Desktop Applications:**
1. User initiates "Connect YouTube Account" from options screen
2. App starts local HTTP server on loopback (127.0.0.1:random_port)
3. Opens browser to Google OAuth consent screen
4. User grants permissions (youtube.upload scope)
5. Google redirects to local server with authorization code
6. App exchanges code for access + refresh tokens using PKCE
7. Tokens stored securely in config directory

**Required Scopes:**
- `https://www.googleapis.com/auth/youtube.upload` - Upload videos
- `https://www.googleapis.com/auth/youtube` - Manage playlists (for adding to playlist)

**Token Storage:**
- Store in `~/.config/kartoza-video-processor/youtube_token.json`
- Encrypt with user's system keyring if available, otherwise plain JSON
- Contains: access_token, refresh_token, expiry, token_type

**Functions:**
```go
type YouTubeAuth struct {
    config      *oauth2.Config
    tokenPath   string
}

func NewYouTubeAuth(clientID, clientSecret string) *YouTubeAuth
func (a *YouTubeAuth) IsAuthenticated() bool
func (a *YouTubeAuth) Authenticate(ctx context.Context) error  // Starts OAuth flow
func (a *YouTubeAuth) GetClient(ctx context.Context) (*http.Client, error)
func (a *YouTubeAuth) Logout() error  // Removes stored tokens
func (a *YouTubeAuth) RefreshToken(ctx context.Context) error
```

### 2. Configuration Updates (`config.go`)

**Add to Config struct:**
```go
type YouTubeConfig struct {
    ClientID          string `json:"client_id,omitempty"`
    ClientSecret      string `json:"client_secret,omitempty"`
    DefaultPlaylistID string `json:"default_playlist_id,omitempty"`
    DefaultPrivacy    string `json:"default_privacy,omitempty"`  // public, unlisted, private
    AutoUpload        bool   `json:"auto_upload,omitempty"`      // Prompt after processing
    UploadEnabled     bool   `json:"upload_enabled,omitempty"`   // Feature enabled
}

// In main Config struct:
type Config struct {
    // ... existing fields ...
    YouTube YouTubeConfig `json:"youtube,omitempty"`
}
```

**Client Credentials:**
- User provides their own OAuth client ID/secret from Google Cloud Console
- Instructions provided in options screen
- Alternatively, app could ship with default credentials (less secure)

### 3. Thumbnail Extraction (`thumbnail.go`)

**Extract thumbnail using FFmpeg:**
```go
func ExtractThumbnail(videoPath string, timestamp time.Duration, outputPath string) error {
    // ffmpeg -i video.mp4 -ss 60 -vframes 1 -q:v 2 thumbnail.jpg
}

func GetVideoDuration(videoPath string) (time.Duration, error) {
    // ffprobe to get duration
}

func ExtractThumbnailAuto(videoPath, outputPath string) error {
    // Extract at 60s or last frame if shorter
    duration, err := GetVideoDuration(videoPath)
    if err != nil {
        return err
    }

    timestamp := 60 * time.Second
    if duration < timestamp {
        timestamp = duration - time.Second  // Last frame
        if timestamp < 0 {
            timestamp = 0
        }
    }

    return ExtractThumbnail(videoPath, timestamp, outputPath)
}
```

### 4. Video Upload (`upload.go`)

```go
type UploadOptions struct {
    VideoPath     string
    Title         string
    Description   string
    Tags          []string
    CategoryID    string        // YouTube category (e.g., "27" for Education)
    PrivacyStatus string        // public, unlisted, private
    PlaylistID    string        // Optional: add to playlist after upload
    ThumbnailPath string        // Optional: custom thumbnail
    NotifySubscribers bool
}

type UploadResult struct {
    VideoID     string
    VideoURL    string
    PlaylistItemID string  // If added to playlist
}

type YouTubeUploader struct {
    service *youtube.Service
}

func NewYouTubeUploader(ctx context.Context, auth *YouTubeAuth) (*YouTubeUploader, error)
func (u *YouTubeUploader) Upload(ctx context.Context, opts UploadOptions, progress func(int64, int64)) (*UploadResult, error)
func (u *YouTubeUploader) SetThumbnail(ctx context.Context, videoID, thumbnailPath string) error
func (u *YouTubeUploader) AddToPlaylist(ctx context.Context, videoID, playlistID string) error
func (u *YouTubeUploader) ListPlaylists(ctx context.Context) ([]Playlist, error)
```

### 5. Playlist Management (`playlist.go`)

```go
type Playlist struct {
    ID          string
    Title       string
    Description string
    ItemCount   int64
}

func (u *YouTubeUploader) ListPlaylists(ctx context.Context) ([]Playlist, error)
func (u *YouTubeUploader) CreatePlaylist(ctx context.Context, title, description string) (*Playlist, error)
```

### 6. TUI Integration

**New Processing Step:**
Add optional "Upload to YouTube" step after processing completes.

```go
const (
    ProcessStepStopping = iota
    ProcessStepAnalyzing
    ProcessStepNormalizing
    ProcessStepMerging
    ProcessStepVertical
    ProcessStepUpload  // NEW
)
```

**Post-Processing Screen:**
After processing completes successfully, show:
1. "Upload to YouTube?" prompt (if YouTube is configured)
2. Allow editing title/description before upload
3. Select privacy status
4. Select/create playlist
5. Show upload progress with percentage

**Options Screen Additions:**
- "YouTube Integration" section
  - Connect/Disconnect YouTube account
  - Set OAuth client ID/secret
  - Default privacy setting
  - Default playlist
  - Enable/disable auto-upload prompt

### 7. New TUI Screens

**YouTubeSetupScreen:**
- Guide user through getting OAuth credentials
- Link to Google Cloud Console instructions
- Input fields for client ID and secret
- "Authenticate" button to start OAuth flow
- Status indicator (connected/not connected)

**YouTubeUploadScreen:**
- Shows video thumbnail preview
- Editable title (pre-filled from recording)
- Editable description (pre-filled)
- Topic tags
- Privacy dropdown (public/unlisted/private)
- Playlist selector
- Upload button
- Progress bar during upload

## User Flow

### First Time Setup:
1. User goes to Options > YouTube Integration
2. Clicks "Setup YouTube"
3. Shown instructions to create Google Cloud project and OAuth credentials
4. Enters Client ID and Client Secret
5. Clicks "Connect Account"
6. Browser opens to Google consent screen
7. User grants permissions
8. App receives tokens and stores them
9. Shows "Connected as: [channel name]"

### Upload Flow:
1. User completes recording and processing
2. Processing complete screen shows "Upload to YouTube?" button
3. User clicks to upload
4. Upload screen shows with pre-filled metadata
5. User can edit title, description, select privacy, select playlist
6. Clicks "Upload"
7. Progress bar shows upload progress
8. On success, shows link to video

## Security Considerations

1. **Token Storage**: Use OS keyring when available (keychain on macOS, libsecret on Linux)
2. **Client Secrets**: Stored in config file - user's responsibility to protect
3. **Token Refresh**: Automatically refresh expired tokens
4. **Minimal Scopes**: Only request necessary permissions

## Error Handling

1. **Authentication Errors**: Clear tokens, prompt re-authentication
2. **Upload Errors**: Show error message, allow retry
3. **Quota Errors**: Inform user about daily limits
4. **Network Errors**: Implement retry with exponential backoff

## Testing

1. Mock YouTube service for unit tests
2. Integration tests with test YouTube account
3. Test token refresh flow
4. Test upload with various video sizes

## File Changes Summary

### New Files:
- `internal/youtube/auth.go`
- `internal/youtube/upload.go`
- `internal/youtube/playlist.go`
- `internal/youtube/thumbnail.go`
- `internal/youtube/config.go`
- `internal/tui/youtube_setup.go`
- `internal/tui/youtube_upload.go`

### Modified Files:
- `internal/config/config.go` - Add YouTubeConfig
- `internal/tui/app.go` - Add YouTube screens and flow
- `internal/tui/options.go` - Add YouTube settings section
- `internal/tui/processing.go` - Add upload step (optional)
- `go.mod` - Add Google API dependencies

## Dependencies to Add:

```
go get golang.org/x/oauth2
go get golang.org/x/oauth2/google
go get google.golang.org/api/youtube/v3
go get google.golang.org/api/option
```

## Estimated Implementation Order:

1. Add YouTube config structure
2. Implement OAuth2 authentication flow
3. Implement thumbnail extraction
4. Implement video upload
5. Implement playlist management
6. Create YouTube setup TUI screen
7. Create YouTube upload TUI screen
8. Integrate into post-processing flow
9. Add to options screen
10. Testing and refinement
