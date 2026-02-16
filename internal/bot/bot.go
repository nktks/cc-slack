package bot

import (
	"context"
	"log"
	"regexp"

	"github.com/nktks/cc-slack/internal/tmux"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

var mentionRe = regexp.MustCompile(`^<@[A-Z0-9]+>\s*`)

// ThreadLookup finds the tmux target for a given thread_ts.
type ThreadLookup interface {
	GetByThreadTS(threadTS string) (tmuxTarget string, ok bool)
}

// Bot listens for app_mention and message events via Slack Socket Mode
// and forwards messages to Claude Code via tmux send-keys.
type Bot struct {
	AppToken    string
	BotToken    string
	AllowedUser string
	Threads     ThreadLookup
}

// Run starts the Socket Mode connection and blocks until ctx is cancelled.
func (b *Bot) Run(ctx context.Context) error {
	api := slack.New(b.BotToken, slack.OptionAppLevelToken(b.AppToken))
	client := socketmode.New(api)
	handler := socketmode.NewSocketmodeHandler(client)

	handler.HandleEvents(slackevents.AppMention, func(evt *socketmode.Event, c *socketmode.Client) {
		c.Ack(*evt.Request)
		b.handleAppMention(evt)
	})

	handler.HandleEvents(slackevents.Message, func(evt *socketmode.Event, c *socketmode.Client) {
		c.Ack(*evt.Request)
		b.handleMessage(evt)
	})

	return handler.RunEventLoopContext(ctx)
}

func (b *Bot) handleAppMention(evt *socketmode.Event) {
	eventsAPI, ok := evt.Data.(slackevents.EventsAPIEvent)
	if !ok {
		log.Printf("[bot] failed to cast EventsAPIEvent")
		return
	}

	mention, ok := eventsAPI.InnerEvent.Data.(*slackevents.AppMentionEvent)
	if !ok {
		log.Printf("[bot] failed to cast AppMentionEvent")
		return
	}

	log.Printf("[bot] app_mention: user=%s thread_ts=%s text=%q", mention.User, mention.ThreadTimeStamp, mention.Text)

	b.forwardToTmux(mention.User, mention.ThreadTimeStamp, mention.Text)
}

func (b *Bot) handleMessage(evt *socketmode.Event) {
	eventsAPI, ok := evt.Data.(slackevents.EventsAPIEvent)
	if !ok {
		log.Printf("[bot] failed to cast EventsAPIEvent")
		return
	}

	msg, ok := eventsAPI.InnerEvent.Data.(*slackevents.MessageEvent)
	if !ok {
		log.Printf("[bot] failed to cast MessageEvent")
		return
	}

	// Ignore bot messages to avoid loops.
	if msg.BotID != "" || msg.SubType != "" {
		return
	}

	log.Printf("[bot] message: user=%s thread_ts=%s text=%q", msg.User, msg.ThreadTimeStamp, msg.Text)

	b.forwardToTmux(msg.User, msg.ThreadTimeStamp, msg.Text)
}

func (b *Bot) forwardToTmux(user, threadTS, text string) {
	// Only handle messages in threads that we created.
	if threadTS == "" {
		log.Printf("[bot] skipped: not in a thread")
		return
	}

	tmuxTarget, ok := b.Threads.GetByThreadTS(threadTS)
	if !ok {
		log.Printf("[bot] skipped: thread_ts=%s not found in store", threadTS)
		return
	}
	if tmuxTarget == "" {
		log.Printf("[bot] skipped: tmux target is empty for thread_ts=%s", threadTS)
		return
	}

	// Only allow messages from the configured user.
	if b.AllowedUser != "" && user != b.AllowedUser {
		log.Printf("[bot] skipped: user %s not allowed (allowed=%s)", user, b.AllowedUser)
		return
	}

	text = StripMention(text)
	if text == "" {
		log.Printf("[bot] skipped: text is empty after stripping mention")
		return
	}

	log.Printf("[bot] sending to tmux target=%s text=%q", tmuxTarget, text)
	if err := tmux.SendKeys(tmuxTarget, text); err != nil {
		log.Printf("[bot] tmux send-keys failed: %v", err)
	}
}

// StripMention removes the leading <@BOTID> mention from a message.
func StripMention(text string) string {
	return mentionRe.ReplaceAllString(text, "")
}
