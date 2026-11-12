package v1

import (
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/job_history"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/notify_center"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/backend"
	"github.com/gin-gonic/gin"
)

// HistoryListHandler GET /v1/history?result=&site=&keyword=&from=2006-01-02&to=2006-01-02&page=&page_size=
// from and to are local dates, both inclusive.
func (cb *ControllerBase) HistoryListHandler(c *gin.Context) {
	f := job_history.Filter{
		Result:  c.Query("result"),
		Site:    c.Query("site"),
		Keyword: c.Query("keyword"),
	}
	f.Page, _ = strconv.Atoi(c.Query("page"))
	f.PageSize, _ = strconv.Atoi(c.Query("page_size"))
	if t, err := time.ParseInLocation("2006-01-02", c.Query("from"), time.Local); err == nil {
		f.From = t
	}
	if t, err := time.ParseInLocation("2006-01-02", c.Query("to"), time.Local); err == nil {
		f.To = t.AddDate(0, 0, 1)
	}
	items, total, err := job_history.List(f)
	if err != nil {
		cb.ErrorProcess(c, "HistoryListHandler", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"total": total, "items": items})
}

// HistoryStatsHandler GET /v1/history/stats?days=14
func (cb *ControllerBase) HistoryStatsHandler(c *gin.Context) {
	days, _ := strconv.Atoi(c.Query("days"))
	if days < 1 || days > 90 {
		days = 14
	}
	stats, err := job_history.GetStats(days)
	if err != nil {
		cb.ErrorProcess(c, "HistoryStatsHandler", err)
		return
	}
	c.JSON(http.StatusOK, stats)
}

type notifyTestResult struct {
	Channel string `json:"channel"`
	Ok      bool   `json:"ok"`
	Error   string `json:"error,omitempty"`
}

// NotifyTestHandler POST /v1/notify/test, body: the notification_settings as they are in the
// form (saved or not). Sends one test message to every filled-in channel and waits for it.
func (cb *ControllerBase) NotifyTestHandler(c *gin.Context) {
	cfg := settings.NotificationSettings{}
	if err := c.ShouldBindJSON(&cfg); err != nil {
		c.JSON(http.StatusBadRequest, backend.ReplyCommon{Message: err.Error()})
		return
	}
	if !cfg.HasChannel() {
		c.JSON(http.StatusBadRequest, backend.ReplyCommon{Message: "请先填写至少一个通知渠道"})
		return
	}
	events := []notify_center.JobEvent{{
		Time: time.Now(), Title: "测试通知", Success: true, Site: "test",
	}}
	res := notify_center.Send(cfg, "ChineseSubFinder 测试通知", "看到这条消息说明通知配置可用。", events)
	out := make([]notifyTestResult, 0, len(res))
	for channel, err := range res {
		r := notifyTestResult{Channel: channel, Ok: err == nil}
		if err != nil {
			r.Error = err.Error()
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Channel < out[j].Channel })
	c.JSON(http.StatusOK, gin.H{"results": out})
}
