# Design Decisions

This document explains the key design decisions made in Kartoza Video Processor and the reasoning behind them.

## Terminal User Interface

### Decision: Use TUI Instead of GUI

**Choice:** Build a terminal-based application rather than a traditional GUI.

**Reasoning:**

1. **Target audience** - Developers and power users comfortable with terminals
2. **Simplicity** - No GUI toolkit dependencies, smaller binary
3. **Remote access** - Works over SSH
4. **Keyboard-first** - Efficient workflow for experienced users
5. **Cross-platform** - Terminal works everywhere

**Trade-offs:**

- Less discoverable for new users
- Limited visual capabilities
- No drag-and-drop

---

### Decision: Elm Architecture (Bubble Tea)

**Choice:** Use the Elm Architecture for UI state management.

**Reasoning:**

1. **Predictability** - Unidirectional data flow
2. **Testability** - Pure functions for Update
3. **Debugging** - Easy to trace state changes
4. **Composition** - Screens combine naturally

**Example:**

```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case KeyPressMsg:
        // Handle input, return new state
        return m.handleKey(msg), nil
    }
    return m, nil
}
```

---

## Recording Architecture

### Decision: Separate FFmpeg Processes

**Choice:** Run screen, audio, and webcam capture as separate FFmpeg processes.

**Reasoning:**

1. **Isolation** - One crash doesn't affect others
2. **Flexibility** - Easy to add/remove streams
3. **Resource management** - OS handles scheduling
4. **Pause support** - SIGSTOP/SIGCONT works cleanly

**Trade-offs:**

- Synchronization complexity
- Multiple file outputs to merge
- Process management overhead

---

### Decision: PID File Communication

**Choice:** Use PID files for tracking recording processes.

**Reasoning:**

1. **Simplicity** - No IPC complexity
2. **Persistence** - Survives TUI restart
3. **Debugging** - Easy to inspect
4. **Recovery** - Can detect orphaned processes

**Files:**

```
/tmp/kvp-video-pid   # Screen capture PID
/tmp/kvp-audio-pid   # Audio capture PID
/tmp/kvp-webcam-pid  # Webcam capture PID
```

---

### Decision: Post-Recording Processing

**Choice:** Process videos after recording completes, not during.

**Reasoning:**

1. **Quality** - No real-time encoding pressure
2. **Flexibility** - Can re-process with different settings
3. **Reliability** - Recording is the critical path
4. **Resources** - Full CPU available for encoding

**Trade-offs:**

- Delay before final video ready
- Raw files take temporary disk space

---

## File Organization

### Decision: Topic-Based Folder Structure

**Choice:** Organize recordings by topic, then by title.

**Structure:**

```
~/Videos/Screencasts/
├── QGIS sketcher sketches/
│   ├── Introduction/
│   └── Advanced/
└── GIS tutorials/
    └── Getting Started/
```

**Reasoning:**

1. **Logical grouping** - Related videos together
2. **Playlist alignment** - Mirrors YouTube playlists
3. **Browsable** - Easy to navigate in file manager
4. **Scalable** - Handles many recordings

---

### Decision: Keep Raw Files

**Choice:** Preserve raw recording files after processing.

**Reasoning:**

1. **Re-processing** - Can recreate with different settings
2. **Quality preservation** - No generation loss
3. **Debugging** - Diagnose issues with originals
4. **Flexibility** - Extract portions without full video

**Trade-offs:**

- Increased disk usage
- User must clean up manually

---

## Configuration

### Decision: JSON Configuration File

**Choice:** Use JSON for configuration storage.

**Reasoning:**

1. **Standard format** - Well understood
2. **Go support** - Native encoding/decoding
3. **Human readable** - Easy to edit manually
4. **Tooling** - Syntax highlighting, validation

**Alternatives considered:**

- YAML - More complex parsing
- TOML - Less common in Go ecosystem
- INI - Limited structure support

---

### Decision: XDG Base Directory

**Choice:** Follow XDG Base Directory specification.

**Paths:**

```
~/.config/kartoza-video-processor/config.json
~/Videos/Screencasts/
```

**Reasoning:**

1. **Standard compliance** - Expected on Linux
2. **Separation** - Config separate from data
3. **Backup friendly** - Clear what to backup
4. **Multi-user** - Works in shared systems

---

## YouTube Integration

### Decision: OAuth Desktop Flow

**Choice:** Use OAuth 2.0 desktop application flow.

**Reasoning:**

1. **Security** - No server needed for tokens
2. **Simplicity** - Standard Google flow
3. **User control** - Clear permission grants
4. **Offline access** - Refresh tokens for long sessions

**Flow:**

1. App starts local HTTP server
2. Browser opens Google auth
3. User grants permissions
4. Redirect to local server
5. App receives authorization code
6. Exchange for tokens

---

### Decision: Resumable Uploads

**Choice:** Use YouTube resumable upload protocol.

**Reasoning:**

1. **Reliability** - Survives network interruptions
2. **Large files** - Handles videos of any size
3. **Progress tracking** - Know exactly where we are
4. **Efficiency** - Only resend failed chunks

---

## Cross-Platform Support

### Decision: Build Tags for Platform Code

**Choice:** Use Go build tags for platform-specific implementations.

**Example:**

```go
// audio_linux.go
//go:build linux

func getDefaultDevice() string {
    // Linux-specific code
}
```

**Reasoning:**

1. **Clean separation** - Platform code isolated
2. **Compile-time** - No runtime checks
3. **Maintainability** - Easy to find platform code
4. **Testing** - Can test platform-specific code

---

### Decision: FFmpeg as External Dependency

**Choice:** Require FFmpeg as system dependency rather than embedding.

**Reasoning:**

1. **Size** - FFmpeg is huge, would bloat binary
2. **Updates** - System FFmpeg gets security updates
3. **Licensing** - Avoids GPL complications
4. **Flexibility** - Users can use custom FFmpeg builds

**Trade-offs:**

- Installation requirement
- Version compatibility concerns

---

## Error Handling

### Decision: Graceful Degradation

**Choice:** Continue operation when non-critical features fail.

**Examples:**

- No audio device → Record without audio
- Desktop notification fails → Continue silently
- Logo file missing → Record without logo

**Reasoning:**

1. **User experience** - Don't block on minor issues
2. **Robustness** - Handle edge cases
3. **Feedback** - Warn but don't prevent

---

### Decision: User-Friendly Error Messages

**Choice:** Transform technical errors into actionable messages.

**Example:**

```go
// Instead of: "ENOENT: audio device /dev/snd/pcm0 not found"
// Show: "No audio device found. Check microphone connection."
```

**Reasoning:**

1. **Accessibility** - Users aren't developers
2. **Actionable** - Tell user what to do
3. **Trust** - Application feels polished

---

## Future Considerations

### Under Consideration

1. **Plugin system** - For custom processing steps
2. **Streaming support** - Direct to YouTube/Twitch
3. **Multi-track audio** - Separate mic and system audio
4. **Template system** - Reusable recording configurations

### Explicitly Not Planned

1. **GUI version** - Out of scope, TUI is the focus
2. **Video editing** - Use dedicated tools
3. **Transcription** - Better done by specialized services
4. **Mobile apps** - Desktop-focused application
