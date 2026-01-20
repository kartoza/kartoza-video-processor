# Audio Package

The `audio` package handles audio capture and post-processing, including normalization.

## Package Location

```
internal/audio/
```

## Responsibility

- Capture audio from microphone
- Normalize audio levels
- Detect audio devices
- Platform-specific audio handling

## Key Files

| File | Purpose |
|------|---------|
| `audio.go` | Core audio logic |
| `audio_linux.go` | Linux-specific (PipeWire/PulseAudio/ALSA) |
| `audio_darwin.go` | macOS-specific (CoreAudio) |
| `audio_windows.go` | Windows-specific (WASAPI) |

## Key Types

### AudioRecorder

Manages audio capture:

```go
type AudioRecorder struct {
    cmd       *exec.Cmd
    outputFile string
    device     string
}
```

### AudioConfig

Configuration for audio capture:

```go
type AudioConfig struct {
    Device     string
    SampleRate int
    Channels   int
    Format     string
}
```

## Core Functions

### Start

Begin audio capture:

```go
func (r *AudioRecorder) Start(outputFile string) error
```

### Stop

End audio capture:

```go
func (r *AudioRecorder) Stop() error
```

### NormalizeAudio

Post-process audio for consistent levels:

```go
func NormalizeAudio(inputFile, outputFile string) error
```

## Platform Implementation

### Linux (PipeWire/PulseAudio)

```go
//go:build linux

func getDefaultDevice() string {
    // Try PipeWire first
    if _, err := exec.LookPath("pw-record"); err == nil {
        return "pipewire"
    }
    // Fall back to PulseAudio
    return "pulse"
}
```

### FFmpeg Command (Linux)

```bash
# PulseAudio
ffmpeg -f pulse -i default -c:a pcm_s16le output.wav

# ALSA
ffmpeg -f alsa -i hw:0 -c:a pcm_s16le output.wav
```

### macOS (CoreAudio)

```go
//go:build darwin

func getDefaultDevice() string {
    return "avfoundation"
}
```

### FFmpeg Command (macOS)

```bash
ffmpeg -f avfoundation -i ":0" -c:a pcm_s16le output.wav
```

### Windows (WASAPI)

```go
//go:build windows

func getDefaultDevice() string {
    return "dshow"
}
```

## Audio Normalization

### Process

1. Analyze input audio for peak level
2. Calculate gain adjustment
3. Apply loudness normalization
4. Output normalized audio

### FFmpeg Normalization

```bash
# Two-pass loudness normalization
ffmpeg -i input.wav -af loudnorm=I=-16:TP=-1.5:LRA=11:print_format=summary -f null -
ffmpeg -i input.wav -af loudnorm=I=-16:TP=-1.5:LRA=11 output.wav
```

### Simple Normalization

```bash
# Single-pass (faster, less accurate)
ffmpeg -i input.wav -af "volume=1.5" output.wav
```

## Error Handling

### Device Detection

```go
func detectAudioDevices() ([]string, error) {
    // List available devices
    cmd := exec.Command("ffmpeg", "-devices", "-hide_banner")
    // Parse output...
}
```

### Fallback Strategy

```go
func (r *AudioRecorder) Start(output string) error {
    devices := []string{"pipewire", "pulse", "alsa"}

    for _, device := range devices {
        if err := r.tryDevice(device, output); err == nil {
            return nil
        }
    }

    return fmt.Errorf("no audio device available")
}
```

## Usage Example

```go
recorder := audio.NewRecorder()

// Start recording
err := recorder.Start("/tmp/audio.wav")
if err != nil {
    log.Printf("Audio unavailable: %v", err)
}

// ... recording ...

// Stop and normalize
recorder.Stop()
audio.NormalizeAudio("/tmp/audio.wav", "/tmp/audio_normalized.wav")
```

## Configuration

### Sample Rates

| Quality | Rate | Use Case |
|---------|------|----------|
| Low | 22050 Hz | Voice only |
| Standard | 44100 Hz | General use |
| High | 48000 Hz | Professional |

### Channels

- **Mono** - Single channel, smaller files
- **Stereo** - Two channels, spatial audio

## Related Packages

- **recorder** - Uses audio for capture
- **merger** - Merges audio with video
