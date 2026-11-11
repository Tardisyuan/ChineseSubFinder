package settings

import (
	"strings"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
)

type EmbySettings struct {
	Enable                bool              `json:"enable"`                   // 是否启用
	ServerType            string            `json:"server_type"`              // emby（默认）或 jellyfin
	AddressUrl            string            `json:"address_url"`              // 内网服务器的 url
	APIKey                string            `json:"api_key"`                  // API key
	MaxRequestVideoNumber int               `json:"max_request_video_number"` // 最大请求获取视频的数量
	SkipWatched           bool              `json:"skip_watched"`             // 是否跳过已经观看的
	MoviePathsMapping     map[string]string `json:"movie_paths_mapping"`      // 电影目录的映射，一旦 common setting 的目录修改，需要提示用户确认映射
	SeriesPathsMapping    map[string]string `json:"series_paths_mapping"`     // 连续剧目录的映射，一旦 common setting 的目录修改，需要提示用户确认映射
	AutoOrManual          bool              `json:"auto_or_manual"`           // 自动或手动模式，自动 IMDB ID 匹配，还是使用手动目录
	Threads               int               `json:"threads"`                  // 同时扫描的并发数
}

func NewEmbySettings() *EmbySettings {
	return &EmbySettings{
		MaxRequestVideoNumber: 500,
		MoviePathsMapping:     make(map[string]string, 0),
		SeriesPathsMapping:    make(map[string]string, 0),
		Threads:               4,
		AutoOrManual:          true,
	}
}

const (
	MediaServerEmby     = "emby"
	MediaServerJellyfin = "jellyfin"
)

// ServerName is "Jellyfin" or "Emby", for log and UI text.
func (e *EmbySettings) ServerName() string {
	if e.IsJellyfin() {
		return "Jellyfin"
	}
	return "Emby"
}

// ServerPath maps a path as CSF sees it to the path the media server sees, using the
// longest matching root in the movie/series mappings. An empty mapping value means the
// media server sees the same path.
func (e *EmbySettings) ServerPath(physicalPath string) string {
	bestRoot, bestServer := "", ""
	for _, m := range []map[string]string{e.MoviePathsMapping, e.SeriesPathsMapping} {
		for root, server := range m {
			r := strings.TrimRight(root, "/")
			if r == "" || server == "" {
				continue
			}
			if (physicalPath == r || strings.HasPrefix(physicalPath, r+"/")) && len(r) > len(bestRoot) {
				bestRoot, bestServer = r, strings.TrimRight(server, "/")
			}
		}
	}
	if bestRoot == "" {
		return physicalPath
	}
	return bestServer + strings.TrimPrefix(physicalPath, bestRoot)
}

// IsJellyfin reports whether the media server is Jellyfin. Empty or unknown means Emby,
// so configs saved before this field existed keep working.
func (e *EmbySettings) IsJellyfin() bool {
	return e.ServerType == MediaServerJellyfin
}

func (e *EmbySettings) Check() {
	if e.MaxRequestVideoNumber < common.EmbyApiGetItemsLimitMin ||
		e.MaxRequestVideoNumber > common.EmbyApiGetItemsLimitMax {

		e.MaxRequestVideoNumber = common.EmbyApiGetItemsLimitMin
	}

	if e.Threads < 1 || e.Threads > 6 {
		e.Threads = 6
	}

}
