package mix_media_info

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMediaInfoFromNfo(t *testing.T) {
	dir := t.TempDir()
	movie := filepath.Join(dir, "Jodorowskys.Dune.2013.mkv")
	if err := os.WriteFile(movie, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	nfo := `<?xml version="1.0" encoding="utf-8"?><movie><title>佐杜洛夫斯基的沙丘</title>` +
		`<originaltitle>Jodorowsky's Dune</originaltitle><year>2013</year><imdbid>tt1935156</imdbid></movie>`
	if err := os.WriteFile(filepath.Join(dir, "movie.nfo"), []byte(nfo), 0644); err != nil {
		t.Fatal(err)
	}
	mi, err := mediaInfoFromNfo(movie, true)
	if err != nil {
		t.Fatal(err)
	}
	if mi.TitleCn != "佐杜洛夫斯基的沙丘" || mi.TitleEn != "Jodorowsky's Dune" || mi.ImdbId != "tt1935156" || mi.Year != "2013" {
		t.Fatalf("got %+v", mi)
	}

	series := filepath.Join(dir, "show", "Season 1")
	if err := os.MkdirAll(series, 0755); err != nil {
		t.Fatal(err)
	}
	ep := filepath.Join(series, "幻视任务 - S01E01 - 第 1 集.mkv")
	_ = os.WriteFile(ep, []byte("x"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "show", "tvshow.nfo"),
		[]byte(`<tvshow><title>幻视任务</title><originaltitle>VisionQuest</originaltitle><year>2026</year></tvshow>`), 0644)
	mi, err = mediaInfoFromNfo(ep, false)
	if err != nil {
		t.Fatal(err)
	}
	if mi.TitleCn != "幻视任务" || mi.TitleEn != "VisionQuest" {
		t.Fatalf("series got %+v", mi)
	}
}
