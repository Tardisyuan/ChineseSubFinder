package v1

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/internal/models"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/cache_center"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/downloader"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/cron_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/scan_logic"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/task_queue"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
	task_queue3 "github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/task_queue"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Config, database and ./cache all go to a throwaway directory.
func TestMain(m *testing.M) {
	root, err := os.MkdirTemp("", "csf-v1-test")
	if err != nil {
		panic(err)
	}
	_ = os.Setenv("HOME", root)
	if err = os.Chdir(root); err != nil {
		panic(err)
	}
	settings.SetConfigRootPath(root)
	gin.SetMode(gin.TestMode)
	code := m.Run()
	_ = os.RemoveAll(root)
	os.Exit(code)
}

func waitStatus(t *testing.T, s *searchTask, want string) searchState {
	for i := 0; i < 200; i++ {
		if st := s.snapshot(); st.Status == want {
			return st
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("status = %q, want %q", s.snapshot().Status, want)
	return searchState{}
}

func TestSearchTaskStates(t *testing.T) {
	s := &searchTask{}
	if st := s.snapshot(); st.Status != searchIdle || st.Candidates == nil {
		t.Fatalf("fresh task: %+v", st)
	}

	// running: a second search is refused and the first one is untouched
	release := make(chan struct{})
	found := []downloader.SubCandidate{{ID: 0, Site: "a"}, {ID: 1, Site: "b"}}
	if !s.start("/m/a.mkv", 0, func() ([]downloader.SubCandidate, error) { <-release; return found, nil }) {
		t.Fatal("first start refused")
	}
	if s.start("/m/b.mkv", 0, func() ([]downloader.SubCandidate, error) { return nil, nil }) {
		t.Fatal("a second search started while one was running")
	}
	if st := s.snapshot(); st.Status != searchRunning || st.VideoFPath != "/m/a.mkv" {
		t.Fatalf("while running: %+v", st)
	}
	if _, _, ok := s.candidate("/m/a.mkv", 0); ok {
		t.Fatal("a candidate was available before the search finished")
	}

	// done: results can be picked, but only for the searched video
	close(release)
	if st := waitStatus(t, s, searchDone); len(st.Candidates) != 2 {
		t.Fatalf("done: %+v", st)
	}
	if c, _, ok := s.candidate("/m/a.mkv", 1); !ok || c.Site != "b" {
		t.Fatalf("candidate 1 = %+v, %v", c, ok)
	}
	if _, _, ok := s.candidate("/m/a.mkv", 7); ok {
		t.Fatal("unknown id accepted")
	}
	if _, _, ok := s.candidate("/m/other.mkv", 1); ok {
		t.Fatal("result of another video accepted")
	}

	// failed, then usable again; the old results are gone
	if !s.start("/m/b.mkv", 1, func() ([]downloader.SubCandidate, error) { return nil, errors.New("no supplier") }) {
		t.Fatal("start after done refused")
	}
	if st := waitStatus(t, s, searchFailed); st.Error != "no supplier" || len(st.Candidates) != 0 || st.VideoType != 1 {
		t.Fatalf("failed: %+v", st)
	}

	// a panic in the search is a failure, not a task stuck in running
	if !s.start("/m/c.mkv", 0, func() ([]downloader.SubCandidate, error) { panic("boom") }) {
		t.Fatal("start after failed refused")
	}
	if st := waitStatus(t, s, searchFailed); st.Error == "" {
		t.Fatalf("panic: %+v", st)
	}
}

func newTestController(t *testing.T) *ControllerBase {
	log := logrus.New()
	return &ControllerBase{log: log, cronHelper: &cron_helper.CronHelper{
		Logger:        log,
		DownloadQueue: task_queue.NewTaskQueue(cache_center.NewCacheCenter(t.Name(), log)),
		Downloader:    &downloader.Downloader{ScanLogic: scan_logic.NewScanLogic(log)},
	}}
}

func post(t *testing.T, h gin.HandlerFunc, body interface{}) map[string]interface{} {
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(b))
	c.Request.Header.Set("Content-Type", "application/json")
	h(c)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	out := map[string]interface{}{}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func newVideo(t *testing.T) string {
	video := filepath.Join(t.TempDir(), "A Movie (2020)", "A.Movie.2020.mkv")
	_ = os.MkdirAll(filepath.Dir(video), 0755)
	if err := os.WriteFile(video, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	return video
}

// Retry puts a failed, downgraded job back to waiting ahead of the normal ones, so the
// queue takes it on the next tick.
func TestRetryJobHandler(t *testing.T) {
	cb := newTestController(t)
	q := cb.cronHelper.DownloadQueue
	job := task_queue3.NewOneJob(common.Movie, newVideo(t), task_queue.DefaultTaskPriorityLevel)
	if ok, err := q.Add(*job); err != nil || !ok {
		t.Fatal(ok, err)
	}
	failed := *job
	failed.JobStatus, failed.ErrorInfo, failed.RetryTimes, failed.DownloadTimes = task_queue3.Failed, "no sub found", 2, 3
	failed.TaskPriority = task_queue.LowTaskPriorityLevel
	if ok, err := q.Update(failed); err != nil || !ok {
		t.Fatal(ok, err)
	}
	if found, _, _ := q.GetOneWaitingJob(); found {
		t.Fatal("a failed job should not be taken by the queue")
	}

	if msg := post(t, cb.RetryJobHandler, gin.H{"id": job.Id})["message"]; msg != "ok" {
		t.Fatalf("retry: %v", msg)
	}
	_, got := q.GetOneJobByID(job.Id)
	if got.JobStatus != task_queue3.Waiting || got.TaskPriority != task_queue.HighTaskPriorityLevel ||
		got.ErrorInfo != "" || got.RetryTimes != 0 || got.DownloadTimes != 3 {
		t.Fatalf("after retry: %+v", got)
	}
	if found, next, _ := q.GetOneWaitingJob(); !found || next.Id != job.Id {
		t.Fatal("the retried job is not the next one taken")
	}
	if msg := post(t, cb.RetryJobHandler, gin.H{"id": "nope"})["message"]; msg != "job not found" {
		t.Fatalf("unknown job: %v", msg)
	}
}

// add-job (MoviePilot) and the UI's add answer "video is locked" for a locked video and
// queue nothing; after unlocking the job is added.
func TestAddJobAnswersLocked(t *testing.T) {
	cb := newTestController(t)
	video := newVideo(t)
	req := gin.H{"video_type": 0, "physical_video_file_full_path": video, "task_priority_level": 3}
	cb.cronHelper.Downloader.ScanLogic.Set(models.NewSkipScanInfoByMovie(video, true))

	for name, h := range map[string]gin.HandlerFunc{"api": cb.AddJobHandler, "ui": cb.VideoListAddHandler} {
		if msg := post(t, h, req)["message"]; msg != msgVideoLocked {
			t.Fatalf("%s add for a locked video: %v", name, msg)
		}
	}
	if n := cb.cronHelper.DownloadQueue.Size(); n != 0 {
		t.Fatalf("%d jobs queued for a locked video", n)
	}

	cb.cronHelper.Downloader.ScanLogic.Set(models.NewSkipScanInfoByMovie(video, false))
	if msg := post(t, cb.AddJobHandler, req)["message"]; msg != "ok" {
		t.Fatalf("add after unlock: %v", msg)
	}
	if n := cb.cronHelper.DownloadQueue.Size(); n != 1 {
		t.Fatalf("queue size after unlock = %d", n)
	}
}

// MoviePilot posts an episode before its scraper wrote the nfo files: the request gets a
// job id at once and the job is queued when the nfo files appear.
func TestAddJobWaitsForSeriesNfo(t *testing.T) {
	nfoWaitTick, nfoWaitMax = 20*time.Millisecond, 5*time.Second
	defer func() { nfoWaitTick, nfoWaitMax = 10*time.Second, 5*time.Minute }()

	cb := newTestController(t)
	root := filepath.Join(t.TempDir(), "Show (2024)")
	video := filepath.Join(root, "Season 1", "Show.S01E02.mkv")
	_ = os.MkdirAll(filepath.Dir(video), 0755)
	if err := os.WriteFile(video, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	reply := post(t, cb.AddJobHandler, gin.H{"video_type": 1, "physical_video_file_full_path": video, "task_priority_level": 3})
	if reply["message"] != msgWaitingNfo || reply["job_id"] == "" {
		t.Fatalf("reply without nfo: %v", reply)
	}
	if n := cb.cronHelper.DownloadQueue.Size(); n != 0 {
		t.Fatalf("queued before the nfo exists: %d", n)
	}

	_ = os.WriteFile(filepath.Join(root, "tvshow.nfo"), []byte(`<tvshow><title>Show</title></tvshow>`), 0644)
	_ = os.WriteFile(filepath.Join(root, "Season 1", "Show.S01E02.nfo"),
		[]byte(`<episodedetails><title>Ep</title><season>1</season><episode>2</episode></episodedetails>`), 0644)
	for i := 0; i < 100 && cb.cronHelper.DownloadQueue.Size() == 0; i++ {
		time.Sleep(20 * time.Millisecond)
	}
	if n := cb.cronHelper.DownloadQueue.Size(); n != 1 {
		t.Fatalf("queue size after the nfo appeared = %d", n)
	}
}
