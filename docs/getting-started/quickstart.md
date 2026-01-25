# Quick Start

Get up and running with your first screen recording in just a few minutes.

## Starting the Application

Launch Kartoza Screencaster from your terminal:

```bash
kvp
```

You'll see a splash screen followed by the main menu.

## Your First Recording

### Step 1: Select "New Recording"

From the main menu, use the arrow keys to navigate and press ++enter++ to select **New Recording**.

<div class="terminal-mockup">
<div class="terminal-header">
<div class="terminal-buttons">
<div class="terminal-button red"></div>
<div class="terminal-button yellow"></div>
<div class="terminal-button green"></div>
</div>
<div class="terminal-title">Main Menu</div>
</div>
<div class="terminal-content"><span class="t-selected">  <span class="t-orange">→ New Recording</span></span>
    <span class="t-blue">Recording History</span>
    <span class="t-blue">Options</span>
    <span class="t-blue">Quit</span>
</div>
</div>

### Step 2: Configure Recording

Fill in the recording details:

1. **Title** - Enter a descriptive title for your video
2. **Episode Number** - Auto-incremented, or set manually
3. **Topic** - Select from predefined topics
4. **Recording Options** - Toggle audio, webcam, screen capture
5. **Monitor** - Select which monitor to record

<div class="terminal-mockup">
<div class="terminal-header">
<div class="terminal-buttons">
<div class="terminal-button red"></div>
<div class="terminal-button yellow"></div>
<div class="terminal-button green"></div>
</div>
<div class="terminal-title">Recording Setup</div>
</div>
<div class="terminal-content"><span class="t-header">━━━━━━━━━━━━━ Recording Setup ━━━━━━━━━━━━━</span>

<span class="t-orange">Title:</span>     <span class="t-white">My First Recording</span>
<span class="t-blue">Episode:</span>   <span class="t-white">1</span>
<span class="t-blue">Topic:</span>     <span class="t-white">QGIS sketches</span>

<span class="t-header">Recording Options</span>
  <span class="t-green">[✓]</span> Record Audio
  <span class="t-gray">[○]</span> Record Webcam
  <span class="t-green">[✓]</span> Record Screen
  <span class="t-gray">[○]</span> Vertical Video
  <span class="t-gray">[○]</span> Add Logo Overlays

<span class="t-header">Monitor Selection</span>
  <span class="t-orange">→</span> <span class="t-white">DP-1 (2560x1440)</span>
    <span class="t-gray">HDMI-1 (1920x1080)</span>

  <span class="t-green">[ Go Live ]</span>    <span class="t-gray">[ Cancel ]</span>
</div>
</div>

### Step 3: Start Recording

Navigate to **Go Live** and press ++enter++. A countdown will appear:

<div class="terminal-mockup">
<div class="terminal-header">
<div class="terminal-buttons">
<div class="terminal-button red"></div>
<div class="terminal-button yellow"></div>
<div class="terminal-button green"></div>
</div>
<div class="terminal-title">Countdown</div>
</div>
<div class="terminal-content" style="text-align: center;">
<span class="t-orange" style="font-size: 1.5em;">
 ███████
       █
       █
 ███████
       █
       █
 ███████
</span>

<span class="t-gray t-dim">Get ready... Recording starts soon!</span>

<span class="t-dim">Press ESC to cancel</span>
</div>
</div>

### Step 4: Recording in Progress

Once recording starts, you'll see the recording screen:

<div class="terminal-mockup">
<div class="terminal-header">
<div class="terminal-buttons">
<div class="terminal-button red"></div>
<div class="terminal-button yellow"></div>
<div class="terminal-button green"></div>
</div>
<div class="terminal-title">Recording</div>
</div>
<div class="terminal-content" style="text-align: center;">
<span class="t-red">● REC</span>  <span class="t-white">00:05:23</span>

  <span class="t-orange">[ Pause ]</span>    <span class="t-red">[ Stop ]</span>

<span class="t-gray">p: pause • s: stop • space: toggle button</span>
</div>
</div>

### Step 5: Stop Recording

Press ++s++ or select **Stop** to end the recording.

The application will automatically process your video:

1. Merge video and audio streams
2. Apply logo overlays (if enabled)
3. Generate metadata
4. Create vertical version (if enabled)

### Step 6: View Your Recording

Your recording is saved to `~/Videos/Screencasts/<topic>/<title>/`

The folder contains:

- `final.mp4` - Your processed video
- `metadata.json` - Recording information
- Raw recording files

## Keyboard Shortcuts Summary

| Key | Action |
|-----|--------|
| ++up++ / ++k++ | Navigate up |
| ++down++ / ++j++ | Navigate down |
| ++enter++ / ++space++ | Select / Confirm |
| ++tab++ | Next field |
| ++shift+tab++ | Previous field |
| ++p++ | Pause recording |
| ++s++ | Stop recording |
| ++q++ / ++esc++ | Quit / Back |

## Next Steps

- **[Recording Setup](../screens/recording-setup.md)** - Learn about all recording options
- **[Options](../screens/options.md)** - Configure application settings
- **[YouTube Upload](../screens/youtube-upload.md)** - Upload to YouTube
