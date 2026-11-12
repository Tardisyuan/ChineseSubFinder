// Package job_history keeps a record of every finished subtitle job in the cache database,
// for the 下载历史 page.
package job_history

import (
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/internal/dao"
	"gorm.io/gorm"
)

const (
	ResultSuccess = "success" // a subtitle was written next to the video
	ResultNoSub   = "no_sub"  // no site had a subtitle
	ResultFailed  = "failed"  // anything else went wrong

	MaxRows    = 5000 // retention: newest rows kept
	MaxAgeDays = 180  // retention: rows older than this are removed
	pruneEvery = 50   // inserts between two retention runs
)

// JobHistory is one finished attempt of one job.
type JobHistory struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	CreatedAt  time.Time `gorm:"index" json:"time"`
	JobId      string    `json:"job_id"`
	VideoType  int       `json:"video_type"` // 0 movie, 1 series
	VideoFPath string    `gorm:"index" json:"video_f_path"`
	Title      string    `json:"title"` // display name
	Season     int       `json:"season"`
	Episode    int       `json:"episode"`
	Result     string    `gorm:"index" json:"result"`
	Site       string    `gorm:"index" json:"site"` // site of the subtitle that was saved
	Sites      string    `json:"sites"`             // every site that delivered a candidate, comma separated
	SubName    string    `json:"sub_name"`
	Error      string    `json:"error"`
}

var (
	getDb       = dao.GetDb // replaced in tests
	migrateOnce sync.Once
	inserts     int
	insertsLock sync.Mutex
)

func db() *gorm.DB {
	d := getDb()
	migrateOnce.Do(func() { _ = d.AutoMigrate(&JobHistory{}) })
	return d
}

// Record stores one entry and now and then applies the retention limits.
func Record(h *JobHistory) error {
	if err := db().Create(h).Error; err != nil {
		return err
	}
	insertsLock.Lock()
	inserts++
	due := inserts%pruneEvery == 0
	insertsLock.Unlock()
	if due {
		return prune(h.ID)
	}
	return nil
}

func prune(newestID uint) error {
	if newestID > MaxRows {
		if err := db().Where("id <= ?", newestID-MaxRows).Delete(&JobHistory{}).Error; err != nil {
			return err
		}
	}
	return db().Where("created_at < ?", time.Now().AddDate(0, 0, -MaxAgeDays)).Delete(&JobHistory{}).Error
}

// Filter narrows List; zero values mean "any".
type Filter struct {
	Result   string
	Site     string
	Keyword  string    // part of the title or path
	From, To time.Time // To is exclusive
	Page     int       // from 1
	PageSize int
}

// List returns one page, newest first, and the number of rows matching the filter.
func List(f Filter) ([]JobHistory, int64, error) {
	q := db().Model(&JobHistory{})
	if f.Result != "" {
		q = q.Where("result = ?", f.Result)
	}
	if f.Site != "" {
		q = q.Where("site = ?", f.Site)
	}
	if f.Keyword != "" {
		like := "%" + f.Keyword + "%"
		q = q.Where("title LIKE ? OR video_f_path LIKE ?", like, like)
	}
	if !f.From.IsZero() {
		q = q.Where("created_at >= ?", f.From)
	}
	if !f.To.IsZero() {
		q = q.Where("created_at < ?", f.To)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if f.PageSize < 1 || f.PageSize > 200 {
		f.PageSize = 20
	}
	if f.Page < 1 {
		f.Page = 1
	}
	rows := make([]JobHistory, 0)
	err := q.Order("id DESC").Offset((f.Page - 1) * f.PageSize).Limit(f.PageSize).Find(&rows).Error
	return rows, total, err
}

type DayStat struct {
	Date    string `json:"date"` // 2006-01-02, local time
	Success int    `json:"success"`
	NoSub   int    `json:"no_sub"`
	Failed  int    `json:"failed"`
}

type SiteStat struct {
	Site    string `json:"site"`
	Offered int    `json:"offered"` // jobs in which the site delivered at least one candidate
	Chosen  int    `json:"chosen"`  // jobs in which its subtitle was the one saved
}

type MissingVideo struct {
	VideoFPath string    `json:"video_f_path"`
	Title      string    `json:"title"`
	Attempts   int       `json:"attempts"`
	LastTime   time.Time `json:"last_time"`
	LastResult string    `json:"last_result"`
	LastError  string    `json:"last_error"`
}

type Stats struct {
	Total        int            `json:"total"`
	Success      int            `json:"success"`
	Days         []DayStat      `json:"days"`  // the last `days` days, oldest first, days without jobs included
	Sites        []SiteStat     `json:"sites"` // most offered first
	Missing      []MissingVideo `json:"missing"`
	MissingCount int            `json:"missing_count"`
}

const maxMissingListed = 200

// GetStats summarises the whole history. A video is "missing" when its latest attempt was not
// a success and the file still exists.
// ponytail: loads every row (at most MaxRows) and counts in Go; move to SQL if MaxRows grows a lot.
func GetStats(days int) (Stats, error) {
	var rows []JobHistory
	if err := db().Order("id").Find(&rows).Error; err != nil {
		return Stats{}, err
	}
	st := Stats{Total: len(rows), Days: make([]DayStat, days), Sites: []SiteStat{}, Missing: []MissingVideo{}}
	dayIndex := make(map[string]int, days)
	today := time.Now()
	for i := 0; i < days; i++ {
		date := today.AddDate(0, 0, i-days+1).Format("2006-01-02")
		st.Days[i].Date = date
		dayIndex[date] = i
	}
	sites := make(map[string]*SiteStat)
	site := func(name string) *SiteStat {
		if sites[name] == nil {
			sites[name] = &SiteStat{Site: name}
		}
		return sites[name]
	}
	latest := make(map[string]*MissingVideo)
	for _, r := range rows {
		if i, ok := dayIndex[r.CreatedAt.Local().Format("2006-01-02")]; ok {
			switch r.Result {
			case ResultSuccess:
				st.Days[i].Success++
			case ResultNoSub:
				st.Days[i].NoSub++
			default:
				st.Days[i].Failed++
			}
		}
		if r.Result == ResultSuccess {
			st.Success++
			if r.Site != "" {
				site(r.Site).Chosen++
			}
		}
		for _, s := range strings.Split(r.Sites, ",") {
			if s != "" {
				site(s).Offered++
			}
		}
		m := latest[r.VideoFPath]
		if m == nil {
			m = &MissingVideo{VideoFPath: r.VideoFPath}
			latest[r.VideoFPath] = m
		}
		m.Attempts++
		m.Title, m.LastTime, m.LastResult, m.LastError = r.Title, r.CreatedAt, r.Result, r.Error
	}
	for _, s := range sites {
		st.Sites = append(st.Sites, *s)
	}
	sort.Slice(st.Sites, func(i, j int) bool {
		if st.Sites[i].Offered != st.Sites[j].Offered {
			return st.Sites[i].Offered > st.Sites[j].Offered
		}
		return st.Sites[i].Site < st.Sites[j].Site
	})
	for _, m := range latest {
		if m.LastResult == ResultSuccess {
			continue
		}
		if _, err := os.Stat(m.VideoFPath); err != nil {
			continue // deleted or moved since
		}
		st.Missing = append(st.Missing, *m)
	}
	st.MissingCount = len(st.Missing)
	sort.Slice(st.Missing, func(i, j int) bool { return st.Missing[i].LastTime.After(st.Missing[j].LastTime) })
	if len(st.Missing) > maxMissingListed {
		st.Missing = st.Missing[:maxMissingListed]
	}
	return st, nil
}
