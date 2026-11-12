package decode

import "testing"

// Names as they come out of subtitle archives. The ones without "." or a space around
// SxxEyy used to parse as season 0 episode 0 with no error.
func TestGetSeasonAndEpisodeFromSubFileNameLoose(t *testing.T) {
	cases := []struct {
		name            string
		full            bool
		season, episode int
	}{
		{"Show.S01E02.1080p.chs.ass", false, 1, 2},
		{"S01E02.chs.ass", false, 1, 2},
		{"Show.S01E02-GRP.chs.ass", false, 1, 2},
		{"Show_S01E02_chs.ass", false, 1, 2},
		{"[site]_1_show.s03e11.ass", false, 3, 11},
		{"简繁 | Show S02E05", false, 2, 5},
		{"Show.S02.1080p.chs.ass", true, 2, 0},
		{"Movie.DTS5E1.2019.ass", false, 0, 0},
		{"Show.2019.chs.ass", false, 0, 0},
	}
	for _, c := range cases {
		full, season, episode, err := GetSeasonAndEpisodeFromSubFileName(c.name)
		if err != nil || full != c.full || season != c.season || episode != c.episode {
			t.Errorf("%s: got full=%v S%dE%d err=%v, want full=%v S%dE%d", c.name, full, season, episode, err, c.full, c.season, c.episode)
		}
	}
}

func TestEpisodeOnlyNames(t *testing.T) {
	cases := []struct {
		name       string
		season, ep int
	}{
		{"幻视任务 第02集.chs.ass", 0, 2},
		{"幻视任务 第2季 第03集.ass", 2, 3},
		{"[SweetSub] VisionQuest - 02 [1080p].ass", 0, 2},
		{"VisionQuest.EP02.chs.srt", 0, 2},
		{"VisionQuest.E02.chs.srt", 0, 2},
		{"VisionQuest.S02.E05.srt", 2, 5},
		{"VisionQuest.1x02.srt", 1, 2},
		{"幻视任务.S01E02.chs.ass", 1, 2},
		{"[Group][Title][05][1080p][CHS].ass", 0, 5},
		{"【字幕组】【标题】【05】【720P】.ass", 0, 5},
		{"[Group] Title [12v2][HEVC].ass", 0, 12},
		{"05.chs.ass", 0, 5},
		{"[Group] Title - 15v2 [1080p].ass", 0, 15},
	}
	for _, c := range cases {
		_, s, e, err := GetSeasonAndEpisodeFromSubFileName(c.name)
		if err != nil || s != c.season || e != c.ep {
			t.Errorf("%s: got S%d E%d err=%v, want S%d E%d", c.name, s, e, err, c.season, c.ep)
		}
	}
	// must not turn resolutions, years or plain titles into episodes
	for _, n := range []string{"Jodorowskys.Dune.2013.1080p.BluRay.x264.srt", "Movie - 2013 [1080p].ass", "Some.Movie.chs.ass", "[Group][Movie][720][x264].ass", "2013.chs.ass"} {
		if _, s, e, _ := GetSeasonAndEpisodeFromSubFileName(n); e != 0 {
			t.Errorf("%s: false episode S%d E%d", n, s, e)
		}
	}
}

func TestAbsoluteEpisodeInName(t *testing.T) {
	cases := []struct {
		name    string
		nfoEp   int
		wantAbs int
	}{
		{"[Group] Title - 15 [1080p].mkv", 3, 15},
		{"[Group] Title - 03 [1080p].mkv", 3, 0},
		{"Title - S02E03 - Name.mkv", 3, 0},
		{"Title.mkv", 3, 0},
	}
	for _, c := range cases {
		if got := AbsoluteEpisodeInName(c.name, c.nfoEp); got != c.wantAbs {
			t.Errorf("%s: got %d want %d", c.name, got, c.wantAbs)
		}
	}
}
