# Config Package

The `config` package manages application configuration, including persistence to disk.

## Package Location

```
internal/config/
```

## Responsibility

- Load and save configuration
- Provide default values
- Manage file paths
- Store YouTube credentials

## Key Files

| File | Purpose |
|------|---------|
| `config.go` | Configuration loading/saving |
| `config_test.go` | Unit tests |

## Key Types

### Config

Main configuration structure:

```go
type Config struct {
    Topics           []models.Topic    `json:"topics"`
    DefaultPresenter string            `json:"default_presenter"`
    LogoDirectory    string            `json:"logo_directory"`
    YouTube          YouTubeConfig     `json:"youtube"`
    Recording        RecordingDefaults `json:"recording"`
}
```

### YouTubeConfig

YouTube API credentials:

```go
type YouTubeConfig struct {
    ClientID            string    `json:"client_id"`
    ClientSecret        string    `json:"client_secret"`
    AccessToken         string    `json:"access_token"`
    RefreshToken        string    `json:"refresh_token"`
    Expiry              time.Time `json:"expiry"`
    DefaultPrivacy      string    `json:"default_privacy"`
    DefaultPlaylistID   string    `json:"default_playlist_id"`
    DefaultPlaylistName string    `json:"default_playlist_name"`
}
```

### RecordingDefaults

Default recording settings:

```go
type RecordingDefaults struct {
    RecordAudio   bool   `json:"record_audio"`
    RecordWebcam  bool   `json:"record_webcam"`
    RecordScreen  bool   `json:"record_screen"`
    VerticalVideo bool   `json:"vertical_video"`
    AddLogos      bool   `json:"add_logos"`
    TitleColor    string `json:"title_color"`
}
```

### GifLoopMode

Animated GIF behavior:

```go
type GifLoopMode string

const (
    GifLoopContinuous GifLoopMode = "continuous"
    GifLoopOnceStart  GifLoopMode = "once_start"
    GifLoopOnceEnd    GifLoopMode = "once_end"
)
```

## Core Functions

### Load

Load configuration from disk:

```go
func Load() (*Config, error)
```

**Behavior:**

1. Determine config file path
2. Read file if exists
3. Parse JSON
4. Apply defaults for missing values
5. Return config

### Save

Persist configuration to disk:

```go
func Save(cfg *Config) error
```

**Behavior:**

1. Marshal to JSON (indented)
2. Create directory if needed
3. Write to file atomically
4. Set appropriate permissions

### GetDefaultVideosDir

Get output directory:

```go
func GetDefaultVideosDir() string
```

Returns: `~/Videos/Screencasts`

### GetCurrentRecordingNumber

Get next episode number:

```go
func GetCurrentRecordingNumber() int
```

## File Paths

### Configuration

```
~/.config/kartoza-screencaster/config.json
```

### PID Files (Runtime)

```
/tmp/kvp-video-pid
/tmp/kvp-audio-pid
/tmp/kvp-webcam-pid
/tmp/kvp-status
```

### Output Directory

```
~/Videos/Screencasts/<topic>/<title>/
```

## Path Constants

```go
const (
    VideoPIDFile   = "/tmp/kvp-video-pid"
    AudioPIDFile   = "/tmp/kvp-audio-pid"
    WebcamPIDFile  = "/tmp/kvp-webcam-pid"
    StatusFile     = "/tmp/kvp-status"
    OutputDirFile  = "/tmp/kvp-output-dir"
)
```

## Default Values

```go
func defaultConfig() *Config {
    return &Config{
        Topics:           models.DefaultTopics(),
        DefaultPresenter: "",
        LogoDirectory:    "",
        YouTube: YouTubeConfig{
            DefaultPrivacy: "unlisted",
        },
        Recording: RecordingDefaults{
            RecordAudio:  true,
            RecordScreen: true,
            RecordWebcam: false,
            AddLogos:     false,
            TitleColor:   "#FF9500",
        },
    }
}
```

## JSON Format

```json
{
  "topics": [
    {
      "name": "QGIS sketches",
      "color": "#FF9500"
    },
    {
      "name": "GIS development tutorials",
      "color": "#0066CC"
    }
  ],
  "default_presenter": "Tim Sketcher",
  "logo_directory": "/home/user/Pictures/logos",
  "youtube": {
    "client_id": "xxx.apps.googleusercontent.com",
    "client_secret": "xxx",
    "access_token": "ya29.xxx",
    "refresh_token": "1//xxx",
    "expiry": "2024-01-15T12:00:00Z",
    "default_privacy": "unlisted",
    "default_playlist_id": "PLxxx",
    "default_playlist_name": "My Playlist"
  },
  "recording": {
    "record_audio": true,
    "record_webcam": false,
    "record_screen": true,
    "vertical_video": false,
    "add_logos": true,
    "title_color": "#FF9500"
  }
}
```

## Error Handling

### First Run (No Config)

```go
func Load() (*Config, error) {
    path := getConfigPath()

    data, err := os.ReadFile(path)
    if os.IsNotExist(err) {
        // Return defaults for new users
        return defaultConfig(), nil
    }
    if err != nil {
        return nil, err
    }

    // Parse existing config
    var cfg Config
    if err := json.Unmarshal(data, &cfg); err != nil {
        return nil, err
    }

    return &cfg, nil
}
```

### Atomic Save

```go
func Save(cfg *Config) error {
    path := getConfigPath()

    // Create directory
    dir := filepath.Dir(path)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return err
    }

    // Marshal config
    data, err := json.MarshalIndent(cfg, "", "  ")
    if err != nil {
        return err
    }

    // Write to temp file first
    tmpFile := path + ".tmp"
    if err := os.WriteFile(tmpFile, data, 0600); err != nil {
        return err
    }

    // Atomic rename
    return os.Rename(tmpFile, path)
}
```

## XDG Compliance

```go
func getConfigPath() string {
    configDir := os.Getenv("XDG_CONFIG_HOME")
    if configDir == "" {
        home, _ := os.UserHomeDir()
        configDir = filepath.Join(home, ".config")
    }
    return filepath.Join(configDir, "kartoza-screencaster", "config.json")
}

func GetDefaultVideosDir() string {
    videosDir := os.Getenv("XDG_VIDEOS_DIR")
    if videosDir == "" {
        home, _ := os.UserHomeDir()
        videosDir = filepath.Join(home, "Videos")
    }
    return filepath.Join(videosDir, "Screencasts")
}
```

## Testing

```go
func TestLoad_NoFile(t *testing.T) {
    // Use temp directory
    os.Setenv("XDG_CONFIG_HOME", t.TempDir())

    cfg, err := Load()
    assert.NoError(t, err)
    assert.NotNil(t, cfg)
    assert.Equal(t, 4, len(cfg.Topics)) // Defaults
}

func TestSave_CreatesDirectory(t *testing.T) {
    tmpDir := t.TempDir()
    os.Setenv("XDG_CONFIG_HOME", tmpDir)

    cfg := &Config{Topics: []models.Topic{{Name: "Test"}}}
    err := Save(cfg)

    assert.NoError(t, err)
    assert.FileExists(t, filepath.Join(tmpDir, "kartoza-screencaster", "config.json"))
}
```

## Usage Example

```go
// Load config
cfg, err := config.Load()
if err != nil {
    log.Fatal(err)
}

// Modify
cfg.DefaultPresenter = "New Name"
cfg.Recording.AddLogos = true

// Save
if err := config.Save(cfg); err != nil {
    log.Fatal(err)
}

// Access paths
outputDir := config.GetDefaultVideosDir()
fmt.Println("Videos saved to:", outputDir)
```

## Related Packages

- **models** - Topic and other shared types
- **youtube** - Uses YouTubeConfig for auth
- **recorder** - Uses paths and defaults
