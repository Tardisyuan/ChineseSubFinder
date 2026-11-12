package http_client_bench

import (
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/sirupsen/logrus"
)

// TestGetPublicIPIsCached: GetPublicIP runs after every downloaded subtitle and must not hit the
// lookup site each time.
func TestGetPublicIPIsCached(t *testing.T) {
	var hits int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&hits, 1)
		_, _ = w.Write([]byte("203.0.113.7"))
	}))
	defer srv.Close()
	l := logrus.New()
	l.SetOutput(ioutil.Discard)
	queue := &settings.TaskQueue{CheckPublicIPTargetSite: srv.URL}
	for i := 0; i < 3; i++ {
		if ip := pkg.GetPublicIP(l, queue); ip != "203.0.113.7" {
			t.Fatalf("GetPublicIP() = %q", ip)
		}
	}
	if n := atomic.LoadInt64(&hits); n != 1 {
		t.Fatalf("3 calls made %d lookups, want 1", n)
	}
}

// newCountingTLSServer is a local HTTPS fake that counts accepted TCP connections
// (= TLS handshakes the client had to pay for).
func newCountingTLSServer() (*httptest.Server, *int64) {
	var conns int64
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	srv.Config.ConnState = func(c net.Conn, state http.ConnState) {
		if state == http.StateNew {
			atomic.AddInt64(&conns, 1)
		}
	}
	srv.StartTLS()
	return srv, &conns
}

// BenchmarkNewHttpClientGet mirrors how the suppliers use NewHttpClient: a fresh client per request.
func BenchmarkNewHttpClientGet(b *testing.B) {
	srv, conns := newCountingTLSServer()
	defer srv.Close()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client, err := pkg.NewHttpClient()
		if err != nil {
			b.Fatal(err)
		}
		resp, err := client.R().Get(srv.URL)
		if err != nil || resp.StatusCode() != 200 {
			b.Fatal(err, resp.StatusCode())
		}
	}
	b.ReportMetric(float64(atomic.LoadInt64(conns))/float64(b.N), "handshakes/op")
}

// TestNewHttpClientReusesConnections fails if every request pays a new TCP+TLS handshake again.
func TestNewHttpClientReusesConnections(t *testing.T) {
	srv, conns := newCountingTLSServer()
	defer srv.Close()
	for i := 0; i < 5; i++ {
		client, err := pkg.NewHttpClient("https://example.invalid/")
		if err != nil {
			t.Fatal(err)
		}
		resp, err := client.R().Get(srv.URL)
		if err != nil || resp.StatusCode() != 200 {
			t.Fatal(err)
		}
		if got := resp.Request.Header.Get("Referer"); got != "https://example.invalid/" {
			t.Fatalf("Referer = %q", got)
		}
	}
	if n := atomic.LoadInt64(conns); n != 1 {
		t.Fatalf("5 sequential requests used %d connections, want 1", n)
	}
}
