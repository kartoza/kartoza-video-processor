# Recorder Package

The `recorder` package orchestrates the recording process, managing FFmpeg processes for screen, audio, and webcam capture.

## Package Location

```
internal/recorder/
```

## Responsibility

- Start/stop/pause recording processes
- Coordinate multiple capture streams
- Monitor recording status
- Trigger post-processing

## Key Files

| File | Purpose |
|------|---------|
| `recorder.go` | Main recording logic |
| `recorder_test.go` | Unit tests |

## Key Types

### Recorder

Main struct managing recording state:

```go
type Recorder struct {
    status        RecordingStatus
    config        *config.Config
    recordingInfo *models.RecordingInfo
    progressChan  chan ProgressUpdate
}
```

### RecordingConfig

Configuration for a recording session:

```go
type RecordingConfig struct {
    Title         string
    Topic         string
    Description   string
    Monitor       models.Monitor
    RecordAudio   bool
    RecordWebcam  bool
    RecordScreen  bool
    VerticalVideo bool
    AddLogos      bool
    LeftLogo      string
    RightLogo     string
    BottomLogo    string
}
```

### ProgressUpdate

Progress reporting for UI updates:

```go
type ProgressUpdate struct {
    Step     int
    Progress float64
    Message  string
    Error    error
}
```

## Core Functions

### Start

Begins a new recording session:

```go
func (r *Recorder) Start(cfg RecordingConfig) error
```

**Process:**

1. Validate configuration
2. Create output directory
3. Start screen capture (FFmpeg)
4. Start audio capture (FFmpeg)
5. Start webcam capture (FFmpeg, optional)
6. Write PID files
7. Initialize recording info

### Stop

Ends the recording and triggers processing:

```go
func (r *Recorder) Stop() error
```

**Process:**

1. Send SIGTERM to all processes
2. Wait for graceful shutdown
3. Verify output files
4. Trigger merger for post-processing
5. Clean up PID files

### Pause / Resume

Pauses and resumes recording:

```go
func (r *Recorder) Pause() error
func (r *Recorder) Resume() error
```

**Implementation:**

- Uses SIGSTOP/SIGCONT signals
- Freezes all capture processes simultaneously
- Timer pauses in UI

### GetStatus

Returns current recording state:

```go
func (r *Recorder) GetStatus() RecordingStatus
```

## FFmpeg Commands

### Screen Capture

```bash
ffmpeg -f x11grab -framerate 30 \
    -video_size 2560x1440 \
    -i :0.0+0,0 \
    -c:v libx264 -preset ultrafast \
    -crf 18 \
    output/video.mkv
```

### Audio Capture

```bash
ffmpeg -f pulse -i default \
    -c:a pcm_s16le \
    output/audio.wav
```

### Webcam Capture

```bash
ffmpeg -f v4l2 -framerate 30 \
    -video_size 640x480 \
    -i /dev/video0 \
    -c:v libx264 -preset ultrafast \
    output/webcam.mkv
```

## Process Management

### PID Files

Location: `/tmp/`

| File | Purpose |
|------|---------|
| `kvp-video-pid` | Screen capture process |
| `kvp-audio-pid` | Audio capture process |
| `kvp-webcam-pid` | Webcam capture process |

### Signal Handling

```go
// Pause recording
syscall.Kill(pid, syscall.SIGSTOP)

// Resume recording
syscall.Kill(pid, syscall.SIGCONT)

// Stop recording
syscall.Kill(pid, syscall.SIGTERM)
```

## Error Handling

### Common Errors

| Error | Cause | Recovery |
|-------|-------|----------|
| `no recording in progress` | Stop without Start | Inform user |
| `monitor not found` | Invalid monitor | Re-select monitor |
| `audio device error` | No microphone | Disable audio |

### Recovery Strategies

```go
func (r *Recorder) Start(cfg RecordingConfig) error {
    // Try to start audio, but don't fail if unavailable
    if cfg.RecordAudio {
        if err := r.startAudio(); err != nil {
            notify.Warning("Audio unavailable", err.Error())
            cfg.RecordAudio = false
        }
    }
    // Continue with recording...
}
```

## Testing

### Unit Tests

```go
func TestRecorder_GetStatus_NoRecording(t *testing.T) {
    rec := New()
    status := rec.GetStatus()
    assert.False(t, status.IsRecording)
}

func TestReadPID_ValidContent(t *testing.T) {
    tmpFile := createTempPIDFile("12345")
    defer os.Remove(tmpFile)

    pid := readPID(tmpFile)
    assert.Equal(t, 12345, pid)
}
```

## Usage Example

```go
rec := recorder.New()

cfg := recorder.RecordingConfig{
    Title:       "My Recording",
    Topic:       "Tutorials",
    RecordAudio: true,
    RecordScreen: true,
    Monitor:     selectedMonitor,
}

if err := rec.Start(cfg); err != nil {
    log.Fatal(err)
}

// ... recording in progress ...

if err := rec.Stop(); err != nil {
    log.Fatal(err)
}
```

## Related Packages

- **audio** - Audio capture implementation
- **webcam** - Webcam capture implementation
- **merger** - Post-processing after recording
- **monitor** - Display detection
