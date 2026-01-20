package youtube

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

// Uploader handles YouTube video uploads
type Uploader struct {
	service *youtube.Service
	auth    *Auth
}

// NewUploader creates a new YouTube uploader
func NewUploader(ctx context.Context, auth *Auth) (*Uploader, error) {
	client, err := auth.GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get authenticated client: %w", err)
	}

	service, err := youtube.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create YouTube service: %w", err)
	}

	return &Uploader{
		service: service,
		auth:    auth,
	}, nil
}

// ProgressReader wraps an io.Reader to report progress
type ProgressReader struct {
	reader       io.Reader
	total        int64
	read         int64
	progressFunc func(read, total int64)
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.read += int64(n)
	if pr.progressFunc != nil {
		pr.progressFunc(pr.read, pr.total)
	}
	return n, err
}

// Upload uploads a video to YouTube
func (u *Uploader) Upload(ctx context.Context, opts UploadOptions, progressFunc func(read, total int64)) (*UploadResult, error) {
	// Validate options
	if opts.VideoPath == "" {
		return nil, fmt.Errorf("video path is required")
	}
	if opts.Title == "" {
		return nil, fmt.Errorf("title is required")
	}

	// Open video file
	file, err := os.Open(opts.VideoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open video file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Get file size for progress
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat video file: %w", err)
	}

	// Create progress reader
	reader := &ProgressReader{
		reader:       file,
		total:        fileInfo.Size(),
		progressFunc: progressFunc,
	}

	// Set default values
	privacyStatus := string(opts.PrivacyStatus)
	if privacyStatus == "" {
		privacyStatus = string(PrivacyUnlisted)
	}

	categoryID := opts.CategoryID
	if categoryID == "" {
		categoryID = DefaultCategoryID
	}

	// Create video metadata
	video := &youtube.Video{
		Snippet: &youtube.VideoSnippet{
			Title:       opts.Title,
			Description: opts.Description,
			Tags:        opts.Tags,
			CategoryId:  categoryID,
		},
		Status: &youtube.VideoStatus{
			PrivacyStatus: privacyStatus,
		},
	}

	// Perform upload
	call := u.service.Videos.Insert([]string{"snippet", "status"}, video)
	call = call.NotifySubscribers(opts.NotifySubscribers)
	call = call.Media(reader)
	call = call.Context(ctx)

	response, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("upload failed: %w", err)
	}

	result := &UploadResult{
		VideoID:  response.Id,
		VideoURL: fmt.Sprintf("https://www.youtube.com/watch?v=%s", response.Id),
	}

	// Set custom thumbnail if provided
	if opts.ThumbnailPath != "" {
		if err := u.SetThumbnail(ctx, response.Id, opts.ThumbnailPath); err != nil {
			// Log but don't fail the upload
			fmt.Printf("Warning: failed to set thumbnail: %v\n", err)
		}
	}

	// Add to playlist if specified
	if opts.PlaylistID != "" {
		playlistItemID, err := u.AddToPlaylist(ctx, response.Id, opts.PlaylistID)
		if err != nil {
			// Log but don't fail the upload
			fmt.Printf("Warning: failed to add to playlist: %v\n", err)
		} else {
			result.PlaylistItemID = playlistItemID
		}
	}

	return result, nil
}

// SetThumbnail sets a custom thumbnail for a video
func (u *Uploader) SetThumbnail(ctx context.Context, videoID, thumbnailPath string) error {
	file, err := os.Open(thumbnailPath)
	if err != nil {
		return fmt.Errorf("failed to open thumbnail: %w", err)
	}
	defer func() { _ = file.Close() }()

	call := u.service.Thumbnails.Set(videoID)
	call = call.Media(file)
	call = call.Context(ctx)

	_, err = call.Do()
	if err != nil {
		return fmt.Errorf("failed to set thumbnail: %w", err)
	}

	return nil
}

// AddToPlaylist adds a video to a playlist
func (u *Uploader) AddToPlaylist(ctx context.Context, videoID, playlistID string) (string, error) {
	playlistItem := &youtube.PlaylistItem{
		Snippet: &youtube.PlaylistItemSnippet{
			PlaylistId: playlistID,
			ResourceId: &youtube.ResourceId{
				Kind:    "youtube#video",
				VideoId: videoID,
			},
		},
	}

	call := u.service.PlaylistItems.Insert([]string{"snippet"}, playlistItem)
	call = call.Context(ctx)

	response, err := call.Do()
	if err != nil {
		return "", fmt.Errorf("failed to add to playlist: %w", err)
	}

	return response.Id, nil
}

// ListPlaylists returns all playlists for the authenticated user
func (u *Uploader) ListPlaylists(ctx context.Context) ([]Playlist, error) {
	var playlists []Playlist
	pageToken := ""

	for {
		call := u.service.Playlists.List([]string{"snippet", "contentDetails"})
		call = call.Mine(true)
		call = call.MaxResults(50)
		call = call.Context(ctx)

		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		response, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("failed to list playlists: %w", err)
		}

		for _, item := range response.Items {
			thumbnailURL := ""
			if item.Snippet.Thumbnails != nil && item.Snippet.Thumbnails.Default != nil {
				thumbnailURL = item.Snippet.Thumbnails.Default.Url
			}

			playlists = append(playlists, Playlist{
				ID:          item.Id,
				Title:       item.Snippet.Title,
				Description: item.Snippet.Description,
				ItemCount:   item.ContentDetails.ItemCount,
				Thumbnails:  thumbnailURL,
			})
		}

		pageToken = response.NextPageToken
		if pageToken == "" {
			break
		}
	}

	return playlists, nil
}

// CreatePlaylist creates a new playlist
func (u *Uploader) CreatePlaylist(ctx context.Context, title, description string, privacy PrivacyStatus) (*Playlist, error) {
	playlist := &youtube.Playlist{
		Snippet: &youtube.PlaylistSnippet{
			Title:       title,
			Description: description,
		},
		Status: &youtube.PlaylistStatus{
			PrivacyStatus: string(privacy),
		},
	}

	call := u.service.Playlists.Insert([]string{"snippet", "status"}, playlist)
	call = call.Context(ctx)

	response, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to create playlist: %w", err)
	}

	return &Playlist{
		ID:          response.Id,
		Title:       response.Snippet.Title,
		Description: response.Snippet.Description,
		ItemCount:   0,
	}, nil
}

// GetUploadQuota returns the remaining upload quota (approximate)
// Note: YouTube doesn't provide exact quota info via API
func (u *Uploader) GetUploadQuota(ctx context.Context) (string, error) {
	// YouTube API doesn't expose quota directly
	// This is a placeholder - in practice you'd track usage yourself
	return "Unknown (check Google Cloud Console)", nil
}

// BuildUploadOptions creates UploadOptions from recording metadata
func BuildUploadOptions(videoPath, title, description, topic string, tags []string, privacy PrivacyStatus) UploadOptions {
	// Add topic to tags if not already present
	topicTag := strings.ToLower(strings.ReplaceAll(topic, " ", "-"))
	hasTopicTag := false
	for _, tag := range tags {
		if strings.EqualFold(tag, topic) || strings.EqualFold(tag, topicTag) {
			hasTopicTag = true
			break
		}
	}
	if !hasTopicTag && topic != "" {
		tags = append(tags, topic)
	}

	// Add default tags
	defaultTags := []string{"kartoza", "screencast"}
	for _, dt := range defaultTags {
		found := false
		for _, t := range tags {
			if strings.EqualFold(t, dt) {
				found = true
				break
			}
		}
		if !found {
			tags = append(tags, dt)
		}
	}

	// Ensure we have a thumbnail path
	thumbnailPath := GetThumbnailPath(videoPath)

	return UploadOptions{
		VideoPath:         videoPath,
		Title:             title,
		Description:       description,
		Tags:              tags,
		CategoryID:        DefaultCategoryID,
		PrivacyStatus:     privacy,
		ThumbnailPath:     thumbnailPath,
		NotifySubscribers: privacy == PrivacyPublic, // Only notify for public videos
	}
}

// ValidateVideoFile checks if the video file is suitable for YouTube upload
func ValidateVideoFile(videoPath string) error {
	info, err := os.Stat(videoPath)
	if err != nil {
		return fmt.Errorf("cannot access video file: %w", err)
	}

	// Check file size (YouTube max is 256GB, but let's be reasonable)
	maxSize := int64(128 * 1024 * 1024 * 1024) // 128GB
	if info.Size() > maxSize {
		return fmt.Errorf("video file is too large (max 128GB)")
	}

	if info.Size() == 0 {
		return fmt.Errorf("video file is empty")
	}

	// Check extension
	ext := strings.ToLower(filepath.Ext(videoPath))
	validExtensions := map[string]bool{
		".mp4":  true,
		".mov":  true,
		".avi":  true,
		".wmv":  true,
		".flv":  true,
		".webm": true,
		".mkv":  true,
		".3gp":  true,
	}

	if !validExtensions[ext] {
		return fmt.Errorf("unsupported video format: %s", ext)
	}

	return nil
}

// GetVideoMetadata retrieves basic video metadata using ffprobe
func GetVideoMetadata(videoPath string) (duration string, resolution string, err error) {
	dur, err := GetVideoDuration(videoPath)
	if err != nil {
		return "", "", err
	}

	hours := int(dur.Hours())
	minutes := int(dur.Minutes()) % 60
	seconds := int(dur.Seconds()) % 60

	if hours > 0 {
		duration = fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
	} else {
		duration = fmt.Sprintf("%d:%02d", minutes, seconds)
	}

	// TODO: Get resolution using ffprobe
	resolution = "Unknown"

	return duration, resolution, nil
}

// UpdateVideoPrivacy updates the privacy status of a YouTube video
func (u *Uploader) UpdateVideoPrivacy(ctx context.Context, videoID string, privacy PrivacyStatus) error {
	// First, get the current video to preserve other metadata
	call := u.service.Videos.List([]string{"status"})
	call = call.Id(videoID)
	call = call.Context(ctx)

	response, err := call.Do()
	if err != nil {
		return fmt.Errorf("failed to get video: %w", err)
	}

	if len(response.Items) == 0 {
		return fmt.Errorf("video not found: %s", videoID)
	}

	video := response.Items[0]
	video.Status.PrivacyStatus = string(privacy)

	// Update the video
	updateCall := u.service.Videos.Update([]string{"status"}, video)
	updateCall = updateCall.Context(ctx)

	_, err = updateCall.Do()
	if err != nil {
		return fmt.Errorf("failed to update video privacy: %w", err)
	}

	return nil
}

// DeleteVideo deletes a video from YouTube
func (u *Uploader) DeleteVideo(ctx context.Context, videoID string) error {
	call := u.service.Videos.Delete(videoID)
	call = call.Context(ctx)

	err := call.Do()
	if err != nil {
		return fmt.Errorf("failed to delete video: %w", err)
	}

	return nil
}

// GetVideoInfo retrieves information about a YouTube video
func (u *Uploader) GetVideoInfo(ctx context.Context, videoID string) (*youtube.Video, error) {
	call := u.service.Videos.List([]string{"snippet", "status", "contentDetails"})
	call = call.Id(videoID)
	call = call.Context(ctx)

	response, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get video info: %w", err)
	}

	if len(response.Items) == 0 {
		return nil, fmt.Errorf("video not found: %s", videoID)
	}

	return response.Items[0], nil
}

// RemoveFromPlaylist removes a video from a playlist
func (u *Uploader) RemoveFromPlaylist(ctx context.Context, playlistItemID string) error {
	call := u.service.PlaylistItems.Delete(playlistItemID)
	call = call.Context(ctx)

	err := call.Do()
	if err != nil {
		return fmt.Errorf("failed to remove from playlist: %w", err)
	}

	return nil
}
