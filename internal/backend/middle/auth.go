package middle

import (
	"net/http"
	"strings"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/backend"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/common"
	"github.com/gin-gonic/gin"
)

// TokenCookieName is the cookie that carries the Web UI access token, for requests the
// browser makes without the Authorization header (<img>, <a href>, subtitle tracks).
const TokenCookieName = "csf_token"

// headerToken returns the second field of "Authorization: <scheme> <token>", or "".
func headerToken(c *gin.Context) string {
	fields := strings.Fields(c.Request.Header.Get("Authorization"))
	if len(fields) != 2 {
		return ""
	}
	return fields[1]
}

// SetTokenCookie stores the Web UI token in the browser, for as long as the token lives.
func SetTokenCookie(c *gin.Context, token string) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(TokenCookieName, token, int(common.AccessTokenLifetime.Seconds()), "/", "", false, true)
}

// checkToken checks the Web UI token and keeps the cookie alive while the token is in use.
func checkToken(c *gin.Context, token string) bool {
	ok, renewed := common.CheckAccessToken(token)
	if renewed {
		SetTokenCookie(c, token)
	}
	return ok
}

func reject(c *gin.Context, message string) {
	c.JSON(http.StatusUnauthorized, backend.ReplyCheckAuth{Message: message})
	c.Abort()
}

func CheckAuth() gin.HandlerFunc {

	return func(context *gin.Context) {
		nowAccessToken := headerToken(context)
		if nowAccessToken == "" {
			reject(context, "Request Header Authorization Error")
			return
		}
		if !checkToken(context, nowAccessToken) {
			reject(context, "AccessToken Error")
			return
		}
		// 向下传递消息
		context.Next()
	}
}

// CheckAuthIfSetup is for the endpoints the setup wizard needs before any user exists:
// open on a fresh install, login required once setup is done.
func CheckAuthIfSetup() gin.HandlerFunc {
	checkAuth := CheckAuth()
	return func(context *gin.Context) {
		userInfo := settings.Get().UserInfo
		if userInfo.Username == "" || userInfo.Password == "" {
			context.Next()
			return
		}
		checkAuth(context)
	}
}

// CheckStaticAuth protects the static file routes (media folders, preview cache). The token
// may come from the Authorization header or from the cookie set at login.
func CheckStaticAuth() gin.HandlerFunc {

	return func(context *gin.Context) {
		nowAccessToken := headerToken(context)
		if nowAccessToken == "" {
			nowAccessToken, _ = context.Cookie(TokenCookieName)
		}
		if !checkToken(context, nowAccessToken) {
			reject(context, "AccessToken Error")
			return
		}
		context.Next()
	}
}

// CheckApiAuth checks the API key of /api/v1. It reads the key from the settings on every
// request, so a key saved by setup or a settings change works without further syncing.
func CheckApiAuth() gin.HandlerFunc {

	return func(context *gin.Context) {
		nowAccessToken := headerToken(context)
		if nowAccessToken == "" {
			reject(context, "Request Header Authorization Error")
			return
		}
		apiKey := settings.Get().ExperimentalFunction.ApiKeySettings
		if apiKey.Enabled == false || apiKey.Key == "" {
			reject(context, "api_key_enabled == false or api_key is empty")
			return
		}
		if nowAccessToken != apiKey.Key {
			reject(context, "AccessToken Error")
			return
		}
		// 向下传递消息
		context.Next()
	}
}
