package sub_parser_hub

import (
	"os"
	"sync"
	"time"

	languageConst "github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/language"
)

// Library scans only need "is this subtitle Chinese, and which language", but finding out
// means a full parse (charset detection, regexes, per-line language detection: ~40 ms and
// ~20 MB for a 200 KB bilingual srt). Results are cached by path, size and mtime, so an
// unchanged file is parsed once per process.
// ponytail: unbounded map, one small entry per subtitle file in the library; add eviction
// if libraries ever reach millions of files.

type chineseCacheEntry struct {
	size    int64
	modTime time.Time
	found   bool
	chinese bool
	lang    languageConst.MyLanguage
}

var chineseCache sync.Map // path -> chineseCacheEntry

// SubChineseInfo reports whether filePath is a supported subtitle (found), whether it has
// Chinese, and its detected language. Errors are not cached.
func (p SubParserHub) SubChineseInfo(filePath string) (found, chinese bool, lang languageConst.MyLanguage, err error) {
	st, statErr := os.Stat(filePath)
	if statErr == nil {
		if v, ok := chineseCache.Load(filePath); ok {
			e := v.(chineseCacheEntry)
			if e.size == st.Size() && e.modTime.Equal(st.ModTime()) {
				return e.found, e.chinese, e.lang, nil
			}
		}
	}
	bFind, info, err := p.DetermineFileTypeFromFile(filePath)
	if err != nil {
		return false, false, languageConst.Unknown, err
	}
	if bFind && info != nil {
		found, lang = true, info.Lang
		chinese = p.IsSubHasChinese(info)
	}
	if statErr == nil {
		chineseCache.Store(filePath, chineseCacheEntry{size: st.Size(), modTime: st.ModTime(), found: found, chinese: chinese, lang: lang})
	}
	return found, chinese, lang, nil
}
