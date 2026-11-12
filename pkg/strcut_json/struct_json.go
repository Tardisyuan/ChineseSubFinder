package strcut_json

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// ToFile 注意传入的不是指针
func ToFile(srcJsonFileFPath string, input interface{}) error {
	jsonBytes, err := json.Marshal(input)
	if err != nil {
		return err
	}

	return WriteFileAtomic(filepath.FromSlash(srcJsonFileFPath), jsonBytes, true)
}

// WriteFileAtomic writes data to a temp file next to path and renames it over path, so a
// crash or kill in the middle leaves the previous file instead of a truncated one (a
// truncated settings file makes the next start panic). The parent folder is created.
// sync also flushes to disk before the rename, which covers power loss; leave it off for
// files that are rewritten very often.
func WriteFileAtomic(path string, data []byte, sync bool) error {
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return err
	}
	// Same mode bits os.Create would give (0666 minus umask), unlike os.CreateTemp's 0600.
	tmpPath := fmt.Sprintf("%s.%d.tmp", path, time.Now().UnixNano())
	file, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		return err
	}
	_, err = file.Write(data)
	if err == nil && sync {
		err = file.Sync()
	}
	if cerr := file.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(tmpPath, path)
	}
	if err != nil {
		_ = os.Remove(tmpPath)
	}
	return err
}

// ToStruct 传入的必须是指针
func ToStruct(desJsonFileFPath string, output interface{}) error {

	file, err := os.Open(filepath.FromSlash(desJsonFileFPath))
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	bytes, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	err = json.Unmarshal(bytes, output)
	if err != nil {
		return err
	}

	return nil
}
