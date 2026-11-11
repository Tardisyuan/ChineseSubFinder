package mark_system

import (
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/language"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/subparser"
)

func sub(site, name string, lang language.MyLanguage) subparser.FileInfo {
	ext := name[len(name)-4:]
	return subparser.FileInfo{FromWhereSite: site, FileFullPath: "/tmp/[" + site + "]_0_" + name, Lang: lang, Ext: ext}
}

func prio(lang string, match bool, order ...string) *settings.SubPriority {
	p := &settings.SubPriority{SiteOrder: order, Language: lang, MatchRelease: &match}
	p.Check()
	return p
}

func TestPickBest(t *testing.T) {
	video := "/media/tv/VisionQuest.S01E01.No.Strings.1080p.DSNP.WEB-DL.DDP5.1.Atmos.H.264-FLUX.mkv"
	other := sub("assrt", "visionquest.s01e01.1080p.web.h264-cakes.chs&eng.ass", language.ChineseSimpleEnglish)
	same := sub("subhd", "VisionQuest.S01E01.No.Strings.1080p.DSNP.WEB-DL.DDP5.1.Atmos.H.264-FLUX.zh.srt", language.ChineseSimple)
	english := sub("subhd", "VisionQuest.S01E01.1080p.DSNP.WEB-DL.H.264-FLUX.en.srt", language.English)
	cases := []struct {
		name string
		subs []subparser.FileInfo
		p    *settings.SubPriority
		vid  string
		want string
	}{
		{"matching release beats language preference", []subparser.FileInfo{other, same, english}, prio("bilingual", true), video, same.FileFullPath},
		{"release match off: bilingual preferred", []subparser.FileInfo{other, same}, prio("bilingual", false), video, other.FileFullPath},
		{"no tags in video name: language decides", []subparser.FileInfo{other, same}, prio("chs", true), "/media/tv/幻视任务 - S01E01 - 第 1 集.mp4", same.FileFullPath},
		{"site order breaks ties", []subparser.FileInfo{
			sub("subhd", "a.ass", language.ChineseSimple), sub("assrt", "b.ass", language.ChineseSimple),
		}, prio("chs", true, "assrt", "subhd"), "/x/a.mkv", "/tmp/[assrt]_0_b.ass"},
		{"english only is never picked", []subparser.FileInfo{english}, prio("bilingual", true), video, ""},
	}
	for _, c := range cases {
		got := pickBest(c.subs, c.vid, 0, c.p)
		gotPath := ""
		if got != nil {
			gotPath = got.FileFullPath
		}
		if gotPath != c.want {
			t.Errorf("%s: got %q want %q", c.name, gotPath, c.want)
		}
	}
}

func TestLangTier(t *testing.T) {
	if !(langTier(language.ChineseTraditional, settings.SubLangCht) > langTier(language.ChineseSimpleEnglish, settings.SubLangCht)) {
		t.Fatal("cht preference should put 繁 above 简英")
	}
	if langTier(language.English, settings.SubLangBilingual) != 0 {
		t.Fatal("english must be 0")
	}
}

func TestSubPriorityCheck(t *testing.T) {
	p := &settings.SubPriority{SiteOrder: []string{"xunlei", "a4k", "xunlei"}}
	p.Check()
	want := []string{"xunlei", "assrt", "shooter"}
	for i := range want {
		if p.SiteOrder[i] != want[i] {
			t.Fatalf("order %v", p.SiteOrder)
		}
	}
	if p.Language != settings.SubLangBilingual || p.MatchRelease == nil || !*p.MatchRelease {
		t.Fatalf("defaults %+v", p)
	}
}
