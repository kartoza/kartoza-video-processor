# Main Menu

The Main Menu is the starting point of Kartoza Video Processor. From here, you can access all major features of the application.

## Screen Preview

<div class="terminal-mockup">
<div class="terminal-header">
<div class="terminal-buttons">
<div class="terminal-button red"></div>
<div class="terminal-button yellow"></div>
<div class="terminal-button green"></div>
</div>
<div class="terminal-title">Kartoza Video Processor - Main Menu</div>
</div>
<div class="terminal-content"><span class="t-header">━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━</span>
<span class="t-header">                    Main Menu</span>
<span class="t-header">━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━</span>

<span class="t-selected">  <span class="t-orange">→ New Recording</span></span>
    <span class="t-blue">Recording History</span>        <span class="t-gray">(42 recordings)</span>
    <span class="t-blue">Options</span>
    <span class="t-blue">Quit</span>



<span class="t-gray">━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━</span>
<span class="t-gray">↑/k: up • ↓/j: down • enter/space: select • q: quit</span>
</div>
</div>

## Menu Items

### New Recording

<span class="status-indicator status-ready"></span> **New Recording**

Opens the [Recording Setup](recording-setup.md) screen where you can configure and start a new screen recording.

**What you can do:**

- Set recording title and episode number
- Choose recording options (audio, webcam, logos)
- Select which monitor to record
- Configure vertical video output

---

### Recording History

<span class="status-indicator status-ready"></span> **Recording History** *(with count)*

Opens the [History](history.md) screen showing all your past recordings. The count in parentheses shows how many recordings are available.

**What you can do:**

- Browse all past recordings
- View recording metadata
- Open recording folder
- Delete old recordings
- Upload recordings to YouTube

---

### Options

<span class="status-indicator status-ready"></span> **Options**

Opens the [Options](options.md) screen for application configuration.

**What you can configure:**

- Topic management (add/remove topics)
- Default presenter name
- Logo directory location
- YouTube API credentials

---

### Quit

<span class="status-indicator status-ready"></span> **Quit**

Exits the application. You can also press ++q++ at any time from this menu.

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| ++up++ / ++k++ | Move selection up |
| ++down++ / ++j++ | Move selection down |
| ++enter++ / ++space++ | Select highlighted item |
| ++q++ / ++ctrl+c++ | Quit application |

## Navigation Flow

```mermaid
graph LR
    A[Main Menu] --> B[New Recording]
    A --> C[Recording History]
    A --> D[Options]
    A --> E[Quit]

    B --> F[Recording Setup]
    C --> G[History Screen]
    D --> H[Options Screen]
```

## External Recording Detection

!!! warning "External Recording Warning"
    If the application detects an existing screen recording process (such as from OBS or another instance), a warning will be displayed at the top of the menu. You should stop the external recording before starting a new one.

<div class="terminal-mockup">
<div class="terminal-header">
<div class="terminal-buttons">
<div class="terminal-button red"></div>
<div class="terminal-button yellow"></div>
<div class="terminal-button green"></div>
</div>
<div class="terminal-title">Warning Example</div>
</div>
<div class="terminal-content"><span class="t-yellow">⚠ Warning: External recording detected (PID: 12345)</span>
<span class="t-yellow">  Stop the external recording before starting a new one.</span>

<span class="t-header">                    Main Menu</span>
...
</div>
</div>

## Next Steps

From the Main Menu, you'll typically want to:

1. **[Start a new recording](recording-setup.md)** - Configure and begin recording
2. **[Review past recordings](history.md)** - Browse your recording library
3. **[Configure options](options.md)** - Set up topics and preferences
