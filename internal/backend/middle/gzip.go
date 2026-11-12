package middle

import (
	"compress/gzip"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

// Gzip compresses the web UI (index, /js, /css) and the JSON API (/v1, /api/v1) for clients
// that accept it: the vendor bundle is 484 KB raw, 159 KB gzipped. Media files, previews,
// HLS segments and subtitle downloads are left alone (already compressed or streamed).
// Responses without a body (HEAD, 204, 304) are passed through untouched.
func Gzip() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !gzipWanted(c.Request) {
			c.Next()
			return
		}
		g := &gzipWriter{ResponseWriter: c.Writer, method: c.Request.Method}
		c.Writer = g
		defer g.close()
		c.Next()
	}
}

func gzipWanted(r *http.Request) bool {
	if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		return false
	}
	p := r.URL.Path
	return p == "/" || strings.HasPrefix(p, "/js/") || strings.HasPrefix(p, "/css/") ||
		strings.HasPrefix(p, "/v1/") || strings.HasPrefix(p, "/api/v1/")
}

var gzipPool = sync.Pool{New: func() interface{} { w, _ := gzip.NewWriterLevel(nil, gzip.DefaultCompression); return w }}

type gzipWriter struct {
	gin.ResponseWriter
	method  string
	decided bool
	skip    bool
	gz      *gzip.Writer
}

func (g *gzipWriter) decide(code int) {
	if g.decided {
		return
	}
	g.decided = true
	h := g.ResponseWriter.Header()
	if g.method == http.MethodHead || code < 200 || code == http.StatusNoContent || code == http.StatusNotModified || code == http.StatusPartialContent ||
		h.Get("Content-Encoding") != "" {
		g.skip = true
		return
	}
	h.Set("Content-Encoding", "gzip")
	h.Add("Vary", "Accept-Encoding")
	h.Del("Content-Length")
}

func (g *gzipWriter) WriteHeader(code int) {
	g.decide(code)
	g.ResponseWriter.WriteHeader(code)
}

func (g *gzipWriter) Write(b []byte) (int, error) {
	if !g.decided {
		g.WriteHeader(g.ResponseWriter.Status())
	}
	if g.skip {
		return g.ResponseWriter.Write(b)
	}
	if g.gz == nil {
		g.gz = gzipPool.Get().(*gzip.Writer)
		g.gz.Reset(g.ResponseWriter)
	}
	return g.gz.Write(b)
}

func (g *gzipWriter) WriteString(s string) (int, error) { return g.Write([]byte(s)) }

func (g *gzipWriter) close() {
	if g.gz != nil {
		_ = g.gz.Close()
		gzipPool.Put(g.gz)
		g.gz = nil
	}
}
