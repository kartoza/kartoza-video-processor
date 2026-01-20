# Development Setup

This guide covers setting up a development environment for Kartoza Video Processor.

## Prerequisites

### Required Tools

| Tool | Version | Purpose |
|------|---------|---------|
| **Go** | 1.21+ | Compilation |
| **FFmpeg** | 4.0+ | Video processing |
| **Git** | 2.0+ | Version control |

### Optional Tools

| Tool | Purpose |
|------|---------|
| **Nix** | Reproducible environment |
| **golangci-lint** | Code linting |
| **goreleaser** | Release automation |

## Setup Methods

### Using Nix (Recommended)

The project includes a Nix flake for reproducible development:

```bash
# Clone the repository
git clone https://github.com/kartoza/kartoza-video-processor.git
cd kartoza-video-processor

# Enter development shell
nix develop

# All dependencies are now available
go build ./cmd/kvp
```

**Benefits of Nix:**

- Exact dependency versions
- Isolated from system packages
- Works on any Linux/macOS

### Manual Setup

#### 1. Install Go

=== "Ubuntu/Debian"
    ```bash
    sudo apt install golang-go
    ```

=== "Fedora"
    ```bash
    sudo dnf install golang
    ```

=== "macOS"
    ```bash
    brew install go
    ```

=== "From source"
    ```bash
    wget https://go.dev/dl/go1.21.linux-amd64.tar.gz
    sudo tar -C /usr/local -xzf go1.21.linux-amd64.tar.gz
    export PATH=$PATH:/usr/local/go/bin
    ```

#### 2. Install FFmpeg

=== "Ubuntu/Debian"
    ```bash
    sudo apt install ffmpeg
    ```

=== "Fedora"
    ```bash
    sudo dnf install ffmpeg
    ```

=== "macOS"
    ```bash
    brew install ffmpeg
    ```

#### 3. Clone and Build

```bash
# Clone repository
git clone https://github.com/kartoza/kartoza-video-processor.git
cd kartoza-video-processor

# Download dependencies
go mod download

# Build
go build -o kvp ./cmd/kvp

# Run
./kvp
```

## Project Structure

```
kartoza-video-processor/
├── cmd/
│   └── kvp/              # Main application entry
│       └── main.go
├── internal/
│   ├── audio/            # Audio capture and processing
│   ├── config/           # Configuration management
│   ├── merger/           # Video post-processing
│   ├── models/           # Shared data structures
│   ├── monitor/          # Display detection
│   ├── notify/           # Desktop notifications
│   ├── recorder/         # Recording orchestration
│   ├── tui/              # Terminal user interface
│   ├── webcam/           # Webcam capture
│   └── youtube/          # YouTube API integration
├── docs/                 # Documentation (MkDocs)
├── flake.nix            # Nix flake definition
├── go.mod               # Go module definition
├── go.sum               # Dependency checksums
└── README.md            # Project overview
```

## Development Workflow

### Building

```bash
# Standard build
go build ./cmd/kvp

# Build with version info
go build -ldflags "-X main.version=dev" ./cmd/kvp

# Build for specific platform
GOOS=linux GOARCH=amd64 go build ./cmd/kvp
GOOS=darwin GOARCH=arm64 go build ./cmd/kvp
GOOS=windows GOARCH=amd64 go build ./cmd/kvp
```

### Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/recorder/...

# Verbose output
go test -v ./...
```

### Linting

```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linter
golangci-lint run

# Run with timeout (for large codebases)
golangci-lint run --timeout=5m
```

### Running

```bash
# Run directly
go run ./cmd/kvp

# Run built binary
./kvp

# Run with debug logging (if implemented)
DEBUG=1 ./kvp
```

## IDE Setup

### VS Code

Install extensions:

- **Go** (golang.go)
- **Even Better TOML** (tamasfe.even-better-toml)

Settings (`.vscode/settings.json`):

```json
{
    "go.lintTool": "golangci-lint",
    "go.lintFlags": ["--fast"],
    "go.useLanguageServer": true,
    "editor.formatOnSave": true,
    "[go]": {
        "editor.defaultFormatter": "golang.go"
    }
}
```

### GoLand / IntelliJ

1. Open project folder
2. Go module detected automatically
3. Configure golangci-lint in Settings → Tools → External Tools

### Neovim

With `nvim-lspconfig`:

```lua
require('lspconfig').gopls.setup{}
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `HOME` | User home directory | System |
| `XDG_CONFIG_HOME` | Config directory | `~/.config` |
| `XDG_VIDEOS_DIR` | Videos directory | `~/Videos` |
| `DEBUG` | Enable debug logging | (unset) |

## Common Tasks

### Adding a New Package

1. Create directory under `internal/`
2. Create main Go file with package declaration
3. Add tests in `*_test.go`
4. Import from other packages as needed

```bash
mkdir internal/newpackage
touch internal/newpackage/newpackage.go
touch internal/newpackage/newpackage_test.go
```

### Updating Dependencies

```bash
# Update all dependencies
go get -u ./...

# Update specific dependency
go get -u github.com/charmbracelet/bubbletea

# Tidy up go.mod
go mod tidy
```

### Creating a Release

```bash
# Tag a version
git tag v1.0.0
git push origin v1.0.0

# Build release binaries (with goreleaser)
goreleaser release --clean

# Or manually
GOOS=linux GOARCH=amd64 go build -o kvp-linux-amd64 ./cmd/kvp
GOOS=darwin GOARCH=amd64 go build -o kvp-darwin-amd64 ./cmd/kvp
GOOS=darwin GOARCH=arm64 go build -o kvp-darwin-arm64 ./cmd/kvp
GOOS=windows GOARCH=amd64 go build -o kvp-windows-amd64.exe ./cmd/kvp
```

## Debugging

### Using Delve

```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug main
dlv debug ./cmd/kvp

# Debug tests
dlv test ./internal/recorder/
```

### Debug Logging

Add debug statements:

```go
import "log"

log.Printf("Debug: value = %v", value)
```

### FFmpeg Debugging

To see FFmpeg output:

```go
cmd.Stdout = os.Stdout
cmd.Stderr = os.Stderr
```

## Documentation

### Building Docs Locally

```bash
# Enter nix shell (includes mkdocs)
nix develop

# Or install manually
pip install mkdocs-material

# Serve docs locally
mkdocs serve

# Build static site
mkdocs build
```

### Documentation Structure

```
docs/
├── index.md              # Home page
├── getting-started/      # Installation, quick start
├── screens/              # Screen documentation
├── workflows/            # Step-by-step guides
├── developer/            # Developer docs
└── assets/               # CSS, images
```

## Troubleshooting

### Go Module Issues

```bash
# Clear module cache
go clean -modcache

# Re-download dependencies
go mod download
```

### Build Failures

```bash
# Check Go version
go version

# Verify dependencies
go mod verify

# Check for missing imports
go mod tidy
```

### Test Failures

```bash
# Run with verbose output
go test -v ./...

# Run single test
go test -v -run TestName ./internal/package/
```
