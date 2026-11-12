package cache_center

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/supplier"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/cache_center/models"
)

func (c *CacheCenter) DownloadFileAdd(subInfo *supplier.SubInfo) error {
	defer c.locker.Unlock()
	c.locker.Lock()

	if subInfo.FileUrl == "" {
		return errors.New("subInfo FileUrl is empty")
	}

	// 只支持秒或者小时为单位
	tmpTTL := time.Duration(settings.Get().AdvancedSettings.DownloadFileCache.TTL) * time.Second
	if settings.Get().AdvancedSettings.DownloadFileCache.Unit == "hour" {
		tmpTTL = time.Duration(settings.Get().AdvancedSettings.DownloadFileCache.TTL) * time.Hour
	} else {
		tmpTTL = time.Duration(settings.Get().AdvancedSettings.DownloadFileCache.TTL) * time.Second
	}

	b, err := json.Marshal(subInfo)
	if err != nil {
		return err
	}

	// 保存到本地文件
	todayString := time.Now().Format("2006-01-02")
	saveFPath := filepath.Join(c.downloadFileSaveRootPath, todayString, subInfo.GetUID())
	err = pkg.WriteFile(saveFPath, b)
	if err != nil {
		return err
	}
	relPath, err := filepath.Rel(c.downloadFileSaveRootPath, saveFPath)
	if err != nil {
		return err
	}

	df := models.DownloadFileInfo{
		UID:            subInfo.GetUID(),
		RelPath:        relPath,
		ExpirationTime: time.Now().Add(tmpTTL),
	}

	// one row per uid: drop an expired or older entry for the same subtitle first
	c.db.Unscoped().Where("uid = ?", df.UID).Delete(&models.DownloadFileInfo{})
	c.db.Save(&df)
	if time.Since(c.lastDownloadFileCleanup) > 24*time.Hour {
		c.downloadFileCleanExpiredLocked()
	}

	return nil
}

// DownloadFileCleanExpired deletes cached downloads whose TTL (advanced_settings.download_file_cache,
// default 180 days) has passed, rows and files, plus empty day folders. Before this the TTL was
// stored but never applied, so download_files/ only grew. Runs at start and at most daily.
func (c *CacheCenter) DownloadFileCleanExpired() {
	defer c.locker.Unlock()
	c.locker.Lock()
	c.downloadFileCleanExpiredLocked()
}

func (c *CacheCenter) downloadFileCleanExpiredLocked() {
	c.lastDownloadFileCleanup = time.Now()
	var expired []models.DownloadFileInfo
	c.db.Where("expiration_time < ?", time.Now()).Find(&expired)
	for _, df := range expired {
		_ = os.Remove(filepath.Join(c.downloadFileSaveRootPath, df.RelPath))
		c.db.Unscoped().Delete(&df)
	}
	days, err := os.ReadDir(c.downloadFileSaveRootPath)
	if err != nil {
		return
	}
	for _, d := range days {
		if !d.IsDir() {
			continue
		}
		dir := filepath.Join(c.downloadFileSaveRootPath, d.Name())
		if entries, err := os.ReadDir(dir); err == nil && len(entries) == 0 {
			_ = os.Remove(dir)
		}
	}
	if len(expired) > 0 && c.Log != nil {
		c.Log.Infoln("download cache: removed", len(expired), "expired files")
	}
}

func (c *CacheCenter) DownloadFileGet(fileUrlUID string) (bool, *supplier.SubInfo, error) {
	defer c.locker.Unlock()
	c.locker.Lock()

	var dfs []models.DownloadFileInfo
	c.db.Where("uid = ?", fileUrlUID).Order("expiration_time desc").Find(&dfs)

	if len(dfs) == 0 {
		return false, nil, nil
	}
	if !dfs[0].ExpirationTime.IsZero() && time.Now().After(dfs[0].ExpirationTime) {
		// expired: download again (the cleanup removes the file later)
		return false, nil, nil
	}

	localFileFPath := filepath.Join(c.downloadFileSaveRootPath, dfs[0].RelPath)
	if pkg.IsFile(localFileFPath) == false {
		return false, nil, nil
	}

	bytes, err := os.ReadFile(localFileFPath)
	if err != nil {
		return false, nil, err
	}

	var subInfo supplier.SubInfo
	err = json.Unmarshal(bytes, &subInfo)
	if err != nil {
		return false, nil, err
	}

	return true, &subInfo, nil
}
