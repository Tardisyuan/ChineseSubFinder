package cachetest

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_parser/ass"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_parser/srt"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/sub_parser_hub"
	"github.com/sirupsen/logrus"
)

func srtBody(line string) string {
	var b strings.Builder
	for i := 1; i <= 40; i++ {
		b.WriteString(strings.Join([]string{
			strconv.Itoa(i), "00:00:0" + strconv.Itoa(i%10) + ",000 --> 00:00:0" + strconv.Itoa(i%10) + ",500", line, "", "",
		}, "\n"))
	}
	return b.String()
}

// Same answer from the cache, and a changed file is parsed again.
func TestSubChineseInfoCache(t *testing.T) {
	log := logrus.New()
	hub := sub_parser_hub.NewSubParserHub(log, ass.NewParser(log), srt.NewParser(log))
	p := filepath.Join(t.TempDir(), "movie.zh.srt")
	if err := os.WriteFile(p, []byte(srtBody("这是一句中文对白，用来检测语言")), 0644); err != nil {
		t.Fatal(err)
	}
	found, zh, _, err := hub.SubChineseInfo(p)
	if err != nil || !found || !zh {
		t.Fatalf("first: found=%v zh=%v err=%v", found, zh, err)
	}
	start := time.Now()
	found, zh, _, err = hub.SubChineseInfo(p)
	if err != nil || !found || !zh || time.Since(start) > 5*time.Millisecond {
		t.Fatalf("cached: found=%v zh=%v err=%v took=%v", found, zh, err, time.Since(start))
	}
	later := time.Now().Add(2 * time.Second)
	if err := os.WriteFile(p, []byte(srtBody("This is an English only line of dialogue here")), 0644); err != nil {
		t.Fatal(err)
	}
	_ = os.Chtimes(p, later, later)
	_, zh, _, err = hub.SubChineseInfo(p)
	if err != nil || zh {
		t.Fatalf("after rewrite: zh=%v err=%v (cache not invalidated)", zh, err)
	}
}
