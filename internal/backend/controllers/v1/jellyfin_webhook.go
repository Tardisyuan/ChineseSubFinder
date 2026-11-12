package v1

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/decode"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/emby_api"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/task_queue"
	backend2 "github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/backend"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
	TTaskqueue "github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/task_queue"
	"github.com/gin-gonic/gin"
)

// flexInt reads a JSON number or a numeric string; a template renders numbers either way.
type flexInt int

func (f *flexInt) UnmarshalJSON(data []byte) error {
	s := strings.Trim(strings.TrimSpace(string(data)), `"`)
	if s == "" || s == "null" {
		*f = 0
		return nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return fmt.Errorf("not a number: %s", data)
	}
	*f = flexInt(n)
	return nil
}

// jellyfinWebhookReq is what the endpoint reads from the Jellyfin Webhook plugin. With
// "Send All Properties" the plugin sends these fields (and many more, which are ignored).
// The plugin has no file path variable, so the path is asked from Jellyfin by ItemId; other
// callers may send Path (the path as the media server sees it) and skip that lookup.
type jellyfinWebhookReq struct {
	NotificationType string  `json:"NotificationType"` // only "ItemAdded" (or empty) is handled
	ItemType         string  `json:"ItemType"`         // "Movie" or "Episode", others are ignored
	ItemId           string  `json:"ItemId"`
	Name             string  `json:"Name"`
	SeriesName       string  `json:"SeriesName"`
	SeasonNumber     flexInt `json:"SeasonNumber"`  // used when the episode has no nfo
	EpisodeNumber    flexInt `json:"EpisodeNumber"` // used when the episode has no nfo
	Path             string  `json:"Path"`
}

// localPathFromServer maps a path as the media server sees it to the path CSF sees: the
// reverse of EmbySettings.ServerPath. The longest matching server root wins; a path under no
// mapped root is returned unchanged (both see the same path).
func localPathFromServer(em *settings.EmbySettings, serverPath string) string {
	bestServer, bestLocal := "", ""
	for _, m := range []map[string]string{em.MoviePathsMapping, em.SeriesPathsMapping} {
		for local, server := range m {
			s := strings.TrimRight(server, "/")
			if s == "" || strings.TrimRight(local, "/") == "" {
				continue
			}
			if (serverPath == s || strings.HasPrefix(serverPath, s+"/")) && len(s) > len(bestServer) {
				bestServer, bestLocal = s, strings.TrimRight(local, "/")
			}
		}
	}
	if bestServer == "" {
		return serverPath
	}
	return bestLocal + strings.TrimPrefix(serverPath, bestServer)
}

// jellyfinJob turns a webhook payload into a queue job. lookupPath asks the media server for
// the item's path. A nil job with status 200 means "nothing to do for this payload".
func jellyfinJob(req jellyfinWebhookReq, em *settings.EmbySettings, lookupPath func(itemId string) (string, error)) (*TTaskqueue.OneJob, int, string) {
	if req.NotificationType != "" && req.NotificationType != "ItemAdded" {
		return nil, http.StatusOK, "ignored: NotificationType " + req.NotificationType
	}
	if req.ItemType != "Movie" && req.ItemType != "Episode" {
		return nil, http.StatusOK, "ignored: ItemType " + req.ItemType
	}
	serverPath := req.Path
	if serverPath == "" {
		if req.ItemId == "" {
			return nil, http.StatusBadRequest, "payload has neither ItemId nor Path"
		}
		var err error
		if serverPath, err = lookupPath(req.ItemId); err != nil {
			return nil, http.StatusBadGateway, "item path lookup failed: " + err.Error()
		}
		if serverPath == "" {
			return nil, http.StatusBadGateway, "media server returned no path for item " + req.ItemId
		}
	}
	localPath := localPathFromServer(em, serverPath)
	if !pkg.IsFile(localPath) {
		return nil, http.StatusNotFound, fmt.Sprintf(
			"video file not found: %s (media server path: %s); check the path mapping in the media server settings", localPath, serverPath)
	}
	// High priority like "立即下载" in the web UI: a new item is searched right away and is
	// not filtered by the scan rules for old videos.
	if req.ItemType == "Movie" {
		return TTaskqueue.NewOneJob(common.Movie, localPath, task_queue.HighTaskPriorityLevel, req.ItemId), http.StatusOK, "ok"
	}
	seriesRoot := decode.GetSeriesDirRootFPath(localPath)
	if seriesRoot == "" {
		return nil, http.StatusUnprocessableEntity, "tvshow.nfo not found two folders above " + localPath
	}
	season, episode := int(req.SeasonNumber), int(req.EpisodeNumber)
	if nfo, err := decode.GetVideoNfoInfo4OneSeriesEpisode(localPath); err == nil {
		season, episode = nfo.Season, nfo.Episode
	} else if season < 1 || episode < 1 {
		// the queue drops series jobs without a season and episode number (specials included)
		return nil, http.StatusUnprocessableEntity, "episode has no nfo and the payload has no SeasonNumber/EpisodeNumber above 0: " + localPath
	}
	job := TTaskqueue.NewOneJob(common.Series, localPath, task_queue.HighTaskPriorityLevel, req.ItemId)
	job.Season, job.Episode, job.SeriesRootDirPath = season, episode, seriesRoot
	return job, http.StatusOK, "ok"
}

// apiKeyOk checks the /api/v1 key from the Authorization header ("Bearer <key>") or, for
// senders that cannot set headers, from the api_key query parameter.
func apiKeyOk(c *gin.Context) bool {
	key := c.Query("api_key")
	if fields := strings.Fields(c.GetHeader("Authorization")); len(fields) == 2 {
		key = fields[1]
	}
	want := settings.Get().ExperimentalFunction.ApiKeySettings
	return want.Enabled && want.Key != "" && subtle.ConstantTimeCompare([]byte(key), []byte(want.Key)) == 1
}

// JellyfinWebhookHandler POST /api/v1/jellyfin-webhook[?api_key=<key>]
// Receives "Item Added" from the Jellyfin Webhook plugin and queues a subtitle job for the
// new movie or episode, like /api/v1/add-job does.
func (cb *ControllerBase) JellyfinWebhookHandler(c *gin.Context) {
	if !apiKeyOk(c) {
		c.JSON(http.StatusUnauthorized, backend2.ReplyCheckAuth{Message: "api key error, or api key not enabled"})
		return
	}
	req := jellyfinWebhookReq{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, backend2.ReplyJobThings{Message: "bad payload: " + err.Error()})
		return
	}
	em := settings.Get().EmbySettings
	job, status, msg := jellyfinJob(req, em, func(itemId string) (string, error) {
		if !em.Enable || em.AddressUrl == "" || em.APIKey == "" {
			return "", fmt.Errorf("media server is not configured in the settings")
		}
		info, err := emby_api.NewEmbyApi(cb.log).GetItemVideoInfo(em, itemId)
		return info.Path, err
	})
	if job == nil {
		cb.log.Infoln("JellyfinWebhook:", req.ItemType, req.SeriesName, req.Name, req.ItemId, "->", msg)
		c.JSON(status, backend2.ReplyJobThings{Message: msg})
		return
	}
	added, err := cb.cronHelper.DownloadQueue.Add(*job)
	if err != nil {
		cb.ErrorProcess(c, "JellyfinWebhookHandler", err)
		return
	}
	if !added {
		msg = "job is already in queue"
	}
	cb.log.Infoln("JellyfinWebhook:", req.ItemType, job.VideoFPath, "->", msg)
	c.JSON(http.StatusOK, backend2.ReplyJobThings{JobID: job.Id, Message: msg})
}
