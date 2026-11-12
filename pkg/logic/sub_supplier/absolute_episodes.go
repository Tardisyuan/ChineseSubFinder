package sub_supplier

import (
	"path/filepath"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/decode"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/sub_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/series"
)

// absoluteEpisodes maps the episode number written in a video file name to the season/episode
// its nfo gives, for the episodes whose release counts across seasons ("[Group] Title - 15"
// for S02E03). Subtitle files numbered the same way are then filed under the right episode.
func absoluteEpisodes(seriesInfo *series.SeriesInfo) map[int]sub_helper.EpisodeRef {
	out := make(map[int]sub_helper.EpisodeRef)
	for _, ep := range seriesInfo.NeedDlEpsKeyList {
		if abs := decode.AbsoluteEpisodeInName(filepath.Base(ep.FileFullPath), ep.Episode); abs > 0 {
			out[abs] = sub_helper.EpisodeRef{Season: ep.Season, Episode: ep.Episode}
		}
	}
	return out
}
