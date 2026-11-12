package middle

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/gin-gonic/gin"
)

func gzipEngine() *gin.Engine {
	gin.SetMode(gin.TestMode)
	e := gin.New()
	e.Use(Gzip())
	body := strings.Repeat("console.log('csf');\n", 500)
	e.StaticFS("/js", http.FS(fstest.MapFS{"app.js": {Data: []byte(body)}}))
	e.GET("/v1/overview", func(c *gin.Context) { c.JSON(200, gin.H{"ok": strings.Repeat("x", 2000)}) })
	e.GET("/movie_dir_0/poster.jpg", func(c *gin.Context) { c.Data(200, "image/jpeg", []byte(body)) })
	e.GET("/v1/empty", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	return e
}

func get(e *gin.Engine, method, path string, hdr map[string]string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, nil)
	r.Header.Set("Accept-Encoding", "gzip")
	for k, v := range hdr {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	e.ServeHTTP(w, r)
	return w
}

func TestGzip(t *testing.T) {
	e := gzipEngine()

	w := get(e, "GET", "/js/app.js", nil)
	if w.Code != 200 || w.Header().Get("Content-Encoding") != "gzip" || w.Header().Get("Content-Length") != "" {
		t.Fatalf("js: code %d enc %q len %q", w.Code, w.Header().Get("Content-Encoding"), w.Header().Get("Content-Length"))
	}
	zr, err := gzip.NewReader(bytes.NewReader(w.Body.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	plain, _ := io.ReadAll(zr)
	if !strings.HasPrefix(string(plain), "console.log") || w.Body.Len() >= len(plain) {
		t.Fatalf("js body not gzip of the file: %d -> %d", len(plain), w.Body.Len())
	}

	if w := get(e, "GET", "/v1/overview", nil); w.Header().Get("Content-Encoding") != "gzip" {
		t.Fatal("api json not compressed")
	}
	if w := get(e, "GET", "/movie_dir_0/poster.jpg", nil); w.Header().Get("Content-Encoding") != "" {
		t.Fatal("media file must not be compressed")
	}
	if w := get(e, "GET", "/v1/empty", nil); w.Code != 204 || w.Body.Len() != 0 || w.Header().Get("Content-Encoding") != "" {
		t.Fatalf("204: code %d body %d enc %q", w.Code, w.Body.Len(), w.Header().Get("Content-Encoding"))
	}
	r := httptest.NewRequest("GET", "/js/app.js", nil) // no Accept-Encoding
	w2 := httptest.NewRecorder()
	e.ServeHTTP(w2, r)
	if w2.Header().Get("Content-Encoding") != "" || !strings.HasPrefix(w2.Body.String(), "console.log") {
		t.Fatal("client without gzip got compressed body")
	}
}
