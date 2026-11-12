package assrt

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func ids(r *SearchSubResult) string {
	out := ""
	for _, s := range r.Sub.Subs {
		id := s.Id
		if id == 0 {
			id = s.FileId
		}
		out += fmt.Sprintf("%d ", int(id))
	}
	return out
}

func reply(t *testing.T, subs string) *SearchSubResult {
	var r SearchSubResult
	if err := json.Unmarshal([]byte(`{"status":0,"sub":{"action":"search","result":"succeed","subs":[`+subs+`]}}`), &r); err != nil {
		t.Fatal(err)
	}
	return &r
}

func TestPreferForVideoMovie(t *testing.T) {
	r := reply(t, `
		{"id":1,"videoname":"Movie.2026.720p.HDTV-AAA","lang":{"desc":"简","langlist":{"langchs":true}}},
		{"id":2,"videoname":"Movie.2026.1080p.AMZN.WEB-DL-FLUX","lang":{"desc":"英","langlist":{"langeng":true}}},
		{"id":3,"native_name":"Movie.2026.1080p.AMZN.WEB-DL-FLUX","lang":{"desc":"简 英 双语","langlist":{"langchs":true,"langeng":true,"langdou":true}}},
		{"fileid":"4","native_name":"Movie 2026","m_lang":"英"},
		{"fileid":"5","native_name":"Movie.2026.1080p.WEB-DL-OTHER","m_lang":"英&nbsp;简&nbsp;繁&nbsp;双语"},
		{"id":6,"native_name":"Movie 2026 no language given"}`)
	preferForVideo(r, "/m/Movie.2026.1080p.AMZN.WEB-DL.DDP5.1-FLUX.mkv", true)
	// same release in Chinese, other Chinese by release match, unknown language, English only
	if got := ids(r); got != "3 5 1 6 2 4 " {
		t.Fatalf("got %s", got)
	}
}

func TestPreferForVideoEpisode(t *testing.T) {
	dir := t.TempDir()
	video := filepath.Join(dir, "Show - S01E02 - Name.1080p.WEB-DL-FLUX.mkv")
	nfo := `<?xml version="1.0"?><episodedetails><season>1</season><episode>2</episode></episodedetails>`
	if err := os.WriteFile(filepath.Join(dir, "Show - S01E02 - Name.1080p.WEB-DL-FLUX.nfo"), []byte(nfo), 0o644); err != nil {
		t.Fatal(err)
	}
	r := reply(t, `
		{"id":1,"native_name":"Show.S01E03.1080p.WEB-DL-FLUX","lang":{"langlist":{"langchs":true}}},
		{"id":2,"native_name":"Show.S01E02.720p.HDTV-AAA","lang":{"langlist":{"langchs":true}}},
		{"id":3,"native_name":"Show.S01.1080p.BluRay 全集","lang":{"langlist":{"langchs":true}}},
		{"id":4,"native_name":"Show.S01E02.1080p.WEB-DL-FLUX","lang":{"langlist":{"langchs":true}}}`)
	preferForVideo(r, video, false)
	// S01E03 is gone; the matching release comes first; the pack stays
	if got := ids(r); got != "4 2 3 " && got != "4 3 2 " {
		t.Fatalf("got %s", got)
	}
}
