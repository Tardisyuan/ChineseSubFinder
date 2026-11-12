package cache_center

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/cache_center/models"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/language"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/supplier"
	"github.com/sirupsen/logrus"
)

// Expired downloads are not served and are deleted (row and file); a fresh save of the same
// subtitle replaces the old row.
func TestDownloadFileTTL(t *testing.T) {
	dir := t.TempDir()
	old, _ := os.Getwd()
	_ = os.Chdir(dir)
	defer os.Chdir(old)
	settings.SetConfigRootPath(filepath.Join(dir, "config"))

	c := NewCacheCenter("ttl_test", logrus.New())
	sub := supplier.NewSubInfo("assrt", 0, "v.mkv", language.ChineseSimple, "https://x/a.srt", 0, 0, ".srt", []byte("data"))
	if err := c.DownloadFileAdd(sub); err != nil {
		t.Fatal(err)
	}
	if found, _, _ := c.DownloadFileGet(sub.GetUID()); !found {
		t.Fatal("fresh entry not found")
	}
	var row models.DownloadFileInfo
	c.db.Where("uid = ?", sub.GetUID()).First(&row)
	file := filepath.Join(c.downloadFileSaveRootPath, row.RelPath)
	c.db.Model(&row).Update("expiration_time", time.Now().Add(-time.Hour))

	if found, _, _ := c.DownloadFileGet(sub.GetUID()); found {
		t.Fatal("expired entry served")
	}
	c.DownloadFileCleanExpired()
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Fatalf("expired file still there: %v", err)
	}
	if err := c.DownloadFileAdd(sub); err != nil {
		t.Fatal(err)
	}
	var n int64
	c.db.Model(&models.DownloadFileInfo{}).Where("uid = ?", sub.GetUID()).Count(&n)
	if found, _, _ := c.DownloadFileGet(sub.GetUID()); !found || n != 1 {
		t.Fatalf("re-added: found=%v rows=%d", found, n)
	}
}
