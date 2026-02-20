package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/nktks/cc-slack/internal/bot"
	"github.com/nktks/cc-slack/internal/ccusage"
	"github.com/nktks/cc-slack/internal/server"
	"github.com/nktks/cc-slack/internal/slack"
	"github.com/robfig/cron/v3"
)

func main() {
	port := flag.String("port", "19999", "server listen port")
	ccusageCron := flag.String("ccusage-cron", "", "cron schedule for ccusage weekly report (e.g. \"0 9 * * 1\")")
	flag.Parse()

	token := envWithFallback("CC_NOTIFY_SLACK_TOKEN", "SLACK_TOKEN")
	channel := envWithFallback("CC_NOTIFY_SLACK_CHANNEL", "SLACK_CHANNEL")
	if token == "" || channel == "" {
		log.Fatal("CC_NOTIFY_SLACK_TOKEN and CC_NOTIFY_SLACK_CHANNEL must be set")
	}

	userID := os.Getenv("CC_NOTIFY_SLACK_USER_ID")

	slackClient := slack.New(token)

	threads := server.NewThreadStore()
	go func() {
		for {
			time.Sleep(1 * time.Hour)
			threads.CleanOlderThan(30 * 24 * time.Hour)
		}
	}()

	h := &server.Handler{
		Slack:         slackClient,
		Channel:       channel,
		UserID:  userID,
		Threads:       threads,
	}

	appToken := os.Getenv("CC_NOTIFY_SLACK_APP_TOKEN")
	if appToken != "" {
		var allowedUser string
		if strings.HasPrefix(channel, "U") {
			allowedUser = channel
		} else {
			allowedUser = userID
		}
		if allowedUser == "" {
			log.Fatal("CC_NOTIFY_SLACK_USER_ID is required when bot is enabled with a channel (non-DM)")
		}

		b := &bot.Bot{
			AppToken:    appToken,
			BotToken:    token,
			AllowedUser: allowedUser,
			Threads:     threads,
		}
		go func() {
			if err := b.Run(context.Background()); err != nil {
				log.Fatalf("bot error: %v", err)
			}
		}()
		log.Printf("bot started (allowed_user=%s)", allowedUser)
	}

	if *ccusageCron != "" {
		c := cron.New()
		_, err := c.AddFunc(*ccusageCron, func() {
			log.Printf("running ccusage weekly report")
			data, err := ccusage.Run()
			if err != nil {
				log.Printf("ccusage run failed: %v", err)
				return
			}
			text, err := ccusage.FormatSlackTable(data)
			if err != nil {
				log.Printf("ccusage format failed: %v", err)
				return
			}
			if _, err := slackClient.PostMessage(channel, text, ""); err != nil {
				log.Printf("ccusage slack post failed: %v", err)
			}
		})
		if err != nil {
			log.Fatalf("invalid ccusage-cron schedule: %v", err)
		}
		c.Start()
		log.Printf("ccusage cron started (schedule=%s)", *ccusageCron)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/hook", h.HandleHook)

	addr := fmt.Sprintf(":%s", *port)
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func envWithFallback(primary, fallback string) string {
	if v := os.Getenv(primary); v != "" {
		return v
	}
	return os.Getenv(fallback)
}
