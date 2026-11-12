package strcut_json

import (
	"os"
	"path/filepath"
	"testing"
)

func TestToFileAtomic(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "sub", "a.json")
	if err := ToFile(p, map[string]int{"a": 1}); err != nil {
		t.Fatal(err)
	}
	if err := ToFile(p, map[string]int{"a": 2}); err != nil {
		t.Fatal(err)
	}
	out := map[string]int{}
	if err := ToStruct(p, &out); err != nil || out["a"] != 2 {
		t.Fatal(err, out)
	}
	// a failed write keeps the old content and leaves no temp file behind
	if err := os.Chmod(filepath.Dir(p), 0555); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(filepath.Dir(p), 0755)
	if os.Getuid() != 0 {
		if err := ToFile(p, map[string]int{"a": 3}); err == nil {
			t.Fatal("want error in read-only folder")
		}
	}
	if err := ToStruct(p, &out); err != nil || out["a"] != 2 {
		t.Fatal(err, out)
	}
	entries, _ := os.ReadDir(filepath.Dir(p))
	if len(entries) != 1 {
		t.Fatal("temp file left behind", entries)
	}
}
