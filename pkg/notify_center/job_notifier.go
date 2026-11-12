package notify_center

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/local_http_proxy_server"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/sirupsen/logrus"
)

// JobEvent is one finished subtitle job, as reported to the user.
type JobEvent struct {
	Time       time.Time `json:"time"`
	Title      string    `json:"title"` // display name, e.g. "Some Show S01E02"
	VideoFPath string    `json:"video_f_path"`
	Success    bool      `json:"success"`
	Site       string    `json:"site,omitempty"`
	SubName    string    `json:"sub_name,omitempty"`
	Error      string    `json:"error,omitempty"`
}

const (
	maxPending      = 500            // events kept for one digest, the rest is only counted
	maxDigestLines  = 10             // events listed in one message
	failureCooldown = 24 * time.Hour // the same video's failure is reported once per this period
	sendTimeout     = 15 * time.Second
)

// JobNotifier collects job events and sends them as digests: the first event goes out after a
// short delay, later ones wait until DigestMinutes have passed since the last message. Add
// never does network I/O, so a slow or dead notifier cannot hold up a download.
type JobNotifier struct {
	mu       sync.Mutex
	pending  []JobEvent
	dropped  int
	timerSet bool
	lastSent time.Time
	failSeen map[string]time.Time
	log      *logrus.Logger

	unit time.Duration                                                             // what one "digest minute" lasts, shortened in tests
	cfg  func() settings.NotificationSettings                                      // read on every use, settings can change at runtime
	send func(cfg settings.NotificationSettings, title, body string, n []JobEvent) // replaced in tests
}

func NewJobNotifier() *JobNotifier {
	n := &JobNotifier{
		failSeen: make(map[string]time.Time),
		unit:     time.Minute,
		cfg: func() settings.NotificationSettings {
			if s := settings.Get().NotificationSettings; s != nil {
				return *s
			}
			return settings.NotificationSettings{}
		},
	}
	n.send = func(cfg settings.NotificationSettings, title, body string, events []JobEvent) {
		n.mu.Lock()
		log := n.log
		n.mu.Unlock()
		for channel, err := range Send(cfg, title, body, events) {
			if err != nil && log != nil {
				log.Errorln("notification", channel, "failed:", err)
			}
		}
	}
	return n
}

// Jobs is the notifier the download pipeline reports to.
var Jobs = NewJobNotifier()

// Add queues one event. It returns immediately.
func (n *JobNotifier) Add(log *logrus.Logger, ev JobEvent) {
	cfg := n.cfg()
	if !cfg.Enable || !cfg.HasChannel() || (ev.Success && !cfg.OnSuccess) || (!ev.Success && !cfg.OnFailure) {
		return
	}
	now := time.Now()
	n.mu.Lock()
	defer n.mu.Unlock()
	n.log = log
	if ev.Success {
		delete(n.failSeen, ev.VideoFPath)
	} else {
		// A video without subtitles is retried on every scan for weeks; report it once a day.
		if t, ok := n.failSeen[ev.VideoFPath]; ok && now.Sub(t) < failureCooldown {
			return
		}
		if len(n.failSeen) > 2000 {
			for k, t := range n.failSeen {
				if now.Sub(t) >= failureCooldown {
					delete(n.failSeen, k)
				}
			}
		}
		n.failSeen[ev.VideoFPath] = now
	}
	if len(n.pending) >= maxPending {
		n.dropped++
	} else {
		n.pending = append(n.pending, ev)
	}
	if n.timerSet {
		return
	}
	n.timerSet = true
	// Wait a moment even when a message is due, so events finishing together share one message.
	delay := n.unit / 12
	if wait := n.lastSent.Add(time.Duration(cfg.DigestMinutes) * n.unit).Sub(now); wait > delay {
		delay = wait
	}
	time.AfterFunc(delay, n.flush)
}

func (n *JobNotifier) flush() {
	n.mu.Lock()
	events, dropped := n.pending, n.dropped
	n.pending, n.dropped, n.timerSet = nil, 0, false
	n.lastSent = time.Now()
	n.mu.Unlock()
	if len(events) == 0 {
		return
	}
	title, body := Digest(events, dropped)
	n.send(n.cfg(), title, body, events)
}

// Digest builds the message for a batch of events. dropped is the number of events that were
// counted but not kept.
func Digest(events []JobEvent, dropped int) (title, body string) {
	okCount := 0
	lines := make([]string, 0, maxDigestLines+1)
	for i, ev := range events {
		if ev.Success {
			okCount++
		}
		if i >= maxDigestLines {
			continue
		}
		line := "[失败] " + ev.Title
		if ev.Success {
			line = "[成功] " + ev.Title
			if ev.Site != "" {
				line += " · " + ev.Site
			}
		} else if ev.Error != "" {
			line += "：" + truncate(ev.Error, 120)
		}
		lines = append(lines, line)
	}
	if more := len(events) - len(lines) + dropped; more > 0 {
		lines = append(lines, fmt.Sprintf("……另有 %d 条", more))
	}
	title = fmt.Sprintf("ChineseSubFinder 字幕下载：成功 %d，失败 %d", okCount, len(events)-okCount)
	return title, strings.Join(lines, "\n")
}

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}

// Send delivers one message to every configured channel and returns the result per channel
// ("webhook", "telegram", "bark"); channels that are not configured are left out.
func Send(cfg settings.NotificationSettings, title, body string, events []JobEvent) map[string]error {
	out := make(map[string]error)
	direct := &http.Client{Timeout: sendTimeout}
	if cfg.WebhookUrl != "" {
		okCount := 0
		for _, ev := range events {
			if ev.Success {
				okCount++
			}
		}
		if events == nil {
			events = []JobEvent{}
		}
		out["webhook"] = postJSON(direct, cfg.WebhookUrl, map[string]interface{}{
			"title": title, "body": body,
			"success": okCount, "failed": len(events) - okCount,
			"events": events,
		})
	}
	if cfg.TelegramBotToken != "" && cfg.TelegramChatId != "" {
		base := strings.TrimRight(cfg.TelegramApiBase, "/")
		if base == "" {
			base = "https://api.telegram.org"
		}
		// Telegram is usually only reachable through the proxy configured for the subtitle sites.
		client := direct
		if proxy := local_http_proxy_server.GetProxyUrl(); proxy != "" && cfg.TelegramApiBase == "" {
			if u, err := url.Parse(proxy); err == nil {
				tr := &http.Transport{Proxy: http.ProxyURL(u)}
				defer tr.CloseIdleConnections()
				client = &http.Client{Timeout: sendTimeout, Transport: tr}
			}
		}
		err := postJSON(client, base+"/bot"+cfg.TelegramBotToken+"/sendMessage", map[string]interface{}{
			"chat_id": cfg.TelegramChatId, "text": title + "\n" + body,
		})
		if err != nil {
			// the token is part of the URL and Go puts the URL into its errors
			err = fmt.Errorf("%s", strings.ReplaceAll(err.Error(), cfg.TelegramBotToken, "***"))
		}
		out["telegram"] = err
	}
	if cfg.BarkUrl != "" {
		out["bark"] = postJSON(direct, strings.TrimRight(cfg.BarkUrl, "/"), map[string]interface{}{
			"title": title, "body": body, "group": "ChineseSubFinder",
		})
	}
	return out
}

func postJSON(client *http.Client, target string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	resp, err := client.Post(target, "application/json; charset=utf-8", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	answer, _ := io.ReadAll(io.LimitReader(resp.Body, 300))
	if resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(answer)))
	}
	return nil
}
