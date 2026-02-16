package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/nktks/cc-slack-notifier/internal/server"
	"github.com/nktks/cc-slack-notifier/internal/slack"
)

func main() {
	port := flag.String("port", "19999", "server listen port")
	flag.Parse()

	token := envWithFallback("CC_NOTIFY_SLACK_TOKEN", "SLACK_TOKEN")
	channel := envWithFallback("CC_NOTIFY_SLACK_CHANNEL", "SLACK_CHANNEL")
	if token == "" || channel == "" {
		log.Fatal("CC_NOTIFY_SLACK_TOKEN and CC_NOTIFY_SLACK_CHANNEL must be set")
	}

	mentionUserID := os.Getenv("CC_NOTIFY_SLACK_MENTION_USER_ID")

	threads := server.NewThreadStore()
	go func() {
		for {
			time.Sleep(1 * time.Hour)
			threads.CleanOlderThan(30 * 24 * time.Hour)
		}
	}()

	h := &server.Handler{
		Slack:         slack.New(token),
		Channel:       channel,
		MentionUserID: mentionUserID,
		Threads:       threads,
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
