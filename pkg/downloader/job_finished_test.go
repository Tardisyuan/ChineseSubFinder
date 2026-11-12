package downloader

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/job_history"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/task_queue"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/subparser"
	taskQueue2 "github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/task_queue"
)

func lastHistory(t *testing.T, video string) job_history.JobHistory {
	t.Helper()
	rows, _, err := job_history.List(job_history.Filter{Keyword: video})
	if err != nil || len(rows) == 0 {
		t.Fatalf("no history for %s: %v", video, err)
	}
	return rows[0]
}

func TestJobFinishedWritesHistory(t *testing.T) {
	d := newTestDownloader(t, &fakeSupplier{})

	// success: the subtitle picked by oneVideoSelectBestSub ends up in the entry
	job := addJob(t, d, common.Movie, task_queue.DefaultTaskPriorityLevel)
	noteChosenSub(job.VideoFPath,
		[]string{"/tmp/[zimuku]_0_a.ass", "/tmp/[subhd]_0_b.srt", "/tmp/[zimuku]_1_c.ass"},
		subparser.FileInfo{FromWhereSite: "zimuku", FileFullPath: "/tmp/[zimuku]_0_a.ass"})
	d.jobFinished(job, nil)
	h := lastHistory(t, job.VideoFPath)
	if h.Result != job_history.ResultSuccess || h.Site != "zimuku" || h.Sites != "zimuku,subhd" || h.SubName != "[zimuku]_0_a.ass" {
		t.Errorf("success entry: %+v", h)
	}
	if h.Title != filepath.Base(filepath.Dir(job.VideoFPath)) {
		t.Errorf("movie title should be the folder name, got %q", h.Title)
	}
	if got := getJob(t, d, job.Id).JobStatus; got != taskQueue2.Done {
		t.Errorf("queue was not updated, status %v", got)
	}

	// nothing found, and nothing left over from the job before
	job = addJob(t, d, common.Series, task_queue.DefaultTaskPriorityLevel)
	d.jobFinished(job, task_queue.ErrNoSubFound)
	h = lastHistory(t, job.VideoFPath)
	if h.Result != job_history.ResultNoSub || h.Site != "" || h.Season != 1 || h.Episode != 2 {
		t.Errorf("no-sub entry: %+v", h)
	}
	if want := filepath.Base(job.SeriesRootDirPath) + " S01E02"; h.Title != want {
		t.Errorf("series title %q, want %q", h.Title, want)
	}

	// any other error; a candidate list is kept, a chosen subtitle is not claimed
	job = addJob(t, d, common.Movie, task_queue.DefaultTaskPriorityLevel)
	noteChosenSub(job.VideoFPath, []string{"/tmp/[subhd]_0_b.srt"}, subparser.FileInfo{FromWhereSite: "subhd"})
	d.jobFinished(job, errors.New("disk full"))
	h = lastHistory(t, job.VideoFPath)
	if h.Result != job_history.ResultFailed || h.Error != "disk full" || h.Site != "" || h.Sites != "subhd" {
		t.Errorf("failed entry: %+v", h)
	}
}
