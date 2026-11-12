package emby_api

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/sirupsen/logrus"
)

// fakeServer records the last request and answers the user-list endpoints the way
// Emby (/emby/Users/Query -> {Items}) and Jellyfin (/Users -> [...]) do.
func fakeServer(t *testing.T, last *http.Request) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*last = *r
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/emby/Users/Query":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"Items": []map[string]string{{"Name": "a", "Id": "u1"}}, "TotalRecordCount": 1})
		case "/Users":
			_ = json.NewEncoder(w).Encode([]map[string]string{{"Name": "a", "Id": "u1"}, {"Name": "b", "Id": "u2"}})
		default:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"Items": []interface{}{}})
		}
	}))
}

func TestServerTypeRouting(t *testing.T) {
	var last http.Request
	srv := fakeServer(t, &last)
	defer srv.Close()
	em := &EmbyApi{log: logrus.New(), timeOut: 5 * time.Second}

	emby := &settings.EmbySettings{AddressUrl: srv.URL + "/", APIKey: "k1"}
	users, err := em.GetUserIdList(emby)
	if err != nil || len(users.Items) != 1 {
		t.Fatalf("emby users: %+v %v", users, err)
	}
	if last.URL.Path != "/emby/Users/Query" || last.URL.Query().Get("api_key") != "k1" {
		t.Fatalf("emby request: %s %s", last.URL.Path, last.URL.RawQuery)
	}

	jf := &settings.EmbySettings{AddressUrl: srv.URL, APIKey: "k2", ServerType: settings.MediaServerJellyfin}
	users, err = em.GetUserIdList(jf)
	if err != nil || len(users.Items) != 2 || users.Items[1].Id != "u2" {
		t.Fatalf("jellyfin users: %+v %v", users, err)
	}
	if last.URL.Path != "/Users" || last.URL.Query().Get("api_key") != "" ||
		last.Header.Get("Authorization") != `MediaBrowser Token="k2"` {
		t.Fatalf("jellyfin request: %s %s %q", last.URL.Path, last.URL.RawQuery, last.Header.Get("Authorization"))
	}

	if _, err = em.GetRecentItemsByUserID(jf, "u1", 10); err != nil {
		t.Fatal(err)
	}
	if last.URL.Path != "/Items" || last.URL.Query().Get("userId") != "u1" || last.URL.Query().Get("Limit") != "10" {
		t.Fatalf("jellyfin user items: %s %s", last.URL.Path, last.URL.RawQuery)
	}

	if _, err = em.GetItemVideoInfoByUserId(emby, "u1", "v9"); err != nil {
		t.Fatal(err)
	}
	if last.URL.Path != "/emby/Users/u1/Items/v9" {
		t.Fatalf("emby user item: %s", last.URL.Path)
	}

	if _, err = em.GetItemVideoInfo(jf, "v9"); err != nil {
		t.Fatal(err)
	}
	if last.URL.Path != "/Items/v9" {
		t.Fatalf("jellyfin item: %s", last.URL.Path)
	}
}

func TestNotifyPathUpdated(t *testing.T) {
	var gotPath, gotAuth string
	var body struct {
		Updates []struct{ Path, UpdateType string }
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotAuth = r.URL.Path, r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	em := &EmbyApi{log: logrus.New(), timeOut: 5 * time.Second}

	jf := &settings.EmbySettings{
		AddressUrl: srv.URL, APIKey: "k", ServerType: settings.MediaServerJellyfin,
		SeriesPathsMapping: map[string]string{"/media/tv show": "/data/tv", "/media": ""},
	}
	dir := jf.ServerPath("/media/tv show/欧美剧/幻视任务 (2026)/Season 1")
	if dir != "/data/tv/欧美剧/幻视任务 (2026)/Season 1" {
		t.Fatalf("mapped %q", dir)
	}
	if p := jf.ServerPath("/media/movie/x"); p != "/media/movie/x" {
		t.Fatalf("empty mapping should keep the path, got %q", p)
	}
	if err := em.NotifyPathUpdated(jf, dir); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/Library/Media/Updated" || gotAuth != `MediaBrowser Token="k"` ||
		len(body.Updates) != 1 || body.Updates[0].Path != dir || body.Updates[0].UpdateType != "Modified" {
		t.Fatalf("request %s %q %+v", gotPath, gotAuth, body)
	}
}

// A refused refresh (bad key, item gone) must come back as an error, not as "refreshed".
func TestUpdateVideoSubListReportsHttpError(t *testing.T) {
	status := http.StatusNotFound
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
	}))
	defer srv.Close()
	em := &EmbyApi{log: logrus.New(), timeOut: 5 * time.Second}
	conf := &settings.EmbySettings{AddressUrl: srv.URL, APIKey: "k"}

	if err := em.UpdateVideoSubList(conf, "gone"); err == nil {
		t.Fatal("HTTP 404 was reported as success")
	}
	status = http.StatusNoContent
	if err := em.UpdateVideoSubList(conf, "v1"); err != nil {
		t.Fatal(err)
	}
}

// With SkipWatched, a video watched by any user is skipped, including the first user.
func TestSkipWatchedAnyUser(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/Users":
			_ = json.NewEncoder(w).Encode([]map[string]string{{"Name": "a", "Id": "u1"}, {"Name": "b", "Id": "u2"}})
		case r.URL.Path == "/Items" && r.URL.Query().Get("userId") == "u1":
			_, _ = w.Write([]byte(`{"Items":[{"Id":"A","Name":"A","UserData":{"Played":true}},{"Id":"B","Name":"B","UserData":{"Played":false}}]}`))
		case r.URL.Path == "/Items" && r.URL.Query().Get("userId") == "u2":
			_, _ = w.Write([]byte(`{"Items":[{"Id":"A","Name":"A","UserData":{"Played":false}},{"Id":"B","Name":"B","UserData":{"Played":true}},{"Id":"C","Name":"C","UserData":{"Played":false}}]}`))
		default:
			_, _ = w.Write([]byte(`{"Items":[]}`))
		}
	}))
	defer srv.Close()
	em := &EmbyApi{log: logrus.New(), timeOut: 5 * time.Second}
	jf := &settings.EmbySettings{AddressUrl: srv.URL, APIKey: "k", ServerType: settings.MediaServerJellyfin}
	items, err := em.GetRecentlyItems(jf, true, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(items.Items) != 1 || items.Items[0].Id != "C" {
		t.Fatalf("want only C, got %+v", items.Items)
	}
}

// Consecutive Emby/Jellyfin calls reuse one connection instead of dialing each time.
func TestEmbyRequestsReuseConnection(t *testing.T) {
	var conns int32
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"Name":"a","Id":"u1"}]`))
	}))
	srv.Config.ConnState = func(_ net.Conn, s http.ConnState) {
		if s == http.StateNew {
			atomic.AddInt32(&conns, 1)
		}
	}
	srv.Start()
	defer srv.Close()
	em := &EmbyApi{log: logrus.New(), timeOut: 5 * time.Second}
	jf := &settings.EmbySettings{AddressUrl: srv.URL, APIKey: "k", ServerType: settings.MediaServerJellyfin}
	for i := 0; i < 5; i++ {
		if _, err := em.GetUserIdList(jf); err != nil {
			t.Fatal(err)
		}
	}
	if n := atomic.LoadInt32(&conns); n != 1 {
		t.Fatalf("5 requests opened %d connections, want 1", n)
	}
}
