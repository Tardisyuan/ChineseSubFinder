package base

import (
	"net/http"

	"github.com/ChineseSubFinder/ChineseSubFinder/internal/backend/middle"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"

	backend2 "github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/backend"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/common"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/gin-gonic/gin"
)

func (cb *ControllerBase) LoginHandler(c *gin.Context) {

	var err error
	defer func() {
		// 统一的异常处理
		cb.ErrorProcess(c, "LoginHandler", err)
	}()
	nowUserInfo := settings.UserInfo{}
	err = c.ShouldBindJSON(&nowUserInfo)
	if err != nil {
		return
	}

	if settings.Get().UserInfo.Username == "" || settings.Get().UserInfo.Password == "" {
		// 配置文件中的账号和密码任意一个未空，提示用户需要进行 setup 流程
		c.JSON(http.StatusNoContent, backend2.ReplyCommon{Message: "You need do `Setup`"})
		return
	}

	// Failed logins are counted per client address; too many in a row lock it out for a while.
	clientIP := c.RemoteIP()
	if wait := loginGuard.blockedFor(clientIP); wait > 0 {
		tooManyAttempts(c, wait)
		return
	}

	if !settings.Get().UserInfo.CheckLogin(nowUserInfo.Username, nowUserInfo.Password) {
		// 账号密码不匹配
		loginGuard.fail(clientIP)
		cb.fileDownloader.Log.Warnln("Login failed from", clientIP)
		c.JSON(http.StatusBadRequest, backend2.ReplyCommon{Message: "Username or Password Error"})
		return
	}
	// 用户账号密码匹配
	loginGuard.success(clientIP)
	nowAccessToken := pkg.GenerateAccessToken()
	common.SetAccessToken(nowAccessToken)
	// The static file routes (posters, subtitles) are requested by the browser without
	// the Authorization header, so the token also goes into a cookie.
	middle.SetTokenCookie(c, nowAccessToken)
	c.JSON(http.StatusOK, backend2.ReplyLogin{AccessToken: nowAccessToken,
		Settings: *settings.Get().GetNoPasswordSettings()})
}
