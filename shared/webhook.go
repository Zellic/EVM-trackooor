package shared

import (
	"encoding/json"
	"log"
	"log/slog"
	"strings"
	"time"

	discordwebhook "evm-trackooor/discord-webhook"
)

// global webhook instance
var DiscordWebhook WebhookInstance

// functions will NOT error if webhook URL is not set

type WebhookInstance struct {
	WebhookURL           string
	Username             string
	Avatar_url           string
	RetrySendingMessages bool
}

func (w WebhookInstance) SendMessage(msg string) error {
	if w.WebhookURL == "" {
		return nil
	}

	// if message is too long, truncate it
	if len(msg) > 2000 {
		Infof(slog.Default(), "Discord message length (%v) too long, truncating", len(msg))
		msg = msg[:1980]
		msg += "... (truncated)"
	}

	hook := discordwebhook.Hook{
		Username:   w.Username,
		Avatar_url: w.Avatar_url,
		Content:    msg,
	}

	payload, err := json.Marshal(hook)
	if err != nil {
		log.Fatal(err)
	}
	err = discordwebhook.ExecuteWebhook(w.WebhookURL, payload)
	if err != nil {
		if strings.Contains(err.Error(), "rate limited") && w.RetrySendingMessages {
			Infof(slog.Default(), "Rate limited, waiting 5s and retrying sending message: %v", msg)
			time.Sleep(5 * time.Second)
			return w.SendMessage(msg)
		}
	}
	return err
}

func (w WebhookInstance) SendEmbedMessages(embeds []discordwebhook.Embed) error {
	if w.WebhookURL == "" {
		return nil
	}

	hook := discordwebhook.Hook{
		Username:   w.Username,
		Avatar_url: w.Avatar_url,
		Embeds:     embeds,
	}

	payload, err := json.Marshal(hook)
	if err != nil {
		log.Fatal(err)
	}

	err = discordwebhook.ExecuteWebhook(w.WebhookURL, payload)
	if err != nil {
		if strings.Contains(err.Error(), "rate limited") && w.RetrySendingMessages {
			Infof(slog.Default(), "Rate limited, waiting 5s and retrying sending embeds (length %v)", len(embeds))
			time.Sleep(5 * time.Second)
			return w.SendEmbedMessages(embeds)
		}
	}
	return err
}

func (w WebhookInstance) SendFile(imgData []byte, imgFilename string) error {
	if w.WebhookURL == "" {
		return nil
	}

	hook := discordwebhook.Hook{
		Username:   w.Username,
		Avatar_url: w.Avatar_url,
		Content:    "",
	}

	payload, err := json.Marshal(hook)
	if err != nil {
		log.Fatal(err)
	}

	err = discordwebhook.SendFileToWebhook(w.WebhookURL, payload, imgData, imgFilename)
	if err != nil {
		if strings.Contains(err.Error(), "rate limited") && w.RetrySendingMessages {
			Infof(slog.Default(), "Rate limited, waiting 5s and retrying sending image")
			time.Sleep(5 * time.Second)
			return w.SendFile(imgData, imgFilename)
		}
	}
	return err
}
