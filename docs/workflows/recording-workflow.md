# Recording a Video

This workflow guide walks you through the complete process of creating a screen recording from start to finish.

## Workflow Overview

```mermaid
graph TD
    A[Main Menu] -->|New Recording| B[Recording Setup]
    B -->|Go Live| C[Countdown]
    C -->|5...4...3...2...1...GO!| D[Recording]
    D -->|Stop| E[Processing]
    E -->|Complete| F{What next?}
    F -->|Upload| G[YouTube Upload]
    F -->|Done| H[Main Menu]
    G -->|Complete| H
```

## Step-by-Step Guide

<div class="workflow-step">
<div class="workflow-step-number">1</div>
<div>
<strong>Start from Main Menu</strong><br>
Launch the application and select <strong>New Recording</strong> from the <a href="../screens/main-menu.md">Main Menu</a>.
</div>
</div>

---

### Configure Your Recording

<div class="workflow-step">
<div class="workflow-step-number">2</div>
<div>
<strong>Set Recording Title</strong><br>
Enter a descriptive title for your video. This becomes the filename and YouTube title.
</div>
</div>

<div class="workflow-step">
<div class="workflow-step-number">3</div>
<div>
<strong>Select Topic</strong><br>
Choose a topic category. Use <kbd>←</kbd>/<kbd>→</kbd> to cycle through options.
</div>
</div>

<div class="workflow-step">
<div class="workflow-step-number">4</div>
<div>
<strong>Configure Recording Options</strong><br>
Toggle the features you need:
<ul>
<li><strong>Record Audio</strong> - Capture microphone</li>
<li><strong>Record Webcam</strong> - Add picture-in-picture</li>
<li><strong>Record Screen</strong> - Capture monitor</li>
<li><strong>Vertical Video</strong> - Create 9:16 version</li>
<li><strong>Add Logos</strong> - Professional branding</li>
</ul>
</div>
</div>

<div class="workflow-step">
<div class="workflow-step-number">5</div>
<div>
<strong>Select Monitor</strong><br>
Choose which monitor to record if you have multiple displays.
</div>
</div>

<div class="workflow-step">
<div class="workflow-step-number">6</div>
<div>
<strong>Press Go Live</strong><br>
Navigate to the <strong>Go Live</strong> button and press <kbd>Enter</kbd>.
</div>
</div>

---

### The Countdown

<div class="workflow-step">
<div class="workflow-step-number">7</div>
<div>
<strong>Prepare During Countdown</strong><br>
A 5-second countdown with audio beeps gives you time to:
<ul>
<li>Position your mouse</li>
<li>Clear your throat</li>
<li>Take a breath</li>
<li>Focus on your content</li>
</ul>
</div>
</div>

!!! tip "Cancelling the Countdown"
    Press ++esc++ at any time to cancel and return to setup with your settings preserved.

---

### During Recording

<div class="workflow-step">
<div class="workflow-step-number">8</div>
<div>
<strong>Record Your Content</strong><br>
The recording screen shows:
<ul>
<li>Blinking <span class="t-red">● REC</span> indicator</li>
<li>Elapsed time counter</li>
<li>Pause and Stop buttons</li>
</ul>
</div>
</div>

**Recording Controls:**

| Key | Action |
|-----|--------|
| ++p++ | Pause/Resume recording |
| ++s++ | Stop recording |

<div class="workflow-step">
<div class="workflow-step-number">9</div>
<div>
<strong>Pause if Needed</strong><br>
Press <kbd>p</kbd> to pause for breaks. The timer freezes and you can resume seamlessly.
</div>
</div>

<div class="workflow-step">
<div class="workflow-step-number">10</div>
<div>
<strong>Stop When Finished</strong><br>
Press <kbd>s</kbd> or select <strong>Stop</strong> to end the recording.
</div>
</div>

---

### Post-Processing

<div class="workflow-step">
<div class="workflow-step-number">11</div>
<div>
<strong>Automatic Processing</strong><br>
The application automatically:
<ol>
<li>Finalizes raw recordings</li>
<li>Normalizes audio levels</li>
<li>Merges video and audio</li>
<li>Adds logo/banner overlays to merged video (if enabled) — visible for the first 15 seconds</li>
<li>Creates vertical version (if enabled) with logos, banner, and title in the lower third</li>
<li>Saves metadata</li>
</ol>
</div>
</div>

### Logo & Banner Placement

When **Add Logos** is enabled, branding overlays are applied to both merged and vertical video outputs.

#### Merged Video (Landscape)

Logo and banner overlays appear for the **first 15 seconds** of the video, then fade out:

- **Left logo** — scaled to 1/8 of the video width, positioned at the top-left corner
- **Right logo** — scaled to 1/8 of the video width, positioned at the top-right corner
- **Banner** — scaled to 1/2 of the video width, positioned at the bottom-left corner

#### Vertical Video (1080×1920)

The vertical video is divided into three zones:

| Zone | Position | Content |
|------|----------|---------|
| Top third | 0–640px | Screen recording (scaled to 1080px width) |
| Middle third | 640–1280px | Webcam feed (scaled to fit) |
| Bottom third | 1280–1920px | Colored background with branding (configurable in Options) |

The bottom third contains:

- **Left logo** — scaled to 360px width, top-left of the branding zone
- **Right logo** — scaled to 360px width, top-right of the branding zone
- **Banner** — scaled to full 1080px width, centered above the title
- **Title text** — centered below the banner

---

### After Processing

<div class="workflow-step">
<div class="workflow-step-number">12</div>
<div>
<strong>Choose Next Action</strong><br>
<ul>
<li><strong>Upload to YouTube</strong> - Continue to <a href="youtube-workflow.md">YouTube workflow</a></li>
<li><strong>Return to Menu</strong> - Save for later</li>
</ul>
</div>
</div>

## Output Files

Your recording is saved to:

```
~/Videos/Screencasts/<topic>/<title>/
├── final.mp4           # Processed video
├── final_vertical.mp4  # Vertical version (if enabled)
├── metadata.json       # Recording information
├── video.mkv           # Raw screen capture
├── audio.wav           # Raw audio
└── webcam.mkv          # Raw webcam (if enabled)
```

## Quick Reference

### Minimum Recording Setup

For a basic screen recording:

1. Set title
2. Ensure "Record Screen" is enabled
3. Select monitor
4. Press "Go Live"

### Full-Featured Recording

For professional videos:

1. Set title and episode number
2. Select appropriate topic
3. Enable audio, screen, and logos
4. Configure logo positions
5. Select monitor
6. Add description
7. Press "Go Live"

## Systray Quick-Record

The system tray icon provides a streamlined recording workflow without the full setup screen.

### First-Run Flow

```mermaid
graph TD
    A[Click systray icon] -->|First time| B{Presets configured?}
    B -->|No| C[Open TUI to Recording Presets]
    C -->|Save| D[TUI auto-closes]
    D --> E[Click systray icon again]
    B -->|Yes| F[5-second countdown with beeps]
    E --> F
    F --> G[Start recording]
```

1. **First recording attempt**: If you haven't configured recording presets, clicking the systray icon opens the TUI directly to the Recording Presets section in Options.
2. **Configure presets**: Toggle Audio, Webcam, Screen, Vertical Video, and Logos as desired, then press Save.
3. **Auto-close**: The TUI closes automatically after saving.
4. **Subsequent recordings**: Click the systray icon to begin a 5-second countdown. During the countdown, audible beeps play and the systray icon displays the current countdown number (5, 4, 3, 2, 1). Recording starts automatically when the countdown reaches zero.
5. **Cancel countdown**: Click the systray icon again during the countdown to cancel it and return to idle.

You can change your presets at any time through the Options screen in the full TUI.

### Stopping a Recording

When you stop a recording from the systray (single click while recording), the TUI opens directly to the recording detail edit page so you can fill in the title, description, presenter and topic. The recording is saved with a "needs metadata" status until you complete this step.

### Processing Screen

After processing completes, the processing screen displays the standard header and footer. The footer shows keyboard shortcuts to preview your recording output:

- **v**: Play the vertical video
- **m**: Play the merged video
- **a**: Play the audio
- **o**: Open the project folder in the file manager

These are the same shortcuts used in the recording detail view in Recording History.

### CLI Presets Mode

You can also open the presets configuration directly from the command line:

```bash
kartoza-screencaster --presets
```

This opens the TUI focused on the Recording Presets section and auto-closes after saving. This is the same mode used by the systray first-run detection.

### CLI Edit Recording Mode

To open the TUI directly to edit the latest recording that needs metadata:

```bash
kartoza-screencaster --edit-recording
```

This is the mode used by the systray after stopping a recording.

---

## Troubleshooting

### No Audio Captured

- Check microphone is connected
- Verify audio system (PipeWire/PulseAudio) is running
- Test with `arecord -l` to list devices

### Black Screen Recording

- Ensure correct monitor is selected
- Check FFmpeg can access display
- On Wayland, ensure screen sharing permissions

### Recording Stutters

- Close unnecessary applications
- Reduce recording resolution
- Disable webcam if not needed

## Related Pages

- **[Recording Setup](../screens/recording-setup.md)** - Detailed field reference
- **[Countdown](../screens/countdown.md)** - Countdown details
- **[Recording](../screens/recording.md)** - Recording screen controls
- **[Processing](../screens/processing.md)** - Processing steps
- **[YouTube Workflow](youtube-workflow.md)** - Upload after recording
