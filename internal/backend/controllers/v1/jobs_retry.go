package v1

import (
	"net/http"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/task_queue"
	backend2 "github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/backend"
	task_queue3 "github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/task_queue"
	"github.com/gin-gonic/gin"
)

// RetryJobHandler POST {id}: retry a failed job now. It goes back to waiting at the high
// priority (taken on the next queue tick, no 12h interval) with error and retry count
// cleared. change-job-status can also set waiting + high, but keeps the old error.
func (cb *ControllerBase) RetryJobHandler(c *gin.Context) {
	var err error
	defer func() {
		cb.ErrorProcess(c, "RetryJobHandler", err)
	}()
	req := backend2.ReqJobLog{}
	if err = c.ShouldBindJSON(&req); err != nil {
		return
	}
	found, job := cb.cronHelper.DownloadQueue.GetOneJobByID(req.Id)
	if found == false {
		c.JSON(http.StatusOK, backend2.ReplyCommon{Message: "job not found"})
		return
	}
	bok, err := cb.cronHelper.DownloadQueue.Update(retryJob(job))
	if err != nil {
		return
	}
	if bok == false {
		c.JSON(http.StatusOK, backend2.ReplyCommon{Message: "update job status failed"})
		return
	}
	c.JSON(http.StatusOK, backend2.ReplyCommon{Message: "ok"})
}

// retryJob puts a job back to waiting ahead of the normal ones, with its failure cleared.
func retryJob(job task_queue3.OneJob) task_queue3.OneJob {
	job.JobStatus = task_queue3.Waiting
	job.TaskPriority = task_queue.HighTaskPriorityLevel
	job.RetryTimes = 0
	job.ErrorInfo = ""
	return job
}
