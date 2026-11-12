package middle

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/common"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/gin-gonic/gin"
)

func TestAuth(t *testing.T) {
	settings.SetConfigRootPath(t.TempDir())
	gin.SetMode(gin.TestMode)
	ok := func(c *gin.Context) { c.String(http.StatusOK, "ok") }
	r := gin.New() // no Recovery: a panic in a middleware fails the test
	r.GET("/api", CheckApiAuth(), ok)
	r.GET("/static", CheckStaticAuth(), ok)
	r.GET("/wizard", CheckAuthIfSetup(), ok)
	r.GET("/v1", CheckAuth(), ok)

	get := func(path, auth, cookie string) int {
		req := httptest.NewRequest("GET", path, nil)
		if auth != "" {
			req.Header.Set("Authorization", auth)
		}
		if cookie != "" {
			req.AddCookie(&http.Cookie{Name: TokenCookieName, Value: cookie})
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}
	want := func(name string, got, code int) {
		t.Helper()
		if got != code {
			t.Errorf("%s: got %d, want %d", name, got, code)
		}
	}

	// fresh install: no user, nobody logged in, no api key
	common.SetAccessToken("")
	want("wizard before setup", get("/wizard", "", ""), 200)
	want("v1 no token", get("/v1", "", ""), 401)
	want("v1 empty token", get("/v1", "Bearer ", ""), 401)
	want("static not logged in", get("/static", "", ""), 401)
	want("static empty cookie", get("/static", "Bearer ", ""), 401)
	want("api single field header", get("/api", "abc", ""), 401) // used to panic
	want("api key disabled", get("/api", "Bearer x", ""), 401)

	// api key enabled but empty must not let anything through
	settings.Get().ExperimentalFunction.ApiKeySettings.Enabled = true
	want("api empty key", get("/api", "Bearer ", ""), 401)
	want("api empty key, any token", get("/api", "Bearer x", ""), 401)

	// the key is read from the settings, no separate sync step
	settings.Get().ExperimentalFunction.ApiKeySettings.Key = "key123"
	want("api good key", get("/api", "Bearer key123", ""), 200)
	want("api bad key", get("/api", "Bearer key12", ""), 401)

	// after setup and login
	settings.Get().UserInfo.Username = "abcd"
	settings.Get().UserInfo.Password = "123456"
	common.SetAccessToken("tok")
	want("wizard after setup, no token", get("/wizard", "", ""), 401)
	want("wizard after setup, token", get("/wizard", "Bearer tok", ""), 200)
	want("v1 token", get("/v1", "Bearer tok", ""), 200)
	want("static cookie", get("/static", "", "tok"), 200)
	want("static header", get("/static", "Bearer tok", ""), 200)
	want("static wrong cookie", get("/static", "", "nope"), 401)
	want("web token is not an api key", get("/api", "Bearer tok", ""), 401)
}
