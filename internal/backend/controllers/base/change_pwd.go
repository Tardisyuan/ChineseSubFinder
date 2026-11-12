package base

import (
	"net/http"

	backend2 "github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/backend"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/common"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/gin-gonic/gin"
)

func (cb *ControllerBase) ChangePwdHandler(c *gin.Context) {

	var err error
	defer func() {
		// 统一的异常处理
		cb.ErrorProcess(c, "ChangePwdHandler", err)
	}()

	changePwd := backend2.ReqChangePwd{}
	err = c.ShouldBindJSON(&changePwd)
	if err != nil {
		return
	}

	if settings.Get().UserInfo.Username == "" || settings.Get().UserInfo.Password == "" {
		// 配置文件中的账号和密码任意一个未空，提示用户需要进行 setup 流程
		c.JSON(http.StatusNoContent, backend2.ReplyCommon{Message: "You need do `Setup`"})
		return
	}

	// Guessing the old password is limited the same way as guessing it at login.
	clientIP := c.RemoteIP()
	if wait := loginGuard.blockedFor(clientIP); wait > 0 {
		tooManyAttempts(c, wait)
		return
	}

	if !settings.Get().UserInfo.CheckPassword(changePwd.OrgPwd) {
		// 原始的密码不对
		loginGuard.fail(clientIP)
		c.JSON(http.StatusNoContent, backend2.ReplyCommon{Message: "Org Password Error"})
	} else {
		// 同意修改密码，Save 会把新密码换成哈希
		loginGuard.success(clientIP)
		settings.Get().UserInfo.Password = changePwd.NewPwd
		err = settings.Get().Save()
		if err != nil {
			return
		}
		// 修改密码成功后，会清理 AccessToken，强制要求重写登录
		common.SetAccessToken("")
		c.JSON(http.StatusOK, backend2.ReplyCommon{Message: "ok, need ReLogin"})
	}
}
