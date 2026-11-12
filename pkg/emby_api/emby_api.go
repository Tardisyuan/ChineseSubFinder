package emby_api

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/emby"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/go-resty/resty/v2"
	"github.com/panjf2000/ants/v2"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

type EmbyApi struct {
	log     *logrus.Logger
	timeOut time.Duration
}

func NewEmbyApi(log *logrus.Logger) *EmbyApi {
	em := EmbyApi{}
	em.log = log
	// 检查是否超过范围
	settings.Get().Check()
	// 强制设置
	em.timeOut = 5 * 60 * time.Second
	return &em
}

// RefreshRecentlyVideoInfo 字幕下载完毕一次，就可以触发一次这个。并发 6 线程去刷新
func (em *EmbyApi) RefreshRecentlyVideoInfo(embySettings *settings.EmbySettings, SkipWatched bool, maxRequestVideoNumber int) error {
	items, err := em.GetRecentlyItems(embySettings, SkipWatched, maxRequestVideoNumber)
	if err != nil {
		return err
	}

	em.log.Debugln("RefreshRecentlyVideoInfo - GetRecentlyItems Count", len(items.Items))

	updateFunc := func(i interface{}) error {
		tmpId := i.(string)
		return em.UpdateVideoSubList(embySettings, tmpId)
	}
	p, err := ants.NewPoolWithFunc(embySettings.Threads, func(inData interface{}) {
		data := inData.(InputData)
		defer data.Wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), em.timeOut)
		defer cancel()

		done := make(chan error, 1)
		panicChan := make(chan interface{}, 1)

		go func() {
			defer func() {
				if p := recover(); p != nil {
					panicChan <- p
				}

			}()

			done <- updateFunc(data.Id)
		}()

		select {
		case err = <-done:
			if err != nil {
				em.log.Errorln("RefreshRecentlyVideoInfo.NewPoolWithFunc got error", err)
			}
			return
		case p := <-panicChan:
			em.log.Errorln("RefreshRecentlyVideoInfo.NewPoolWithFunc got panic", p)
		case <-ctx.Done():
			em.log.Errorln("RefreshRecentlyVideoInfo.NewPoolWithFunc got time out", ctx.Err())
			return
		}
	})
	if err != nil {
		return err
	}
	defer p.Release()
	wg := sync.WaitGroup{}
	for _, item := range items.Items {
		wg.Add(1)
		err = p.Invoke(InputData{Id: item.Id, Wg: &wg})
		if err != nil {
			em.log.Errorln("RefreshRecentlyVideoInfo ants.Invoke", err)
		}
	}
	wg.Wait()

	return nil
}

func (em *EmbyApi) GetRecentItemsByUserID(embySettings *settings.EmbySettings, userId string, maxRequestVideoNumber int) (emby.EmbyRecentlyItems, error) {

	var tmpRecItems emby.EmbyRecentlyItems
	// 获取指定用户的视频列表
	_, err := em.newRequest(embySettings).
		SetQueryParams(map[string]string{
			"IsUnaired":        "false",
			"Limit":            fmt.Sprintf("%d", maxRequestVideoNumber),
			"Recursive":        "true",
			"SortOrder":        "Descending",
			"IncludeItemTypes": "Episode,Movie",
			"Filters":          "IsNotFolder",
			"SortBy":           "DateCreated",
		}).
		SetResult(&tmpRecItems).
		Get(em.userItemsUrl(embySettings, userId, ""))
	if err != nil {
		return emby.EmbyRecentlyItems{}, err
	}

	return tmpRecItems, nil
}

// GetRecentlyItems 获取近期的视频(根据 SkipWatched 的情况，如果不跳过，那么就是获取所有用户的列表，如果是跳过，那么就会单独读取每个用户的再交叉判断)
// 在 API 调试界面 -- ItemsService
func (em *EmbyApi) GetRecentlyItems(embySettings *settings.EmbySettings, SkipWatched bool, maxRequestVideoNumber int) (emby.EmbyRecentlyItems, error) {

	var recItems emby.EmbyRecentlyItems
	recItems.Items = make([]emby.EmbyRecentlyItem, 0)
	var recItemMap = make(map[string]emby.EmbyRecentlyItem)
	var recItemExsitMap = make(map[string]emby.EmbyRecentlyItem)
	var err error
	if SkipWatched == false {
		em.log.Debugln("Emby Setting SkipWatched = false")

		// 默认是不指定某一个User的视频列表
		_, err = em.newRequest(embySettings).
			SetQueryParams(map[string]string{
				"IsUnaired":        "false",
				"Limit":            fmt.Sprintf("%d", maxRequestVideoNumber),
				"Recursive":        "true",
				"SortOrder":        "Descending",
				"IncludeItemTypes": "Episode,Movie",
				"Filters":          "IsNotFolder",
				"SortBy":           "DateCreated",
			}).
			SetResult(&recItems).
			Get(em.url(embySettings, "/Items"))

		if err != nil {
			return emby.EmbyRecentlyItems{}, err
		}
	} else {
		em.log.Debugln("Emby Setting SkipWatched = true")

		var userIds emby.EmbyUsers
		userIds, err = em.GetUserIdList(embySettings)
		if err != nil {
			return emby.EmbyRecentlyItems{}, err
		}

		for _, item := range userIds.Items {

			tmpRecItems, err := em.GetRecentItemsByUserID(embySettings, item.Id, maxRequestVideoNumber)
			if err != nil {
				return emby.EmbyRecentlyItems{}, err
			}
			// 相同的视频项目，需要判断是否已经看过了，看过的需要排除
			// 项目是否相同可以通过 Id 判断
			for _, recentlyItem := range tmpRecItems.Items {
				// Skip a video that any user has watched. The old bookkeeping re-added every
				// item at the end of the loop, so a video watched only by the first user was
				// never skipped.
				if recentlyItem.UserData.Played {
					recItemExsitMap[recentlyItem.Id] = recentlyItem
				}
				recItemMap[recentlyItem.Id] = recentlyItem
			}
		}

		for id := range recItemExsitMap {
			em.log.Debugln("Skip Watched Video:", recItemMap[id].Type, recItemMap[id].Name)
			delete(recItemMap, id)
		}

		for _, item := range recItemMap {
			recItems.Items = append(recItems.Items, item)
		}

		recItems.TotalRecordCount = len(recItemMap)
	}

	return recItems, nil
}

// GetUserIdList 获取所有的 UserId
func (em *EmbyApi) GetUserIdList(embySettings *settings.EmbySettings) (emby.EmbyUsers, error) {
	var recItems emby.EmbyUsers

	if embySettings.IsJellyfin() {
		// Jellyfin has no /Users/Query; GET /Users returns a bare array.
		var users []struct {
			Name string `json:"Name"`
			Id   string `json:"Id"`
		}
		_, err := em.newRequest(embySettings).SetResult(&users).Get(em.url(embySettings, "/Users"))
		if err != nil {
			return emby.EmbyUsers{}, err
		}
		for _, u := range users {
			recItems.Items = append(recItems.Items, struct {
				Name string `json:"Name"`
				Id   string `json:"Id"`
			}{Name: u.Name, Id: u.Id})
		}
		recItems.TotalRecordCount = len(recItems.Items)
		return recItems, nil
	}

	_, err := em.newRequest(embySettings).
		SetResult(&recItems).
		Get(em.url(embySettings, "/Users/Query"))

	if err != nil {
		return emby.EmbyUsers{}, err
	}
	return recItems, nil
}

// GetItemAncestors 获取父级信息，在 API 调试界面 -- LibraryService
func (em *EmbyApi) GetItemAncestors(embySettings *settings.EmbySettings, id string) ([]emby.EmbyItemsAncestors, error) {

	var recItems []emby.EmbyItemsAncestors

	_, err := em.newRequest(embySettings).
		SetResult(&recItems).
		Get(em.url(embySettings, "/Items/") + id + "/Ancestors")
	if err != nil {
		return nil, err
	}

	return recItems, nil
}

// GetItemVideoInfo 在 API 调试界面 -- UserLibraryService，如果是电影，那么是可以从 ProviderIds 得到 IMDB ID 的
// 如果是连续剧，那么不能使用一集的ID取获取，需要是这个剧集的 ID，注意一季的ID也是不行的
func (em *EmbyApi) GetItemVideoInfo(embySettings *settings.EmbySettings, id string) (emby.EmbyVideoInfo, error) {

	var recItem emby.EmbyVideoInfo

	// Emby answers item lookups on /LiveTv/Programs/{id}; Jellyfin uses /Items/{id}.
	itemPath := "/LiveTv/Programs/"
	if embySettings.IsJellyfin() {
		itemPath = "/Items/"
	}
	_, err := em.newRequest(embySettings).
		SetResult(&recItem).
		Get(em.url(embySettings, itemPath) + id)
	if err != nil {
		return emby.EmbyVideoInfo{}, err
	}

	return recItem, nil
}

// GetItemVideoInfoByUserId 可以拿到这个视频的选择字幕Index，配合 GetItemVideoInfo 使用。 在 API 调试界面 -- UserLibraryService
func (em *EmbyApi) GetItemVideoInfoByUserId(embySettings *settings.EmbySettings, userId, videoId string) (emby.EmbyVideoInfoByUserId, error) {

	var recItem emby.EmbyVideoInfoByUserId

	_, err := em.newRequest(embySettings).
		SetResult(&recItem).
		Get(em.userItemsUrl(embySettings, userId, videoId))
	if err != nil {
		return emby.EmbyVideoInfoByUserId{}, err
	}

	return recItem, nil
}

// UpdateVideoSubList 更新字幕列表， 在 API 调试界面 -- ItemRefreshService
func (em *EmbyApi) UpdateVideoSubList(embySettings *settings.EmbySettings, id string) error {

	resp, err := em.newRequest(embySettings).
		SetQueryParams(map[string]string{
			"Recursive": "true",
		}).
		Post(em.url(embySettings, "/Items/") + id + "/Refresh")
	if err != nil {
		return err
	}
	// 401 (bad key) and 404 (item gone) are not transport errors; without this check the
	// caller logged the refresh as done.
	if resp.StatusCode() >= 300 {
		return fmt.Errorf("%s /Items/%s/Refresh: HTTP %d %s", embySettings.ServerName(), id, resp.StatusCode(), resp.String())
	}

	return nil
}

// NotifyPathUpdated tells Emby/Jellyfin that something under path changed, so it rescans
// just that folder and picks up new subtitles. Works without the media server's item ID.
// path must be the path as the media server sees it.
func (em *EmbyApi) NotifyPathUpdated(embySettings *settings.EmbySettings, path string) error {
	body := map[string]interface{}{
		"Updates": []map[string]string{{"Path": path, "UpdateType": "Modified"}},
	}
	resp, err := em.newRequest(embySettings).
		SetHeader("Content-Type", "application/json").
		SetBody(body).
		Post(em.url(embySettings, "/Library/Media/Updated"))
	if err != nil {
		return err
	}
	if resp.StatusCode() >= 300 {
		return fmt.Errorf("%s /Library/Media/Updated: HTTP %d %s", embySettings.ServerName(), resp.StatusCode(), resp.String())
	}
	return nil
}

// GetSubFileData 下载字幕 subExt -> .ass or .srt , 在 API 调试界面 -- SubtitleService
func (em *EmbyApi) GetSubFileData(embySettings *settings.EmbySettings, videoId, mediaSourceId, subIndex, subExt string) (string, error) {

	response, err := em.newRequest(embySettings).
		Get(em.url(embySettings, "/Videos/") + videoId + "/" + mediaSourceId + "/Subtitles/" + subIndex + "/Stream" + subExt)
	if err != nil {
		return "", err
	}

	return response.String(), nil
}

// url builds an API URL. Emby serves its API under /emby; Jellyfin serves it at the root.
func (em *EmbyApi) url(embySettings *settings.EmbySettings, path string) string {
	base := strings.TrimRight(embySettings.AddressUrl, "/")
	if embySettings.IsJellyfin() {
		return base + path
	}
	return base + "/emby" + path
}

// userItemsUrl is a user's item list (itemId == "") or one item as seen by that user.
// Jellyfin 10.9+ moved these from /Users/{userId}/Items to /Items?userId=.
func (em *EmbyApi) userItemsUrl(embySettings *settings.EmbySettings, userId, itemId string) string {
	if embySettings.IsJellyfin() {
		path := "/Items"
		if itemId != "" {
			path += "/" + itemId
		}
		return em.url(embySettings, path) + "?userId=" + url.QueryEscape(userId)
	}
	path := "/Users/" + userId + "/Items"
	if itemId != "" {
		path += "/" + itemId
	}
	return em.url(embySettings, path)
}

// newRequest attaches the API key. Jellyfin 10.11 disables the legacy api_key query
// parameter by default, so it gets the Authorization header instead.
func (em *EmbyApi) newRequest(embySettings *settings.EmbySettings) *resty.Request {
	r := em.createClient().R()
	if embySettings.IsJellyfin() {
		return r.SetHeader("Authorization", `MediaBrowser Token="`+embySettings.APIKey+`"`)
	}
	return r.SetQueryParam("api_key", embySettings.APIKey)
}

// embyTransport is shared by every Emby/Jellyfin request. Upstream (issue #140) built a new
// transport per request and turned keep-alive off so idle connections would not pile up; with
// one shared, bounded pool keep-alive is safe and a refresh of many items reuses connections.
// The media server is on the LAN, so no proxy.
var embyTransport = &http.Transport{
	Proxy:               nil,
	MaxIdleConns:        20,
	MaxIdleConnsPerHost: 10,
	IdleConnTimeout:     90 * time.Second,
}

func (em *EmbyApi) createClient() *resty.Client {
	return resty.New().SetTransport(embyTransport).SetTimeout(em.timeOut)
}

type InputData struct {
	Id string
	Wg *sync.WaitGroup
}
