package pkg

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/unit_test_helper"
)

func TestFileNameIsBDMV(t *testing.T) {

	rootDir := unit_test_helper.TestData(t, []string{"movies", "失控玩家 (2021)"}, 4, false)
	dbmvFPath := filepath.Join(rootDir, "CERTIFICATE", "id.bdmv")
	bok, fakeVideoFPath := FileNameIsBDMV(dbmvFPath)
	if bok == false {
		t.Fatal("FileNameIsBDMV error")
	}
	println(fakeVideoFPath)
}

func TestGetRestOfDaySec(t *testing.T) {

	rest := GetRestOfDaySec()
	if rest < 0 || rest >= 24*time.Hour {
		t.Fatalf("GetRestOfDaySec() = %v, want within [0, 24h)", rest)
	}
}
