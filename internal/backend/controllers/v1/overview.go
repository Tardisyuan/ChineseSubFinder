package v1

import (
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/emby_api"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/tmdb_api"
	TTaskqueue "github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/task_queue"
	"github.com/gin-gonic/gin"
)

type overviewSupplier struct {
	Name       string `json:"name"`
	Valid      bool   `json:"valid"`
	Speed      int64  `json:"speed"`
	TodayCount int    `json:"today_count"`
	DailyLimit int    `json:"daily_limit"` // -1 = unlimited
}

type overviewService struct {
	Enabled bool   `json:"enabled"`
	Name    string `json:"name"`
	Ok      bool   `json:"ok"`
}

type overviewReply struct {
	Today              string             `json:"today"`
	TodayDownloads     int                `json:"today_downloads"`
	Suppliers          []overviewSupplier `json:"suppliers"`
	SuppliersCheckedAt string             `json:"suppliers_checked_at"`
	Jobs               map[string]int     `json:"jobs"`
	DoneToday          int                `json:"done_today"`
	NextScanAt         string             `json:"next_scan_at"`
	MediaServer        overviewService    `json:"media_server"`
	Tmdb               overviewService    `json:"tmdb"`
}

// External checks (media server, TMDB) are cached so refreshing the page does not hit them every time.
const overviewServiceTTL = 2 * time.Minute

var overviewCache struct {
	sync.Mutex
	at          time.Time
	key         string
	mediaServer bool
	tmdb        bool
}

func timeString(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}

func (cb *ControllerBase) OverviewHandler(c *gin.Context) {
	now := time.Now()
	today := now.Format("2006-01-02")
	reply := overviewReply{Today: today, Jobs: map[string]int{}}

	// Suppliers: last check result plus today's download count and limit.
	downloader := cb.cronHelper.Downloader
	counts := cb.cronHelper.FileDownloader.CacheCenter.DailyDownloadCountsByDay(today)
	limits := settings.Get().AdvancedSettings.SuppliersSettings
	limitOf := map[string]int{
		limits.Xunlei.Name: limits.Xunlei.DailyDownloadLimit, limits.Shooter.Name: limits.Shooter.DailyDownloadLimit,
		limits.Assrt.Name: limits.Assrt.DailyDownloadLimit, limits.SubHD.Name: limits.SubHD.DailyDownloadLimit,
	}
	if downloader != nil {
		status, at := downloader.SiteStatus()
		reply.SuppliersCheckedAt = timeString(at)
		for _, s := range status.SubSiteStatus {
			limit, ok := limitOf[s.Name]
			if !ok {
				limit = -1
			}
			reply.Suppliers = append(reply.Suppliers, overviewSupplier{
				Name: s.Name, Valid: s.Valid, Speed: s.Speed, TodayCount: counts[s.Name], DailyLimit: limit,
			})
		}
	}
	sort.Slice(reply.Suppliers, func(i, j int) bool { return reply.Suppliers[i].Name < reply.Suppliers[j].Name })
	for _, n := range counts {
		reply.TodayDownloads += n
	}

	// Download queue by status, and how many finished today.
	if q := cb.cronHelper.DownloadQueue; q != nil {
		for _, st := range []TTaskqueue.JobStatus{TTaskqueue.Waiting, TTaskqueue.Downloading, TTaskqueue.Done, TTaskqueue.Failed, TTaskqueue.Ignore} {
			_, jobs, err := q.GetJobsByStatus(st)
			if err != nil {
				continue
			}
			reply.Jobs[st.String()] = len(jobs)
			if st == TTaskqueue.Done {
				for _, j := range jobs {
					if time.Time(j.UpdateTime).Format("2006-01-02") == today {
						reply.DoneToday++
					}
				}
			}
		}
	}
	reply.NextScanAt = timeString(cb.cronHelper.NextScanTime())

	// Media server and TMDB reachability, cached.
	emby := settings.Get().EmbySettings
	tmdbSet := settings.Get().AdvancedSettings.TmdbApiSettings
	reply.MediaServer = overviewService{Enabled: emby.Enable, Name: "Emby"}
	if emby.IsJellyfin() {
		reply.MediaServer.Name = "Jellyfin"
	}
	reply.Tmdb = overviewService{Enabled: tmdbSet.Enable && tmdbSet.ApiKey != "", Name: "TMDB"}

	cacheKey := emby.AddressUrl + "|" + emby.APIKey + "|" + emby.ServerType + "|" + tmdbSet.ApiKey
	overviewCache.Lock()
	if overviewCache.key != cacheKey || now.Sub(overviewCache.at) > overviewServiceTTL {
		overviewCache.mediaServer, overviewCache.tmdb = false, false
		if reply.MediaServer.Enabled {
			users, err := emby_api.NewEmbyApi(cb.log).GetUserIdList(emby)
			overviewCache.mediaServer = err == nil && len(users.Items) > 0
		}
		if reply.Tmdb.Enabled {
			if t, err := tmdb_api.NewTmdbHelper(cb.log, tmdbSet.ApiKey, tmdbSet.UseAlternateBaseURL); err == nil {
				overviewCache.tmdb = t.Alive()
			}
		}
		overviewCache.key, overviewCache.at = cacheKey, now
	}
	reply.MediaServer.Ok = reply.MediaServer.Enabled && overviewCache.mediaServer
	reply.Tmdb.Ok = reply.Tmdb.Enabled && overviewCache.tmdb
	overviewCache.Unlock()

	c.JSON(http.StatusOK, reply)
}
