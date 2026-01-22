# Changelog

All notable changes to Kartoza Video Processor will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.6.0] - 2026-01-22

### Added

#### Multi-Platform Syndication System
Announce your YouTube video uploads across 8 social media and communication platforms with a single action:

- **Mastodon** - Federated social network with OAuth2 authentication, supports any instance
- **Bluesky** - Decentralized AT Protocol network with app password authentication
- **LinkedIn** - Professional networking with OAuth2 and rich post previews
- **Telegram** - Bot-based posting to channels and groups with Markdown support
- **Signal** - End-to-end encrypted messaging via signal-cli integration
- **ntfy.sh** - Push notifications with click-through actions (self-hosted option)
- **Google Chat** - Workspace integration via incoming webhooks
- **WordPress** - Blog posts via REST API with app passwords

Key syndication features:
- Multi-account support for each platform
- Enable/disable individual accounts
- Platform-specific post formatting with character limits
- Automatic thumbnail upload where supported
- OAuth2 token refresh and session management
- Comprehensive setup documentation with step-by-step guides

#### Multi-Account YouTube Support
- Manage multiple YouTube accounts directly within the TUI
- Add, edit, and delete YouTube OAuth credentials
- Switch between accounts when uploading
- In-app account management (no manual JSON editing required)

#### History Screen Improvements
- New status column showing recording state (Processing, Ready, Uploaded, etc.)
- Error tracking with visual indicators for failed operations
- Media playback keybindings:
  - `p` - Play merged video
  - `v` - Play vertical video
  - `a` - Play audio file
  - `s` - Play screen recording

#### Recording Setup Enhancements
- Real-time spell checking for titles and descriptions
- Improved form styling with better visual feedback
- Enhanced text input handling

#### Documentation
- Comprehensive MkDocs documentation site
- Detailed setup guides for all syndication platforms
- Screen-by-screen user documentation
- Developer architecture guides

### Fixed
- All linting issues resolved
- Text input handling in form fields
- Layout consistency across all TUI screens

## [0.5.0] - 2026-01-17

### Added
- Experimental cross-platform support for macOS and Windows
- Platform-specific implementations for screen recording

## [0.4.1] - 2026-01-16

### Fixed
- Pause/resume/stop functionality bugs
- YouTube upload progress display

## [0.4.0] - 2026-01-15

### Added
- YouTube upload integration
- Playlist management
- Recording history with metadata

### Fixed
- Stop-start-stop processing bug
- Reprocess feature for failed recordings

## [0.3.0] - 2026-01-12

### Added
- Options screen with configurable settings
- Recording setup form with title/description
- Countdown timer before recording

## [0.2.0] - 2026-01-08

### Added
- Processing screen with progress indicators
- Audio normalization (EBU R128)
- Vertical video generation with webcam overlay

## [0.1.0] - 2026-01-05

### Added
- Initial release
- Multi-monitor screen recording
- Webcam capture at 60fps
- Audio recording with noise reduction
- Beautiful TUI interface
- CLI mode for scripting

[0.6.0]: https://github.com/kartoza/kartoza-video-processor/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/kartoza/kartoza-video-processor/compare/v0.4.1...v0.5.0
[0.4.1]: https://github.com/kartoza/kartoza-video-processor/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/kartoza/kartoza-video-processor/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/kartoza/kartoza-video-processor/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/kartoza/kartoza-video-processor/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/kartoza/kartoza-video-processor/releases/tag/v0.1.0
