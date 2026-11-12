package mark_system

import "testing"

func TestReleaseScoreForTitles(t *testing.T) {
	// a title ending in ".2160p" must keep that tag: it is not a file extension
	if got := ReleaseScore("/m/Movie.2026.2160p.WEB-DL.mkv", "简繁 | Movie.2026.2160p"); got != 1 {
		t.Errorf("resolution in a title: got %d", got)
	}
	video := "/m/Show.S01E01.1080p.DSNP.WEB-DL.DDP5.1.H.264-FLUX.mkv"
	a := ReleaseScore(video, "Show.S01E01.1080p.DSNP.WEB-DL.DDP5.1.H.264-FLUX")
	b := ReleaseScore(video, "Show.S01E01.720p.HDTV-XYZ")
	if a <= b || b != 0 {
		t.Errorf("same release (%d) must beat another release (%d)", a, b)
	}
}
