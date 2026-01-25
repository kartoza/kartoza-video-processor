# Kartoza Screencaster

<div class="hero" markdown>
# Kartoza Screencaster
**A beautiful terminal-based screen recording application with professional video processing and YouTube integration.**
</div>

Welcome to the official documentation for **Kartoza Screencaster** - a powerful, elegant TUI (Terminal User Interface) application designed for creating professional screen recordings, processing videos with logos and effects, and seamlessly uploading to YouTube.

## Features at a Glance

<div class="feature-grid" markdown>

<div class="feature-card" markdown>
### Screen Recording
Capture any monitor with high-quality video encoding using FFmpeg. Support for multiple monitors and custom resolutions.
</div>

<div class="feature-card" markdown>
### Audio Recording
Simultaneous audio capture from your microphone with automatic normalization and noise reduction.
</div>

<div class="feature-card" markdown>
### Webcam Overlay
Include your webcam feed as a picture-in-picture overlay in your recordings.
</div>

<div class="feature-card" markdown>
### Logo Overlays
Add professional branding with customizable logo positions - left, right, and bottom corners with animated GIF support.
</div>

<div class="feature-card" markdown>
### Vertical Video
Automatic conversion to vertical format (9:16) perfect for YouTube Shorts, TikTok, and Instagram Reels.
</div>

<div class="feature-card" markdown>
### YouTube Integration
Direct upload to YouTube with playlist management, privacy controls, and automatic metadata.
</div>

</div>

## Quick Preview

Here's what the main menu looks like:

<div class="terminal-mockup">
<div class="terminal-header">
<div class="terminal-buttons">
<div class="terminal-button red"></div>
<div class="terminal-button yellow"></div>
<div class="terminal-button green"></div>
</div>
<div class="terminal-title">Kartoza Screencaster</div>
</div>
<div class="terminal-content"><span class="t-header">━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━</span>
<span class="t-header">                    Main Menu</span>
<span class="t-header">━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━</span>

<span class="t-selected">  <span class="t-orange">→ New Recording</span></span>
    <span class="t-blue">Recording History</span>
    <span class="t-blue">Options</span>
    <span class="t-blue">Quit</span>

<span class="t-gray">↑/k: up • ↓/j: down • enter/space: select • q: quit</span>
</div>
</div>

## Getting Started

1. **[Installation](getting-started/installation.md)** - Install Kartoza Screencaster on your system
2. **[Quick Start](getting-started/quickstart.md)** - Create your first recording in minutes

## User Guide

Learn about each screen in the application:

- **[Main Menu](screens/main-menu.md)** - Navigate the application
- **[Recording Setup](screens/recording-setup.md)** - Configure your recording
- **[Countdown](screens/countdown.md)** - Prepare for recording
- **[Recording](screens/recording.md)** - Manage active recordings
- **[Processing](screens/processing.md)** - Post-processing steps
- **[History](screens/history.md)** - Browse past recordings
- **[Options](screens/options.md)** - Application settings
- **[YouTube Setup](screens/youtube-setup.md)** - Connect to YouTube
- **[YouTube Upload](screens/youtube-upload.md)** - Upload your videos

## Workflows

Step-by-step guides for common tasks:

- **[Recording a Video](workflows/recording-workflow.md)** - Complete recording workflow
- **[Uploading to YouTube](workflows/youtube-workflow.md)** - YouTube upload process

## Developer Guide

For contributors and developers:

- **[Architecture](developer/architecture.md)** - System design and architecture
- **[Development Setup](developer/setup.md)** - Set up your development environment
- **[Libraries](developer/libraries.md)** - Third-party libraries used
- **[Modules](developer/modules/index.md)** - Detailed module documentation
- **[Design Decisions](developer/design-decisions.md)** - Why things are built the way they are

## Requirements

| Component | Requirement |
|-----------|-------------|
| **OS** | Linux (primary), macOS, Windows (experimental) |
| **Go** | 1.21 or later |
| **FFmpeg** | Required for all video operations |
| **Audio** | PipeWire, PulseAudio, or ALSA |

## Support

- **Issues**: [GitHub Issues](https://github.com/kartoza/kartoza-screencaster/issues)
- **Discussions**: [GitHub Discussions](https://github.com/kartoza/kartoza-screencaster/discussions)

---

<div style="text-align: center; margin-top: 2rem;">
<p>Made with :orange_heart: by <a href="https://kartoza.com">Kartoza</a></p>
</div>
