package downloader

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/job_history"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/notify_center"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/task_queue"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/subparser"
	taskQueue2 "github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/task_queue"
)

// chosenSub is what oneVideoSelectBestSub picked for one video.
type chosenSub struct {
	site, name string
	sites      []string // every site that delivered a candidate
}

// chosenSubs: video path -> chosenSub, filled while a job runs and emptied when it finishes.
// Only one job downloads at a time (downloaderLock), so entries always belong to that job.
var chosenSubs sync.Map

var subSiteRe = regexp.MustCompile(`^\[(\w+)\]_`) // same naming rule as sub_parser_hub.getFromWhereSite

// noteChosenSub remembers which subtitle was picked for a video, for the history entry.
func noteChosenSub(videoFPath string, candidates []string, chosen subparser.FileInfo) {
	c := chosenSub{site: chosen.FromWhereSite, name: chosen.Name}
	if c.name == "" {
		c.name = filepath.Base(chosen.FileFullPath)
	}
	seen := make(map[string]bool)
	for _, f := range candidates {
		if m := subSiteRe.FindStringSubmatch(filepath.Base(f)); m != nil && !seen[m[1]] {
			seen[m[1]] = true
			c.sites = append(c.sites, m[1])
		}
	}
	chosenSubs.Store(videoFPath, c)
}

// jobFinished updates the job in the queue (as AutoDetectUpdateJobStatus did at these places
// before) and then records the outcome in the history and hands it to the notifier. The
// record keeping must never break or delay a download: errors are only logged and the
// notifier sends in the background.
func (d *Downloader) jobFinished(job taskQueue2.OneJob, jobErr error) {
	d.downloadQueue.AutoDetectUpdateJobStatus(job, jobErr)
	d.recordOutcome(job, jobErr)
	// queue jobs run one at a time: drop anything a job that ended early left behind
	chosenSubs.Range(func(k, _ interface{}) bool { chosenSubs.Delete(k); return true })
}

// recordOutcome writes the history entry and hands the event to the notifier. It is also used
// for a subtitle picked by hand in the library search, which has no queue job of its own.
func (d *Downloader) recordOutcome(job taskQueue2.OneJob, jobErr error) {
	defer func() {
		if p := recover(); p != nil {
			d.log.Errorln("jobFinished panic:", p)
		}
	}()
	entry := job_history.JobHistory{
		JobId:      job.Id,
		VideoType:  int(job.VideoType),
		VideoFPath: job.VideoFPath,
		Title:      jobTitle(job),
		Season:     job.Season,
		Episode:    job.Episode,
		Result:     job_history.ResultSuccess,
	}
	if v, ok := chosenSubs.Load(job.VideoFPath); ok {
		c := v.(chosenSub)
		entry.Sites = strings.Join(c.sites, ",")
		if jobErr == nil {
			entry.Site, entry.SubName = c.site, c.name
		}
	}
	chosenSubs.Delete(job.VideoFPath)
	if jobErr != nil {
		entry.Error = jobErr.Error()
		entry.Result = job_history.ResultFailed
		if errors.Is(jobErr, task_queue.ErrNoSubFound) || errors.Is(jobErr, common.AllSiteDownloadSubNotFound) {
			entry.Result = job_history.ResultNoSub
		}
	}
	if err := job_history.Record(&entry); err != nil {
		d.log.Errorln("job_history.Record", job.VideoFPath, err)
	}
	notify_center.Jobs.Add(d.log, notify_center.JobEvent{
		Time:       time.Now(),
		Title:      entry.Title,
		VideoFPath: entry.VideoFPath,
		Success:    jobErr == nil,
		Site:       entry.Site,
		SubName:    entry.SubName,
		Error:      entry.Error,
	})
}

// jobTitle is the name shown in the history and in notifications: the movie folder, or the
// series folder with SxxEyy.
func jobTitle(job taskQueue2.OneJob) string {
	if job.VideoType == common.Series && job.SeriesRootDirPath != "" {
		return fmt.Sprintf("%s S%02dE%02d", filepath.Base(job.SeriesRootDirPath), job.Season, job.Episode)
	}
	if dir := filepath.Base(filepath.Dir(job.VideoFPath)); dir != "" && dir != "." && dir != string(filepath.Separator) {
		return dir
	}
	return filepath.Base(job.VideoFPath)
}
