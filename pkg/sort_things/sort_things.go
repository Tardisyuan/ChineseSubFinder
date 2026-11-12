package sort_things

import (
	"os"
	"sort"
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/decode"
)

type PathSlice struct {
	Path string
}
type PathSlices []PathSlice

func (a PathSlices) Len() int           { return len(a) }
func (a PathSlices) Less(i, j int) bool { return len(a[i].Path) < len(a[j].Path) }
func (a PathSlices) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

// SortStringSliceByLength 排序得到匹配上的路径，最长的那个
func SortStringSliceByLength(m []string) PathSlices {
	p := make(PathSlices, len(m))
	i := 0
	for _, v := range m {
		p[i] = PathSlice{v}
		i++
	}
	sort.Sort(sort.Reverse(p))
	return p
}

// -----------------------------------------------------------------------------

// SortByModTime 根据文件的 Mod Time 进行排序，递减
// Each file is stat'ed once up front; the old Less() stat'ed both files (twice each) on every
// comparison, i.e. ~4*n*log2(n) stats for n files, which dominated a library scan.
func SortByModTime(fileList []string) []string {

	byModTime := byModTime{paths: append(make([]string, 0, len(fileList)), fileList...), modTimes: make([]time.Time, len(fileList))}
	for i, p := range byModTime.paths {
		byModTime.modTimes[i] = GetFileModTime(p)
	}
	sort.Sort(sort.Reverse(byModTime))

	return byModTime.paths
}

type byModTime struct {
	paths    []string
	modTimes []time.Time
}

func (fis byModTime) Len() int {
	return len(fis.paths)
}

func (fis byModTime) Swap(i, j int) {
	fis.paths[i], fis.paths[j] = fis.paths[j], fis.paths[i]
	fis.modTimes[i], fis.modTimes[j] = fis.modTimes[j], fis.modTimes[i]
}

func (fis byModTime) Less(i, j int) bool {
	return fis.modTimes[i].Before(fis.modTimes[j])
}

func GetFileModTime(fileFPath string) time.Time {

	if IsFile(fileFPath) == true {
		// 存在
		fi, err := os.Stat(fileFPath)
		if err != nil {
			return time.Time{}
		}

		return fi.ModTime()
	} else {
		// 不存在才需要考虑蓝光情况
		bok, idBDMVFPath, _ := decode.IsFakeBDMVWorked(fileFPath)
		if bok == false {
			// 也不是蓝光
			return time.Time{}
		}
		// 获取这个蓝光 ID BDMV 文件的时间
		fInfo, err := os.Stat(idBDMVFPath)
		if err != nil {
			return time.Time{}
		}
		return fInfo.ModTime()
	}
}

// -----------------------------------------------------------------------------

// IsFile 存在且是文件
func IsFile(filePath string) bool {
	s, err := os.Stat(filePath)
	if err != nil {
		return false
	}
	return !s.IsDir()
}
