package syndication

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

// DefaultPostTemplate is the default template for syndication posts
const DefaultPostTemplate = `{{.Title}}

{{.Description}}

{{if .VideoURL}}Watch: {{.VideoURL}}{{end}}
{{if .Tags}}
{{.FormattedTags}}{{end}}`

// ShortPostTemplate is for platforms with character limits
const ShortPostTemplate = `{{.Title}}

{{if .VideoURL}}{{.VideoURL}}{{end}}
{{if .Tags}}
{{.FormattedTags}}{{end}}`

// PostBuilder builds formatted post content for different platforms
type PostBuilder struct {
	content  *PostContent
	template string
}

// NewPostBuilder creates a new post builder
func NewPostBuilder(content *PostContent) *PostBuilder {
	return &PostBuilder{
		content:  content,
		template: DefaultPostTemplate,
	}
}

// WithTemplate sets a custom template
func (pb *PostBuilder) WithTemplate(tmpl string) *PostBuilder {
	if tmpl != "" {
		pb.template = tmpl
	}
	return pb
}

// WithShortTemplate uses the short template for character-limited platforms
func (pb *PostBuilder) WithShortTemplate() *PostBuilder {
	pb.template = ShortPostTemplate
	return pb
}

// templateData prepares data for template execution
type templateData struct {
	Title         string
	Description   string
	VideoURL      string
	ThumbnailPath string
	Tags          []string
	CustomMessage string
	FormattedTags string
}

// Build renders the post content using the template
func (pb *PostBuilder) Build() (string, error) {
	tmpl, err := template.New("post").Parse(pb.template)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	data := templateData{
		Title:         pb.content.Title,
		Description:   pb.content.Description,
		VideoURL:      pb.content.VideoURL,
		ThumbnailPath: pb.content.ThumbnailPath,
		Tags:          pb.content.Tags,
		CustomMessage: pb.content.CustomMessage,
		FormattedTags: pb.formatTags(),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	// Clean up extra whitespace
	result := strings.TrimSpace(buf.String())
	// Remove multiple consecutive newlines
	for strings.Contains(result, "\n\n\n") {
		result = strings.ReplaceAll(result, "\n\n\n", "\n\n")
	}

	return result, nil
}

// BuildWithCustomMessage prepends a custom message if provided
func (pb *PostBuilder) BuildWithCustomMessage() (string, error) {
	base, err := pb.Build()
	if err != nil {
		return "", err
	}

	if pb.content.CustomMessage != "" {
		return pb.content.CustomMessage + "\n\n" + base, nil
	}
	return base, nil
}

// BuildForPlatform builds content optimized for a specific platform
func (pb *PostBuilder) BuildForPlatform(platform PlatformType, maxLength int) (string, error) {
	var content string
	var err error

	// Use short template for platforms with strict limits
	if maxLength > 0 && maxLength < 500 {
		pb.WithShortTemplate()
	}

	if pb.content.CustomMessage != "" {
		content, err = pb.BuildWithCustomMessage()
	} else {
		content, err = pb.Build()
	}

	if err != nil {
		return "", err
	}

	// Truncate if needed
	if maxLength > 0 && len(content) > maxLength {
		content = truncateText(content, maxLength, pb.content.VideoURL)
	}

	return content, nil
}

// formatTags formats tags as hashtags
func (pb *PostBuilder) formatTags() string {
	if len(pb.content.Tags) == 0 {
		return ""
	}

	var hashtags []string
	for _, tag := range pb.content.Tags {
		// Clean the tag and add hashtag prefix
		cleaned := strings.TrimPrefix(tag, "#")
		cleaned = strings.ReplaceAll(cleaned, " ", "")
		if cleaned != "" {
			hashtags = append(hashtags, "#"+cleaned)
		}
	}

	return strings.Join(hashtags, " ")
}

// truncateText truncates text to maxLength while preserving the URL
func truncateText(text string, maxLength int, preserveURL string) string {
	if len(text) <= maxLength {
		return text
	}

	// Reserve space for URL and ellipsis if URL should be preserved
	urlSpace := 0
	if preserveURL != "" && strings.Contains(text, preserveURL) {
		urlSpace = len(preserveURL) + 5 // URL + newline + ellipsis + space
	}

	// Find where to cut
	cutPoint := maxLength - 3 // Room for "..."
	if urlSpace > 0 {
		cutPoint -= urlSpace
	}

	if cutPoint < 50 {
		cutPoint = 50 // Minimum content
	}

	// Try to cut at a word boundary
	if idx := strings.LastIndex(text[:cutPoint], " "); idx > cutPoint-20 {
		cutPoint = idx
	}

	truncated := strings.TrimSpace(text[:cutPoint]) + "..."

	// Append URL if it was supposed to be preserved but got cut
	if preserveURL != "" && !strings.Contains(truncated, preserveURL) {
		truncated += "\n\n" + preserveURL
	}

	return truncated
}

// BuildMarkdown builds the post with Markdown formatting (for platforms that support it)
func (pb *PostBuilder) BuildMarkdown() (string, error) {
	var parts []string

	// Title as bold
	if pb.content.Title != "" {
		parts = append(parts, "**"+pb.content.Title+"**")
	}

	// Custom message
	if pb.content.CustomMessage != "" {
		parts = append(parts, pb.content.CustomMessage)
	}

	// Description
	if pb.content.Description != "" {
		parts = append(parts, pb.content.Description)
	}

	// Video link
	if pb.content.VideoURL != "" {
		parts = append(parts, fmt.Sprintf("[Watch Video](%s)", pb.content.VideoURL))
	}

	// Tags
	if tags := pb.formatTags(); tags != "" {
		parts = append(parts, tags)
	}

	return strings.Join(parts, "\n\n"), nil
}

// BuildHTML builds the post with HTML formatting (for WordPress, etc.)
func (pb *PostBuilder) BuildHTML() (string, error) {
	var parts []string

	// Custom message
	if pb.content.CustomMessage != "" {
		parts = append(parts, "<p>"+escapeHTML(pb.content.CustomMessage)+"</p>")
	}

	// Video embed
	if pb.content.VideoURL != "" {
		// Extract video ID for YouTube embed
		if videoID := extractYouTubeID(pb.content.VideoURL); videoID != "" {
			parts = append(parts, fmt.Sprintf(
				`<div class="video-container"><iframe width="560" height="315" src="https://www.youtube.com/embed/%s" frameborder="0" allowfullscreen></iframe></div>`,
				videoID,
			))
		} else {
			parts = append(parts, fmt.Sprintf(`<p><a href="%s">Watch Video</a></p>`, escapeHTML(pb.content.VideoURL)))
		}
	}

	// Description
	if pb.content.Description != "" {
		// Convert newlines to paragraphs
		paragraphs := strings.Split(pb.content.Description, "\n\n")
		for _, p := range paragraphs {
			p = strings.TrimSpace(p)
			if p != "" {
				parts = append(parts, "<p>"+escapeHTML(p)+"</p>")
			}
		}
	}

	// Tags
	if len(pb.content.Tags) > 0 {
		var tagLinks []string
		for _, tag := range pb.content.Tags {
			cleaned := strings.TrimPrefix(tag, "#")
			tagLinks = append(tagLinks, fmt.Sprintf(`<span class="tag">#%s</span>`, escapeHTML(cleaned)))
		}
		parts = append(parts, `<p class="tags">`+strings.Join(tagLinks, " ")+`</p>`)
	}

	return strings.Join(parts, "\n"), nil
}

// escapeHTML escapes HTML special characters
func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	return s
}

// extractYouTubeID extracts the video ID from a YouTube URL
func extractYouTubeID(url string) string {
	// Handle various YouTube URL formats
	patterns := []string{
		"youtube.com/watch?v=",
		"youtu.be/",
		"youtube.com/embed/",
	}

	for _, pattern := range patterns {
		if idx := strings.Index(url, pattern); idx != -1 {
			start := idx + len(pattern)
			// Extract ID (ends at & or end of string)
			end := start
			for end < len(url) && url[end] != '&' && url[end] != '?' {
				end++
			}
			if end > start && end-start <= 15 {
				return url[start:end]
			}
		}
	}
	return ""
}

// BuildTelegramMessage builds a message formatted for Telegram
func (pb *PostBuilder) BuildTelegramMessage() string {
	var parts []string

	// Title in bold
	if pb.content.Title != "" {
		parts = append(parts, "*"+escapeTelegramMarkdown(pb.content.Title)+"*")
	}

	// Custom message
	if pb.content.CustomMessage != "" {
		parts = append(parts, escapeTelegramMarkdown(pb.content.CustomMessage))
	}

	// Description (truncated for readability)
	if pb.content.Description != "" {
		desc := pb.content.Description
		if len(desc) > 300 {
			desc = desc[:297] + "..."
		}
		parts = append(parts, escapeTelegramMarkdown(desc))
	}

	// Video link
	if pb.content.VideoURL != "" {
		parts = append(parts, pb.content.VideoURL)
	}

	// Tags
	if tags := pb.formatTags(); tags != "" {
		parts = append(parts, tags)
	}

	return strings.Join(parts, "\n\n")
}

// escapeTelegramMarkdown escapes Telegram MarkdownV2 special characters
func escapeTelegramMarkdown(s string) string {
	// Characters that need escaping in MarkdownV2
	specialChars := []string{"_", "[", "]", "(", ")", "~", "`", ">", "#", "+", "-", "=", "|", "{", "}", ".", "!"}
	for _, char := range specialChars {
		s = strings.ReplaceAll(s, char, "\\"+char)
	}
	return s
}

// BuildNtfyMessage builds a message for ntfy.sh
func (pb *PostBuilder) BuildNtfyMessage() (title, message string) {
	title = pb.content.Title

	var parts []string
	if pb.content.CustomMessage != "" {
		parts = append(parts, pb.content.CustomMessage)
	}
	if pb.content.Description != "" {
		desc := pb.content.Description
		if len(desc) > 200 {
			desc = desc[:197] + "..."
		}
		parts = append(parts, desc)
	}

	message = strings.Join(parts, "\n\n")
	return
}

// BuildGoogleChatCard builds a Google Chat card structure
func (pb *PostBuilder) BuildGoogleChatCard() map[string]interface{} {
	card := map[string]interface{}{
		"cards": []map[string]interface{}{
			{
				"header": map[string]interface{}{
					"title": pb.content.Title,
				},
				"sections": []map[string]interface{}{},
			},
		},
	}

	sections := []map[string]interface{}{}

	// Thumbnail if available
	if pb.content.ThumbnailPath != "" {
		// Note: For actual implementation, thumbnail would need to be uploaded
		// or a URL would need to be provided
	}

	// Description section
	if pb.content.Description != "" {
		desc := pb.content.Description
		if len(desc) > 500 {
			desc = desc[:497] + "..."
		}
		sections = append(sections, map[string]interface{}{
			"widgets": []map[string]interface{}{
				{
					"textParagraph": map[string]interface{}{
						"text": desc,
					},
				},
			},
		})
	}

	// Custom message
	if pb.content.CustomMessage != "" {
		sections = append(sections, map[string]interface{}{
			"widgets": []map[string]interface{}{
				{
					"textParagraph": map[string]interface{}{
						"text": pb.content.CustomMessage,
					},
				},
			},
		})
	}

	// Button to video
	if pb.content.VideoURL != "" {
		sections = append(sections, map[string]interface{}{
			"widgets": []map[string]interface{}{
				{
					"buttons": []map[string]interface{}{
						{
							"textButton": map[string]interface{}{
								"text": "Watch Video",
								"onClick": map[string]interface{}{
									"openLink": map[string]interface{}{
										"url": pb.content.VideoURL,
									},
								},
							},
						},
					},
				},
			},
		})
	}

	card["cards"].([]map[string]interface{})[0]["sections"] = sections
	return card
}
