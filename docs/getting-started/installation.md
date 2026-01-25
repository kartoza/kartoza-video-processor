# Installation

This guide covers installing Kartoza Screencaster on different operating systems.

## Prerequisites

Before installing, ensure you have the following dependencies:

| Dependency | Purpose | Check Command |
|------------|---------|---------------|
| **FFmpeg** | Video/audio processing | `ffmpeg -version` |
| **Go 1.21+** | Building from source | `go version` |

### Audio Backend (Linux)

One of the following audio systems:

- **PipeWire** (recommended) - Modern audio system
- **PulseAudio** - Traditional Linux audio
- **ALSA** - Direct hardware access

## Installation Methods

### Using Nix (Recommended)

If you have Nix installed with flakes enabled:

```bash
# Run directly without installing
nix run github:kartoza/kartoza-screencaster

# Or install to your profile
nix profile install github:kartoza/kartoza-screencaster
```

### From Source

Clone and build the project:

```bash
# Clone the repository
git clone https://github.com/kartoza/kartoza-screencaster.git
cd kartoza-screencaster

# Build the application
go build -o kvp ./cmd/kvp

# Run the application
./kvp
```

### Using Go Install

```bash
go install github.com/kartoza/kartoza-screencaster/cmd/kvp@latest
```

### Pre-built Binaries

Download pre-built binaries from the [Releases page](https://github.com/kartoza/kartoza-screencaster/releases).

Available binaries:

| Platform | Architecture | Filename |
|----------|--------------|----------|
| Linux | x86_64 | `kvp-linux-amd64` |
| macOS | Intel | `kvp-darwin-amd64` |
| macOS | Apple Silicon | `kvp-darwin-arm64` |
| Windows | x86_64 | `kvp-windows-amd64.exe` |

## Platform-Specific Notes

### Linux

Install FFmpeg using your package manager:

=== "Ubuntu/Debian"
    ```bash
    sudo apt install ffmpeg
    ```

=== "Fedora"
    ```bash
    sudo dnf install ffmpeg
    ```

=== "Arch Linux"
    ```bash
    sudo pacman -S ffmpeg
    ```

### macOS

Install FFmpeg using Homebrew:

```bash
brew install ffmpeg
```

!!! warning "Experimental Support"
    macOS support is experimental. Screen recording requires screen recording permissions in System Preferences > Security & Privacy > Privacy > Screen Recording.

### Windows

!!! warning "Experimental Support"
    Windows support is experimental. Some features may not work correctly.

Install FFmpeg:

1. Download from [ffmpeg.org](https://ffmpeg.org/download.html)
2. Extract to a folder (e.g., `C:\ffmpeg`)
3. Add the `bin` folder to your PATH

## Verifying Installation

After installation, verify everything works:

```bash
# Check kvp is installed
kvp --version

# Check FFmpeg is available
ffmpeg -version
```

## Next Steps

Once installed, proceed to the [Quick Start](quickstart.md) guide to create your first recording.
