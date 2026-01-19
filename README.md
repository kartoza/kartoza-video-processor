# Kartoza Video Processor

A cross-platform screen recording tool with multi-monitor support, audio processing, and webcam integration. Supports Linux (Wayland/X11), Windows, and macOS.

## Features

- **Multi-monitor screen recording** with automatic cursor detection
- **Separate audio recording** with background noise reduction
- **Webcam recording** with real-time 60fps capture
- **Audio normalization** using EBU R128 loudness standards
- **Vertical video creation** with webcam overlay (perfect for social media)
- **Hardware acceleration** support (optional VAAPI encoding on Linux)
- **Desktop notifications** for recording status
- **Cross-platform support** - Linux, Windows, and macOS

## Requirements

### All Platforms
- `ffmpeg` - Video/audio processing and merging
- `ffprobe` - Video metadata extraction

### Linux-specific
- **Wayland**: `wl-screenrec` - Wayland screen recorder
- **X11**: Uses ffmpeg with x11grab (no additional dependencies)
- `pw-record` (PipeWire) - Audio capture
- `notify-send` - Desktop notifications (optional)

### Windows-specific
- Uses ffmpeg with `gdigrab` for screen capture
- Uses ffmpeg with `dshow` for audio and webcam
- No additional dependencies beyond ffmpeg

### macOS-specific
- Uses ffmpeg with `avfoundation` for screen, audio, and webcam
- No additional dependencies beyond ffmpeg

## Installation

### From Source

#### Linux
```bash
# Clone the repository
git clone https://github.com/kartoza/kartoza-video-processor.git
cd kartoza-video-processor

# Build
make build

# Install
sudo make install
```

#### Windows
```bash
# Clone the repository
git clone https://github.com/kartoza/kartoza-video-processor.git
cd kartoza-video-processor

# Build (requires Go installed)
go build -o kartoza-video-processor.exe

# Install ffmpeg from https://ffmpeg.org/download.html
# Add kartoza-video-processor.exe to your PATH
```

#### macOS
```bash
# Clone the repository
git clone https://github.com/kartoza/kartoza-video-processor.git
cd kartoza-video-processor

# Build (requires Go installed)
go build -o kartoza-video-processor

# Install ffmpeg via Homebrew
brew install ffmpeg

# Move binary to PATH
sudo mv kartoza-video-processor /usr/local/bin/
```

### Using Nix Flakes (Linux only)

```bash
# Run directly
nix run github:kartoza/kartoza-video-processor

# Install to profile
nix profile install github:kartoza/kartoza-video-processor
```

### Development

```bash
# Enter development shell (Linux)
nix develop

# Or use direnv (Linux)
direnv allow

# Or use standard Go tools (all platforms)
go run .
```

## Usage

### TUI Mode

Launch the interactive terminal interface:

```bash
kartoza-video-processor
```

Press `Space` or `Enter` to toggle recording.

### CLI Mode

```bash
# Toggle recording
kartoza-video-processor toggle

# Start recording
kartoza-video-processor start

# Start with options
kartoza-video-processor start --monitor DP-1 --no-webcam --hw-accel

# Stop recording
kartoza-video-processor stop

# Check status
kartoza-video-processor status

# List monitors
kartoza-video-processor monitors
```

### CLI Options

```
Flags:
  -m, --monitor string        Monitor name to record (default: monitor with cursor)
      --no-audio              Disable audio recording
      --no-webcam             Disable webcam recording
      --hw-accel              Enable hardware acceleration (VAAPI)
  -o, --output string         Output directory (default: ~/Videos/Screencasts)
      --webcam-device string  Webcam device (default: auto-detect)
      --webcam-fps int        Webcam framerate (default: 60)
      --audio-device string   PipeWire audio device (default: @DEFAULT_SOURCE@)
```

## Output Files

Recordings are saved to `~/Videos/Screencasts/` with the following files:

- `screenrecording-{monitor}-{timestamp}.mp4` - Raw screen capture
- `screenrecording-{monitor}-{timestamp}.wav` - Raw audio
- `screenrecording-{monitor}-{timestamp}-merged.mp4` - Final video with audio
- `screenrecording-{monitor}-{timestamp}-vertical.mp4` - Vertical video with webcam (if available)

## Audio Processing

The tool automatically processes audio with:

1. **Noise reduction** - Removes background noise using FFT-based denoising
2. **Highpass filter** - Removes low-frequency rumble (< 200 Hz)
3. **Two-pass loudness normalization** - EBU R128 compliant
   - Target loudness: -14 LUFS (louder than broadcast, perfect for screen recordings)
   - True peak: -1.5 dB (prevents clipping)
   - Loudness range: 11 LU (preserves dynamic range)

## Configuration

Configuration is stored in `~/.config/kartoza-video-processor/config.json`:

```json
{
  "output_dir": "/home/user/Videos/Screencasts",
  "default_options": {
    "no_audio": false,
    "no_webcam": false,
    "hw_accel": false,
    "webcam_fps": 60,
    "audio_device": "@DEFAULT_SOURCE@"
  },
  "audio_processing": {
    "denoise_enabled": true,
    "highpass_freq": 200,
    "normalize_enabled": true,
    "target_loudness": -14.0
  }
}
```

**Platform-specific paths:**
- Linux: `~/.config/kartoza-video-processor/config.json`
- Windows: `%APPDATA%\kartoza-video-processor\config.json`
- macOS: `~/Library/Application Support/kartoza-video-processor/config.json`

## Platform-Specific Notes

### Linux
- **Wayland**: Requires `wl-screenrec` for optimal screen recording
- **X11**: Uses ffmpeg's `x11grab` - works out of the box
- **Audio**: Uses PipeWire's `pw-record` for audio capture
- **Hardware acceleration**: VAAPI encoding available with `--hw-accel` flag

### Windows
- **Screen recording**: Uses ffmpeg's `gdigrab` to capture the desktop
- **Audio**: Uses ffmpeg's DirectShow (`dshow`) for microphone input
- **Webcam**: Uses ffmpeg's DirectShow (`dshow`) for webcam capture
- **Note**: Ensure ffmpeg is installed and available in your PATH

### macOS
- **Screen recording**: Uses ffmpeg's AVFoundation to capture screens
- **Audio**: Uses ffmpeg's AVFoundation for microphone input
- **Webcam**: Uses ffmpeg's AVFoundation for webcam capture
- **Note**: macOS may require screen recording permissions - grant access when prompted
- **Note**: Install ffmpeg via Homebrew: `brew install ffmpeg`

## Keybindings (TUI)

| Key | Action |
|-----|--------|
| `Space` / `Enter` | Toggle recording |
| `q` | Quit |
| `?` | Toggle help |

## Hyprland Integration (Linux)

Add to your Hyprland config:

```conf
# Toggle screen recording
bind = $mainMod, R, exec, kartoza-video-processor toggle
```

## Building

```bash
# Build for current platform
make build

# Build static binary
make static

# Run tests
make test

# Run linter
make lint

# Build all platforms
make release
```

## License

MIT License - see [LICENSE](LICENSE) for details.

## Author

Tim Sutton - [Kartoza](https://kartoza.com)
