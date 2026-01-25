# Syndication Setup

The Syndication Setup screen allows you to configure accounts for announcing your YouTube videos across multiple social media and communication platforms. Once configured, you can automatically share your video announcements to all enabled platforms with a single action.

## Supported Platforms

| Platform | Auth Method | Character Limit | Key Features |
|----------|-------------|-----------------|--------------|
| **Mastodon** | OAuth2 | 500 chars | Federated, any instance, images |
| **Bluesky** | App Password | 300 chars | Decentralized, AT Protocol |
| **LinkedIn** | OAuth2 | 3000 chars | Professional network, rich previews |
| **Telegram** | Bot Token | 4096 chars | Markdown formatting, channels |
| **Signal** | signal-cli | Unlimited | End-to-end encrypted, groups |
| **ntfy.sh** | HTTP Token | Unlimited | Push notifications, click actions |
| **Google Chat** | Webhook | 4096 chars | Cards, buttons, workspace integration |
| **WordPress** | App Password | Unlimited | Full HTML, featured images, SEO |

## Accessing Syndication Setup

Navigate to: **Main Menu** → **Options** → **Syndication Setup**

## Screen Navigation

### Platform List

When you enter Syndication Setup, you'll see all supported platforms with account counts:

```
┌─────────────────────────────────────────┐
│           SYNDICATION SETUP             │
├─────────────────────────────────────────┤
│  Select Platform                        │
│  Configure accounts to announce videos  │
│                                         │
│  > 🐘 Mastodon (2 accounts, 1 enabled)  │
│    🦋 Bluesky (1 accounts, 1 enabled)   │
│    💼 LinkedIn (0 accounts)             │
│    ✈️ Telegram (1 accounts, 1 enabled)  │
│    📱 Signal (0 accounts)               │
│    🔔 ntfy.sh (0 accounts)              │
│    💬 Google Chat (0 accounts)          │
│    📝 WordPress (0 accounts)            │
├─────────────────────────────────────────┤
│  ↑/↓: select • enter: manage • q: back │
└─────────────────────────────────────────┘
```

### Account List

Press ++enter++ on a platform to view and manage its accounts:

```
┌─────────────────────────────────────────┐
│        🐘 MASTODON ACCOUNTS             │
├─────────────────────────────────────────┤
│  > Personal Account [enabled]           │
│    Work Account [disabled]              │
├─────────────────────────────────────────┤
│  n: add • e: edit • d: delete           │
│  c: connect • t: toggle • esc: back     │
└─────────────────────────────────────────┘
```

| Key | Action |
|-----|--------|
| ++n++ / ++a++ | Add new account |
| ++e++ | Edit selected account |
| ++d++ / ++delete++ | Delete selected account |
| ++c++ | Connect/authenticate account |
| ++t++ | Toggle enabled/disabled |
| ++up++ / ++down++ | Navigate accounts |
| ++esc++ | Back to platform list |

---

## Platform Setup Guides

### Mastodon

Mastodon is a federated social network. You can connect to any Mastodon instance.

#### Prerequisites

- A Mastodon account on any instance
- Ability to create an application on your instance

#### Step 1: Register an OAuth Application

1. Log into your Mastodon instance (e.g., `mastodon.social`, `fosstodon.org`)
2. Go to **Settings** → **Development** → **New Application**
   - Direct URL: `https://YOUR_INSTANCE/settings/applications`
3. Fill in the application details:
   - **Application name**: `Kartoza Screencaster`
   - **Application website**: `https://github.com/kartoza/kartoza-screencaster` (optional)
   - **Redirect URI**: `urn:ietf:wg:oauth:2.0:oob`
   - **Scopes**: Select the following:
     - [x] `read` - Read account information
     - [x] `write:statuses` - Post statuses
     - [x] `write:media` - Upload media attachments
4. Click **Submit**
5. Copy the **Client ID** and **Client Secret**

#### Step 2: Add Account in App

1. Navigate to **Syndication Setup** → **Mastodon**
2. Press ++n++ to add a new account
3. Fill in the fields:
   - **Account Name**: A friendly name (e.g., "Personal Mastodon")
   - **Instance URL**: Your instance domain (e.g., `mastodon.social`)
   - **Client ID**: Paste from step 1
   - **Client Secret**: Paste from step 1
4. Press ++enter++ to save

#### Step 3: Authorize the App

1. Select your account and press ++c++ to connect
2. A browser window will open to your instance's authorization page
3. Click **Authorize** to grant permissions
4. Copy the authorization code displayed
5. Paste the code in the app when prompted

!!! success "Connected"
    Your Mastodon account is now ready for syndication!

---

### Bluesky

Bluesky uses the AT Protocol for decentralized social networking.

#### Prerequisites

- A Bluesky account (handle ending in `.bsky.social` or custom domain)

#### Step 1: Generate an App Password

1. Log into [bsky.app](https://bsky.app)
2. Click your avatar → **Settings**
3. Navigate to **App Passwords** (under Advanced)
4. Click **Add App Password**
5. Enter a name: `Kartoza Screencaster`
6. Click **Create App Password**
7. **Copy the password immediately** - it won't be shown again!

!!! warning "App Password Security"
    App passwords provide full access to your account. Store them securely and never share them. You can revoke app passwords at any time from Settings.

#### Step 2: Add Account in App

1. Navigate to **Syndication Setup** → **Bluesky**
2. Press ++n++ to add a new account
3. Fill in the fields:
   - **Account Name**: A friendly name (e.g., "My Bluesky")
   - **Handle**: Your full handle (e.g., `username.bsky.social`)
   - **App Password**: Paste the app password from step 1
4. Press ++enter++ to save

#### Step 3: Verify Connection

1. Select your account and press ++c++ to test the connection
2. The app will create a session with Bluesky's servers

!!! tip "Session Management"
    Bluesky sessions are automatically refreshed. You shouldn't need to reconnect unless you revoke the app password.

---

### LinkedIn

LinkedIn requires OAuth2 authentication with a registered application.

#### Prerequisites

- A LinkedIn account
- A LinkedIn Company Page (if posting as organization) or personal profile

#### Step 1: Create a LinkedIn App

1. Go to [LinkedIn Developer Portal](https://www.linkedin.com/developers/)
2. Click **Create App**
3. Fill in the app details:
   - **App name**: `Kartoza Screencaster`
   - **LinkedIn Page**: Select your company page (required)
   - **Privacy policy URL**: Can use your website
   - **App logo**: Upload any square image
4. Click **Create app**

#### Step 2: Configure Products and Permissions

1. In your app dashboard, go to the **Products** tab
2. Request access to:
   - **Share on LinkedIn** - Required for posting
   - **Sign In with LinkedIn using OpenID Connect** - Required for authentication
3. Go to the **Auth** tab
4. Under **OAuth 2.0 settings**, add redirect URL:
   ```
   http://localhost:8089/callback
   ```
5. Copy the **Client ID** and **Client Secret**

!!! note "Approval Process"
    LinkedIn may require manual approval for certain API products. The "Share on LinkedIn" product typically requires a review that can take a few days.

#### Step 3: Add Account in App

1. Navigate to **Syndication Setup** → **LinkedIn**
2. Press ++n++ to add a new account
3. Fill in the fields:
   - **Account Name**: A friendly name (e.g., "Company LinkedIn")
   - **Client ID**: Paste from developer portal
   - **Client Secret**: Paste from developer portal
4. Press ++enter++ to save

#### Step 4: Authorize the App

1. Select your account and press ++c++ to connect
2. A browser window opens to LinkedIn's authorization page
3. Sign in and click **Allow**
4. Copy the authorization code from the redirect URL
5. Paste the code in the app

---

### Telegram

Telegram uses bot tokens for automated posting to channels and groups.

#### Prerequisites

- A Telegram account
- A channel or group where you want to post
- Admin rights in the target channel/group

#### Step 1: Create a Telegram Bot

1. Open Telegram and search for **@BotFather**
2. Start a chat and send `/newbot`
3. Follow the prompts:
   - Enter a name for your bot (e.g., "Video Announcements")
   - Enter a username ending in `bot` (e.g., `kartoza_videos_bot`)
4. **Copy the HTTP API token** provided by BotFather

!!! example "Bot Token Format"
    ```
    123456789:ABCdefGHIjklMNOpqrsTUVwxyz
    ```

#### Step 2: Get Your Chat ID

**For Channels:**

1. Add your bot as an administrator to your channel
2. Post any message to the channel
3. Visit: `https://api.telegram.org/bot<YOUR_TOKEN>/getUpdates`
4. Look for `"chat":{"id":-100XXXXXXXXXX}` - this is your channel ID

**For Groups:**

1. Add your bot to your group
2. Send a message mentioning the bot: `@your_bot test`
3. Visit the same URL as above
4. The group ID will appear (usually starts with `-`)

**For Private Messages:**

1. Start a chat with your bot
2. Send any message
3. Check the getUpdates URL for your personal chat ID

!!! tip "Multiple Destinations"
    You can send to multiple channels/groups by entering comma-separated chat IDs.

#### Step 3: Add Account in App

1. Navigate to **Syndication Setup** → **Telegram**
2. Press ++n++ to add a new account
3. Fill in the fields:
   - **Account Name**: A friendly name (e.g., "Announcements Channel")
   - **Bot Token**: Paste from BotFather
   - **Chat IDs**: Enter one or more chat IDs, comma-separated
     - Example: `-1001234567890,-1009876543210`
4. Press ++enter++ to save

#### Step 4: Test Connection

1. Select your account and press ++c++ to send a test message
2. Check your Telegram channel/group for the test message

---

### Signal

Signal uses signal-cli for sending encrypted messages.

#### Prerequisites

- A phone number for Signal registration
- `signal-cli` installed and in your PATH

#### Step 1: Install signal-cli

=== "Linux (Manual)"

    ```bash
    # Download the latest release
    wget https://github.com/AsamK/signal-cli/releases/download/v0.13.0/signal-cli-0.13.0-Linux.tar.gz

    # Extract
    tar xf signal-cli-0.13.0-Linux.tar.gz

    # Move to PATH
    sudo mv signal-cli-0.13.0/bin/signal-cli /usr/local/bin/
    sudo mv signal-cli-0.13.0/lib /usr/local/lib/signal-cli
    ```

=== "macOS (Homebrew)"

    ```bash
    brew install signal-cli
    ```

=== "Nix"

    ```bash
    nix-env -iA nixpkgs.signal-cli
    # Or in your flake/configuration
    ```

#### Step 2: Register Your Number

```bash
# Request verification code via SMS
signal-cli -a +1234567890 register

# Or request via voice call
signal-cli -a +1234567890 register --voice

# Verify with the code you receive
signal-cli -a +1234567890 verify 123456
```

!!! warning "Number Restrictions"
    Signal may require CAPTCHA verification. If you see a CAPTCHA error, visit the URL provided and complete verification, then run:
    ```bash
    signal-cli -a +1234567890 register --captcha "signalcaptcha://..."
    ```

#### Step 3: Get Group IDs (Optional)

To send to Signal groups:

```bash
# List all groups you're a member of
signal-cli -a +1234567890 listGroups -d

# Output shows group ID in base64 format
```

#### Step 4: Add Account in App

1. Navigate to **Syndication Setup** → **Signal**
2. Press ++n++ to add a new account
3. Fill in the fields:
   - **Account Name**: A friendly name
   - **Signal Number**: Your registered number (e.g., `+1234567890`)
   - **Recipients**: Comma-separated phone numbers or group IDs
     - Example: `+1987654321,group.XXXXXXXX==`
4. Press ++enter++ to save

---

### ntfy.sh

ntfy.sh provides simple HTTP-based push notifications.

#### Prerequisites

- None for public topics
- A ntfy.sh account for private topics (optional)

#### Step 1: Choose a Topic

Topics are like channels. Anyone who knows the topic name can subscribe.

**Public Topics:**

- Choose a unique, hard-to-guess topic name
- Example: `kartoza-videos-abc123xyz`

**Private Topics (ntfy.sh account required):**

1. Create an account at [ntfy.sh](https://ntfy.sh)
2. Go to **Account** → **Access Tokens**
3. Generate a new token
4. Reserve your topic name in account settings

!!! tip "Self-Hosted ntfy"
    You can also run your own ntfy server. Just change the Server URL in the account configuration.

#### Step 2: Add Account in App

1. Navigate to **Syndication Setup** → **ntfy.sh**
2. Press ++n++ to add a new account
3. Fill in the fields:
   - **Account Name**: A friendly name
   - **Topic**: Your topic name (e.g., `my-video-announcements`)
   - **Server URL**: `https://ntfy.sh` (or your self-hosted URL)
   - **Access Token**: Leave empty for public topics, or enter token for private
4. Press ++enter++ to save

#### Step 3: Subscribe to Notifications

Tell your audience to subscribe:

- **Web**: Visit `https://ntfy.sh/YOUR_TOPIC`
- **Android**: Install ntfy app, add topic
- **iOS**: Install ntfy app, add topic
- **Desktop**: Use web interface or CLI

```bash
# CLI subscription
ntfy subscribe YOUR_TOPIC
```

---

### Google Chat

Google Chat uses incoming webhooks for posting to spaces.

#### Prerequisites

- A Google Workspace account
- A Google Chat space where you have permissions to add webhooks

#### Step 1: Create a Webhook

1. Open [Google Chat](https://chat.google.com)
2. Open the space where you want to post
3. Click the space name at the top → **Apps & integrations**
4. Click **+ Add webhooks**
5. Enter a name: `Kartoza Screencaster`
6. Optionally add an avatar URL
7. Click **Save**
8. **Copy the webhook URL**

!!! warning "Webhook URL Security"
    Anyone with the webhook URL can post to your space. Keep it confidential and don't commit it to version control.

#### Step 2: Add Account in App

1. Navigate to **Syndication Setup** → **Google Chat**
2. Press ++n++ to add a new account
3. Fill in the fields:
   - **Account Name**: A friendly name (e.g., "Team Updates Space")
   - **Webhook URL**: Paste the full webhook URL
4. Press ++enter++ to save

#### Step 3: Test Connection

1. Select your account and press ++c++ to send a test message
2. Check your Google Chat space

---

### WordPress

WordPress uses application passwords for REST API authentication.

#### Prerequisites

- A WordPress site (self-hosted or WordPress.com Business plan)
- Administrator or Editor role on the site
- WordPress 5.6+ (for application passwords)

#### Step 1: Enable Application Passwords

Application passwords are enabled by default on WordPress 5.6+. If disabled:

1. Add to your theme's `functions.php` or a plugin:
   ```php
   add_filter('wp_is_application_passwords_available', '__return_true');
   ```

#### Step 2: Generate an Application Password

1. Log into your WordPress admin dashboard
2. Go to **Users** → **Profile** (or click your username)
3. Scroll down to **Application Passwords**
4. Enter a name: `Kartoza Screencaster`
5. Click **Add New Application Password**
6. **Copy the password immediately** - it won't be shown again!

!!! note "Password Format"
    The password is displayed with spaces for readability (e.g., `XXXX XXXX XXXX`). You can enter it with or without spaces - both work.

#### Step 3: Add Account in App

1. Navigate to **Syndication Setup** → **WordPress**
2. Press ++n++ to add a new account
3. Fill in the fields:
   - **Account Name**: A friendly name (e.g., "Company Blog")
   - **Site URL**: Your WordPress URL (e.g., `https://example.com`)
   - **Username**: Your WordPress username
   - **App Password**: Paste the application password
   - **Post Status**: `draft` (review before publishing) or `publish` (immediate)
4. Press ++enter++ to save

#### Step 4: Test Connection

1. Select your account and press ++c++ to verify authentication
2. The app will attempt to access the WordPress REST API

!!! tip "Post Categories"
    Posts are created in the default category. You can edit them in WordPress to add categories, tags, and featured images before publishing.

---

## Configuration Storage

All syndication settings are stored locally:

```
~/.config/kartoza-screencaster/
├── config.json                          # Main config with account settings
├── syndication_token_synd_xxxxx.json    # OAuth tokens (Mastodon, LinkedIn)
└── syndication_session_synd_xxxxx.json  # Bluesky sessions
```

!!! warning "Credential Security"
    - Config files contain sensitive credentials
    - Files are only readable by your user (`chmod 600`)
    - Never commit these files to version control
    - Back up securely if needed

---

## Troubleshooting

### Common Issues

**"Connection failed" for OAuth platforms:**

- Verify Client ID and Secret are correct
- Check that redirect URI matches exactly
- Ensure required scopes/products are enabled

**"Unauthorized" for Telegram:**

- Verify bot token is correct
- Ensure bot is admin in the channel
- Check chat ID format (channels start with `-100`)

**"signal-cli not found":**

- Ensure signal-cli is in your PATH
- Check with: `which signal-cli`
- Verify number is registered: `signal-cli -a +NUMBER listGroups`

**LinkedIn posts not appearing:**

- Check if "Share on LinkedIn" product is approved
- Verify account has posting permissions
- Try posting to personal profile first

**WordPress authentication failed:**

- Verify site URL is correct (include https://)
- Check username spelling
- Regenerate application password if needed
- Ensure WordPress 5.6+ or application passwords plugin

---

## Related Pages

- **[Options](options.md)** - Application settings
- **[History](history.md)** - View recordings and syndicate
- **[YouTube Upload](youtube-upload.md)** - Upload videos
- **[YouTube Setup](youtube-setup.md)** - Configure YouTube accounts
