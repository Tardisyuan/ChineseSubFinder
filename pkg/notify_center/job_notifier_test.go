package notify_center

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
)

type sent struct {
	title, body string
	events      int
}

func testNotifier(cfg settings.NotificationSettings) (*JobNotifier, func() []sent) {
	var mu sync.Mutex
	var got []sent
	n := NewJobNotifier()
	n.unit = 60 * time.Millisecond // one "minute"
	n.cfg = func() settings.NotificationSettings { return cfg }
	n.send = func(_ settings.NotificationSettings, title, body string, events []JobEvent) {
		mu.Lock()
		defer mu.Unlock()
		got = append(got, sent{title, body, len(events)})
	}
	return n, func() []sent {
		mu.Lock()
		defer mu.Unlock()
		return append([]sent(nil), got...)
	}
}

func ev(path string, ok bool) JobEvent {
	return JobEvent{Title: path, VideoFPath: path, Success: ok, Site: "zimuku", Error: "No Sub Found"}
}

func TestBatching(t *testing.T) {
	cfg := settings.NotificationSettings{Enable: true, OnSuccess: true, OnFailure: true, DigestMinutes: 5, WebhookUrl: "x"}
	n, got := testNotifier(cfg)

	// a big scan: 300 jobs finish within the first "minutes"
	n.Add(nil, ev("/m/0", false))
	time.Sleep(30 * time.Millisecond) // the first message leaves on its own
	start := time.Now()
	for i := 1; i < 300; i++ {
		n.Add(nil, ev(fmt.Sprintf("/m/%d", i), i%3 != 0))
	}
	if d := time.Since(start); d > 50*time.Millisecond {
		t.Errorf("Add must not block, 299 calls took %v", d)
	}
	time.Sleep(500 * time.Millisecond) // > 5 "minutes"
	msgs := got()
	if len(msgs) != 2 {
		t.Fatalf("300 events should give 2 messages (first + digest), got %d: %+v", len(msgs), msgs)
	}
	if msgs[0].events != 1 || msgs[1].events != 299 {
		t.Errorf("events per message: %d, %d", msgs[0].events, msgs[1].events)
	}
	if !strings.Contains(msgs[1].title, "成功 200，失败 99") {
		t.Errorf("digest title: %q", msgs[1].title)
	}
	if lines := strings.Split(msgs[1].body, "\n"); len(lines) != maxDigestLines+1 || lines[maxDigestLines] != "……另有 289 条" {
		t.Errorf("digest body should list %d events and a rest line, got:\n%s", maxDigestLines, msgs[1].body)
	}
}

func TestPendingCapAndFailureCooldown(t *testing.T) {
	cfg := settings.NotificationSettings{Enable: true, OnSuccess: true, OnFailure: true, DigestMinutes: 1, WebhookUrl: "x"}
	n, got := testNotifier(cfg)
	// the same failing video is reported once, however often it is retried
	for i := 0; i < 5; i++ {
		n.Add(nil, ev("/missing.mkv", false))
	}
	for i := 0; i < maxPending+50; i++ {
		n.Add(nil, ev(fmt.Sprintf("/ok/%d", i), true))
	}
	time.Sleep(150 * time.Millisecond)
	msgs := got()
	if len(msgs) != 1 || msgs[0].events != maxPending {
		t.Fatalf("want one message with %d kept events, got %+v", maxPending, msgs)
	}
	if !strings.Contains(msgs[0].title, "失败 1") {
		t.Errorf("the repeated failure should count once: %q", msgs[0].title)
	}
	if !strings.Contains(msgs[0].body, "……另有 541 条") { // 500 kept - 10 listed + 51 not kept
		t.Errorf("rest line wrong:\n%s", msgs[0].body)
	}
}

func TestSwitches(t *testing.T) {
	for name, cfg := range map[string]settings.NotificationSettings{
		"disabled":   {OnSuccess: true, OnFailure: true, WebhookUrl: "x"},
		"no channel": {Enable: true, OnSuccess: true, OnFailure: true},
		"all off":    {Enable: true, WebhookUrl: "x"},
	} {
		n, got := testNotifier(cfg)
		n.Add(nil, ev("/a", true))
		n.Add(nil, ev("/b", false))
		time.Sleep(40 * time.Millisecond)
		if len(got()) != 0 {
			t.Errorf("%s: nothing should be sent, got %+v", name, got())
		}
	}
}

func TestSendChannels(t *testing.T) {
	var mu sync.Mutex
	bodies := map[string]map[string]interface{}{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		var v map[string]interface{}
		_ = json.Unmarshal(data, &v)
		mu.Lock()
		bodies[r.Method+" "+r.URL.Path] = v
		mu.Unlock()
		if r.URL.Path == "/fail" {
			http.Error(w, "nope", http.StatusForbidden)
		}
	}))
	defer srv.Close()

	events := []JobEvent{ev("/a.mkv", true), ev("/b.mkv", false)}
	title, body := Digest(events, 0)
	res := Send(settings.NotificationSettings{
		WebhookUrl:       srv.URL + "/hook",
		TelegramBotToken: "123:ABC", TelegramChatId: "42", TelegramApiBase: srv.URL,
		BarkUrl: srv.URL + "/devicekey/",
	}, title, body, events)
	for _, c := range []string{"webhook", "telegram", "bark"} {
		if err, ok := res[c]; !ok || err != nil {
			t.Errorf("%s: ok=%v err=%v", c, ok, err)
		}
	}
	if h := bodies["POST /hook"]; h["success"] != float64(1) || h["failed"] != float64(1) || len(h["events"].([]interface{})) != 2 {
		t.Errorf("webhook payload: %v", h)
	}
	if tg := bodies["POST /bot123:ABC/sendMessage"]; tg["chat_id"] != "42" || !strings.Contains(tg["text"].(string), "[成功] /a.mkv · zimuku") {
		t.Errorf("telegram payload: %v", tg)
	}
	if b := bodies["POST /devicekey"]; b["title"] != title || b["body"] != body {
		t.Errorf("bark payload: %v", b)
	}

	// errors come back per channel, and never contain the bot token
	res = Send(settings.NotificationSettings{
		WebhookUrl:       srv.URL + "/fail",
		TelegramBotToken: "123:SECRET", TelegramChatId: "42", TelegramApiBase: "http://127.0.0.1:1",
	}, "t", "b", nil)
	if res["webhook"] == nil || !strings.Contains(res["webhook"].Error(), "403") {
		t.Errorf("webhook error: %v", res["webhook"])
	}
	if res["telegram"] == nil || strings.Contains(res["telegram"].Error(), "SECRET") {
		t.Errorf("telegram error should be set and redacted: %v", res["telegram"])
	}
	if _, ok := res["bark"]; ok {
		t.Error("bark is not configured and must be left out")
	}
}
