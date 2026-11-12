package url_connectedness_helper

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUrlConnectednessTest(t *testing.T) {

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/missing" {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	deadUrl := srv.URL
	defer srv.Close()

	// 200 is "connected"
	got, _, err := UrlConnectednessTest(srv.URL, "")
	if err != nil || got != true {
		t.Fatalf("live url: got = %v, err = %v, want true, nil", got, err)
	}
	// a non-200 answer is "not connected" without an error
	got, _, err = UrlConnectednessTest(srv.URL+"/missing", "")
	if err != nil || got != false {
		t.Fatalf("404 url: got = %v, err = %v, want false, nil", got, err)
	}
	// only http proxies are supported
	got, _, err = UrlConnectednessTest(srv.URL, "socks5://127.0.0.1:1")
	if err == nil || got != false {
		t.Fatalf("socks5 proxy: got = %v, err = %v, want false and an error", got, err)
	}
	// nothing listens any more
	srv.Close()
	got, _, err = UrlConnectednessTest(deadUrl, "")
	if err == nil || got != false {
		t.Fatalf("dead url: got = %v, err = %v, want false and an error", got, err)
	}
}
