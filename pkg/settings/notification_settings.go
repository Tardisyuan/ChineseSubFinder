package settings

// NotificationSettings configures the messages sent when a subtitle job finishes.
// Everything is off by default; a config file without this section loads with these defaults.
type NotificationSettings struct {
	Enable           bool   `json:"enable"`
	OnSuccess        bool   `json:"on_success"`         // a subtitle was downloaded
	OnFailure        bool   `json:"on_failure"`         // a job failed or found no subtitle
	DigestMinutes    int    `json:"digest_minutes"`     // at most one message per this many minutes, 0 = no batching
	WebhookUrl       string `json:"webhook_url"`        // generic webhook, POST JSON
	TelegramBotToken string `json:"telegram_bot_token"` // Telegram bot preset
	TelegramChatId   string `json:"telegram_chat_id"`
	TelegramApiBase  string `json:"telegram_api_base"` // empty = https://api.telegram.org, for a reverse proxy
	BarkUrl          string `json:"bark_url"`          // Bark preset: https://api.day.app/<device key>
}

func NewNotificationSettings() *NotificationSettings {
	return &NotificationSettings{OnSuccess: true, OnFailure: true, DigestMinutes: 5}
}

// HasChannel reports whether at least one destination is filled in.
func (n NotificationSettings) HasChannel() bool {
	return n.WebhookUrl != "" || (n.TelegramBotToken != "" && n.TelegramChatId != "") || n.BarkUrl != ""
}
