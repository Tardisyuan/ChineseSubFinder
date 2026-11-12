package sub_helper

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/supplier"
	"github.com/sirupsen/logrus"
)

func zipOf(t *testing.T, names ...string) []byte {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	body := []byte(strings.Repeat("1\n00:00:01,000 --> 00:00:02,000\n你好\n\n", 60)) // > 1000 bytes, smaller files are skipped
	for _, n := range names {
		f, err := w.Create(n)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = f.Write(body)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func archive(t *testing.T, site string, season, episode int, names ...string) supplier.SubInfo {
	s := supplier.NewSubInfo(site, 0, "pack.zip", 0, "http://x/"+site+names[0], 0, 0, ".zip", zipOf(t, names...))
	s.Season, s.Episode = season, episode
	return *s
}

// organized returns key -> sorted original file names (site prefix removed).
func organized(t *testing.T, subs []supplier.SubInfo, absEps ...map[int]EpisodeRef) map[string][]string {
	wd, _ := os.Getwd()
	if err := os.Chdir(t.TempDir()); err != nil { // the tmp folder lives under the cwd
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(wd) }()
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)
	got, err := OrganizeDlSubFiles(log, "Show", subs, false, absEps...)
	if err != nil {
		t.Fatal(err)
	}
	out := make(map[string][]string)
	for k, files := range got {
		for _, f := range files {
			name := filepath.Base(f)
			out[k] = append(out[k], name[strings.Index(name, "_0_")+3:])
		}
		sort.Strings(out[k])
	}
	return out
}

func same(a []string, b ...string) bool {
	return strings.Join(a, "|") == strings.Join(b, "|")
}

// A season pack is split per episode by the names inside it, whatever the naming style.
func TestOrganizeSeasonPack(t *testing.T) {
	got := organized(t, []supplier.SubInfo{
		archive(t, "subhd", 2, 0, "Show.S02E01.chs.ass", "Show.S02E02.chs.ass", "第03集.ass", "[Grp][Show][04][1080p].ass", "05.chs.ass"),
	})
	for key, want := range map[string]string{"S2E1": "Show.S02E01.chs.ass", "S2E2": "Show.S02E02.chs.ass",
		"S2E3": "第03集.ass", "S2E4": "[Grp][Show][04][1080p].ass", "S2E5": "05.chs.ass"} {
		if !same(got[key], want) {
			t.Errorf("%s: got %v want %s", key, got[key], want)
		}
	}
}

// An archive downloaded for one episode whose files do not name the episode belongs to
// that episode; it used to land in S0E0 and was never used.
func TestOrganizeSingleEpisodeArchiveWithoutEpisodeInNames(t *testing.T) {
	got := organized(t, []supplier.SubInfo{
		archive(t, "assrt", 1, 5, "Show.1080p.WEB-DL.chs.ass", "Show.1080p.WEB-DL.cht.ass"),
	})
	if !same(got["S1E5"], "Show.1080p.WEB-DL.chs.ass", "Show.1080p.WEB-DL.cht.ass") || len(got["S0E0"]) != 0 {
		t.Fatalf("got %v", got)
	}
	// names that differ only in digits are an unknown numbering: do not file them all under E5
	got = organized(t, []supplier.SubInfo{archive(t, "assrt", 1, 5, "Show part1 x.ass", "Show part2 x.ass")})
	if len(got["S1E5"]) != 0 {
		t.Fatalf("numbered files must not be taken as one episode: %v", got)
	}
	// a pack keeps its own episode numbers even when it was fetched for episode 5
	got = organized(t, []supplier.SubInfo{archive(t, "assrt", 1, 5, "Show.S01E05.ass", "Show.S01E06.ass")})
	if !same(got["S1E5"], "Show.S01E05.ass") || !same(got["S1E6"], "Show.S01E06.ass") {
		t.Fatalf("got %v", got)
	}
}

// Absolute numbering: the video is "[Grp] Show - 15.mkv" but the nfo says S02E03.
func TestOrganizeAbsoluteEpisodeNumbers(t *testing.T) {
	abs := map[int]EpisodeRef{15: {Season: 2, Episode: 3}}
	got := organized(t, []supplier.SubInfo{
		archive(t, "subhd", 2, 0, "[Grp] Show - 14 [1080p].ass", "[Grp] Show - 15 [1080p].ass", "Show.S02E15.ass"),
	}, abs)
	if !same(got["S2E3"], "[Grp] Show - 15 [1080p].ass") {
		t.Errorf("S2E3: %v", got["S2E3"])
	}
	// an explicit S02E15 is not an absolute number and stays where it is
	if !same(got["S2E15"], "Show.S02E15.ass") || !same(got["S2E14"], "[Grp] Show - 14 [1080p].ass") {
		t.Errorf("got %v", got)
	}
}
