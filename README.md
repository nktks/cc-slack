# cc-slack-notifier

A Go tool that sends Slack notifications via Claude Code [Hooks](https://code.claude.com/docs/en/hooks).

It notifies the event name and the last user prompt (truncated to 100 characters) on the following events:

- **PermissionRequest** - When Claude Code requests permission
- **Stop** - When Claude Code finishes responding
- **TaskCompleted** - When a task is marked as completed

## Notification example

```
[PermissionRequest] "Implement a Slack notification tool in Go that sends notifications when..."
```

## Prerequisites

### Slack Bot setup

1. Create a Slack App at https://api.slack.com/apps
2. Under **OAuth & Permissions**, add the following Bot Token Scope:
   - `chat:write`
3. Install the app to your workspace and copy the **Bot User OAuth Token** (`xoxb-...`)
4. Invite the bot to the target channel (if sending to a channel)

## Environment variables

| Variable | Fallback | Description |
|---|---|---|
| `CC_NOTIFY_SLACK_TOKEN` | `SLACK_TOKEN` | Slack Bot User OAuth Token (`xoxb-...`) |
| `CC_NOTIFY_SLACK_CHANNEL` | `SLACK_CHANNEL` | Target channel ID or user ID |

Set `CC_NOTIFY_SLACK_CHANNEL` to a Slack user ID (e.g. `U1234567890`) to send as a direct message.

Add these to your shell profile (e.g. `~/.zshrc`) so they are inherited by hooks:

```bash
export CC_NOTIFY_SLACK_TOKEN=xoxb-your-bot-token
export CC_NOTIFY_SLACK_CHANNEL=U1234567890  # User ID for DM
```

## Claude Code Hook configuration

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
            "command": "go run github.com/nktks/cc-slack-notifier@latest"
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
            "command": "go run github.com/nktks/cc-slack-notifier@latest"
          }
        ]
      }
    ],
    "TaskCompleted": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "go run github.com/nktks/cc-slack-notifier@latest"
          }
        ]
      }
    ]
  }
}
```

You can also add hooks interactively by typing `/hooks` in Claude Code.

## References

- [Hooks guide](https://code.claude.com/docs/en/hooks-guide)
- [Hooks reference](https://code.claude.com/docs/en/hooks)
