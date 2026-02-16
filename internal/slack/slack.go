package slack

import (
	"fmt"

	slackapi "github.com/slack-go/slack"
)

// Client is the interface for posting Slack messages.
type Client interface {
	PostMessage(channel, text, threadTS string) (ts string, err error)
}

type client struct {
	api *slackapi.Client
}

// New creates a Slack client with the given bot token.
func New(token string) Client {
	return &client{
		api: slackapi.New(token),
	}
}

func (c *client) PostMessage(channel, text, threadTS string) (string, error) {
	var opts []slackapi.MsgOption
	opts = append(opts, slackapi.MsgOptionText(text, false))
	if threadTS != "" {
		opts = append(opts, slackapi.MsgOptionTS(threadTS))
	}

	_, ts, err := c.api.PostMessage(channel, opts...)
	if err != nil {
		return "", fmt.Errorf("slack API error: %w", err)
	}
	return ts, nil
}
