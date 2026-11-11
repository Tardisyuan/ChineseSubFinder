package mark_system

import (
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_parser/ass"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_parser/srt"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/sub_parser_hub"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/subparser"
	"github.com/sirupsen/logrus"
)

// MarkingSystem 评价系统，解决字幕排序优先级问题。排序规则见 ranking.go，偏好读取自
// settings.AdvancedSettings（SubPriority、SubTypePriority），每次挑选时实时读取。
type MarkingSystem struct {
	log          *logrus.Logger
	subParserHub *sub_parser_hub.SubParserHub
}

func NewMarkingSystem(log *logrus.Logger) *MarkingSystem {
	return &MarkingSystem{
		log:          log,
		subParserHub: sub_parser_hub.NewSubParserHub(log, ass.NewParser(log), srt.NewParser(log)),
	}
}

func currentPrefs() (int, *settings.SubPriority) {
	adv := settings.Get().AdvancedSettings
	prio := adv.SubPriority
	if prio == nil {
		prio = settings.NewSubPriority()
	}
	return adv.SubTypePriority, prio
}

// SelectOneSubFile 选择最优的一个字幕文件
func (m MarkingSystem) SelectOneSubFile(videoPath string, organizeSubFiles []string) *subparser.FileInfo {
	subTypePriority, prio := currentPrefs()
	all := make([]subparser.FileInfo, 0)
	for _, infos := range m.parseSubFileInfo(organizeSubFiles) {
		all = append(all, infos...)
	}
	return pickBest(all, videoPath, subTypePriority, prio)
}

// SelectEachSiteTop1SubFile 每个网站最优的文件
func (m MarkingSystem) SelectEachSiteTop1SubFile(videoPath string, organizeSubFiles []string) ([]string, []subparser.FileInfo) {
	subTypePriority, prio := currentPrefs()
	var outSiteName = make([]string, 0)
	var outSubParserFileInfos = make([]subparser.FileInfo, 0)
	for siteName, infos := range m.parseSubFileInfo(organizeSubFiles) {
		if best := pickBest(infos, videoPath, subTypePriority, prio); best != nil {
			outSiteName = append(outSiteName, siteName)
			outSubParserFileInfos = append(outSubParserFileInfos, *best)
		}
	}
	return outSiteName, outSubParserFileInfos
}

// parseSubFileInfo 从文件解析字幕信息
func (m MarkingSystem) parseSubFileInfo(organizeSubFiles []string) map[string][]subparser.FileInfo {
	// 一个网站可能就算取了 Top1 字幕，也可能是返回一个压缩包，然后解压完就是多个字幕，所以
	var subInfoDict = make(map[string][]subparser.FileInfo)
	// 拿到现有的字幕列表，开始抉择
	// 先判断当前字幕是什么语言（如果是简体，还需要考虑，判断这个字幕是简体还是繁体）
	for _, oneSubFileFullPath := range organizeSubFiles {
		bFind, subFileInfo, err := m.subParserHub.DetermineFileTypeFromFile(oneSubFileFullPath)
		if err != nil {
			m.log.Errorln("DetermineFileTypeFromFile", oneSubFileFullPath, err)
			continue
		}
		if bFind == false {
			m.log.Warnln("DetermineFileTypeFromFile", oneSubFileFullPath, "not support SubType")
			continue
		}
		_, ok := subInfoDict[subFileInfo.FromWhereSite]
		if ok == false {
			// 新建
			subInfoDict[subFileInfo.FromWhereSite] = make([]subparser.FileInfo, 0)
		}
		// 添加
		subInfoDict[subFileInfo.FromWhereSite] = append(subInfoDict[subFileInfo.FromWhereSite], *subFileInfo)
	}
	return subInfoDict
}
