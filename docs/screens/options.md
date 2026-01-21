# Options

The Options screen allows you to configure application-wide settings including topics, default presenter, logo directory, and YouTube integration.

## Screen Preview

<div class="terminal-mockup">
<div class="terminal-header">
<div class="terminal-buttons">
<div class="terminal-button red"></div>
<div class="terminal-button yellow"></div>
<div class="terminal-button green"></div>
</div>
<div class="terminal-title">Options</div>
</div>
<div class="terminal-content"><span class="t-header">━━━━━━━━━━━━━━━━━━ Options ━━━━━━━━━━━━━━━━━━</span>

<span class="t-header">Topics</span>
<span class="t-selected"><span class="t-orange">→</span> <span class="t-white">QGIS sketches</span></span>
  <span class="t-blue">GIS development tutorials</span>
  <span class="t-blue">Open source workflows</span>
  <span class="t-blue">General tutorials</span>

<span class="t-gray">New topic:</span> <span class="t-white">█</span>
<span class="t-gray">[ Add ] [ Remove Selected ]</span>

<span class="t-header">Defaults</span>
<span class="t-blue">Default Presenter:</span> <span class="t-white">Tim Sketcher</span>

<span class="t-header">Paths</span>
<span class="t-blue">Logo Directory:</span>    <span class="t-cyan">/home/user/Pictures/logos</span>
                    <span class="t-gray">[ Browse... ]</span>

<span class="t-header">Integrations</span>
<span class="t-blue">YouTube:</span>           <span class="t-green">● Connected</span>
                    <span class="t-gray">[ Configure YouTube ]</span>

  <span class="t-green">[ Save ]</span>

<span class="t-gray">━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━</span>
<span class="t-gray">tab: next • shift+tab: prev • enter: select • esc: cancel</span>
</div>
</div>

## Configuration Sections

### Topics Management

Topics are categories for organizing your recordings. Each topic creates a separate folder in your output directory.

#### Topic List

<span class="t-header">**Topics**</span>

Displays all configured topics. Use ++up++ / ++down++ to navigate.

**Default Topics:**

- QGIS sketches
- GIS development tutorials
- Open source workflows
- General tutorials

---

#### Add Topic

<span class="t-gray">**New topic:**</span> *Text Input*

Enter a new topic name and press ++enter++ or select **[ Add ]** to add it.

**Rules:**

- Maximum 50 characters
- Must be unique
- Becomes a folder name (avoid special characters)

---

#### Remove Topic

<span class="t-gray">**[ Remove Selected ]**</span>

Removes the currently selected topic from the list.

!!! warning "Recordings Not Deleted"
    Removing a topic only removes it from the configuration. Existing recordings in that topic's folder are preserved.

---

### Default Presenter

<span class="t-blue">**Default Presenter:**</span> *Text Input*

Sets the default presenter name used in recording metadata and YouTube uploads.

**Used in:**

- Video metadata
- YouTube video description
- Recording history display

---

### Logo Directory

<span class="t-blue">**Logo Directory:**</span> *Path / File Browser*

Specifies the folder containing logo images for video overlays.

**Selecting a Directory:**

1. Press ++enter++ on **[ Browse... ]**
2. Navigate using the file browser
3. Select the directory containing your logos
4. Press ++enter++ to confirm

#### File Browser

<div class="terminal-mockup">
<div class="terminal-header">
<div class="terminal-buttons">
<div class="terminal-button red"></div>
<div class="terminal-button yellow"></div>
<div class="terminal-button green"></div>
</div>
<div class="terminal-title">Select Logo Directory</div>
</div>
<div class="terminal-content"><span class="t-header">Select Directory</span>

<span class="t-gray">Path:</span> <span class="t-cyan">/home/user/Pictures/logos</span>

<span class="t-blue">📁</span> <span class="t-gray">..</span>
<span class="t-selected"><span class="t-orange">→</span> <span class="t-blue">📁</span> <span class="t-white">logos</span></span>
<span class="t-blue">📁</span> <span class="t-gray">screenshots</span>
<span class="t-blue">📁</span> <span class="t-gray">wallpapers</span>
<span class="t-gray">📄</span> <span class="t-gray">profile.png</span>

<span class="t-gray">━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━</span>
<span class="t-gray">↑/↓: navigate • enter: select/open • esc: cancel</span>
</div>
</div>

**Supported Logo Formats:**

| Format | Extension | Notes |
|--------|-----------|-------|
| PNG | `.png` | Recommended for static logos |
| GIF | `.gif` | Animated logos supported |
| SVG | `.svg` | Vector graphics |
| JPEG | `.jpg`, `.jpeg` | Not recommended (no transparency) |

---

### YouTube Integration

<span class="t-blue">**YouTube:**</span> *Status / Configuration*

Shows YouTube connection status and provides access to setup.

**Status Indicators:**

| Status | Meaning |
|--------|---------|
| <span class="t-green">● Connected</span> | YouTube API configured and authenticated |
| <span class="t-yellow">● Not configured</span> | API credentials not set |
| <span class="t-red">● Auth expired</span> | Need to re-authenticate |

Press ++enter++ on **[ Configure YouTube ]** to open the [YouTube Setup](youtube-setup.md) screen.

---

### Save Button

<span class="t-green">**[ Save ]**</span>

Saves all configuration changes to disk.

**Configuration File Location:**

```
~/.config/kartoza-video-processor/config.json
```

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| ++tab++ | Next field |
| ++shift+tab++ | Previous field |
| ++up++ / ++down++ | Navigate lists |
| ++enter++ | Select / Confirm |
| ++esc++ | Cancel / Back |

## Field Navigation Order

1. Topic list
2. Add topic input
3. Add button
4. Remove button
5. Default presenter input
6. Logo directory browse
7. YouTube configure
8. Save button

## Configuration File

Settings are stored in JSON format:

```json
{
  "topics": [
    {"name": "QGIS sketches", "color": "#FF9500"},
    {"name": "GIS development tutorials", "color": "#0066CC"},
    {"name": "Open source workflows", "color": "#00CC66"},
    {"name": "General tutorials", "color": "#9966FF"}
  ],
  "default_presenter": "Tim Sketcher",
  "logo_directory": "/home/user/Pictures/logos",
  "youtube": {
    "client_id": "...",
    "client_secret": "...",
    "access_token": "...",
    "refresh_token": "..."
  }
}
```

## Workflow Position

This screen is accessed from:

- **[Main Menu](main-menu.md)** → Select "Options"

From here you can navigate to:

- **[YouTube Setup](youtube-setup.md)** → Configure YouTube API
- **[Main Menu](main-menu.md)** → Save and return

## Related Pages

- **[Main Menu](main-menu.md)** - Return to navigation
- **[YouTube Setup](youtube-setup.md)** - Configure YouTube credentials
- **[Recording Setup](recording-setup.md)** - Uses topics and logos
