package task_queue

import (
	"errors"
	"io/ioutil"
	"os"
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/cache_center"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
	task_queue2 "github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/task_queue"
	"github.com/sirupsen/logrus"
)

// A job that keeps failing moves from the default level to the first-retry level, then one
// level further back each time it exceeds MaxRetryTimes, and never ahead of the default.
func TestFailingJobIsDowngraded(t *testing.T) {
	dir := t.TempDir()
	old, _ := os.Getwd()
	_ = os.Chdir(dir)
	defer func() { _ = os.Chdir(old) }()
	settings.SetConfigRootPath(dir)
	l := logrus.New()
	l.SetOutput(ioutil.Discard)
	tq := NewTaskQueue(cache_center.NewCacheCenter("retry", l))
	defer tq.Close()

	job := task_queue2.NewOneJob(common.Movie, "/media/movies/a/a.mkv", DefaultTaskPriorityLevel)
	if ok, err := tq.Add(*job); err != nil || !ok {
		t.Fatal(ok, err)
	}
	maxRetry := settings.Get().AdvancedSettings.TaskQueue.MaxRetryTimes
	fail := errors.New("no sub found")
	priorities := []int{}
	for i := 0; i < 2*(maxRetry+1)+1; i++ {
		_, cur := tq.GetOneJobByID(job.Id)
		tq.AutoDetectUpdateJobStatus(cur, fail)
		_, cur = tq.GetOneJobByID(job.Id)
		priorities = append(priorities, cur.TaskPriority)
	}
	last := priorities[len(priorities)-1]
	if priorities[0] != FirstRetryTaskPriorityLevel || last <= FirstRetryTaskPriorityLevel {
		t.Fatalf("priorities over failures: %v (want start at %d and move back)", priorities, FirstRetryTaskPriorityLevel)
	}
	for _, p := range priorities {
		if p < DefaultTaskPriorityLevel {
			t.Fatalf("a failing job moved ahead of the default level: %v", priorities)
		}
	}
}

// A job whose series folder changed after it was queued can still be deleted.
func TestDeleteAfterSeriesPathChange(t *testing.T) {
	dir := t.TempDir()
	old, _ := os.Getwd()
	_ = os.Chdir(dir)
	defer func() { _ = os.Chdir(old) }()
	settings.SetConfigRootPath(dir)
	l := logrus.New()
	l.SetOutput(ioutil.Discard)
	tq := NewTaskQueue(cache_center.NewCacheCenter("regroup", l))
	defer tq.Close()

	job := task_queue2.NewOneJob(common.Series, "/media/tv/Old Name/Season 1/e01.mkv", DefaultTaskPriorityLevel)
	job.SeriesRootDirPath = "/media/tv/Old Name"
	if ok, err := tq.Add(*job); err != nil || !ok {
		t.Fatal(ok, err)
	}
	job.SeriesRootDirPath = "/media/tv/New Name"
	if ok, err := tq.Update(*job); err != nil || !ok {
		t.Fatal(ok, err)
	}
	if ok, err := tq.Del(job.Id); err != nil || !ok {
		t.Fatalf("delete after path change: ok=%v err=%v", ok, err)
	}
	if tq.IsExist(job.Id) {
		t.Fatal("job still in queue")
	}
}
