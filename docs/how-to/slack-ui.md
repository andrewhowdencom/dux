# How to Configure the Slack UI

Dux natively supports operating within a Slack workspace. You can run Dux as a Slack App, interacting within direct messages, threads, and channels.

This guide outlines how to create the necessary Slack App, configure the proper permissions, and configure Dux to run it using either **Socket Mode** or **Webhooks**.

---

## 1. Create a Slack App

The easiest way to configure a Slack App for Dux is to use an App Manifest. This automatically configures scopes, events, socket mode, and slash commands.

1. Go to the [Slack API Apps page](https://api.slack.com/apps) and click **Create New App**.
2. Select **From an app manifest**, select the relevant workspace, and paste the following YAML:

```yaml
display_information:
  name: Dux Agent
  description: LLM-powered assistant built with Dux
  background_color: "#181e29"
features:
  app_home:
    home_tab_enabled: false
    messages_tab_enabled: true
    messages_tab_read_only_enabled: false
  bot_user:
    display_name: Dux
    always_online: true
  slash_commands:
    - command: /new
      url: https://dux.local/  # Ignored in socket mode, but required by API 
      description: Reset the current Dux conversational session
      should_escape: false
oauth_config:
  scopes:
    bot:
      - app_mentions:read
      - channels:history
      - chat:write
      - chat:write.customize
      - commands
      - groups:history
      - im:history
      - mpim:history
settings:
  event_subscriptions:
    bot_events:
      - app_mention
      - message.channels
      - message.groups
      - message.im
      - message.mpim
  interactivity:
    is_enabled: true
    is_env_aware: false
  org_deploy_enabled: false
  socket_mode_enabled: true
  token_rotation_enabled: false
```

3. Review the summary and click **Create**.

*If you prefer to configure this manually from scratch, continue reading.*

---

## 2. Configure Scopes (Permissions)

To allow Dux to read channel messages, reply, and request approval through Interactive Blocks, you need the following **Bot Token Scopes** (Found under *OAuth & Permissions*):

* `app_mentions:read`: To know when people ping the bot.
* `channels:history`: To read public channel messages the bot is in.
* `groups:history`: To read private channel messages.
* `mpim:history`: To read group messages.
* `im:history`: To read Direct Messages.
* `chat:write`: To send messages.
* `chat:write.customize`: To impersonate the Agent's configured name and icon natively in Slack.
* `commands`: To allow Dux to capture Slack Slash Commands (e.g., `/new`).

---

## 3. Choose your Connection Architecture

You can expose your bot to Slack using either **Socket Mode** (daemon, behind-the-firewall) or **Webhooks** (public endpoint). Socket Mode is generally recommended for internal tools.

### Option A: Socket Mode (Recommended)

Socket Mode requires an active daemon connection and does not require a public HTTP endpoint.

1. In the Slack App Dashboard sidebar, click **Socket Mode** and enable it.
2. It will prompt you to generate an **App-Level Token**. Generate it, give it the `connections:write` scope, and copy the token (starts with `xapp-`).
3. Dux will use this App Token alongside the Bot Token.

### Option B: Webhooks (Events API)

If you have a public endpoint (or are using something like `ngrok`), you can use Slack's Event API.

1. Find your **Signing Secret** under *Basic Information* in the Slack dashboard.
2. Enable *Event Subscriptions* and provide your public URL (e.g., `https://my-dux-url.com/`).
3. You will need to explicitly subscribe to `message.channels`, `message.im`, etc.

---

## 4. Install the App

Go to **Install App** in the dashboard sidebar and click "Install to Workspace". This will generate your **Bot User OAuth Token** (starts with `xoxb-`).

---

## 5. Dux Configuration

Add a `slack` UI entry to your Dux `config.yaml` using the tokens you generated.

### Example for Socket Mode

```yaml
ui:
  - type: slack
    agent: "customer-support-agent"  # (Optional) specific agent to bind
    configuration:
      bot_token: "xoxb-your-bot-token"
      app_token: "xapp-your-app-token"
      reply_mode: "mentioned"        # Only reply if explicitly @ mentioned ("always" or "mentioned")
```

### Example for Webhooks

```yaml
ui:
  - type: slack
    agent: "internal-sre-agent"
    configuration:
      bot_token: "xoxb-"
      signing_secret: "your-signing-secret"
      webhook_address: ":8443"        # Port Dux binds to locally
      reply_mode: "always"            # Responds to all messages in a channel
```

## 6. Run Dux!

Start the Slack integration:

```bash
dux serve slack
```

Once running:
* **@ Mention** the bot in a channel to talk in a thread.
* **DM** the bot to maintain a 1:1 conversation state.
* Use **Slash Commands** like `/new` to drop the current working memory and start fresh!
