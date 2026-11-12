package sub_helper

import (
	"path/filepath"
	"regexp"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/decode"
)

// EpisodeRef is the season/episode a subtitle file belongs to.
type EpisodeRef struct {
	Season, Episode int
}

// absEpisode resolves an episode-only file name (season 0) through the absolute-number map.
func absEpisode(absEps []map[int]EpisodeRef, season, episode int) (EpisodeRef, bool) {
	if len(absEps) < 1 || season != 0 || episode <= 0 {
		return EpisodeRef{}, false
	}
	ref, ok := absEps[0][episode]
	return ref, ok
}

var regDigits = regexp.MustCompile(`\d+`)

// namesCarryNoEpisode tells whether the subtitle files of one archive can all be taken as
// the episode the archive was downloaded for: no name has a recognisable episode, and no
// two names differ only in their digits (which would be an episode numbering this code
// does not know, e.g. "Show 01.ass" / "Show 02.ass", so better unused than misfiled).
func namesCarryNoEpisode(files []string) bool {
	seen := make(map[string]bool)
	for _, f := range files {
		name := filepath.Base(f)
		if _, _, eps, err := decode.GetSeasonAndEpisodeFromSubFileName(name); err != nil || eps > 0 {
			return false
		}
		key := regDigits.ReplaceAllString(name, "#")
		if key != name && seen[key] {
			return false
		}
		seen[key] = true
	}
	return true
}
