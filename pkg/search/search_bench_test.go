package search

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/sirupsen/logrus"
)

// makeFakeMovieLib builds n movie folders, each with a sparse .mkv, an .nfo, a poster and a .srt.
func makeFakeMovieLib(tb testing.TB, n int) string {
	root := tb.TempDir()
	base := time.Now().Add(-time.Hour * 24 * 365)
	for i := 0; i < n; i++ {
		dir := filepath.Join(root, fmt.Sprintf("Movie %04d (2020)", i))
		if err := os.MkdirAll(dir, 0755); err != nil {
			tb.Fatal(err)
		}
		video := filepath.Join(dir, fmt.Sprintf("Movie %04d (2020).mkv", i))
		f, err := os.Create(video)
		if err != nil {
			tb.Fatal(err)
		}
		_ = f.Truncate(2000) // SkipFileInfo drops files < 1000 bytes
		_ = f.Close()
		// pseudo-random but deterministic mod times
		mt := base.Add(time.Duration((i*7919)%n) * time.Minute)
		_ = os.Chtimes(video, mt, mt)
		for _, name := range []string{"movie.nfo", "poster.jpg", fmt.Sprintf("Movie %04d (2020).chinese(简,subhd).default.srt", i)} {
			_ = ioutil.WriteFile(filepath.Join(dir, name), []byte("x"), 0644)
		}
	}
	return root
}

func quietLogger() *logrus.Logger {
	l := logrus.New()
	l.SetOutput(ioutil.Discard)
	return l
}

// BenchmarkMatchedVideoFileFromDirs measures a full movie-library scan (walk + newest-first sort)
// over 3000 local fake movies.
func BenchmarkMatchedVideoFileFromDirs(b *testing.B) {
	settings.SetConfigRootPath(b.TempDir())
	root := makeFakeMovieLib(b, 3000)
	l := quietLogger()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m, err := MatchedVideoFileFromDirs(l, []string{root})
		if err != nil {
			b.Fatal(err)
		}
		v, _ := m.Get(root)
		if len(v.([]string)) != 3000 {
			b.Fatalf("got %d files", len(v.([]string)))
		}
	}
}
