package notify_center

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestNewNotifyCenter(t *testing.T) {

	var gotPaths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPaths = append(gotPaths, r.URL.Path)
	}))
	defer srv.Close()

	center := NewNotifyCenter(logrus.New(), srv.URL+"/")
	center.Add("groupName", "Info哈哈")
	center.Send()
	if len(gotPaths) != 1 || gotPaths[0] != "/groupName/Info哈哈" {
		t.Fatalf("webhook got %q, want one request for /groupName/Info哈哈", gotPaths)
	}

	center.Clear()
	center.Send()
	if len(gotPaths) != 1 {
		t.Fatalf("Send() after Clear() must not send, got %q", gotPaths)
	}
}
