package downloader

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/cache_center"
	embyHelper "github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/emby_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/file_downloader"
	subSupplier "github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_supplier"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/media_info_dealers"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/scan_logic"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/task_queue"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/series"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/supplier"
	taskQueue2 "github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/task_queue"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

// Config, database and ./cache all go to a throwaway directory.
func TestMain(m *testing.M) {
	root, err := os.MkdirTemp("", "csf-downloader-test")
	if err != nil {
		panic(err)
	}
	_ = os.Setenv("HOME", root)
	if err = os.Chdir(root); err != nil {
		panic(err)
	}
	settings.SetConfigRootPath(root)
	code := m.Run()
	_ = os.RemoveAll(root)
	os.Exit(code)
}

// fakeSupplier finds nothing; started/release let a test hold a job in mid-download.
type fakeSupplier struct {
	log     *logrus.Logger
	started chan struct{}
	release chan struct{}
}

func (f *fakeSupplier) CheckAlive() (bool, int64)    { return true, 0 }
func (f *fakeSupplier) IsAlive() bool                { return true }
func (f *fakeSupplier) GetSupplierName() string      { return "fake" }
func (f *fakeSupplier) OverDailyDownloadLimit() bool { return false }
func (f *fakeSupplier) GetLogger() *logrus.Logger    { return f.log }
func (f *fakeSupplier) GetSubListFromFile4Movie(string) ([]supplier.SubInfo, error) {
	if f.started != nil {
		f.started <- struct{}{}
		<-f.release
	}
	return nil, nil
}
func (f *fakeSupplier) GetSubListFromFile4Series(*series.SeriesInfo) ([]supplier.SubInfo, error) {
	return nil, errors.New("not used")
}
func (f *fakeSupplier) GetSubListFromFile4Anime(*series.SeriesInfo) ([]supplier.SubInfo, error) {
	return nil, errors.New("not used")
}

func newTestDownloader(t *testing.T, sup *fakeSupplier) *Downloader {
	log := logrus.New()
	sup.log = log
	center := cache_center.NewCacheCenter(t.Name(), log)
	d := &Downloader{
		log:            log,
		fileDownloader: &file_downloader.FileDownloader{Log: log, CacheCenter: center},
		downloadQueue:  task_queue.NewTaskQueue(center),
		subSupplierHub: subSupplier.NewSubSupplierHub(sup),
		ScanLogic:      scan_logic.NewScanLogic(log),
	}
	d.ctx, d.cancel = context.WithCancel(context.Background())
	t.Cleanup(d.cancel)
	return d
}

// addJob puts a job for a freshly created video file into the queue.
func addJob(t *testing.T, d *Downloader, videoType common.VideoType, priority int) taskQueue2.OneJob {
	dir := t.TempDir()
	video := filepath.Join(dir, "Some Show S01E02.mkv")
	if err := os.WriteFile(video, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	job := taskQueue2.NewOneJob(videoType, video, priority)
	if videoType == common.Series {
		job.Season, job.Episode, job.SeriesRootDirPath = 1, 2, dir // dir has no tvshow.nfo
	}
	if ok, err := d.downloadQueue.Add(*job); err != nil || !ok {
		t.Fatal("add job", ok, err)
	}
	return *job
}

func getJob(t *testing.T, d *Downloader, id string) taskQueue2.OneJob {
	ok, job := d.downloadQueue.GetOneJobByID(id)
	if !ok {
		t.Fatal("job is gone")
	}
	return job
}

// With no alive supplier the job used to be marked Downloading and then left that way.
func TestQueueLeavesJobWaitingWithoutSuppliers(t *testing.T) {
	sup := &fakeSupplier{}
	d := newTestDownloader(t, sup)
	d.subSupplierHub.DelSubSupplier(sup)
	job := addJob(t, d, common.Movie, task_queue.HighTaskPriorityLevel)

	d.queueDownloaderLocal()

	if got := getJob(t, d, job.Id).JobStatus; got != taskQueue2.Waiting {
		t.Fatalf("job status = %v, want Waiting", got)
	}
}

// A series whose info cannot be read used to be returned untouched, so the queue took the
// same job again on every tick and never got past it.
func TestQueueRecordsSeriesInfoError(t *testing.T) {
	d := newTestDownloader(t, &fakeSupplier{})
	job := addJob(t, d, common.Series, task_queue.DefaultTaskPriorityLevel)

	d.queueDownloaderLocal()

	got := getJob(t, d, job.Id)
	if got.DownloadTimes != 1 || got.ErrorInfo == "" {
		t.Fatalf("failure not recorded: DownloadTimes=%d ErrorInfo=%q", got.DownloadTimes, got.ErrorInfo)
	}
	if found, next, _ := d.downloadQueue.GetOneJob(); found {
		t.Fatalf("job %s would be retried at once", next.VideoFPath)
	}
}

// An unreachable media server used to stop the queue at the played check.
func TestQueueGoesOnWhenMediaServerIsDown(t *testing.T) {
	sup := &fakeSupplier{}
	d := newTestDownloader(t, sup)
	em := settings.Get().EmbySettings
	oldUrl := em.AddressUrl
	em.AddressUrl = "http://127.0.0.1:1" // nothing listens here
	defer func() { em.AddressUrl = oldUrl }()
	d.embyHelper = embyHelper.NewEmbyHelper(&media_info_dealers.Dealers{Logger: d.log})

	dir := t.TempDir()
	video := filepath.Join(dir, "movie.mkv")
	if err := os.WriteFile(video, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	job := taskQueue2.NewOneJob(common.Movie, video, task_queue.HighTaskPriorityLevel, "item-1")
	if ok, err := d.downloadQueue.Add(*job); err != nil || !ok {
		t.Fatal("add job", ok, err)
	}

	d.queueDownloaderLocal()

	if got := getJob(t, d, job.Id); got.DownloadTimes != 1 {
		t.Fatalf("job was not processed: status=%v DownloadTimes=%d", got.JobStatus, got.DownloadTimes)
	}
}

// Cancel must not make the queue return while the job goroutine is still running: the lock
// would be free and the next job would run next to it, sharing (and wiping) the tmp folder.
func TestQueueWaitsForRunningJobOnCancel(t *testing.T) {
	sup := &fakeSupplier{started: make(chan struct{}), release: make(chan struct{})}
	d := newTestDownloader(t, sup)
	job := addJob(t, d, common.Movie, task_queue.HighTaskPriorityLevel)

	returned := make(chan struct{})
	go func() {
		d.queueDownloaderLocal()
		close(returned)
	}()
	<-sup.started
	d.Cancel()
	select {
	case <-returned:
		t.Fatal("queueDownloaderLocal returned while the job was still downloading")
	case <-time.After(300 * time.Millisecond):
	}
	close(sup.release)
	select {
	case <-returned:
	case <-time.After(10 * time.Second):
		t.Fatal("queueDownloaderLocal did not return after the job finished")
	}
	if got := getJob(t, d, job.Id); got.JobStatus != taskQueue2.Waiting || got.ErrorInfo != task_queue.ErrNoSubFound.Error() {
		t.Fatalf("job status = %v, error = %q", got.JobStatus, got.ErrorInfo)
	}
}

func TestNoSubSavedErrIsNeverNil(t *testing.T) {
	if noSubSavedErr(nil) != task_queue.ErrNoSubFound {
		t.Fatal("nil must become ErrNoSubFound, or the job is marked Done")
	}
	e := errors.New("x")
	if noSubSavedErr(e) != e {
		t.Fatal("a real error must be kept")
	}
}
