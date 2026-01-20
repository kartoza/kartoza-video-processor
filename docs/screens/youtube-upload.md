# YouTube Upload

The YouTube Upload screen handles the video upload process to YouTube, including metadata configuration and progress tracking.

## Screen Preview

<div class="terminal-mockup">
<div class="terminal-header">
<div class="terminal-buttons">
<div class="terminal-button red"></div>
<div class="terminal-button yellow"></div>
<div class="terminal-button green"></div>
</div>
<div class="terminal-title">YouTube Upload</div>
</div>
<div class="terminal-content"><span class="t-header">━━━━━━━━━━━━━━━ Upload to YouTube ━━━━━━━━━━━━━━</span>

<span class="t-orange">Title:</span>
<span class="t-cyan">┌────────────────────────────────────────────────┐</span>
<span class="t-cyan">│</span> <span class="t-white">Introduction to QGIS sketcher - Episode 42</span>    <span class="t-cyan">│</span>
<span class="t-cyan">└────────────────────────────────────────────────┘</span>

<span class="t-blue">Description:</span>
<span class="t-gray">┌────────────────────────────────────────────────┐</span>
<span class="t-gray">│</span> <span class="t-white">Learn how to use the sketcher tool in QGIS</span>    <span class="t-gray">│</span>
<span class="t-gray">│</span> <span class="t-white">for creating quick sketches and annotations.</span>  <span class="t-gray">│</span>
<span class="t-gray">│</span>                                                <span class="t-gray">│</span>
<span class="t-gray">│</span> <span class="t-white">Presented by Tim Sketcher</span>                     <span class="t-gray">│</span>
<span class="t-gray">└────────────────────────────────────────────────┘</span>

<span class="t-blue">Privacy:</span>     <span class="t-white">Unlisted</span>       <span class="t-gray">←/→ to change</span>
<span class="t-blue">Playlist:</span>    <span class="t-white">QGIS Tutorials</span> <span class="t-gray">←/→ to change</span>

<span class="t-gray">File:</span> <span class="t-cyan">~/Videos/Screencasts/.../final.mp4</span> <span class="t-gray">(245 MB)</span>

  <span class="t-green">[ Upload ]</span>    <span class="t-gray">[ Cancel ]</span>

<span class="t-gray">━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━</span>
<span class="t-gray">tab: next field • enter: upload • esc: cancel</span>
</div>
</div>

## Upload Configuration

### Title

<span class="t-orange">**Title:**</span> *Text Input*

The video title displayed on YouTube. Pre-filled from recording metadata.

**Character Limit:** 100 characters

---

### Description

<span class="t-blue">**Description:**</span> *Text Area*

The video description. Pre-filled from recording metadata.

**Best Practices:**

- Include relevant keywords
- Add timestamps for sections
- Credit resources used
- Include links to related content

---

### Privacy

<span class="t-blue">**Privacy:**</span> *Selection*

| Setting | Icon | Description |
|---------|------|-------------|
| **Public** | 🌐 | Anyone can search and watch |
| **Unlisted** | 🔗 | Only accessible via direct link |
| **Private** | 🔒 | Only you can watch |

Use ++left++ / ++right++ to change.

---

### Playlist

<span class="t-blue">**Playlist:**</span> *Selection*

Select a playlist to add the video to. The video will be appended to the playlist after upload.

**Options:**

- Your existing playlists
- "(none)" - Don't add to a playlist

---

### File Information

<span class="t-gray">**File:**</span> *Display Only*

Shows the video file path and size that will be uploaded.

## Upload Progress

After pressing **[ Upload ]**:

<div class="terminal-mockup">
<div class="terminal-header">
<div class="terminal-buttons">
<div class="terminal-button red"></div>
<div class="terminal-button yellow"></div>
<div class="terminal-button green"></div>
</div>
<div class="terminal-title">Uploading</div>
</div>
<div class="terminal-content"><span class="t-header">━━━━━━━━━━━━━━━ Uploading Video ━━━━━━━━━━━━━━━</span>

<span class="t-white">Introduction to QGIS sketcher - Episode 42</span>

<span class="t-cyan">Uploading...</span> <span class="t-white">67%</span>
<span class="t-blue">████████████████████████████░░░░░░░░░░░░░░</span>

<span class="t-gray">164 MB / 245 MB</span>
<span class="t-gray">Speed: 12.5 MB/s</span>
<span class="t-gray">ETA: ~6 seconds</span>

<span class="t-gray">━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━</span>
<span class="t-gray">Please wait... Do not close the application</span>
</div>
</div>

### Progress Information

| Field | Description |
|-------|-------------|
| **Percentage** | Upload completion (0-100%) |
| **Progress Bar** | Visual progress indicator |
| **Bytes** | Uploaded / Total bytes |
| **Speed** | Current upload speed |
| **ETA** | Estimated time remaining |

## Upload Complete

After successful upload:

<div class="terminal-mockup">
<div class="terminal-header">
<div class="terminal-buttons">
<div class="terminal-button red"></div>
<div class="terminal-button yellow"></div>
<div class="terminal-button green"></div>
</div>
<div class="terminal-title">Upload Complete</div>
</div>
<div class="terminal-content"><span class="t-header">━━━━━━━━━━━━━━━ Upload Complete ━━━━━━━━━━━━━━━</span>

<span class="t-green">✓ Video uploaded successfully!</span>

<span class="t-blue">Title:</span>    <span class="t-white">Introduction to QGIS sketcher - Episode 42</span>
<span class="t-blue">Privacy:</span>  <span class="t-white">Unlisted</span>
<span class="t-blue">Playlist:</span> <span class="t-white">QGIS Tutorials</span>

<span class="t-header">Video URL</span>
<span class="t-cyan">https://youtu.be/dQw4w9WgXcQ</span>

  <span class="t-green">[ Open in Browser ]</span>    <span class="t-blue">[ Done ]</span>

<span class="t-gray">━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━</span>
<span class="t-gray">enter: select • q: done</span>
</div>
</div>

### Post-Upload Actions

| Action | Description |
|--------|-------------|
| **Open in Browser** | Opens the YouTube video page |
| **Done** | Returns to previous screen |

## Error Handling

### Upload Failed

<div class="terminal-mockup">
<div class="terminal-header">
<div class="terminal-buttons">
<div class="terminal-button red"></div>
<div class="terminal-button yellow"></div>
<div class="terminal-button green"></div>
</div>
<div class="terminal-title">Upload Error</div>
</div>
<div class="terminal-content"><span class="t-red">✗ Upload Failed</span>

<span class="t-red">Error: Network connection lost</span>

<span class="t-gray">The upload was interrupted. Your video was not</span>
<span class="t-gray">published to YouTube.</span>

  <span class="t-yellow">[ Retry ]</span>    <span class="t-gray">[ Cancel ]</span>
</div>
</div>

**Common Errors:**

| Error | Cause | Solution |
|-------|-------|----------|
| Network error | Connection lost | Check internet, retry |
| Auth expired | Token expired | Re-authenticate in Options |
| Quota exceeded | API limit reached | Wait 24 hours |
| Invalid video | File corrupt | Re-export video |

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| ++tab++ | Next field |
| ++shift+tab++ | Previous field |
| ++left++ / ++right++ | Change selection |
| ++enter++ | Upload / Select |
| ++esc++ | Cancel |

## Workflow Position

This screen is accessed from:

- **[Processing](processing.md)** → "Upload to YouTube"
- **[History](history.md)** → Press ++u++ on a recording

After upload:

- **[Main Menu](main-menu.md)** → Return to start
- **Browser** → View video on YouTube

## Technical Details

- **API:** YouTube Data API v3
- **Upload Method:** Resumable uploads
- **Chunk Size:** 256 KB (for resume support)
- **Retry Logic:** Automatic retry on transient failures

## Related Pages

- **[YouTube Setup](youtube-setup.md)** - Configure credentials first
- **[Processing](processing.md)** - Process before uploading
- **[History](history.md)** - Upload past recordings
