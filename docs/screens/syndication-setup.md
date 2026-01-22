# Syndication Setup

The Syndication Setup screen allows you to configure accounts for announcing your YouTube videos across multiple social media and communication platforms.

## Supported Platforms

| Platform | Auth Method | Key Features |
|----------|-------------|--------------|
| **Mastodon** | OAuth2 | 500 char limit, images, any instance |
| **Bluesky** | App Password | 300 char limit, images, AT Protocol |
| **LinkedIn** | OAuth2 | Rich posts, article previews |
| **Telegram** | Bot Token | Markdown, images, multiple chats |
| **Signal** | signal-cli | Secure messaging, groups |
| **ntfy.sh** | HTTP (optional token) | Push notifications, click actions |
| **Google Chat** | Webhooks | Cards, buttons |
| **WordPress** | App Password | Full HTML, featured images |

## Screen Flow

### 1. Platform List

When you enter Syndication Setup, you'll see a list of all supported platforms:

```
Syndication Setup
Configure accounts to announce your videos

  > Mastodon (2 accounts, 1 enabled)
    Bluesky (1 accounts, 1 enabled)
    LinkedIn (0 accounts)
    Telegram (1 accounts, 1 enabled)
    Signal (0 accounts)
    ntfy.sh (0 accounts)
    Google Chat (0 accounts)
    WordPress (0 accounts)

up/down: select • enter: manage accounts • q: back
```

### 2. Account List

Press **Enter** on a platform to manage its accounts:

```
Mastodon Accounts

  > Tim's Mastodon [enabled]
    Work Account [disabled]

n: add • e: edit • d: delete • c: connect • t: toggle • esc: back
```

| Key | Action |
|-----|--------|
| ++n++ / ++a++ | Add new account |
| ++e++ | Edit selected account |
| ++d++ / ++delete++ | Delete selected account |
| ++c++ | Connect/authenticate |
| ++t++ | Toggle enabled/disabled |
| ++up++ / ++down++ | Navigate accounts |
| ++esc++ | Back to platform list |

### 3. Add/Edit Account

Each platform has specific fields to configure:

#### Mastodon
- **Account Name**: Friendly name for the account
- **Instance URL**: e.g., `mastodon.social`, `fosstodon.org`
- **Client ID**: From your Mastodon app registration
- **Client Secret**: From your Mastodon app registration

#### Bluesky
- **Account Name**: Friendly name
- **Handle**: Your Bluesky handle (e.g., `user.bsky.social`)
- **App Password**: Generate at bsky.app Settings > App Passwords

#### LinkedIn
- **Account Name**: Friendly name
- **Client ID**: From LinkedIn Developer Console
- **Client Secret**: From LinkedIn Developer Console

#### Telegram
- **Account Name**: Friendly name
- **Bot Token**: From @BotFather
- **Chat IDs**: Comma-separated channel/group IDs

#### Signal
- **Account Name**: Friendly name
- **Signal Number**: Your registered phone number (+1234567890)
- **Recipients**: Comma-separated phone numbers or group IDs

#### ntfy.sh
- **Account Name**: Friendly name
- **Topic**: The notification topic name
- **Server URL**: Default: `https://ntfy.sh`
- **Access Token**: Optional, for private topics

#### Google Chat
- **Account Name**: Friendly name
- **Webhook URL**: From Google Chat space settings

#### WordPress
- **Account Name**: Friendly name
- **Site URL**: Your WordPress site URL
- **Username**: WordPress username
- **App Password**: From WordPress User > App Passwords
- **Post Status**: `draft` or `publish`

## Authentication

### OAuth Platforms (Mastodon, LinkedIn)

1. Configure credentials and save the account
2. Select the account and press ++c++ to connect
3. A browser window will open for authorization
4. Paste the authorization code when prompted

### App Password Platforms (Bluesky, WordPress)

1. Generate an app password in the platform's settings
2. Enter the credentials and save
3. Press ++c++ to verify the connection

### Token/API Platforms (Telegram, ntfy, Google Chat)

1. Enter the required tokens/URLs
2. Save the account
3. Press ++c++ to test the connection

### Signal

Requires `signal-cli` to be installed and registered:

```bash
# Install signal-cli
# Register your number
signal-cli -a +1234567890 register
signal-cli -a +1234567890 verify CODE
```

## Registering OAuth Apps

### Mastodon

1. Go to your instance's settings (e.g., `https://mastodon.social/settings/applications`)
2. Click "New Application"
3. Set:
   - Application name: `Kartoza Video Processor`
   - Redirect URI: `urn:ietf:wg:oauth:2.0:oob`
   - Scopes: `read`, `write:statuses`, `write:media`
4. Copy Client ID and Client Secret

### LinkedIn

1. Go to [LinkedIn Developer Portal](https://www.linkedin.com/developers/)
2. Create a new app
3. Request access to `w_member_social` and `r_liteprofile` products
4. Add redirect URL: `http://localhost:8089/callback`
5. Copy Client ID and Client Secret

### Bluesky

1. Go to [bsky.app](https://bsky.app) Settings
2. Navigate to App Passwords
3. Create a new app password
4. Use your handle and the app password

## Configuration Storage

Account credentials are stored in:

```
~/.config/kartoza-video-processor/config.json
```

OAuth tokens and sessions are stored per-account:

```
~/.config/kartoza-video-processor/
├── syndication_token_synd_xxxxx.json    # OAuth tokens
└── syndication_session_synd_xxxxx.json  # Bluesky sessions
```

## Workflow Position

Access this screen from:

- **[Options](options.md)** > Syndication Setup

After setup:

- Use **[History](history.md)** to syndicate past recordings
- Syndication is automatically offered after YouTube uploads

## Related Pages

- **[Options](options.md)** - Application settings
- **[History](history.md)** - View and syndicate recordings
- **[YouTube Upload](youtube-upload.md)** - Upload and syndicate
