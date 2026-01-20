# Recording

The Recording screen is displayed while your recording is in progress. It provides real-time status information and controls for pausing or stopping the recording.

## Screen Preview

<div class="terminal-mockup">
<div class="terminal-header">
<div class="terminal-buttons">
<div class="terminal-button red"></div>
<div class="terminal-button yellow"></div>
<div class="terminal-button green"></div>
</div>
<div class="terminal-title">Recording in Progress</div>
</div>
<div class="terminal-content" style="text-align: center; padding: 40px 20px;">
<span style="font-size: 1.2em; line-height: 1.1;">
<span class="t-white">
  ██████████████████████████      ████
  █                    █    ██    █
  █   ████████████     █  ██      █
  █   █          █     █ █        █
  █   █    ●     █     ██         █
  █   █          █     █ █        █
  █   ████████████     █  ██      █
  █                    █    ██    █
  ██████████████████████████      ████
</span>
</span>

<div style="margin-top: 20px;">
<span class="t-red" style="animation: pulse 1s infinite;">●</span> <span class="t-red">REC</span>
<span class="t-white" style="margin-left: 20px; font-size: 1.5em;">00:05:23</span>
</div>

<div style="margin-top: 30px;">
  <span class="t-orange" style="border: 2px solid #ff9500; padding: 8px 20px;">[ Pause ]</span>
  <span style="margin: 0 20px;"></span>
  <span class="t-gray" style="border: 2px solid #666; padding: 8px 20px;">[ Stop ]</span>
</div>

<div style="margin-top: 20px;">
<span class="t-gray">p: pause • s: stop • ←/→: select button • space: activate</span>
</div>
</div>
</div>

## Screen Elements

### Camera Icon

A large video camera icon is displayed in the center, providing visual confirmation that recording is active.

### Status Indicator

<span class="status-indicator status-recording"></span> **REC**

The blinking "REC" indicator confirms the recording is active. It blinks every 500ms to clearly show the recording state.

### Elapsed Time

**00:05:23**

Shows the current recording duration in `HH:MM:SS` format. Updates every second.

### Control Buttons

Two action buttons are available:

| Button | Description |
|--------|-------------|
| **[ Pause ]** | Temporarily pause the recording |
| **[ Stop ]** | End the recording |

Use ++left++ / ++right++ to select between buttons, then ++space++ or ++enter++ to activate.

## Paused State

When paused, the display changes:

<div class="terminal-mockup">
<div class="terminal-header">
<div class="terminal-buttons">
<div class="terminal-button red"></div>
<div class="terminal-button yellow"></div>
<div class="terminal-button green"></div>
</div>
<div class="terminal-title">Recording Paused</div>
</div>
<div class="terminal-content" style="text-align: center; padding: 40px 20px;">
<span style="font-size: 1.2em; line-height: 1.1;">
<span class="t-yellow">
    ████████      ████████
    ████████      ████████
    ████████      ████████
    ████████      ████████
    ████████      ████████
    ████████      ████████
    ████████      ████████
    ████████      ████████
    ████████      ████████
</span>
</span>

<div style="margin-top: 20px;">
<span class="t-yellow">⏸</span> <span class="t-yellow">PAUSED</span>
<span class="t-white" style="margin-left: 20px; font-size: 1.5em;">00:05:23</span>
</div>

<div style="margin-top: 30px;">
  <span class="t-green" style="border: 2px solid #00cc66; padding: 8px 20px;">[ Resume ]</span>
  <span style="margin: 0 20px;"></span>
  <span class="t-gray" style="border: 2px solid #666; padding: 8px 20px;">[ Stop ]</span>
</div>

<div style="margin-top: 20px;">
<span class="t-gray">p: resume • s: stop • ←/→: select button • space: activate</span>
</div>
</div>
</div>

**Paused State Changes:**

- Camera icon replaced with pause icon (two vertical bars)
- "REC" changes to "PAUSED" (yellow)
- "Pause" button changes to "Resume"
- Timer freezes at pause point
- FFmpeg processes receive SIGSTOP signal

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| ++p++ | Toggle pause/resume |
| ++s++ | Stop recording |
| ++left++ / ++right++ | Select button |
| ++space++ / ++enter++ | Activate selected button |

## Recording Processes

While recording, the following processes run simultaneously:

| Process | Description |
|---------|-------------|
| **Screen capture** | FFmpeg capturing the selected monitor |
| **Audio capture** | FFmpeg capturing microphone input |
| **Webcam capture** | FFmpeg capturing webcam (if enabled) |

Process IDs are stored in temporary files:

- `/tmp/kvp-video-pid`
- `/tmp/kvp-audio-pid`
- `/tmp/kvp-webcam-pid`

## Pause Functionality

### How Pause Works

1. **SIGSTOP** signal sent to all recording processes
2. Timer display freezes
3. UI updates to paused state
4. Processes remain in memory, ready to resume

### Resume Behavior

1. **SIGCONT** signal sent to all processes
2. Timer continues from pause point
3. UI returns to recording state
4. No gaps in the final video

!!! note "Pause Limitations"
    The pause feature uses Unix signals and works best on Linux. Behavior may vary on other platforms.

## Stopping the Recording

When you stop the recording:

1. **SIGTERM** sent to all recording processes
2. Processes cleanly terminate and finalize files
3. Screen transitions to [Processing](processing.md) screen
4. Post-processing begins automatically

## File Outputs

During recording, files are written to:

```
~/Videos/Screencasts/<topic>/<title>/
├── video.mkv       # Raw screen capture
├── audio.wav       # Raw audio capture
├── webcam.mkv      # Raw webcam (if enabled)
└── metadata.json   # Recording information
```

## Workflow Position

<div class="workflow-step">
<div class="workflow-step-number">1</div>
<div>
<strong>Previous:</strong> <a href="countdown.md">Countdown</a> → Countdown completes
</div>
</div>

<div class="workflow-step">
<div class="workflow-step-number">2</div>
<div>
<strong>Current:</strong> Recording (this screen)
</div>
</div>

<div class="workflow-step">
<div class="workflow-step-number">3</div>
<div>
<strong>Next:</strong> <a href="processing.md">Processing</a> → After pressing Stop
</div>
</div>

## Technical Details

- **Display refresh**: Every 100ms for smooth timer updates
- **Blink interval**: 500ms for REC indicator
- **Signal handling**: SIGSTOP/SIGCONT for pause, SIGTERM for stop
- **Process monitoring**: Regular checks for process health

## Related Pages

- **[Countdown](countdown.md)** - Before recording starts
- **[Processing](processing.md)** - After recording ends
- **[Recording Setup](recording-setup.md)** - Configuration options
