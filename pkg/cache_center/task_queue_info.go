package cache_center

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/strcut_json"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/cache_center/models"
)

func (c *CacheCenter) TaskQueueClear() error {

	// 没有必要删除 DB 中的数据，直接删除外部的缓存文件即可
	err := pkg.ClearFolder(c.taskQueueSaveRootPath)
	if err != nil {
		return err
	}
	return nil
}

func (c *CacheCenter) TaskQueueSave(taskPriority int, taskQueueBytes []byte) error {
	defer c.locker.Unlock()
	c.locker.Lock()

	var taskQueues []models.TaskQueueInfo
	c.db.Where("priority = ?", taskPriority).Find(&taskQueues)
	// 写入到本地存储
	saveFPath := filepath.Join(c.taskQueueSaveRootPath, fmt.Sprintf("%d", taskPriority)+".tq")
	// atomic: a kill in the middle of a plain write left a truncated queue file
	err := strcut_json.WriteFileAtomic(saveFPath, taskQueueBytes, false)
	if err != nil {
		return err
	}
	relPath, err := filepath.Rel(c.taskQueueSaveRootPath, saveFPath)
	if err != nil {
		return err
	}

	if len(taskQueues) == 0 {
		// 不存在，需要新建
		nowTaskQueue := models.TaskQueueInfo{
			Priority: taskPriority,
			RelPath:  relPath,
		}
		c.db.Save(&nowTaskQueue)
	} else if taskQueues[0].RelPath != relPath {
		// 存在，需要更新 (the path is fixed per priority, so normally there is nothing to write;
		// skipping the no-op UPDATE saves one sqlite commit per queue change)
		taskQueues[0].RelPath = relPath
		c.db.Save(&taskQueues[0])
	}

	return nil
}

func (c *CacheCenter) TaskQueueRead() (map[int][]byte, error) {
	defer c.locker.Unlock()
	c.locker.Lock()

	var taskQueues []models.TaskQueueInfo
	c.db.Find(&taskQueues)

	outTaskQueueBytes := make(map[int][]byte, 0)
	for _, taskQueue := range taskQueues {

		oneTaskQueueFPath := filepath.Join(c.taskQueueSaveRootPath, taskQueue.RelPath)
		if pkg.IsFile(oneTaskQueueFPath) == false {
			continue
		}
		bytes, err := os.ReadFile(oneTaskQueueFPath)
		if err != nil {
			return nil, err
		}

		outTaskQueueBytes[taskQueue.Priority] = bytes
	}

	return outTaskQueueBytes, nil
}
