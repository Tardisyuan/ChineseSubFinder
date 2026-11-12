package task_queue

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/cache_center"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
	task_queue2 "github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/task_queue"
	"github.com/sirupsen/logrus"
)

// Jobs added to the queue must still be there after the queue is reopened.
func TestAddedJobsSurviveReopen(t *testing.T) {
	dir := t.TempDir()
	old, _ := os.Getwd()
	_ = os.Chdir(dir) // the cache center lives in ./cache
	defer func() { _ = os.Chdir(old) }()
	settings.SetConfigRootPath(dir)
	l := logrus.New()
	l.SetOutput(ioutil.Discard)

	tq := NewTaskQueue(cache_center.NewCacheCenter("reopen", l))
	for j := 0; j < 3; j++ {
		job := task_queue2.NewOneJob(common.Movie, fmt.Sprintf("/media/movies/m%d/m%d.mkv", j, j), DefaultTaskPriorityLevel)
		if ok, err := tq.Add(*job); err != nil || !ok {
			t.Fatal(ok, err)
		}
	}
	tq.Close()

	tq = NewTaskQueue(cache_center.NewCacheCenter("reopen", l))
	defer tq.Close()
	if _, jobs, err := tq.GetAllJobs(); err != nil || len(jobs) != 3 {
		t.Fatalf("got %d jobs after reopen (err %v), want 3", len(jobs), err)
	}
}

// BenchmarkAddJobs measures what a library scan pays to queue n episodes. Every Add() rewrites
// the whole priority bucket (JSON file + sqlite row), so the total cost grows with n squared.
func BenchmarkAddJobs(b *testing.B) {
	for _, batch := range []bool{false, true} {
		for _, n := range []int{250, 1000, 4000} {
			batch := batch
			b.Run(fmt.Sprintf("batch=%v/n=%d", batch, n), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					b.StopTimer()
					dir := b.TempDir()
					old, _ := os.Getwd()
					_ = os.Chdir(dir) // the cache center lives in ./cache
					settings.SetConfigRootPath(dir)
					l := logrus.New()
					l.SetOutput(ioutil.Discard)
					center := cache_center.NewCacheCenter(fmt.Sprintf("bench%d", time.Now().UnixNano()), l)
					tq := NewTaskQueue(center)
					jobs := make([]task_queue2.OneJob, n)
					for j := range jobs {
						jobs[j] = *task_queue2.NewOneJob(common.Series,
							fmt.Sprintf("/media/series/Some Show (2020)/Season 1/Some Show - S01E%04d - Title WEBDL-1080p.mkv", j),
							DefaultTaskPriorityLevel)
						jobs[j].SeriesRootDirPath = "/media/series/Some Show (2020)"
					}
					b.StartTimer()
					if batch {
						tq.BeginBatch()
					}
					for j := range jobs {
						if _, err := tq.Add(jobs[j]); err != nil {
							b.Fatal(err)
						}
					}
					if batch {
						if err := tq.EndBatch(); err != nil {
							b.Fatal(err)
						}
					}
					b.StopTimer()
					tq.Close()
					_ = os.Chdir(old)
				}
			})
		}
	}
}

// Jobs added inside a batch are on disk after EndBatch.
func TestBatchedAddSurvivesReopen(t *testing.T) {
	dir := t.TempDir()
	old, _ := os.Getwd()
	_ = os.Chdir(dir)
	defer func() { _ = os.Chdir(old) }()
	settings.SetConfigRootPath(dir)
	l := logrus.New()
	l.SetOutput(ioutil.Discard)

	tq := NewTaskQueue(cache_center.NewCacheCenter("batch", l))
	tq.BeginBatch()
	for j := 0; j < 5; j++ {
		job := task_queue2.NewOneJob(common.Movie, fmt.Sprintf("/media/movies/b%d/b%d.mkv", j, j), DefaultTaskPriorityLevel)
		if ok, err := tq.Add(*job); err != nil || !ok {
			t.Fatal(ok, err)
		}
	}
	if err := tq.EndBatch(); err != nil {
		t.Fatal(err)
	}
	tq.Close()

	tq = NewTaskQueue(cache_center.NewCacheCenter("batch", l))
	defer tq.Close()
	if _, jobs, err := tq.GetAllJobs(); err != nil || len(jobs) != 5 {
		t.Fatalf("got %d jobs after reopen (err %v), want 5", len(jobs), err)
	}
}
