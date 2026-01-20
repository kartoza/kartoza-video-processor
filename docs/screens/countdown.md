# Countdown

The Countdown screen provides a visual and audio countdown before recording begins, giving you time to prepare.

## Screen Preview

<div class="terminal-mockup">
<div class="terminal-header">
<div class="terminal-buttons">
<div class="terminal-button red"></div>
<div class="terminal-button yellow"></div>
<div class="terminal-button green"></div>
</div>
<div class="terminal-title">Countdown</div>
</div>
<div class="terminal-content" style="text-align: center; padding: 40px;">

<span class="t-orange" style="font-size: 1.2em; line-height: 1.2;">
 ███████
       █
       █
 ███████
       █
       █
 ███████
</span>

<span class="t-gray t-dim" style="display: block; margin-top: 20px;">Get ready... Recording starts soon!</span>

<span class="t-dim" style="display: block; margin-top: 10px;">Press ESC to cancel</span>
</div>
</div>

## Countdown Sequence

The countdown runs from **5** to **1**, then displays **GO!** before recording begins.

### Visual Display

Each number is displayed as a large ASCII art digit:

| Count | Color | Audio |
|-------|-------|-------|
| 5 | Orange | 880 Hz beep |
| 4 | Orange | 784 Hz beep |
| 3 | Dark Orange | 698 Hz beep |
| 2 | Dark Orange | 622 Hz beep |
| 1 | Red | 554 Hz beep |
| GO! | Green | None |

### Number Display Examples

<div class="terminal-mockup">
<div class="terminal-header">
<div class="terminal-buttons">
<div class="terminal-button red"></div>
<div class="terminal-button yellow"></div>
<div class="terminal-button green"></div>
</div>
<div class="terminal-title">Countdown Numbers</div>
</div>
<div class="terminal-content" style="display: flex; justify-content: space-around;">
<span class="t-orange" style="line-height: 1.1;">
 ███████
 █
 █
 ███████
       █
       █
 ███████
</span>
<span class="t-red" style="line-height: 1.1;">
    █
   ██
    █
    █
    █
    █
   ███
</span>
<span class="t-green" style="line-height: 1.1;">
  ██████   ██████  ██
 ██       ██    ██ ██
 ██   ███ ██    ██ ██
 ██    ██ ██    ██ ██
 ██    ██ ██    ██
  ██████   ██████  ██
</span>
</div>
</div>

## Audio Beeps

The countdown includes descending-frequency audio beeps to help you track progress even when not looking at the screen.

**Frequency Mapping:**

```
5 → 880 Hz (A5)
4 → 784 Hz (G5)
3 → 698 Hz (F5)
2 → 622 Hz (D#5)
1 → 554 Hz (C#5)
```

### Audio System Priority

The application tries multiple audio backends in order:

1. **FFmpeg + PipeWire** (pw-cat)
2. **FFmpeg + ALSA** (aplay)
3. **speaker-test** (ALSA direct)
4. **PulseAudio** (paplay with system sounds)
5. **Console bell** (fallback)

!!! note "Silent Systems"
    If your system doesn't have audio configured, the countdown will still work visually.

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| ++esc++ | Cancel countdown and return to setup |
| ++q++ | Cancel countdown and return to setup |

## Cancellation

If you press ++esc++ or ++q++ during the countdown:

1. Countdown immediately stops
2. No recording processes are started
3. You return to the [Recording Setup](recording-setup.md) screen
4. All your settings are preserved

## After Countdown

When the countdown completes:

1. **GO!** is displayed briefly in green
2. Recording processes start (screen, audio, webcam)
3. Screen transitions to [Recording](recording.md) screen
4. Status indicator shows "REC" with elapsed time

## Timing

| Event | Duration |
|-------|----------|
| Each number (5-1) | 1 second |
| "GO!" display | 0.5 seconds |
| Total countdown | ~5.5 seconds |

## Workflow Position

<div class="workflow-step">
<div class="workflow-step-number">1</div>
<div>
<strong>Previous:</strong> <a href="recording-setup.md">Recording Setup</a> → Press "Go Live"
</div>
</div>

<div class="workflow-step">
<div class="workflow-step-number">2</div>
<div>
<strong>Current:</strong> Countdown (this screen)
</div>
</div>

<div class="workflow-step">
<div class="workflow-step-number">3</div>
<div>
<strong>Next:</strong> <a href="recording.md">Recording</a> → After countdown completes
</div>
</div>

## Technical Details

The countdown is implemented using:

- **7-segment style ASCII art** for digit display
- **tea.Tick** for timing intervals
- **goroutines** for non-blocking audio playback
- **Alternative screen buffer** for clean display

## Related Pages

- **[Recording Setup](recording-setup.md)** - Configure before countdown
- **[Recording](recording.md)** - Active recording screen
