package v1

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/downloader"
	backend2 "github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/backend"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
	task_queue3 "github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/task_queue"
	"github.com/gin-gonic/gin"
)

// In-app subtitle search with manual pick. A search runs the live suppliers and can take
// tens of seconds, so it is a background task the UI polls; only one runs at a time and
// only the last result is kept.

const (
	searchIdle    = "idle"
	searchRunning = "running"
	searchDone    = "done"
	searchFailed  = "failed"

	// reply message of the add-job endpoints for a locked video (字幕已满意，不再自动更换)
	msgVideoLocked = "video is locked"
)

// searchState is what the status endpoint returns.
type searchState struct {
	Status     string                    `json:"status"`
	VideoFPath string                    `json:"video_f_path"`
	VideoType  int                       `json:"video_type"` // 0 movie, 1 series
	Error      string                    `json:"error"`
	Candidates []downloader.SubCandidate `json:"candidates"`
}

// searchTask: idle -> running -> done | failed, and from done/failed to running again.
type searchTask struct {
	mu    sync.Mutex
	state searchState
}

var subSearch = &searchTask{}

// start begins a search unless one is running; run is called in its own goroutine.
func (s *searchTask) start(videoFPath string, videoType int, run func() ([]downloader.SubCandidate, error)) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state.Status == searchRunning {
		return false
	}
	s.state = searchState{Status: searchRunning, VideoFPath: videoFPath, VideoType: videoType}
	go func() {
		var candidates []downloader.SubCandidate
		var err error
		func() {
			defer func() {
				if p := recover(); p != nil {
					err = fmt.Errorf("panic while searching: %v", p)
				}
			}()
			candidates, err = run()
		}()
		s.mu.Lock()
		defer s.mu.Unlock()
		if err != nil {
			s.state.Status, s.state.Error = searchFailed, err.Error()
			return
		}
		s.state.Status, s.state.Candidates = searchDone, candidates
	}()
	return true
}

func (s *searchTask) snapshot() searchState {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.state
	if st.Status == "" {
		st.Status = searchIdle
	}
	if st.Candidates == nil {
		st.Candidates = make([]downloader.SubCandidate, 0)
	}
	return st
}

// candidate returns a result of the finished search for videoFPath.
func (s *searchTask) candidate(videoFPath string, id int) (downloader.SubCandidate, searchState, bool) {
	st := s.snapshot()
	if st.Status == searchDone && st.VideoFPath == videoFPath {
		for _, c := range st.Candidates {
			if c.ID == id {
				return c, st, true
			}
		}
	}
	return downloader.SubCandidate{}, st, false
}

func toVideoType(t int) common.VideoType {
	if t == 1 {
		return common.Series
	}
	return common.Movie
}

// SubSearchStartHandler POST {video_f_path, video_type}: message is "ok", or "busy" with
// the video being searched.
func (cb *ControllerBase) SubSearchStartHandler(c *gin.Context) {
	var err error
	defer func() {
		cb.ErrorProcess(c, "SubSearchStartHandler", err)
	}()
	req := struct {
		VideoFPath string `json:"video_f_path"`
		VideoType  int    `json:"video_type"`
	}{}
	if err = c.ShouldBindJSON(&req); err != nil {
		return
	}
	if pkg.IsFile(req.VideoFPath) == false {
		err = fmt.Errorf("video file not found: %s", req.VideoFPath)
		return
	}
	msg := "ok"
	if subSearch.start(req.VideoFPath, req.VideoType, func() ([]downloader.SubCandidate, error) {
		return cb.cronHelper.Downloader.SearchCandidates(toVideoType(req.VideoType), req.VideoFPath)
	}) == false {
		msg = "busy"
	}
	c.JSON(http.StatusOK, gin.H{"message": msg, "video_f_path": subSearch.snapshot().VideoFPath})
}

// SubSearchStatusHandler GET: state of the current or last search.
func (cb *ControllerBase) SubSearchStatusHandler(c *gin.Context) {
	c.JSON(http.StatusOK, subSearch.snapshot())
}

// SubSearchPickHandler POST {video_f_path, id, lock}: saves one result of the last search
// next to its video. A queued job for the video is marked done.
func (cb *ControllerBase) SubSearchPickHandler(c *gin.Context) {
	var err error
	defer func() {
		cb.ErrorProcess(c, "SubSearchPickHandler", err)
	}()
	req := struct {
		VideoFPath string `json:"video_f_path"`
		ID         int    `json:"id"`
		Lock       bool   `json:"lock"`
	}{}
	if err = c.ShouldBindJSON(&req); err != nil {
		return
	}
	picked, st, ok := subSearch.candidate(req.VideoFPath, req.ID)
	if ok == false {
		err = fmt.Errorf("搜索结果已失效，请重新搜索")
		return
	}
	if err = cb.cronHelper.Downloader.SaveCandidate(toVideoType(st.VideoType), st.VideoFPath, picked.Path, req.Lock); err != nil {
		return
	}
	if _, jobs, _ := cb.cronHelper.DownloadQueue.GetAllJobs(); jobs != nil {
		for _, job := range jobs {
			if job.VideoFPath == st.VideoFPath && job.JobStatus != task_queue3.Done && job.JobStatus != task_queue3.Downloading {
				job.JobStatus, job.ErrorInfo = task_queue3.Done, ""
				if _, err2 := cb.cronHelper.DownloadQueue.Update(job); err2 != nil {
					cb.log.Errorln("SubSearchPickHandler update job", err2)
				}
			}
		}
	}
	c.JSON(http.StatusOK, backend2.ReplyCommon{Message: "ok"})
}
