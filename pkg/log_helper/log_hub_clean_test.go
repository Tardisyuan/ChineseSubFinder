package log_helper

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
)

// The oldest logs go first, whatever their names sort like.
func TestCleanOnceLogsOldestFirst(t *testing.T) {
	logs := filepath.Join(pkg.ConfigRootDirFPath(), "Logs")
	if err := os.MkdirAll(logs, 0755); err != nil {
		t.Skip("config folder not writable here:", err)
	}
	if existing, _ := filepath.Glob(filepath.Join(logs, "Once-*.log")); len(existing) > 0 {
		t.Skip("real job logs present, not touching them")
	}
	defer func() {
		left, _ := filepath.Glob(filepath.Join(logs, "Once-*.log"))
		for _, p := range left {
			_ = os.Remove(p)
		}
	}()
	old := time.Now().Add(-time.Hour)
	for i := 0; i < onceLogMaxCount+2; i++ {
		p := filepath.Join(logs, fmt.Sprintf("Once-%05d.log", i))
		if err := os.WriteFile(p, nil, 0644); err != nil {
			t.Fatal(err)
		}
		// the two names that sort last are the oldest files
		if i >= onceLogMaxCount {
			_ = os.Chtimes(p, old, old)
		}
	}
	CleanAndLoadOnceLogs()
	left, _ := filepath.Glob(filepath.Join(logs, "Once-*.log"))
	if len(left) != onceLogMaxCount {
		t.Fatal("left", len(left))
	}
	if _, err := os.Stat(filepath.Join(logs, fmt.Sprintf("Once-%05d.log", onceLogMaxCount))); err == nil {
		t.Fatal("oldest log was kept")
	}
}
