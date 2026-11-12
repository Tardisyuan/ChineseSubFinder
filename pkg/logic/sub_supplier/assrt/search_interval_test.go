package assrt

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/sirupsen/logrus"
)

// The search interval must separate consecutive API calls, but must not delay the caller after
// the last one.
func TestSearchIntervalOnlyBetweenCalls(t *testing.T) {
	var mu sync.Mutex
	var hits []time.Time
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		hits = append(hits, time.Now())
		mu.Unlock()
		_, _ = w.Write([]byte(`{"sub":{"action":"search","subs":{},"result":"succeed"},"status":0}`))
	}))
	defer srv.Close()

	settings.SetConfigRootPath(t.TempDir())
	settings.Get().AdvancedSettings.SuppliersSettings.Assrt.RootUrl = srv.URL

	const interval = 300 * time.Millisecond
	s := &Supplier{log: logrus.New(), theSearchInterval: interval}
	lastApiCallEnd = time.Time{}

	start := time.Now()
	if _, err := s.getSubByKeyWord("a"); err != nil {
		t.Fatal(err)
	}
	if d := time.Since(start); d >= interval {
		t.Fatalf("a single call took %v, it should not wait for the interval", d)
	}
	if _, err := s.getSubDetail(1); err != nil {
		t.Fatal(err)
	}
	if _, err := s.getSubByKeyWord("b"); err != nil {
		t.Fatal(err)
	}
	if len(hits) != 3 {
		t.Fatalf("got %d requests", len(hits))
	}
	for i := 1; i < len(hits); i++ {
		if gap := hits[i].Sub(hits[i-1]); gap < interval {
			t.Fatalf("requests %d and %d only %v apart, want >= %v", i-1, i, gap, interval)
		}
	}
	if total := time.Since(start); total >= 3*interval {
		t.Fatalf("3 calls took %v, the old sleep-after-every-call behaviour (>= %v) is back", total, 3*interval)
	}
}
