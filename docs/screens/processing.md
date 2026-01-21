# Processing

The Processing screen shows the progress of post-recording video processing. This is where your raw recordings are merged, enhanced, and finalized.

## Screen Preview

<div class="terminal-mockup">
<div class="terminal-header">
<div class="terminal-buttons">
<div class="terminal-button red"></div>
<div class="terminal-button yellow"></div>
<div class="terminal-button green"></div>
</div>
<div class="terminal-title">Processing Video</div>
</div>
<div class="terminal-content"><span class="t-header">━━━━━━━━━━━━━━━ Processing Video ━━━━━━━━━━━━━━━</span>

<span class="t-green">✓</span> <span class="t-white">Finalizing screen recording</span>       <span class="t-green">Done</span>
<span class="t-green">✓</span> <span class="t-white">Processing audio</span>                   <span class="t-green">Done</span>
<span class="t-green">✓</span> <span class="t-white">Merging video and audio</span>            <span class="t-green">Done</span>
<span class="t-cyan">◐</span> <span class="t-orange">Adding logo overlays</span>               <span class="t-cyan">47%</span>
    <span class="t-blue">████████████████████░░░░░░░░░░░░░░░░░░░░</span>
<span class="t-gray">○</span> <span class="t-gray">Creating vertical version</span>          <span class="t-gray">Pending</span>
<span class="t-gray">○</span> <span class="t-gray">Saving metadata</span>                    <span class="t-gray">Pending</span>



<span class="t-gray">━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━</span>
<span class="t-gray">Processing... Please wait</span>
</div>
</div>

## Processing Steps

The processing pipeline runs the following steps in order:

### 1. Finalizing Screen Recording

<span class="t-green">✓</span> **Finalizing screen recording**

Ensures the raw screen capture file is properly closed and finalized.

**Actions:**

- Waits for FFmpeg to complete writing
- Verifies file integrity
- Checks container format

---

### 2. Processing Audio

<span class="t-green">✓</span> **Processing audio**

Normalizes and optimizes the audio track.

**Actions:**

- Audio normalization (consistent volume)
- Noise reduction (if enabled)
- Format conversion to AAC

---

### 3. Merging Video and Audio

<span class="t-green">✓</span> **Merging video and audio**

Combines the separate video and audio streams into a single file.

**Actions:**

- Synchronizes audio with video
- Re-encodes to H.264/AAC
- Applies basic quality settings

---

### 4. Adding Webcam Overlay

<span class="t-gray">○</span> **Adding webcam overlay** *(conditional)*

*Only runs if webcam recording was enabled.*

**Actions:**

- Positions webcam in corner
- Scales webcam to appropriate size
- Composites over main video

---

### 5. Adding Logo Overlays

<span class="t-cyan">◐</span> **Adding logo overlays** *(conditional)*

*Only runs if logo overlays were enabled.*

**Actions:**

- Positions logos at configured corners
- Handles animated GIFs (loop mode)
- Adds title text overlay
- Composites all layers

**Logo Positions:**

```
┌────────────────────────────────────┐
│ [Left Logo]            [Right Logo]│
│                                    │
│          Recording Content         │
│                                    │
│           [Bottom Logo]            │
└────────────────────────────────────┘
```

---

### 6. Creating Vertical Version

<span class="t-gray">○</span> **Creating vertical version** *(conditional)*

*Only runs if vertical video was enabled.*

**Actions:**

- Crops to 9:16 aspect ratio
- Centers content
- Creates separate `_vertical.mp4` file

---

### 7. Saving Metadata

<span class="t-gray">○</span> **Saving metadata**

Writes recording information to a JSON file.

**Metadata includes:**

- Recording title and description
- Duration and file sizes
- Topic and episode number
- Timestamps
- Processing settings used

---

## Step Status Icons

| Icon | Status | Meaning |
|------|--------|---------|
| <span class="t-green">✓</span> | Complete | Step finished successfully |
| <span class="t-cyan">◐</span> | Running | Step currently in progress |
| <span class="t-gray">○</span> | Pending | Step waiting to run |
| <span class="t-red">✗</span> | Failed | Step encountered an error |
| <span class="t-yellow">⊘</span> | Skipped | Step not needed for this recording |

## Progress Bar

For long-running steps, a progress bar shows completion percentage:

<div class="terminal-mockup">
<div class="terminal-header">
<div class="terminal-buttons">
<div class="terminal-button red"></div>
<div class="terminal-button yellow"></div>
<div class="terminal-button green"></div>
</div>
<div class="terminal-title">Progress Examples</div>
</div>
<div class="terminal-content"><span class="t-blue">0%   ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░</span>
<span class="t-blue">25%  ██████████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░</span>
<span class="t-blue">50%  ████████████████████░░░░░░░░░░░░░░░░░░░░</span>
<span class="t-blue">75%  ██████████████████████████████░░░░░░░░░░</span>
<span class="t-green">100% ████████████████████████████████████████</span>
</div>
</div>

## Processing Complete

When all steps finish successfully:

<div class="terminal-mockup">
<div class="terminal-header">
<div class="terminal-buttons">
<div class="terminal-button red"></div>
<div class="terminal-button yellow"></div>
<div class="terminal-button green"></div>
</div>
<div class="terminal-title">Processing Complete</div>
</div>
<div class="terminal-content"><span class="t-header">━━━━━━━━━━━━━━━ Processing Complete ━━━━━━━━━━━━━</span>

<span class="t-green">✓</span> <span class="t-white">Finalizing screen recording</span>       <span class="t-green">Done</span>
<span class="t-green">✓</span> <span class="t-white">Processing audio</span>                   <span class="t-green">Done</span>
<span class="t-green">✓</span> <span class="t-white">Merging video and audio</span>            <span class="t-green">Done</span>
<span class="t-green">✓</span> <span class="t-white">Adding logo overlays</span>               <span class="t-green">Done</span>
<span class="t-green">✓</span> <span class="t-white">Creating vertical version</span>          <span class="t-green">Done</span>
<span class="t-green">✓</span> <span class="t-white">Saving metadata</span>                    <span class="t-green">Done</span>

<span class="t-green">━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━</span>
<span class="t-green">✓ All processing complete!</span>

  <span class="t-green">[ Upload to YouTube ]</span>    <span class="t-blue">[ Return to Menu ]</span>

<span class="t-gray">━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━</span>
<span class="t-gray">enter: select • q: quit</span>
</div>
</div>

**Options after completion:**

| Button | Description |
|--------|-------------|
| **Upload to YouTube** | Navigate to [YouTube Upload](youtube-upload.md) to share your recording |
| **Return to Menu** | Go back to [Main Menu](main-menu.md) |

!!! note "YouTube Button"
    The "Upload to YouTube" button only appears if YouTube is configured in [Options](options.md). Use ++left++ / ++right++ or ++tab++ to switch between buttons, then ++enter++ to confirm.

## Error Handling

If a step fails:

<div class="terminal-mockup">
<div class="terminal-header">
<div class="terminal-buttons">
<div class="terminal-button red"></div>
<div class="terminal-button yellow"></div>
<div class="terminal-button green"></div>
</div>
<div class="terminal-title">Processing Error</div>
</div>
<div class="terminal-content"><span class="t-green">✓</span> <span class="t-white">Finalizing screen recording</span>       <span class="t-green">Done</span>
<span class="t-green">✓</span> <span class="t-white">Processing audio</span>                   <span class="t-green">Done</span>
<span class="t-red">✗</span> <span class="t-red">Merging video and audio</span>            <span class="t-red">Failed</span>
    <span class="t-red">Error: FFmpeg returned exit code 1</span>

<span class="t-gray">━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━</span>
<span class="t-red">✗ Processing failed</span>

  <span class="t-yellow">[ Retry ]</span>    <span class="t-blue">[ Return to Menu ]</span>
</div>
</div>

## Output Files

After successful processing:

```
~/Videos/Screencasts/<topic>/<title>/
├── final.mp4           # Main processed video
├── final_vertical.mp4  # Vertical version (if enabled)
├── metadata.json       # Recording information
├── video.mkv           # Raw screen capture (preserved)
├── audio.wav           # Raw audio (preserved)
└── webcam.mkv          # Raw webcam (if used)
```

## Workflow Position

<div class="workflow-step">
<div class="workflow-step-number">1</div>
<div>
<strong>Previous:</strong> <a href="recording.md">Recording</a> → Stop recording
</div>
</div>

<div class="workflow-step">
<div class="workflow-step-number">2</div>
<div>
<strong>Current:</strong> Processing (this screen)
</div>
</div>

<div class="workflow-step">
<div class="workflow-step-number">3</div>
<div>
<strong>Next:</strong> <a href="youtube-upload.md">YouTube Upload</a> or <a href="main-menu.md">Main Menu</a>
</div>
</div>

## Technical Details

- **Encoding**: H.264 video, AAC audio
- **Quality**: CRF 23 (good quality, reasonable size)
- **Container**: MP4 for maximum compatibility
- **Progress reporting**: Via FFmpeg progress callback

## Related Pages

- **[Recording](recording.md)** - The recording that produces these files
- **[YouTube Upload](youtube-upload.md)** - Upload the processed video
- **[History](history.md)** - View past processed recordings
