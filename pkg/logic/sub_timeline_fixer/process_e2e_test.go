package sub_timeline_fixer

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/sirupsen/logrus"
)

func srtTime(sec float64) string {
	d := time.Duration(sec * float64(time.Second))
	return fmt.Sprintf("%02d:%02d:%02d,%03d", int(d.Hours()), int(d.Minutes())%60, int(d.Seconds())%60, d.Milliseconds()%1000)
}

// makeSrt writes the same dialogue timing with every line moved by shift seconds.
func makeSrt(t *testing.T, path, text string, shift float64) {
	r := rand.New(rand.NewSource(7))
	var b strings.Builder
	at := 20.0
	for i := 1; i <= 150; i++ {
		length := 1 + r.Float64()*2.5
		fmt.Fprintf(&b, "%d\n%s --> %s\n%s %d\n\n", i, srtTime(at+shift), srtTime(at+length+shift), text, i)
		at += length + 0.3 + r.Float64()*4
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

// End to end with the real ffmpeg/ffprobe: a video with an embedded English subtitle and a
// downloaded Chinese subtitle that runs 3.5 s late. Process must move it back in place and
// keep a backup. Skipped when ffmpeg is not installed (it is in the Docker image).
// CSF_FIXER_VIDEO=/path/to/video.mkv CSF_FIXER_SUB=/path/to/sub.srt runs it on real files instead
// (copies are made, the originals are not touched) and only reports timing and the offset.
func TestProcessEndToEnd(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe not installed")
	}
	dir := t.TempDir()
	wd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil { // the fixer cache lives under the cwd
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(wd) }()
	settings.SetConfigRootPath(dir)

	video := filepath.Join(dir, "Show - S01E01.mkv")
	sub := filepath.Join(dir, "Show - S01E01.chinese(简,subhd).srt")
	const shift = 3.5
	realVideo, realSub := os.Getenv("CSF_FIXER_VIDEO"), os.Getenv("CSF_FIXER_SUB")
	if realVideo != "" && realSub != "" {
		video = realVideo
		data, err := os.ReadFile(realSub)
		if err != nil {
			t.Fatal(err)
		}
		sub = filepath.Join(dir, filepath.Base(realSub))
		if err = os.WriteFile(sub, data, 0o644); err != nil {
			t.Fatal(err)
		}
	} else {
		base := filepath.Join(dir, "base.srt")
		makeSrt(t, base, "Some English line number", 0)
		makeSrt(t, sub, "这是一句中文对白", shift)
		out, err := exec.Command("ffmpeg", "-v", "error", "-y",
			"-f", "lavfi", "-i", "color=c=black:s=160x90:r=2:d=600",
			"-f", "lavfi", "-i", "anullsrc=r=16000:cl=mono",
			"-i", base, "-shortest",
			"-c:v", "mpeg4", "-c:a", "aac", "-c:s", "srt", "-metadata:s:s:0", "language=eng", video).CombinedOutput()
		if err != nil {
			t.Skipf("cannot build the test video with this ffmpeg: %v %s", err, out)
		}
	}
	before, _ := os.ReadFile(sub)

	log := logrus.New()
	log.SetLevel(logrus.WarnLevel)
	helper := NewSubTimelineFixerHelperEx(log, *settings.NewTimelineFixerSettings())
	if !helper.Check() {
		t.Fatal("Check() must find ffmpeg")
	}
	start := time.Now()
	if err := helper.Process(video, sub); err != nil {
		t.Fatal(err)
	}
	t.Logf("Process took %v", time.Since(start))
	after, _ := os.ReadFile(sub)
	if realVideo != "" {
		t.Logf("changed=%v first cue before/after: %q / %q", string(before) != string(after), firstCue(string(before)), firstCue(string(after)))
		return
	}
	// first line starts at 20.0 s in the video; the download had it at 23.5 s
	want, _ := time.ParseDuration("20s")
	got := cueStart(t, firstCue(string(after)))
	if diff := got - want; diff < -300*time.Millisecond || diff > 300*time.Millisecond {
		t.Fatalf("first cue at %v after fixing, want about %v (was %v)", got, want, cueStart(t, firstCue(string(before))))
	}
	if cue := firstCue(string(after)); !strings.HasPrefix(cue, "00:00:") {
		t.Errorf("an .srt cue needs two-digit hours, got %q", cue)
	}
	if _, err := os.Stat(sub + ".csf-bk"); err != nil {
		t.Errorf("no backup of the original subtitle: %v", err)
	}
}

func firstCue(srt string) string {
	for _, line := range strings.Split(srt, "\n") {
		if strings.Contains(line, "-->") {
			return strings.TrimSpace(line)
		}
	}
	return ""
}

func cueStart(t *testing.T, cue string) time.Duration {
	var h, m, s, ms int
	if _, err := fmt.Sscanf(cue, "%d:%d:%d,%d", &h, &m, &s, &ms); err != nil {
		t.Fatalf("cue %q: %v", cue, err)
	}
	return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute + time.Duration(s)*time.Second + time.Duration(ms)*time.Millisecond
}
