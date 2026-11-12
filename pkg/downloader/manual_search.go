package downloader

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/ChineseSubFinder/ChineseSubFinder/internal/models"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/decode"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/language"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/series_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_parser/ass"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_parser/srt"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	subcommon "github.com/ChineseSubFinder/ChineseSubFinder/pkg/sub_formatter/common"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/sub_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/sub_parser_hub"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
	taskQueue2 "github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/task_queue"
)

// SubCandidate is one subtitle file found by a manual search, for the user to pick from.
type SubCandidate struct {
	ID        int      `json:"id"`
	Site      string   `json:"site"`       // supplier it came from
	Language  string   `json:"language"`   // 简体 / 简英 / ...
	IsChinese bool     `json:"is_chinese"` // the automatic ranking never picks non-Chinese ones
	Ext       string   `json:"ext"`
	Name      string   `json:"name"`
	Best      bool     `json:"best"`    // the one the automatic ranking would save
	Preview   []string `json:"preview"` // first dialogue lines
	Path      string   `json:"-"`
}

var sitePrefix = regexp.MustCompile(`^\[\w+\]_\d+_`)

// ass override tags ("{\fs14}") and line breaks ("\N"), dropped from the preview
var assTags = regexp.MustCompile(`\{[^}]*\}|\\[Nnh]`)

const previewLines = 8

// ManualSearchDir is where the candidates of the last manual search are kept. It is next
// to, not inside, the tmp folder, which is wiped after every download.
func ManualSearchDir() (string, error) {
	tmp, err := pkg.GetRootTmpFolder()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(tmp), "manual_search"), nil
}

// SearchCandidates runs the live suppliers for one video, the same way the queue does
// (supplier cache and daily limits apply), and returns every subtitle file found instead
// of saving the best one. The files are copied to ManualSearchDir, replacing the previous
// search. It holds the queue lock, so it waits for a running download and none starts
// meanwhile.
func (d *Downloader) SearchCandidates(videoType common.VideoType, videoFPath string) ([]SubCandidate, error) {
	d.downloaderLock.Lock()
	defer d.downloaderLock.Unlock()

	hub := d.subSupplierHub
	if hub == nil || len(hub.Suppliers) < 1 {
		return nil, errors.New("没有可用的字幕源：字幕源检测还未完成，或全部不可用/超出今日下载次数")
	}
	// same cleanup as after a queue job, still under the lock
	defer func() {
		if err := pkg.ClearRootTmpFolder(); err != nil {
			d.log.Errorln("ClearRootTmpFolder", err)
		}
	}()

	var files []string
	var err error
	if videoType == common.Movie {
		files, err = hub.DownloadSub4Movie(videoFPath, 0)
		if err != nil {
			return nil, err
		}
	} else {
		eps, err := decode.GetVideoNfoInfo4OneSeriesEpisode(videoFPath)
		if err != nil {
			return nil, fmt.Errorf("读取这一集的 nfo 信息失败: %v", err)
		}
		seriesDir := decode.GetSeriesDirRootFPath(videoFPath)
		if seriesDir == "" {
			return nil, fmt.Errorf("找不到连续剧的根目录（tvshow.nfo）: %s", videoFPath)
		}
		// forced: an episode that already has a subtitle is searched as well
		seriesInfo, err := series_helper.ReadSeriesInfoFromDir(d.fileDownloader.MediaInfoDealers, seriesDir,
			settings.Get().AdvancedSettings.TaskQueue.ExpirationTime, true, false,
			map[int][]int{eps.Season: {eps.Episode}})
		if err != nil {
			return nil, err
		}
		organized, err := hub.DownloadSub4Series(seriesDir, seriesInfo, 0)
		if err != nil {
			return nil, err
		}
		files = append(files, organized[pkg.GetEpisodeKeyName(eps.Season, eps.Episode)]...)
		// files of a full-season pack that belong to this episode
		for _, f := range organized[pkg.GetEpisodeKeyName(eps.Season, 0)] {
			_, s, e, err := decode.GetSeasonAndEpisodeFromSubFileName(filepath.Base(f))
			if err == nil && s == eps.Season && e == eps.Episode {
				files = append(files, f)
			}
		}
	}
	return d.keepCandidates(videoFPath, files)
}

// keepCandidates copies the found files out of the tmp folder and describes them.
func (d *Downloader) keepCandidates(videoFPath string, files []string) ([]SubCandidate, error) {
	keepDir, err := ManualSearchDir()
	if err != nil {
		return nil, err
	}
	if err = os.RemoveAll(keepDir); err != nil {
		return nil, err
	}
	if err = os.MkdirAll(keepDir, os.ModePerm); err != nil {
		return nil, err
	}
	best := d.mk.SelectOneSubFile(videoFPath, files)
	parser := sub_parser_hub.NewSubParserHub(d.log, ass.NewParser(d.log), srt.NewParser(d.log))

	out := make([]SubCandidate, 0)
	seen := make(map[string]bool)
	for _, f := range files {
		if seen[f] {
			continue
		}
		seen[f] = true
		ok, info, err := parser.DetermineFileTypeFromFile(f)
		if err != nil || !ok {
			d.log.Warnln("manual search: not a usable subtitle", f, err)
			continue
		}
		// the "[site]_N_" prefix is kept in the file name: the site is read back from it
		kept := filepath.Join(keepDir, fmt.Sprintf("%d", len(out)), filepath.Base(f))
		if err = os.MkdirAll(filepath.Dir(kept), os.ModePerm); err == nil {
			err = pkg.CopyFile(f, kept)
		}
		if err != nil {
			d.log.Errorln("manual search: keep", f, err)
			continue
		}
		c := SubCandidate{
			ID:        len(out),
			Site:      info.FromWhereSite,
			Language:  language.Lang2ChineseString(info.Lang),
			IsChinese: language.HasChineseLang(info.Lang),
			Ext:       info.Ext,
			Name:      sitePrefix.ReplaceAllString(filepath.Base(f), ""),
			Best:      best != nil && best.FileFullPath == f,
			Preview:   make([]string, 0),
			Path:      kept,
		}
		for _, dlg := range info.Dialogues {
			if len(c.Preview) >= previewLines {
				break
			}
			for _, line := range dlg.Lines {
				c.Preview = append(c.Preview, dlg.StartTime+"  "+strings.TrimSpace(assTags.ReplaceAllString(line, " ")))
			}
		}
		out = append(out, c)
	}
	// recommended first, then Chinese, then the rest; ID still names the file
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Best != out[j].Best {
			return out[i].Best
		}
		return out[i].IsChinese && !out[j].IsChinese
	})
	return out, nil
}

// SaveCandidate writes a picked subtitle next to the video the way the queue does: old
// default marks removed, normal naming, timeline fix / encoding / 简繁 settings applied.
// With lock the video is locked afterwards, so the automatic download leaves it alone.
func (d *Downloader) SaveCandidate(videoType common.VideoType, videoFPath, subFPath string, lock bool) error {
	ok, info, err := sub_parser_hub.NewSubParserHub(d.log, ass.NewParser(d.log), srt.NewParser(d.log)).DetermineFileTypeFromFile(subFPath)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("不支持的字幕格式: " + filepath.Base(subFPath))
	}
	if err = sub_helper.SearchVideoMatchSubFileAndRemoveExtMark(d.log, videoFPath); err != nil {
		d.log.Errorln("SearchVideoMatchSubFileAndRemoveExtMark,", videoFPath, err)
	}
	// same rule as oneVideoSelectBestSub: the "normal" formatter has no default mark
	setDefault := d.subNameFormatter != subcommon.Normal
	if err = d.SaveSubHelper.WriteSubFile2VideoPath(videoFPath, *info, "", setDefault, false); err != nil {
		return err
	}
	if lock {
		if videoType == common.Movie {
			d.ScanLogic.Set(models.NewSkipScanInfoByMovie(videoFPath, true))
		} else {
			d.ScanLogic.Set(models.NewSkipScanInfoBySeriesEx(videoFPath, true))
		}
	}
	job := taskQueue2.OneJob{VideoFPath: videoFPath, VideoType: videoType}
	d.refreshMediaServer(job)
	// a manual pick counts as a finished download in the history and notifications
	noteChosenSub(videoFPath, []string{subFPath}, *info)
	d.recordOutcome(job, nil)
	return nil
}
