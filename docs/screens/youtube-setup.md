# YouTube Setup

The YouTube Setup screen guides you through connecting your YouTube account to enable direct video uploads.

## Screen Preview

<div class="terminal-mockup">
<div class="terminal-header">
<div class="terminal-buttons">
<div class="terminal-button red"></div>
<div class="terminal-button yellow"></div>
<div class="terminal-button green"></div>
</div>
<div class="terminal-title">YouTube Setup</div>
</div>
<div class="terminal-content"><span class="t-header">━━━━━━━━━━━━━━━ YouTube Setup ━━━━━━━━━━━━━━━</span>

<span class="t-white">Step 1: Enter YouTube API Credentials</span>

<span class="t-gray">To upload videos to YouTube, you need to create</span>
<span class="t-gray">a project in Google Cloud Console and obtain</span>
<span class="t-gray">OAuth 2.0 credentials.</span>

<span class="t-orange">Client ID:</span>
<span class="t-cyan">┌────────────────────────────────────────────────┐</span>
<span class="t-cyan">│</span> <span class="t-white">123456789-abc123def456.apps.googleusercontent</span> <span class="t-cyan">│</span>
<span class="t-cyan">└────────────────────────────────────────────────┘</span>

<span class="t-blue">Client Secret:</span>
<span class="t-gray">┌────────────────────────────────────────────────┐</span>
<span class="t-gray">│</span> <span class="t-gray">••••••••••••••••••••••••</span>                    <span class="t-gray">│</span>
<span class="t-gray">└────────────────────────────────────────────────┘</span>

  <span class="t-green">[ Connect ]</span>    <span class="t-gray">[ Cancel ]</span>

<span class="t-gray">━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━</span>
<span class="t-gray">tab: next field • enter: connect • esc: cancel</span>
</div>
</div>

## Setup Process

### Step 1: Obtain API Credentials

Before using this screen, you need to create OAuth credentials in Google Cloud Console.

!!! info "Google Cloud Console Setup"
    1. Go to [Google Cloud Console](https://console.cloud.google.com/)
    2. Create a new project or select existing
    3. Enable the **YouTube Data API v3**
    4. Go to **Credentials** → **Create Credentials** → **OAuth 2.0 Client ID**
    5. Select **Desktop Application**
    6. Copy the **Client ID** and **Client Secret**

---

### Step 2: Enter Credentials

<span class="t-orange">**Client ID:**</span> *Text Input*

Paste your OAuth 2.0 Client ID from Google Cloud Console.

**Format:** `123456789-xxxxxx.apps.googleusercontent.com`

---

<span class="t-blue">**Client Secret:**</span> *Text Input (masked)*

Paste your OAuth 2.0 Client Secret.

**Security:** The secret is masked with dots for privacy.

---

### Step 3: Authenticate

Press **[ Connect ]** to begin OAuth authentication.

<div class="terminal-mockup">
<div class="terminal-header">
<div class="terminal-buttons">
<div class="terminal-button red"></div>
<div class="terminal-button yellow"></div>
<div class="terminal-button green"></div>
</div>
<div class="terminal-title">OAuth Authentication</div>
</div>
<div class="terminal-content"><span class="t-header">━━━━━━━━━━━━━━━ Authentication ━━━━━━━━━━━━━━━</span>

<span class="t-green">Please open this URL in your browser:</span>

<span class="t-cyan">https://accounts.google.com/o/oauth2/auth?...</span>

<span class="t-gray">A browser window should open automatically.</span>
<span class="t-gray">If not, copy and paste the URL above.</span>

<span class="t-yellow">Waiting for authentication...</span>

<span class="t-gray">━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━</span>
<span class="t-gray">esc: cancel authentication</span>
</div>
</div>

**Authentication Flow:**

1. Browser opens Google sign-in page
2. Select your YouTube account
3. Grant permissions to the application
4. Browser redirects to local callback
5. Application receives authentication tokens

---

### Step 4: Verification

After successful authentication:

<div class="terminal-mockup">
<div class="terminal-header">
<div class="terminal-buttons">
<div class="terminal-button red"></div>
<div class="terminal-button yellow"></div>
<div class="terminal-button green"></div>
</div>
<div class="terminal-title">Connected</div>
</div>
<div class="terminal-content"><span class="t-header">━━━━━━━━━━━━━━━ YouTube Connected ━━━━━━━━━━━━━━</span>

<span class="t-green">✓ Successfully connected to YouTube!</span>

<span class="t-blue">Channel:</span>      <span class="t-white">Tim's Tech Channel</span>
<span class="t-blue">Channel ID:</span>   <span class="t-gray">UC_xxxxxxxxxxxx</span>

<span class="t-header">Default Settings</span>
<span class="t-blue">Privacy:</span>      <span class="t-white">Unlisted</span>  <span class="t-gray">←/→ to change</span>
<span class="t-blue">Playlist:</span>     <span class="t-white">QGIS Tutorials</span>  <span class="t-gray">←/→ to change</span>

<span class="t-header">Playlists</span>
  <span class="t-orange">→</span> <span class="t-white">QGIS Tutorials</span> <span class="t-gray">(42 videos)</span>
    <span class="t-blue">GIS Tips</span> <span class="t-gray">(15 videos)</span>
    <span class="t-blue">Open Source GIS</span> <span class="t-gray">(28 videos)</span>
  <span class="t-green">[ + Create Playlist ]</span>

  <span class="t-green">[ Save & Close ]</span>    <span class="t-red">[ Disconnect ]</span>

<span class="t-gray">━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━</span>
</div>
</div>

## Connected State

Once connected, you can configure:

### Default Privacy

<span class="t-blue">**Privacy:**</span> *Selection*

Default privacy setting for new uploads.

| Setting | Description |
|---------|-------------|
| **Public** | Anyone can find and watch |
| **Unlisted** | Only people with the link can watch |
| **Private** | Only you can watch |

Use ++left++ / ++right++ to change.

---

### Default Playlist

<span class="t-blue">**Playlist:**</span> *Selection*

Default playlist for new uploads. Videos are automatically added.

---

### Playlist Management

View and manage your YouTube playlists.

#### Create New Playlist

<div class="terminal-mockup">
<div class="terminal-header">
<div class="terminal-buttons">
<div class="terminal-button red"></div>
<div class="terminal-button yellow"></div>
<div class="terminal-button green"></div>
</div>
<div class="terminal-title">Create Playlist</div>
</div>
<div class="terminal-content"><span class="t-header">━━━━━━━━━━━━ Create New Playlist ━━━━━━━━━━━━</span>

<span class="t-orange">Title:</span>
<span class="t-cyan">┌────────────────────────────────────────────────┐</span>
<span class="t-cyan">│</span> <span class="t-white">New Tutorial Series</span>                           <span class="t-cyan">│</span>
<span class="t-cyan">└────────────────────────────────────────────────┘</span>

<span class="t-blue">Description:</span>
<span class="t-gray">┌────────────────────────────────────────────────┐</span>
<span class="t-gray">│</span> <span class="t-gray">A collection of tutorials about...</span>            <span class="t-gray">│</span>
<span class="t-gray">└────────────────────────────────────────────────┘</span>

<span class="t-blue">Privacy:</span> <span class="t-white">Unlisted</span>  <span class="t-gray">←/→ to change</span>

  <span class="t-green">[ Create ]</span>    <span class="t-gray">[ Cancel ]</span>
</div>
</div>

---

### Disconnect

<span class="t-red">**[ Disconnect ]**</span>

Removes YouTube credentials and disconnects your account.

!!! warning "Requires Re-authentication"
    After disconnecting, you'll need to go through the full setup process again to reconnect.

## Error Handling

### Invalid Credentials

<div class="terminal-mockup">
<div class="terminal-header">
<div class="terminal-buttons">
<div class="terminal-button red"></div>
<div class="terminal-button yellow"></div>
<div class="terminal-button green"></div>
</div>
<div class="terminal-title">Error</div>
</div>
<div class="terminal-content"><span class="t-red">✗ Authentication Failed</span>

<span class="t-red">Error: Invalid client credentials</span>

<span class="t-gray">Please verify your Client ID and Client Secret</span>
<span class="t-gray">are correct and try again.</span>

  <span class="t-blue">[ Try Again ]</span>    <span class="t-gray">[ Cancel ]</span>
</div>
</div>

### Authentication Timeout

If the browser authentication takes too long, a timeout error is displayed. You can retry by pressing **[ Try Again ]**.

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| ++tab++ | Next field |
| ++shift+tab++ | Previous field |
| ++enter++ | Connect / Select |
| ++left++ / ++right++ | Change selection |
| ++esc++ | Cancel / Back |

## Required Permissions

The application requests the following YouTube API scopes:

| Scope | Permission |
|-------|------------|
| `youtube.upload` | Upload videos |
| `youtube.readonly` | Read channel info |
| `youtube` | Manage playlists |

## Token Storage

OAuth tokens are stored securely in:

```
~/.config/kartoza-video-processor/config.json
```

**Stored data:**

- Access token (short-lived)
- Refresh token (long-lived)
- Token expiry time

!!! note "Token Refresh"
    Access tokens are automatically refreshed using the refresh token. You shouldn't need to re-authenticate unless you disconnect or revoke access.

## Workflow Position

This screen is accessed from:

- **[Options](options.md)** → Select "Configure YouTube"

After setup, return to:

- **[Options](options.md)** → Continue configuration
- **[Main Menu](main-menu.md)** → Start using YouTube features

## Related Pages

- **[Options](options.md)** - Application settings
- **[YouTube Upload](youtube-upload.md)** - Upload videos after setup
- **[History](history.md)** - Upload past recordings
