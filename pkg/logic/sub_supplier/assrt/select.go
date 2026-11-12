package assrt

import (
	"sort"
	"strings"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/decode"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/mark_system"
)

// preferForVideo reorders a search reply before the first Topic results are downloaded (each
// download costs a rate-limited detail call, so only one or two are fetched). The search
// returns up to 15 subtitles for every release of the title; taken in reply order, the one
// downloaded was often timed for another release, in English only, or for another episode.
// Order: Chinese before unknown before English-only, then release match with the video file,
// then the reply order. For an episode, results that name a different SxxEyy are dropped.
func preferForVideo(r *SearchSubResult, videoFPath string, isMovie bool) {
	season, episode := 0, 0
	if !isMovie {
		if nfo, err := decode.GetVideoNfoInfo4OneSeriesEpisode(videoFPath); err == nil {
			season, episode = nfo.Season, nfo.Episode
		}
	}
	subs := r.Sub.Subs
	scores := make(map[int]int, len(subs)) // by subtitle id
	kept := subs[:0]
	for _, sub := range subs {
		if episode > 0 && (isOtherEpisode(sub.Videoname, season, episode) || isOtherEpisode(sub.NativeName, season, episode)) {
			continue
		}
		score := mark_system.ReleaseScore(videoFPath, sub.Videoname)
		if s := mark_system.ReleaseScore(videoFPath, sub.NativeName); s > score {
			score = s
		}
		l := sub.Lang.Langlist
		desc := sub.Lang.Desc + sub.MLang
		switch {
		case l.Langchs || l.Langcht || l.Langdou || strings.ContainsAny(desc, "简繁双"):
			score += 1000
		case l.Langeng || strings.TrimSpace(desc) != "":
			score -= 1000 // a language is given and it is not Chinese
		}
		id := int(sub.Id)
		if id == 0 {
			id = int(sub.FileId)
		}
		scores[id] = score
		kept = append(kept, sub)
	}
	key := func(i int) int {
		if kept[i].Id != 0 {
			return scores[int(kept[i].Id)]
		}
		return scores[int(kept[i].FileId)]
	}
	sort.SliceStable(kept, func(i, j int) bool { return key(i) > key(j) })
	r.Sub.Subs = kept
}

// isOtherEpisode: the name states an SxxEyy and it is not the wanted one. Packs ("S01") and
// names without an episode pass.
func isOtherEpisode(name string, season, episode int) bool {
	_, s, e, err := decode.GetSeasonAndEpisodeFromSubFileName(name)
	return err == nil && s > 0 && e > 0 && (s != season || e != episode)
}
