package mark_system

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/language"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/subparser"
)

// Subtitles are compared in this order, each step only breaking ties of the one before:
//  1. release match: how many release tags (group, web source, source type, resolution)
//     the subtitle name shares with the video name; a matching release is what keeps the
//     timing in sync. Skipped when the setting is off or the video name has no tags.
//  2. language: the preferred one of bilingual / simplified / traditional.
//  3. format: the "字幕格式下载优先级" setting (srt or ass), if set.
//  4. site: position in the configured site order.
// Non-Chinese subtitles are never picked.

type rankKey [4]int

func (a rankKey) better(b rankKey) bool {
	for i := range a {
		if a[i] != b[i] {
			return a[i] > b[i]
		}
	}
	return false
}

// pickBest returns the highest-ranked Chinese subtitle, or nil.
func pickBest(infos []subparser.FileInfo, videoPath string, subTypePriority int, prio *settings.SubPriority) *subparser.FileInfo {
	videoTags := releaseTags(filepath.Base(videoPath))
	var best *subparser.FileInfo
	var bestKey rankKey
	for i := range infos {
		lt := langTier(infos[i].Lang, prio.Language)
		if lt == 0 {
			continue
		}
		key := rankKey{0, lt, formatScore(infos[i].Ext, subTypePriority), siteScore(infos[i].FromWhereSite, prio.SiteOrder)}
		if prio.MatchRelease == nil || *prio.MatchRelease {
			key[0] = releaseScore(videoTags, releaseTags(filepath.Base(infos[i].FileFullPath)))
		}
		if best == nil || key.better(bestKey) {
			best, bestKey = &infos[i], key
		}
	}
	return best
}

// langTier: 4 = preferred ... 1 = least preferred Chinese, 0 = no Chinese.
func langTier(l language.MyLanguage, pref string) int {
	simple, trad, bilingual := false, false, false
	switch l {
	case language.ChineseSimple:
		simple = true
	case language.ChineseTraditional:
		trad = true
	case language.ChineseSimpleEnglish, language.ChineseSimpleJapanese, language.ChineseSimpleKorean:
		simple, bilingual = true, true
	case language.ChineseTraditionalEnglish, language.ChineseTraditionalJapanese, language.ChineseTraditionalKorean:
		trad, bilingual = true, true
	default:
		return 0
	}
	switch pref {
	case settings.SubLangChs: // 简 > 简双语 > 繁 > 繁双语
		return pick(simple && !bilingual, simple, trad && !bilingual)
	case settings.SubLangCht: // 繁 > 繁双语 > 简 > 简双语
		return pick(trad && !bilingual, trad, simple && !bilingual)
	default: // 简双语 > 繁双语 > 简 > 繁
		return pick(simple && bilingual, trad && bilingual, simple)
	}
}

// pick maps the first true condition to 4, 3, 2; anything else Chinese is 1.
func pick(first, second, third bool) int {
	switch {
	case first:
		return 4
	case second:
		return 3
	case third:
		return 2
	}
	return 1
}

func formatScore(ext string, subTypePriority int) int {
	ext = strings.ToLower(ext)
	switch subTypePriority {
	case 1:
		if ext == common.SubExtSRT {
			return 1
		}
	case 2:
		if ext == common.SubExtASS || ext == common.SubExtSSA {
			return 1
		}
	}
	return 0
}

func siteScore(site string, order []string) int {
	for i, s := range order {
		if s == site {
			return len(order) - i
		}
	}
	return 0
}

var (
	tagSplit     = regexp.MustCompile(`[\s._\[\]()]+`)
	resolutions  = map[string]bool{"2160p": true, "1080p": true, "1080i": true, "720p": true, "576p": true, "480p": true}
	webSources   = map[string]bool{"dsnp": true, "amzn": true, "nf": true, "atvp": true, "hmax": true, "max": true, "hulu": true, "pcok": true, "itunes": true, "cr": true, "pmtp": true, "stan": true, "bcore": true}
	sourceTypes  = map[string]string{"web-dl": "webdl", "webdl": "webdl", "webrip": "webrip", "bluray": "bluray", "blu-ray": "bluray", "bdrip": "bdrip", "brrip": "bdrip", "remux": "remux", "hdtv": "hdtv", "dvdrip": "dvdrip", "uhd": "uhd"}
	groupPattern = regexp.MustCompile(`-([A-Za-z0-9]{2,})$`)
)

type tags struct {
	group   string
	set     map[string]int // tag -> weight
	present bool
}

// releaseTags pulls release group, web source, source type and resolution out of a
// scene-style name such as "Show.S01E01.1080p.DSNP.WEB-DL.DDP5.1.H.264-FLUX".
func releaseTags(name string) tags {
	base := strings.TrimSuffix(name, filepath.Ext(name))
	// subtitle names often end in a language tag: ".zh", ".chs&eng", ".en"
	for {
		ext := strings.ToLower(filepath.Ext(base))
		if ext == "" || len(ext) > 9 || strings.ContainsAny(ext, "0123456789") {
			break
		}
		if strings.Contains("|.zh|.chs|.cht|.en|.eng|.sc|.tc|.zh-en|.chs&eng|.cht&eng|.default|.forced|", "|"+ext+"|") {
			base = strings.TrimSuffix(base, filepath.Ext(base))
			continue
		}
		break
	}
	t := tags{set: map[string]int{}}
	// a release group has letters; "-2013" or "-1" is not one
	if m := groupPattern.FindStringSubmatch(base); m != nil && strings.ContainsAny(strings.ToLower(m[1]), "abcdefghijklmnopqrstuvwxyz") {
		t.group = strings.ToLower(m[1])
	}
	for _, tok := range tagSplit.Split(strings.ToLower(base), -1) {
		if resolutions[tok] {
			t.set[tok] = 1
		}
		if webSources[tok] {
			t.set[tok] = 2
		}
		if v, ok := sourceTypes[tok]; ok {
			t.set[v] = 2
		}
		// "web-dl" survives the split; "WEB-DL-FLUX" style tails do not, so check prefixes too
		for k, v := range sourceTypes {
			if strings.HasPrefix(tok, k+"-") {
				t.set[v] = 2
			}
		}
	}
	t.present = t.group != "" || len(t.set) > 0
	return t
}

// releaseScore weighs shared tags: release group 4, web source / source type 2, resolution 1.
func releaseScore(video, sub tags) int {
	if !video.present {
		return 0
	}
	score := 0
	if video.group != "" && video.group == sub.group {
		score += 4
	}
	for k, w := range video.set {
		if _, ok := sub.set[k]; ok {
			score += w
		}
	}
	return score
}
