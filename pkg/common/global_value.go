package common

import (
	"crypto/subtle"
	"sync"
	"time"
)

// AccessTokenLifetime is how long the Web UI token stays valid without being used.
// Every authenticated request pushes the expiry out again.
const AccessTokenLifetime = 30 * 24 * time.Hour

// SetAccessToken 设置 Web UI 访问的 Token
func SetAccessToken(newToken string) {

	defer mutexAccessToken.Unlock()
	mutexAccessToken.Lock()
	accessToken = newToken
	accessTokenExpire = timeNow().Add(AccessTokenLifetime)
}

// GetAccessToken 获取 Web UI 访问的 Token，过期后为空
func GetAccessToken() string {

	defer mutexAccessToken.Unlock()
	mutexAccessToken.Lock()
	if timeNow().After(accessTokenExpire) {
		accessToken = ""
	}
	return accessToken
}

// CheckAccessToken reports whether token is the current, unexpired Web UI token and
// refreshes its lifetime. renewed is true at most once a day: the caller then re-sends
// the cookie so that the browser's copy does not expire before the server's.
func CheckAccessToken(token string) (ok bool, renewed bool) {

	defer mutexAccessToken.Unlock()
	mutexAccessToken.Lock()
	now := timeNow()
	if now.After(accessTokenExpire) {
		accessToken = ""
	}
	if token == "" || accessToken == "" || subtle.ConstantTimeCompare([]byte(token), []byte(accessToken)) != 1 {
		return false, false
	}
	if accessTokenExpire.Sub(now) < AccessTokenLifetime-24*time.Hour {
		accessTokenExpire = now.Add(AccessTokenLifetime)
		renewed = true
	}
	return true, renewed
}

var (
	accessToken       = ""
	accessTokenExpire time.Time
	mutexAccessToken  sync.Mutex
	timeNow           = time.Now // replaced in tests
)
