package emby_api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
