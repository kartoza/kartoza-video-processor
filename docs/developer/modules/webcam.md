# Webcam Package

The `webcam` package handles webcam video capture for picture-in-picture overlays.

## Package Location

```
internal/webcam/
```

## Responsibility

- Capture video from webcam
- Detect available cameras
- Platform-specific webcam handling

## Key Files

| File | Purpose |
|------|---------|
| `webcam.go` | Core webcam logic |
| `webcam_linux.go` | Linux-specific (V4L2) |
| `webcam_darwin.go` | macOS-specific (AVFoundation) |
| `webcam_windows.go` | Windows-specific (DirectShow) |

## Key Types

### WebcamRecorder

Manages webcam capture:

```go
type WebcamRecorder struct {
    cmd        *exec.Cmd
    outputFile string
    device     string
    width      int
    height     int
    framerate  int
}
```

### WebcamInfo

Information about a webcam device:

```go
type WebcamInfo struct {
    Device     string
    Name       string
    Resolutions []Resolution
}

type Resolution struct {
    Width  int
    Height int
}
```

## Core Functions

### ListDevices

Enumerate available webcams:

```go
func ListDevices() ([]WebcamInfo, error)
```

### Start

Begin webcam capture:

```go
func (w *WebcamRecorder) Start(outputFile string) error
```

### Stop

End webcam capture:

```go
func (w *WebcamRecorder) Stop() error
```

## Platform Implementation

### Linux (V4L2)

```go
//go:build linux

func ListDevices() ([]WebcamInfo, error) {
    // List /dev/video* devices
    matches, _ := filepath.Glob("/dev/video*")
    // Query each device with v4l2-ctl
}

func (w *WebcamRecorder) buildCommand() *exec.Cmd {
    return exec.Command("ffmpeg",
        "-f", "v4l2",
        "-framerate", fmt.Sprintf("%d", w.framerate),
        "-video_size", fmt.Sprintf("%dx%d", w.width, w.height),
        "-i", w.device,
        "-c:v", "libx264",
        "-preset", "ultrafast",
        w.outputFile,
    )
}
```

### macOS (AVFoundation)

```go
//go:build darwin

func ListDevices() ([]WebcamInfo, error) {
    // Use ffmpeg -f avfoundation -list_devices true
    cmd := exec.Command("ffmpeg",
        "-f", "avfoundation",
        "-list_devices", "true",
        "-i", "",
    )
}

func (w *WebcamRecorder) buildCommand() *exec.Cmd {
    return exec.Command("ffmpeg",
        "-f", "avfoundation",
        "-framerate", fmt.Sprintf("%d", w.framerate),
        "-video_size", fmt.Sprintf("%dx%d", w.width, w.height),
        "-i", w.device,
        "-c:v", "libx264",
        "-preset", "ultrafast",
        w.outputFile,
    )
}
```

### Windows (DirectShow)

```go
//go:build windows

func ListDevices() ([]WebcamInfo, error) {
    // Use ffmpeg -f dshow -list_devices true
    cmd := exec.Command("ffmpeg",
        "-f", "dshow",
        "-list_devices", "true",
        "-i", "dummy",
    )
}
```

## FFmpeg Commands

### Linux

```bash
ffmpeg -f v4l2 \
    -framerate 30 \
    -video_size 640x480 \
    -i /dev/video0 \
    -c:v libx264 -preset ultrafast \
    webcam.mkv
```

### macOS

```bash
ffmpeg -f avfoundation \
    -framerate 30 \
    -video_size 640x480 \
    -i "0" \
    -c:v libx264 -preset ultrafast \
    webcam.mkv
```

### Windows

```bash
ffmpeg -f dshow \
    -framerate 30 \
    -video_size 640x480 \
    -i video="Integrated Camera" \
    -c:v libx264 -preset ultrafast \
    webcam.mkv
```

## Configuration

### Resolution Presets

| Preset | Resolution | Use Case |
|--------|------------|----------|
| Low | 320x240 | Small overlay |
| Standard | 640x480 | Default |
| HD | 1280x720 | Large overlay |

### Framerate

| Value | Effect |
|-------|--------|
| 15 fps | Lower CPU, choppy |
| 30 fps | Smooth (default) |
| 60 fps | Very smooth, high CPU |

## Error Handling

### Device Busy

```go
func (w *WebcamRecorder) Start(output string) error {
    if err := w.cmd.Start(); err != nil {
        if strings.Contains(err.Error(), "Device or resource busy") {
            return fmt.Errorf("webcam in use by another application")
        }
        return err
    }
    return nil
}
```

### No Webcam Found

```go
func ListDevices() ([]WebcamInfo, error) {
    devices := findDevices()
    if len(devices) == 0 {
        return nil, fmt.Errorf("no webcam devices found")
    }
    return devices, nil
}
```

## Usage Example

```go
// List available webcams
devices, err := webcam.ListDevices()
if err != nil {
    log.Printf("No webcam: %v", err)
    return
}

// Start recording
recorder := webcam.NewRecorder(devices[0].Device, 640, 480, 30)
err = recorder.Start("/tmp/webcam.mkv")

// ... recording ...

recorder.Stop()
```

## Related Packages

- **recorder** - Uses webcam for capture
- **merger** - Overlays webcam on screen recording
