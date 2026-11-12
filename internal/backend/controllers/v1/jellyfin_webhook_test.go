package v1

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/task_queue"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
	"github.com/gin-gonic/gin"
)

func TestLocalPathFromServer(t *testing.T) {
	em := &settings.EmbySettings{
		MoviePathsMapping: map[string]string{
			"/media/movies":    "/data/movies",
			"/media/movies/4k": "/data/movies-4k/", // trailing slash and a more specific root
			"/media/same":      "",                 // empty = both sides see the same path
		},
		SeriesPathsMapping: map[string]string{"/media/tv/": "/mnt/user/tv"},
	}
	for server, want := range map[string]string{
		"/data/movies/A (2020)/A.mkv":   "/media/movies/A (2020)/A.mkv",
		"/data/movies-4k/B/B.mkv":       "/media/movies/4k/B/B.mkv",
		"/data/movies-4k":               "/media/movies/4k",
		"/data/moviesX/C.mkv":           "/data/moviesX/C.mkv", // prefix of a name is not a match
		"/mnt/user/tv/Show/S01/e.mkv":   "/media/tv/Show/S01/e.mkv",
		"/media/same/D/D.mkv":           "/media/same/D/D.mkv",
		"/elsewhere/E.mkv":              "/elsewhere/E.mkv",
		"/data/movies/电影 (2021)/电影.mkv": "/media/movies/电影 (2021)/电影.mkv",
	} {
		if got := localPathFromServer(em, server); got != want {
			t.Errorf("%s -> %s, want %s", server, got, want)
		}
		// and back again with the existing forward mapping
		if want != server && em.ServerPath(want) != server {
			t.Errorf("round trip of %s gives %s", server, em.ServerPath(want))
		}
	}
}

// What the plugin posts with "Send All Properties" (shortened), and a template that quotes numbers.
const (
	payloadMovie   = `{"ServerName":"jf","NotificationType":"ItemAdded","Name":"Some Movie","ItemId":"aaaa1111","ItemType":"Movie","Year":2020,"Video_0_Codec":"hevc"}`
	payloadEpisode = `{"NotificationType":"ItemAdded","Name":"Pilot","ItemId":"bbbb2222","ItemType":"Episode","SeriesName":"Some Show","SeasonNumber":1,"EpisodeNumber":2,"SeasonNumber00":"01"}`
	payloadQuoted  = `{"ItemType":"Episode","ItemId":"cccc3333","SeasonNumber":"3","EpisodeNumber":"14"}`
)

func parsePayload(t *testing.T, s string) jellyfinWebhookReq {
	t.Helper()
	req := jellyfinWebhookReq{}
	if err := json.Unmarshal([]byte(s), &req); err != nil {
		t.Fatal(err)
	}
	return req
}

func touch(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestJellyfinJob(t *testing.T) {
	root := t.TempDir() // CSF side; Jellyfin sees /data/movies and /data/tv
	em := &settings.EmbySettings{
		MoviePathsMapping:  map[string]string{filepath.Join(root, "movies"): "/data/movies"},
		SeriesPathsMapping: map[string]string{filepath.Join(root, "tv"): "/data/tv"},
	}
	movie := filepath.Join(root, "movies", "Some Movie (2020)", "Some Movie (2020).mkv")
	ep := filepath.Join(root, "tv", "Some Show", "Season 01", "Some Show S01E02.mkv")
	ep14 := filepath.Join(root, "tv", "Some Show", "Season 03", "Some Show S03E14.mkv")
	touch(t, movie, "x")
	touch(t, ep, "x")
	touch(t, ep14, "x")
	touch(t, filepath.Join(root, "tv", "Some Show", "tvshow.nfo"), "<tvshow><title>Some Show</title></tvshow>")
	// this episode has an nfo, and the nfo wins over the payload numbers
	touch(t, filepath.Join(root, "tv", "Some Show", "Season 01", "Some Show S01E02.nfo"),
		"<episodedetails><title>Pilot</title><season>1</season><episode>2</episode></episodedetails>")

	serverPaths := map[string]string{
		"aaaa1111": "/data/movies/Some Movie (2020)/Some Movie (2020).mkv",
		"bbbb2222": "/data/tv/Some Show/Season 01/Some Show S01E02.mkv",
		"cccc3333": "/data/tv/Some Show/Season 03/Some Show S03E14.mkv",
		"gone":     "/data/movies/Gone/Gone.mkv",
	}
	lookup := func(id string) (string, error) {
		if p, ok := serverPaths[id]; ok {
			return p, nil
		}
		return "", errors.New("item not found")
	}

	job, status, msg := jellyfinJob(parsePayload(t, payloadMovie), em, lookup)
	if job == nil || status != 200 {
		t.Fatalf("movie: status %d, %s", status, msg)
	}
	if job.VideoType != common.Movie || job.VideoFPath != movie || job.MediaServerInsideVideoID != "aaaa1111" ||
		job.TaskPriority != task_queue.HighTaskPriorityLevel {
		t.Errorf("movie job: %+v", job)
	}

	job, status, msg = jellyfinJob(parsePayload(t, payloadEpisode), em, lookup)
	if job == nil || status != 200 {
		t.Fatalf("episode: status %d, %s", status, msg)
	}
	if job.VideoType != common.Series || job.VideoFPath != ep || job.Season != 1 || job.Episode != 2 ||
		job.SeriesRootDirPath != filepath.Join(root, "tv", "Some Show") {
		t.Errorf("episode job: %+v", job)
	}

	// no episode nfo: season and episode come from the payload, quoted numbers included
	job, status, msg = jellyfinJob(parsePayload(t, payloadQuoted), em, lookup)
	if job == nil || job.Season != 3 || job.Episode != 14 || job.VideoFPath != ep14 {
		t.Fatalf("episode without nfo: status %d, %s, %+v", status, msg, job)
	}

	// Path in the payload: no lookup
	req := jellyfinWebhookReq{ItemType: "Movie", Path: serverPaths["aaaa1111"]}
	job, _, _ = jellyfinJob(req, em, func(string) (string, error) { t.Error("lookup not expected"); return "", nil })
	if job == nil || job.VideoFPath != movie {
		t.Errorf("payload with Path: %+v", job)
	}

	for name, tc := range map[string]struct {
		payload string
		status  int
	}{
		"other notification": {`{"NotificationType":"PlaybackStart","ItemType":"Movie","ItemId":"aaaa1111"}`, 200},
		"series item":        {`{"NotificationType":"ItemAdded","ItemType":"Series","ItemId":"x"}`, 200},
		"season item":        {`{"NotificationType":"ItemAdded","ItemType":"Season","ItemId":"x"}`, 200},
		"no id, no path":     {`{"ItemType":"Movie"}`, 400},
		"unknown item":       {`{"ItemType":"Movie","ItemId":"nope"}`, 502},
		"file not on disk":   {`{"ItemType":"Movie","ItemId":"gone"}`, 404},
		"episode no numbers": {`{"ItemType":"Episode","ItemId":"cccc3333"}`, 422},
		"special, no nfo":    {`{"ItemType":"Episode","ItemId":"cccc3333","SeasonNumber":0,"EpisodeNumber":1}`, 422},
		"episode no tvshow":  {`{"ItemType":"Episode","Path":"/data/movies/Some Movie (2020)/Some Movie (2020).mkv","EpisodeNumber":1}`, 422},
	} {
		job, status, msg = jellyfinJob(parsePayload(t, tc.payload), em, lookup)
		if job != nil || status != tc.status {
			t.Errorf("%s: job %v, status %d (%s), want no job and %d", name, job != nil, status, msg, tc.status)
		}
	}
}

func TestApiKeyOk(t *testing.T) {
	settings.SetConfigRootPath(t.TempDir())
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/hook", func(c *gin.Context) {
		if apiKeyOk(c) {
			c.Status(http.StatusOK)
		} else {
			c.Status(http.StatusUnauthorized)
		}
	})
	post := func(target, auth string) int {
		req := httptest.NewRequest("POST", target, nil)
		if auth != "" {
			req.Header.Set("Authorization", auth)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}
	key := &settings.Get().ExperimentalFunction.ApiKeySettings
	if post("/hook?api_key=", "") != 401 || post("/hook", "Bearer x") != 401 {
		t.Error("with the API key disabled nothing may pass")
	}
	key.Enabled = true
	if post("/hook?api_key=", "") != 401 || post("/hook", "") != 401 {
		t.Error("an empty key must not match an empty setting")
	}
	key.Key = "key123"
	for target, want := range map[string]int{"/hook?api_key=key123": 200, "/hook?api_key=key12": 401, "/hook": 401} {
		if got := post(target, ""); got != want {
			t.Errorf("%s: %d, want %d", target, got, want)
		}
	}
	if post("/hook", "Bearer key123") != 200 || post("/hook?api_key=key123", "Bearer wrong") != 401 {
		t.Error("header key: good must pass, and a wrong header is not rescued by the query")
	}
}
