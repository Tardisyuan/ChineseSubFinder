package downloader

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/internal/models"
	markSystem "github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/mark_system"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/save_sub_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/scan_logic"
	subcommon "github.com/ChineseSubFinder/ChineseSubFinder/pkg/sub_formatter/common"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/sub_formatter/emby"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/task_queue"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/language"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/supplier"
	taskQueue2 "github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/task_queue"
)

// subsSupplier is a fakeSupplier that finds the given subtitles for any movie.
type subsSupplier struct {
	fakeSupplier
	subs []supplier.SubInfo
}

func (s *subsSupplier) GetSubListFromFile4Movie(string) ([]supplier.SubInfo, error) {
	return s.subs, nil
}

func srtOf(lines ...string) []byte {
	out := ""
	for i := 0; i < 30; i++ {
		out += strings.Join([]string{string(rune('1' + i%9)), "00:00:0" + string(rune('0'+i%10)) + ",000 --> 00:00:0" + string(rune('0'+i%10)) + ",900", lines[i%len(lines)], "", ""}, "\n")
	}
	return []byte(out)
}

func newSearchDownloader(t *testing.T) (*Downloader, string) {
	sup := &subsSupplier{subs: []supplier.SubInfo{
		*supplier.NewSubInfo("siteA", 1, "The.Movie.2020.eng.srt", language.English, "http://a/1", 0, 0, ".srt",
			srtOf("Hello there, how are you today?", "This is only an English line.")),
		*supplier.NewSubInfo("siteB", 1, "The.Movie.2020.chs.srt", language.ChineseSimple, "http://b/1", 0, 0, ".srt",
			srtOf("你好，今天过得怎么样？", "这是一句简体中文的台词。")),
	}}
	d := newTestDownloader(t, &sup.fakeSupplier)
	log := d.log
	d.subSupplierHub.Suppliers[0] = sup
	d.mk = markSystem.NewMarkingSystem(log)
	d.subFormatter = emby.NewFormatter()
	d.subNameFormatter = subcommon.FormatterName(d.subFormatter.GetFormatterFormatterName())
	d.SaveSubHelper = save_sub_helper.NewSaveSubHelper(log, d.subFormatter, nil)

	video := filepath.Join(t.TempDir(), "The Movie (2020)", "The.Movie.2020.mkv")
	if err := os.MkdirAll(filepath.Dir(video), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(video, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	return d, video
}

// A manual search returns every file found, marks the one the ranking would save, keeps
// the files after the tmp folder is wiped, and a picked one is saved and locks the video.
func TestSearchCandidatesAndSave(t *testing.T) {
	d, video := newSearchDownloader(t)

	got, err := d.SearchCandidates(common.Movie, video)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("candidates = %+v, want 2", got)
	}
	first, second := got[0], got[1]
	if !first.Best || !first.IsChinese || first.Site != "siteB" || first.Ext != ".srt" || len(first.Preview) == 0 {
		t.Fatalf("first candidate should be the recommended Chinese one from siteB: %+v", first)
	}
	if second.Best || second.IsChinese || second.Site != "siteA" {
		t.Fatalf("second candidate should be the English one from siteA: %+v", second)
	}
	for _, c := range got {
		if _, err := os.Stat(c.Path); err != nil {
			t.Fatalf("candidate file was not kept: %v", err)
		}
	}

	// the user may pick the one the ranking would never take
	if err = d.SaveCandidate(common.Movie, video, second.Path, true); err != nil {
		t.Fatal(err)
	}
	saved, _ := filepath.Glob(filepath.Join(filepath.Dir(video), "*.srt"))
	if len(saved) != 1 || !strings.HasPrefix(filepath.Base(saved[0]), "The.Movie.2020.") {
		t.Fatalf("saved subtitles = %v, want one named after the video", saved)
	}
	if !d.ScanLogic.Get(0, video) {
		t.Fatal("video is not locked after a pick with lock")
	}
	// the lock is in the database: a new instance (app restart) still sees it
	if !scan_logic.NewScanLogic(d.log).Get(0, video) {
		t.Fatal("lock did not survive a restart")
	}
}

// A locked video is not downloaded even for a high priority job (UI 插队 / MoviePilot).
func TestQueueSkipsLockedVideoAtHighPriority(t *testing.T) {
	d, _ := newSearchDownloader(t)
	job := addJob(t, d, common.Movie, task_queue.HighTaskPriorityLevel)
	d.ScanLogic.Set(models.NewSkipScanInfoByMovie(job.VideoFPath, true))

	d.queueDownloaderLocal()

	if got := getJob(t, d, job.Id); got.JobStatus != taskQueue2.Ignore || got.DownloadTimes != 0 {
		t.Fatalf("locked job: status=%v downloads=%d, want Ignore and 0", got.JobStatus, got.DownloadTimes)
	}
	subs, _ := filepath.Glob(filepath.Join(filepath.Dir(job.VideoFPath), "*.srt"))
	if len(subs) != 0 {
		t.Fatalf("a subtitle was written for a locked video: %v", subs)
	}
}
