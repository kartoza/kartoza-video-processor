# History

The History screen displays all your past recordings in a searchable, sortable table. From here you can browse, manage, and upload your recording library.

## Screen Preview

<div class="terminal-mockup">
<div class="terminal-header">
<div class="terminal-buttons">
<div class="terminal-button red"></div>
<div class="terminal-button yellow"></div>
<div class="terminal-button green"></div>
</div>
<div class="terminal-title">Recording History</div>
</div>
<div class="terminal-content"><span class="t-header">━━━━━━━━━━━━━━━ Recording History ━━━━━━━━━━━━━━━</span>

<span class="t-white">Status   Topic            Date         Duration    Size    </span>
<span class="t-gray">────────────────────────────────────────────────────────────</span>
<span class="t-selected"><span class="t-orange">→</span> <span class="t-green">✓ Done</span>   <span class="t-white">QGIS</span>    <span class="t-cyan">2024-01-15</span>   <span class="t-white">12:34</span>       <span class="t-green">245 MB</span></span>
  <span class="t-green">✓ Done</span>   <span class="t-blue">GIS development</span>  <span class="t-cyan">2024-01-14</span>   <span class="t-white">08:22</span>       <span class="t-green">156 MB</span>
  <span class="t-red">✗ Error</span>  <span class="t-blue">Open source</span>      <span class="t-cyan">2024-01-13</span>   <span class="t-white">15:47</span>       <span class="t-green">312 MB</span>
  <span class="t-green">✓ Done</span>   <span class="t-blue">QGIS</span>    <span class="t-cyan">2024-01-12</span>   <span class="t-white">05:18</span>       <span class="t-green">98 MB</span>
  <span class="t-orange">⟳ Proc</span>   <span class="t-blue">General tutorials</span><span class="t-cyan">2024-01-10</span>   <span class="t-white">20:05</span>       <span class="t-green">425 MB</span>

<span class="t-gray">────────────────────────────────────────────────────────────</span>
<span class="t-gray">Showing 1-5 of 42 recordings</span>

<span class="t-gray">━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━</span>
<span class="t-gray">↑/k: up • ↓/j: down • enter: view • o: open folder • u: upload • d: delete • q: back</span>
</div>
</div>

## Table Columns

### Status

<span class="t-green">**Status**</span>

Shows the current processing status of each recording:

| Icon | Status | Description |
|------|--------|-------------|
| <span class="t-green">✓ Done</span> | Completed | Recording processed successfully |
| <span class="t-red">✗ Error</span> | Failed | Processing encountered an error |
| <span class="t-orange">⟳ Proc</span> | Processing | Currently being processed |
| <span class="t-red">● Rec</span> | Recording | Currently being recorded |
| <span class="t-orange">⏸ Pause</span> | Paused | Recording is paused |

**YouTube Upload Indicator:**

A 🎬 (clapper) icon appears next to the status when the recording has been uploaded to YouTube. For example: <span class="t-green">✓ Done</span>🎬

---

### Topic

<span class="t-blue">**Topic**</span>

The category/topic assigned during recording setup. Recordings are organized into folders by topic.

---

### Date

<span class="t-cyan">**Date**</span>

The date the recording was created, formatted as `YYYY-MM-DD`.

---

### Duration

<span class="t-white">**Duration**</span>

The length of the recording in `MM:SS` or `HH:MM:SS` format.

---

### Size

<span class="t-green">**Size**</span>

The file size of the final processed video. Displayed in human-readable format (KB, MB, GB).

---

### Folder

<span class="t-gray">**Folder**</span>

The path to the recording's output folder. Shown when selected.

## Actions

### View Details

Press ++enter++ to view detailed information about the selected recording.

<div class="terminal-mockup">
<div class="terminal-header">
<div class="terminal-buttons">
<div class="terminal-button red"></div>
<div class="terminal-button yellow"></div>
<div class="terminal-button green"></div>
</div>
<div class="terminal-title">Recording Details</div>
</div>
<div class="terminal-content"><span class="t-header">━━━━━━━━━━━━ Recording Details ━━━━━━━━━━━━</span>

<span class="t-orange">Title:</span>       <span class="t-white">Introduction to sketcher sketches</span>
<span class="t-blue">Topic:</span>       <span class="t-white">QGIS sketches</span>
<span class="t-blue">Episode:</span>     <span class="t-white">#42</span>
<span class="t-blue">Date:</span>        <span class="t-cyan">2024-01-15 14:32:05</span>
<span class="t-blue">Duration:</span>    <span class="t-white">12 minutes 34 seconds</span>
<span class="t-blue">Size:</span>        <span class="t-green">245 MB</span>

<span class="t-header">Description</span>
<span class="t-gray">Learn how to use the sketcher tool in QGIS for</span>
<span class="t-gray">creating quick sketches and annotations...</span>

<span class="t-header">Files</span>
<span class="t-gray">• final.mp4 (245 MB)</span>
<span class="t-gray">• final_vertical.mp4 (180 MB)</span>
<span class="t-gray">• metadata.json</span>

  <span class="t-blue">[ Back ]</span>    <span class="t-green">[ Upload ]</span>    <span class="t-cyan">[ Open Folder ]</span>
</div>
</div>

---

### Open Folder

Press ++o++ to open the recording's folder in your system file manager.

**Behavior:**

- Opens folder containing `final.mp4` and related files
- Uses system default file manager (xdg-open on Linux)
- Allows manual file management

---

### Upload to YouTube

Press ++u++ to upload the selected recording to YouTube.

**Requirements:**

- YouTube must be configured in [Options](options.md)
- OAuth authentication must be complete
- `final.mp4` must exist

Navigates to [YouTube Upload](youtube-upload.md) screen.

---

### Play Video

There are three keybindings to play different versions of your recording:

| Key | Action |
|-----|--------|
| ++v++ | Play vertical video (falls back to merged if no vertical exists) |
| ++m++ | Play merged video (screen + audio combined) |
| ++a++ | Play normalized audio (falls back to original if normalization wasn't applied) |

**Behavior:**

- Uses `xdg-open` on Linux to launch the default media player
- Does not block the application - you can continue using the app while playing
- Only available for completed recordings
- Audio playback prefers the normalized audio track (`audio-normalized.wav`) if available

---

### View Error Details (Failed Recordings)

When viewing a recording that failed during processing, you'll see additional error information:

<div class="terminal-mockup">
<div class="terminal-header">
<div class="terminal-buttons">
<div class="terminal-button red"></div>
<div class="terminal-button yellow"></div>
<div class="terminal-button green"></div>
</div>
<div class="terminal-title">Recording Details (Failed)</div>
</div>
<div class="terminal-content"><span class="t-header">━━━━━━━━━━━━ Recording Details ━━━━━━━━━━━━</span>

<span class="t-orange">Title:</span>       <span class="t-white">Introduction to sketcher sketches</span>
<span class="t-blue">Topic:</span>       <span class="t-white">QGIS sketches</span>
...
<span class="t-header">────────────────────────────────────────────</span>
<span class="t-red-bg">✗ Processing Failed</span>

<span class="t-gray">Error:</span>  <span class="t-red">failed to merge recordings: ffmpeg error</span>

<span class="t-gray">Details:</span>
<span class="t-gray">Video post-processing failed.</span>
<span class="t-gray">Error: ffmpeg exited with code 1...</span>

<span class="t-orange">Press 'v' to view full error details and traceback</span>

<span class="t-gray">e: Edit • r: Reprocess • v: View Error Details • Esc: Back</span>
</div>
</div>

Press ++v++ to view the full error details with scrollable traceback information:

<div class="terminal-mockup">
<div class="terminal-header">
<div class="terminal-buttons">
<div class="terminal-button red"></div>
<div class="terminal-button yellow"></div>
<div class="terminal-button green"></div>
</div>
<div class="terminal-title">Error Details</div>
</div>
<div class="terminal-content"><span class="t-header">━━━━━━━━━━━━━━━ Error Details ━━━━━━━━━━━━━━━</span>

<span class="t-gray">Lines 1-25 of 45 ↓</span>

<span class="t-orange">Recording:</span> <span class="t-white">Introduction to sketcher sketches</span>
<span class="t-gray">Folder: 001-introduction-to-sketcher</span>

<span class="t-red">ERROR SUMMARY:</span>
  • failed to merge recordings: ffmpeg error

<span class="t-orange">DETAILED ERROR INFORMATION:</span>
<span class="t-gray">────────────────────────────────────────────────────────────</span>
<span class="t-gray">Video post-processing failed.</span>

<span class="t-gray">Error: ffmpeg exited with code 1</span>

<span class="t-gray">Processing Context:</span>
<span class="t-gray">  - Video file: /home/user/Videos/.../screen.mp4</span>
<span class="t-gray">  - Audio file: /home/user/Videos/.../audio.wav</span>
<span class="t-gray">  - Output directory: /home/user/Videos/...</span>
<span class="t-gray">...</span>

<span class="t-gray">↑/↓: Scroll • PgUp/PgDn: Page • r: Reprocess • Esc: Back</span>
</div>
</div>

**Error Details Include:**

- **Error Summary**: The primary error message
- **Processing Context**: Details about input files, output directory, and processing options
- **Possible Causes**: Suggestions based on the error type
- **Suggested Actions**: Steps to resolve the issue
- **Stack Trace**: Technical debugging information for bug reports

!!! tip "Recovering from Errors"
    You can try to fix the issue and press ++r++ to reprocess the recording. Common fixes include:

    - Ensuring sufficient disk space
    - Checking that input files exist
    - Restarting the application

---

### Edit Recording

Press ++e++ from the detail view to edit the recording's metadata.

<div class="terminal-mockup">
<div class="terminal-header">
<div class="terminal-buttons">
<div class="terminal-button red"></div>
<div class="terminal-button yellow"></div>
<div class="terminal-button green"></div>
</div>
<div class="terminal-title">Edit Recording</div>
</div>
<div class="terminal-content"><span class="t-header">━━━━━━━━━━━━━━━ Edit Recording ━━━━━━━━━━━━━━━</span>

<span class="t-gray">Folder:</span>    <span class="t-blue">001-introduction-to-qgis</span>
<span class="t-gray">Date:</span>      <span class="t-blue">2024-01-15</span>
<span class="t-gray">Duration:</span>  <span class="t-blue">12:34</span>

<span class="t-gray">────────────────────────────────────────────────────</span>

<span class="t-orange">Title:</span>
<span class="t-gray">┌────────────────────────────────────────────┐</span>
<span class="t-gray">│</span> <span class="t-white">Introduction to QGIS</span>              <span class="t-gray">│</span>
<span class="t-gray">└────────────────────────────────────────────┘</span>

<span class="t-gray">Description:</span>
<span class="t-gray">┌────────────────────────────────────────────┐</span>
<span class="t-gray">│</span> <span class="t-white">Learn how to get started with QGIS...</span> <span class="t-gray">│</span>
<span class="t-gray">└────────────────────────────────────────────┘</span>
<span class="t-orange">⚠ "color" → consider "colour" (UK spelling)</span>

<span class="t-gray">Topic:</span>     <span class="t-orange-bg"> QGIS </span>  <span class="t-gray"> GIS development </span>  <span class="t-gray"> General </span>

<span class="t-gray">Presenter:</span>
<span class="t-gray">┌────────────────────────────────────────────┐</span>
<span class="t-gray">│</span> <span class="t-white">Tim Sketcher</span>                       <span class="t-gray">│</span>
<span class="t-gray">└────────────────────────────────────────────┘</span>

<span class="t-gray">━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━</span>
<span class="t-gray">Tab: Next • ←/→: Topic • Ctrl+S: Save • Esc: Cancel</span>
</div>
</div>

**Editable Fields:**

| Field | Description |
|-------|-------------|
| **Title** | The video title (max 100 characters) |
| **Description** | The video description (max 2000 characters) |
| **Topic** | The category for the recording |
| **Presenter** | The presenter's name |

**Spell Check:**

The title and description fields are automatically spell-checked in real time:

- **UK English**: US spellings are flagged with UK alternatives (e.g., "color" → "colour")
- **Grammar**: Common grammar issues are detected (e.g., "a" vs "an", double spaces)
- **GIS Terms**: Common GIS and QGIS terminology is recognized and not flagged

Warnings appear as ⚠ messages below each field. Up to 3 issues are shown for the description field.

**Navigation:**

| Key | Action |
|-----|--------|
| ++tab++ | Move to next field |
| ++shift+tab++ | Move to previous field |
| ++left++ / ++right++ | Change topic selection |
| ++ctrl+s++ | Save changes |
| ++esc++ | Cancel editing |

---

### Delete Recording

Press ++d++ to delete the selected recording.

**Confirmation Dialog:**

<div class="terminal-mockup">
<div class="terminal-header">
<div class="terminal-buttons">
<div class="terminal-button red"></div>
<div class="terminal-button yellow"></div>
<div class="terminal-button green"></div>
</div>
<div class="terminal-title">Confirm Delete</div>
</div>
<div class="terminal-content"><span class="t-red">⚠ Delete Recording?</span>

<span class="t-white">This will permanently delete:</span>
<span class="t-gray">• Introduction to sketcher sketches</span>
<span class="t-gray">• All associated files (245 MB)</span>

<span class="t-red">This action cannot be undone!</span>

  <span class="t-red">[ Delete ]</span>    <span class="t-blue">[ Cancel ]</span>
</div>
</div>

!!! warning "Permanent Action"
    Deletion removes all files in the recording folder. This cannot be undone.

## Navigation

### Scrolling

| Key | Action |
|-----|--------|
| ++up++ / ++k++ | Move selection up |
| ++down++ / ++j++ | Move selection down |
| ++page-up++ | Scroll up one page |
| ++page-down++ | Scroll down one page |
| ++home++ | Go to first recording |
| ++end++ | Go to last recording |

### Actions

| Key | Action |
|-----|--------|
| ++enter++ | View recording details |
| ++e++ | Edit recording metadata |
| ++o++ | Open folder in file manager |
| ++u++ | Upload to YouTube |
| ++v++ | Play vertical video (completed) / View error details (failed) |
| ++m++ | Play merged video (completed recordings) |
| ++a++ | Play audio only (completed recordings) |
| ++r++ | Reprocess recording |
| ++d++ | Delete recording |
| ++q++ / ++esc++ | Return to main menu |

## Empty State

When no recordings exist:

<div class="terminal-mockup">
<div class="terminal-header">
<div class="terminal-buttons">
<div class="terminal-button red"></div>
<div class="terminal-button yellow"></div>
<div class="terminal-button green"></div>
</div>
<div class="terminal-title">Recording History</div>
</div>
<div class="terminal-content" style="text-align: center; padding: 60px 20px;"><span class="t-gray">No recordings yet.</span>

<span class="t-gray">Create your first recording from the main menu!</span>

  <span class="t-blue">[ Return to Menu ]</span>
</div>
</div>

## File Organization

Recordings are stored in:

```
~/Videos/Screencasts/
├── QGIS sketches/
│   ├── Introduction to sketcher/
│   │   ├── final.mp4
│   │   ├── final_vertical.mp4
│   │   └── metadata.json
│   └── Advanced sketching/
│       ├── final.mp4
│       └── metadata.json
├── GIS development tutorials/
│   └── ...
└── General tutorials/
    └── ...
```

## Keyboard Shortcuts Summary

| Key | Action |
|-----|--------|
| ++up++ / ++k++ | Move up |
| ++down++ / ++j++ | Move down |
| ++enter++ | View details |
| ++e++ | Edit recording metadata |
| ++o++ | Open folder |
| ++u++ | Upload to YouTube |
| ++v++ | Play vertical video / View error details |
| ++m++ | Play merged video |
| ++a++ | Play normalized audio |
| ++r++ | Reprocess recording |
| ++d++ | Delete recording |
| ++q++ / ++esc++ | Back to menu |

## Workflow Position

This screen is accessed from:

- **[Main Menu](main-menu.md)** → Select "Recording History"
- **[Processing](processing.md)** → "Return to Menu" then "Recording History"

## Related Pages

- **[Main Menu](main-menu.md)** - Return to main navigation
- **[YouTube Upload](youtube-upload.md)** - Upload recordings
- **[Processing](processing.md)** - How recordings are created
