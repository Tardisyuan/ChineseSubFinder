package assrt

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/sirupsen/logrus"
)

// A non-JSON reply (rate limit, quota, HTML error page) used to crash getSubByKeyWord
// with a nil pointer while it was logging the error.
func TestGetSubByKeyWordBadResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("<html>busy</html>"))
	}))
	defer srv.Close()

	settings.SetConfigRootPath(t.TempDir())
	settings.Get().AdvancedSettings.SuppliersSettings.Assrt.RootUrl = srv.URL

	s := &Supplier{log: logrus.New()}
	res, err := s.getSubByKeyWord("幻视任务 S01E01")
	if err == nil || res != nil {
		t.Fatalf("want error, got %v %v", res, err)
	}
	if !strings.Contains(err.Error(), "HTTP 503") || !strings.Contains(err.Error(), "busy") {
		t.Fatalf("error should carry status and body: %v", err)
	}
}

// Shape seen on 2026-11-11: string "fileid" and string "revision" alongside the old fields.
func TestSearchReplyWithStringNumbers(t *testing.T) {
	body := `{"status":0,"sub":{"keyword":"幻视任务 S01E02","action":"search","result":"succeed","subs":[
		{"fileid":"802296","m_lang":"英&nbsp;简&nbsp;繁&nbsp;双语","m_source":"官方译本",
		 "native_name":"VisionQuest S01E02","revision":"0","vote_score":"3","subtype":"ASS"},
		{"id":801234,"revision":2,"native_name":"old shape"}]}}`
	var r SearchSubResult
	if err := json.Unmarshal([]byte(body), &r); err != nil {
		t.Fatal(err)
	}
	if len(r.Sub.Subs) != 2 || r.Sub.Subs[0].FileId != 802296 || r.Sub.Subs[0].Id != 0 ||
		r.Sub.Subs[0].VoteScore != 3 || r.Sub.Subs[1].Id != 801234 || r.Sub.Subs[1].Revision != 2 {
		t.Fatalf("parsed %+v", r.Sub.Subs)
	}
}
