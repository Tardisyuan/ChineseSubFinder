package marktest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/sub_helper"
	"github.com/sirupsen/logrus"
)

// Removing .default from one video's subtitles must not touch a neighbour whose name only
// starts with the same text (Alien vs Aliens).
func TestRemoveExtMarkOnlyOwnSubtitles(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"Alien.mkv", "Alien.chs.default.ass", "Aliens.mkv", "Aliens.chs.default.ass"} {
		if err := os.WriteFile(filepath.Join(dir, n), make([]byte, 2048), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := sub_helper.SearchVideoMatchSubFileAndRemoveExtMark(logrus.New(), filepath.Join(dir, "Alien.mkv")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "Alien.chs.ass")); err != nil {
		t.Errorf("own subtitle not renamed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "Aliens.chs.default.ass")); err != nil {
		t.Errorf("neighbour's subtitle was changed: %v", err)
	}
}
