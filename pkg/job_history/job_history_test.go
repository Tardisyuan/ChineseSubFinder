package job_history

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func useTempDb(t *testing.T) *gorm.DB {
	t.Helper()
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "h.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = d.AutoMigrate(&JobHistory{}); err != nil {
		t.Fatal(err)
	}
	old := getDb
	getDb = func() *gorm.DB { return d }
	t.Cleanup(func() {
		getDb = old
		if s, err := d.DB(); err == nil {
			_ = s.Close()
		}
	})
	return d
}

func TestListAndStats(t *testing.T) {
	useTempDb(t)
	dir := t.TempDir()
	video := func(name string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	a, b, c := video("a.mkv"), video("b.mkv"), video("c.mkv")
	gone := filepath.Join(dir, "gone.mkv")
	now := time.Now()
	add := func(path, result, site, sites string, age time.Duration) {
		t.Helper()
		err := Record(&JobHistory{CreatedAt: now.Add(-age), VideoFPath: path, Title: filepath.Base(path),
			Result: result, Site: site, Sites: sites})
		if err != nil {
			t.Fatal(err)
		}
	}
	day := 24 * time.Hour
	add(a, ResultNoSub, "", "", 3*day)                   // a: failed first ...
	add(a, ResultSuccess, "zimuku", "zimuku,subhd", day) // ... then got one
	add(b, ResultNoSub, "", "", 2*day)                   // b: still missing, two attempts
	add(b, ResultFailed, "", "subhd", 0)
	add(c, ResultSuccess, "subhd", "subhd", 0)
	add(gone, ResultNoSub, "", "", 0) // file no longer exists, not "missing"
	add(a, ResultSuccess, "zimuku", "zimuku", 40*day)

	rows, total, err := List(Filter{})
	if err != nil || total != 7 || len(rows) != 7 {
		t.Fatalf("List all: %d rows, total %d, err %v", len(rows), total, err)
	}
	if rows[0].ID < rows[1].ID {
		t.Error("List must return newest first")
	}
	for name, tc := range map[string]struct {
		f    Filter
		want int64
	}{
		"result":  {Filter{Result: ResultSuccess}, 3},
		"site":    {Filter{Site: "subhd"}, 1},
		"keyword": {Filter{Keyword: "b.mk"}, 2},
		"from":    {Filter{From: now.Add(-36 * time.Hour)}, 4},
		"range":   {Filter{From: now.Add(-4 * day), To: now.Add(-36 * time.Hour)}, 2},
		"both":    {Filter{Result: ResultNoSub, From: now.Add(-36 * time.Hour)}, 1},
	} {
		if _, got, err := List(tc.f); err != nil || got != tc.want {
			t.Errorf("filter %s: got %d, want %d, err %v", name, got, tc.want, err)
		}
	}
	if rows, total, _ = List(Filter{Page: 2, PageSize: 3}); len(rows) != 3 || total != 7 {
		t.Errorf("paging: %d rows, total %d", len(rows), total)
	}

	st, err := GetStats(7)
	if err != nil {
		t.Fatal(err)
	}
	if st.Total != 7 || st.Success != 3 {
		t.Errorf("total %d success %d", st.Total, st.Success)
	}
	if len(st.Days) != 7 || st.Days[6].Date != now.Format("2006-01-02") {
		t.Fatalf("days: %+v", st.Days)
	}
	if d := st.Days[6]; d.Success != 1 || d.NoSub != 1 || d.Failed != 1 {
		t.Errorf("today: %+v", d)
	}
	if d := st.Days[5]; d.Success != 1 || d.NoSub+d.Failed != 0 {
		t.Errorf("yesterday: %+v", d)
	}
	// subhd delivered candidates in 3 jobs and was saved once; zimuku 2 and 2
	if len(st.Sites) != 2 || st.Sites[0] != (SiteStat{"subhd", 3, 1}) || st.Sites[1] != (SiteStat{"zimuku", 2, 2}) {
		t.Errorf("sites: %+v", st.Sites)
	}
	if st.MissingCount != 1 || st.Missing[0].VideoFPath != b || st.Missing[0].Attempts != 2 || st.Missing[0].LastResult != ResultFailed {
		t.Errorf("missing: %+v", st.Missing)
	}
}

func TestRetention(t *testing.T) {
	d := useTempDb(t)
	old := JobHistory{CreatedAt: time.Now().AddDate(0, 0, -MaxAgeDays-1), VideoFPath: "/old"}
	if err := Record(&old); err != nil {
		t.Fatal(err)
	}
	rows := make([]JobHistory, MaxRows+20)
	for i := range rows {
		rows[i] = JobHistory{VideoFPath: fmt.Sprintf("/v/%d", i), Result: ResultSuccess}
	}
	if err := d.CreateInBatches(rows, 500).Error; err != nil {
		t.Fatal(err)
	}
	last := JobHistory{VideoFPath: "/last"}
	if err := d.Create(&last).Error; err != nil {
		t.Fatal(err)
	}
	if err := prune(last.ID); err != nil {
		t.Fatal(err)
	}
	var count int64
	d.Model(&JobHistory{}).Count(&count)
	if count != MaxRows {
		t.Errorf("rows after prune: %d, want %d", count, MaxRows)
	}
	d.Model(&JobHistory{}).Where("video_f_path = ?", "/old").Count(&count)
	if count != 0 {
		t.Error("the row past the age limit is still there")
	}
	d.Model(&JobHistory{}).Where("video_f_path = ?", "/last").Count(&count)
	if count != 1 {
		t.Error("the newest row was removed")
	}

	// Record itself prunes every pruneEvery inserts
	if err := d.Create(&JobHistory{CreatedAt: time.Now().AddDate(-1, 0, 0), VideoFPath: "/old2"}).Error; err != nil {
		t.Fatal(err)
	}
	for i := 0; i < pruneEvery; i++ {
		if err := Record(&JobHistory{VideoFPath: "/r"}); err != nil {
			t.Fatal(err)
		}
	}
	d.Model(&JobHistory{}).Count(&count)
	d2 := int64(0)
	d.Model(&JobHistory{}).Where("video_f_path = ?", "/old2").Count(&d2)
	if d2 != 0 || count > MaxRows+pruneEvery {
		t.Errorf("Record did not prune: old2=%d rows=%d", d2, count)
	}
}
