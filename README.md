# cc-slack

A local HTTP server that sends Slack notifications from Claude Code [Hooks](https://code.claude.com/docs/en/hooks), with optional Socket Mode bot to forward Slack replies back to Claude Code via tmux.

## Notification

Hooks forward events via `curl` to the server, which posts to Slack with session-based threading.

Supported events:

- **PermissionRequest** - When Claude Code requests permission (includes tool name, input details, and permission choices)
- **Stop** - When Claude Code finishes responding

Notifications include:

- Event name
- User prompt (truncated to 100 characters, shown only in the first message of a thread)
- Last assistant response
- Tool-specific details for PermissionRequest (command, file path, question/options)
- Permission choices (Yes/No) for PermissionRequest

Messages within the same session are grouped into a Slack thread. Thread replies omit the Prompt line since it is already visible in the parent message.

When the channel is set to a user ID (`U...`), the bot auto-mentions the user to ensure mobile push notifications for thread replies.

### Notification examples

**Stop:**
```
[Stop]
Prompt: "Implement a Slack notification tool in Go..."
Response: I have implemented the tool. The main.go file contains...
```

**PermissionRequest (Bash):**
```
[PermissionRequest] Bash
> go test ./...
> 1. Yes
> 2. Yes, and don't ask again for this session
> 3. No
Prompt: "Run the test suite and fix failures"
Response: Running the tests now.
```

**PermissionRequest (Write):**
```
[PermissionRequest] Write
> /path/to/main.go
> 1. Yes
> 2. Yes, allow all edits during this session
> 3. No
Prompt: "Create main.go"
Response: I'll create the file.
```

**PermissionRequest (AskUserQuestion):**
```
[PermissionRequest] AskUserQuestion
> Pick one 1. A 2. B 3. Type something. 4. Chat about this
Prompt: "Help me choose"
```

## Reply Bot (Socket Mode)

When `CC_NOTIFY_SLACK_APP_TOKEN` is set, the server also runs a Slack Socket Mode bot. The bot listens for `app_mention` and `message` events in notification threads and forwards the message text to the corresponding Claude Code session via `tmux send-keys`. `@mention` is not required — any message in a notification thread will be forwarded.

The bot forwards a message only when all of the following conditions are met:

- The message is in a thread (not a top-level message)
- The thread was created by cc-slack (tracked in the in-memory thread store)
- The sender matches the allowed user
- The message is not from a bot (bot messages are ignored to avoid loops)

Requirements:

- Claude Code must be running inside a tmux session
- The hook command must include the `X-Tmux-Target` header (see hook config below)

## Prerequisites

### Slack Bot setup

1. Create a Slack App at https://api.slack.com/apps
2. Under **OAuth & Permissions**, add the following Bot Token Scope:
   - `chat:write`
3. Install the app to your workspace and copy the **Bot User OAuth Token** (`xoxb-...`)
4. Invite the bot to the target channel (if sending to a channel)

#### Socket Mode setup (optional, for reply bot)

5. Under **Event Subscriptions**, enable events and subscribe to bot events:
   - `app_mention` (required for receiving replies in threads)
   - `message.im` (required to enable the Messages Tab for DM-based notifications)
6. Under **App Home** → **Show Tabs**, enable **Messages Tab** and check "Allow users to send Slash commands and messages from the messages tab"
7. Under **Socket Mode**, enable Socket Mode and generate an **App-Level Token** (`xapp-...`) with `connections:write` scope

## Usage

### 1. Start the server

```bash
CC_NOTIFY_SLACK_TOKEN=xoxb-your-bot-token \
CC_NOTIFY_SLACK_CHANNEL=C1234567890 \
CC_NOTIFY_SLACK_USER_ID=U1234567890 \
go run github.com/nktks/cc-slack/cmd/server@latest
```

With reply bot enabled:

```bash
CC_NOTIFY_SLACK_TOKEN=xoxb-your-bot-token \
CC_NOTIFY_SLACK_CHANNEL=C1234567890 \
CC_NOTIFY_SLACK_USER_ID=U1234567890 \
CC_NOTIFY_SLACK_APP_TOKEN=xapp-your-app-token \
go run github.com/nktks/cc-slack/cmd/server@latest
```

Use `-port` flag to change the listen port (default: `19999`):

```bash
go run github.com/nktks/cc-slack/cmd/server@latest -port 18888
```

### 2. Configure Claude Code hooks

Add the following to `~/.claude/settings.json`:

```json
{
  "hooks": {
    "PermissionRequest": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "bash -c 'curl -sf -X POST -H \"Content-Type: application/json\" -H \"X-Tmux-Target: $(tmux display-message -p \"#{session_name}:#{window_index}.#{pane_index}\" 2>/dev/null)\" -d @- http://localhost:19999/hook'"
          }
        ]
      }
    ],
    "Stop": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "bash -c 'curl -sf -X POST -H \"Content-Type: application/json\" -H \"X-Tmux-Target: $(tmux display-message -p \"#{session_name}:#{window_index}.#{pane_index}\" 2>/dev/null)\" -d @- http://localhost:19999/hook'"
          }
        ]
      }
    ]
  }
}
```

If you don't use the reply bot, you can use a simpler hook command without the `X-Tmux-Target` header:

```json
{
  "type": "command",
  "command": "curl -sf -X POST -H 'Content-Type: application/json' -d @- http://localhost:19999/hook"
}
```

You can also add hooks interactively by typing `/hooks` in Claude Code.

## Configuration

### Environment variables

| Variable | Fallback | Required | Description |
|---|---|---|---|
| `CC_NOTIFY_SLACK_TOKEN` | `SLACK_TOKEN` | Yes | Slack Bot User OAuth Token (`xoxb-...`) |
| `CC_NOTIFY_SLACK_CHANNEL` | `SLACK_CHANNEL` | Yes | Target channel ID (`C...`) or user ID (`U...`) for DM |
| `CC_NOTIFY_SLACK_USER_ID` | - | No | User ID (`U...`) to mention in notifications. Required when bot is enabled with a channel |
| `CC_NOTIFY_SLACK_APP_TOKEN` | - | No | Slack App-Level Token (`xapp-...`) to enable Socket Mode reply bot |

### Flags

| Flag | Default | Description |
|---|---|---|
| `-port` | `19999` | Server listen port |

### Mention behavior

- If `CC_NOTIFY_SLACK_USER_ID` is set, that user is mentioned in all notifications.
- If the channel is a user ID (starting with `U`) and no explicit mention user is set, the channel user is auto-mentioned.
- Otherwise, no mention is added.

### Allowed user (reply bot)

- If `CC_NOTIFY_SLACK_CHANNEL` starts with `U`, that user ID is used as the allowed user.
- Otherwise, `CC_NOTIFY_SLACK_USER_ID` is used (required when bot is enabled with a channel).

## Architecture

```
Claude Code Hook → curl → HTTP Server (:19999) → Slack API (notifications)

Slack Socket Mode ← app_mention / message events
        ↓
  tmux send-keys → Claude Code (tmux session)
```

The server holds session-to-thread mappings in memory (including the tmux target pane), so all notifications from the same Claude Code session are grouped into a single Slack thread. When the bot receives an `app_mention` in a known thread, it forwards the message to the corresponding tmux pane. Old thread mappings are cleaned up after 30 days.

## References

- [Hooks guide](https://code.claude.com/docs/en/hooks-guide)
- [Hooks reference](https://code.claude.com/docs/en/hooks)
