package archive_helper

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mholt/archiver/v3"
)

type fakeInfo struct{ name string }

func (f fakeInfo) Name() string       { return f.name }
func (f fakeInfo) Size() int64        { return 1 }
func (f fakeInfo) Mode() os.FileMode  { return 0644 }
func (f fakeInfo) ModTime() time.Time { return time.Time{} }
func (f fakeInfo) IsDir() bool        { return false }
func (f fakeInfo) Sys() interface{}   { return nil }

func entry(name string) archiver.File {
	return archiver.File{FileInfo: fakeInfo{name}, ReadCloser: io.NopCloser(strings.NewReader("x"))}
}

// rar entries keep their in-archive path; one starting with ../ must not leave the folder.
func TestProcessOneFileRejectsPathTraversal(t *testing.T) {
	root := t.TempDir()
	des := filepath.Join(root, "a", "unzip")
	if err := processOneFile(entry("../../evil.ass"), false, des); err == nil {
		t.Fatal("entry outside of the target folder was accepted")
	}
	if _, err := os.Stat(filepath.Join(root, "evil.ass")); err == nil {
		t.Fatal("file was written outside of the target folder")
	}
	// nested folders inside the archive still work
	if err := processOneFile(entry("Season 1/chs/a.ass"), false, des); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(des, "Season 1", "chs", "a.ass")); err != nil {
		t.Fatal(err)
	}
}

func TestIsWantedArchiveExtName(t *testing.T) {
	for _, name := range []string{"a.zip", "a.tar", "a.RAR", "a.7z"} {
		if !IsWantedArchiveExtName(name) {
			t.Fatal(name, "should be an archive")
		}
	}
	if IsWantedArchiveExtName("a.ass") {
		t.Fatal("a.ass is not an archive")
	}
}
