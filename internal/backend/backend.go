package backend

import (
	"context"
	"errors"
	"fmt"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/pre_job"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/internal/backend/middle"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/local_http_proxy_server"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"

	"github.com/ChineseSubFinder/ChineseSubFinder/frontend/dist"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/sirupsen/logrus"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/cron_helper"
)

type BackEnd struct {
	logger        *logrus.Logger
	cronHelper    *cron_helper.CronHelper
	httpPort      int
	running       bool
	srv           *http.Server
	locker        sync.Mutex
	restartSignal chan interface{}
	preJob        *pre_job.PreJob
}

func NewBackEnd(
	logger *logrus.Logger,
	cronHelper *cron_helper.CronHelper,
	httpPort int,
	restartSignal chan interface{}) *BackEnd {

	return &BackEnd{
		logger:        logger,
		cronHelper:    cronHelper,
		httpPort:      httpPort,
		restartSignal: restartSignal,
	}
}

func (b *BackEnd) start() {

	defer b.locker.Unlock()
	b.locker.Lock()

	if b.running == true {
		b.logger.Debugln("Http Server is already running")
		return
	}
	b.running = true
	// ----------------------------------------
	// 设置代理
	err := local_http_proxy_server.SetProxyInfo(settings.Get().AdvancedSettings.ProxySettings.GetInfos())
	if err != nil {
		b.logger.Errorln("Set Local Http Proxy Server Error:", err)
		return
	}
	// -----------------------------------------
	// 设置跨域
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = ioutil.Discard
	engine := gin.Default()
	// 默认所有都通过
	engine.Use(cors.Default())
	engine.Use(middle.Gzip())
	// 初始化路由
	b.preJob = pre_job.NewPreJob(b.logger)
	cbBase, v1Router := InitRouter(engine, b.cronHelper, b.restartSignal, b.preJob)
	// -----------------------------------------
	// 静态文件服务器
	engine.GET("/", func(c *gin.Context) {
		c.Header("content-type", "text/html;charset=utf-8")
		c.String(http.StatusOK, string(dist.SpaIndexHtml))
	})
	// js/css/fonts file names carry a content hash, so the browser may keep them for good.
	// Embedded files have no mod time, so without this header nothing can be revalidated and
	// every page load downloads the whole UI again.
	engine.Use(func(c *gin.Context) {
		for _, prefix := range []string{dist.SpaFolderJS, dist.SpaFolderCSS, dist.SpaFolderFonts} {
			if strings.HasPrefix(c.Request.URL.Path, prefix+"/") {
				c.Header("Cache-Control", "public, max-age=31536000, immutable")
			}
		}
	})
	engine.StaticFS(dist.SpaFolderJS, dist.Assets(dist.SpaFolderName+dist.SpaFolderJS, dist.SpaJS))
	engine.StaticFS(dist.SpaFolderCSS, dist.Assets(dist.SpaFolderName+dist.SpaFolderCSS, dist.SpaCSS))
	engine.StaticFS(dist.SpaFolderFonts, dist.Assets(dist.SpaFolderName+dist.SpaFolderFonts, dist.SpaFonts))
	engine.StaticFS(dist.SpaFolderIcons, dist.Assets(dist.SpaFolderName+dist.SpaFolderIcons, dist.SpaIcons))
	engine.StaticFS(dist.SpaFolderImages, dist.Assets(dist.SpaFolderName+dist.SpaFolderImages, dist.SpaImages))
	// -----------------------------------------
	// 用于预览视频和字幕的静态文件服务
	previewCacheFolder, err := pkg.GetVideoAndSubPreviewCacheFolder()
	if err != nil {
		b.logger.Errorln("GetVideoAndSubPreviewCacheFolder Error:", err)
		return
	}
	engine.Group("/", middle.CheckStaticAuth()).StaticFS("/static/preview", http.Dir(previewCacheFolder))
	// -----------------------------------------
	// api 服务
	engine.Any("/api", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/")
	})
	// listen and serve on 0.0.0.0:8080(default)
	b.srv = &http.Server{
		Addr:    fmt.Sprintf(":%d", b.httpPort),
		Handler: engine,
	}
	go func() {
		b.doPreJob()
		b.doCornJob()
	}()
	// 启动 http server
	go func() {
		b.logger.Infoln("Try Start Http Server At Port", b.httpPort)
		if err := b.srv.ListenAndServe(); err != nil && errors.Is(err, http.ErrServerClosed) == false {
			b.logger.Errorln("Start Server Error:", err)
		}
		defer func() {
			cbBase.Close()
			v1Router.Close()
		}()
	}()
}

// Stop shuts the http server down (waits up to 5 seconds for requests in flight). Used by
// Restart and by the SIGTERM handler in main.
func (b *BackEnd) Stop() {

	b.locker.Lock()
	defer func() {
		b.locker.Unlock()
	}()
	if b.running == false {
		b.logger.Debugln("Http Server is not running")
		return
	}
	b.running = false

	// buffered and never closed: after a timeout the goroutine below still sends once
	exitOk := make(chan interface{}, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go func() {
		if err := b.srv.Shutdown(ctx); err != nil {
			b.logger.Errorln("Http Server Shutdown:", err)
		}
		exitOk <- true
	}()
	select {
	case <-ctx.Done():
		b.logger.Warningln("Http Server Shutdown timeout of 5 seconds.")
	case <-exitOk:
		b.logger.Infoln("Http Server Shutdown Successfully")
	}
	b.logger.Infoln("Http Server Shutdown Done.")
}

func (b *BackEnd) Restart() {

	for {
		select {
		case <-b.restartSignal:
			{
				b.Stop()
				b.start()
			}
		}
	}
}

// doPreJob 前置的任务，热修复、字幕修改文件名格式、提前下载好浏览器
func (b *BackEnd) doPreJob() {

	if settings.Get().UserInfo.Username == "" || settings.Get().UserInfo.Password == "" {
		// 如果没有完成，那么就不执行初始化
		b.logger.Infoln("Need do Setup, then do PreJob")
	} else {
		// 启动程序只会执行一次，用 Once 控制
		// 前置的任务，热修复、字幕修改文件名格式、提前下载好浏览器
		if settings.Get().SpeedDevMode == true {
			return
		}
		if pkg.LiteMode() == false {
			return
		}
		b.logger.Infoln("Setup is Done")
		b.logger.Infoln("PreJob Will Start...")
		// 不启用 Chrome 相关操作
		err := b.preJob.HotFix().ChangeSubNameFormat().Wait()
		if err != nil {
			b.logger.Errorln("pre_job", err)
		}
	}
}

// doCornJob 定时任务
func (b *BackEnd) doCornJob() {
	// 需要使用 go 来执行，因为这个函数是阻塞的
	// 启动定时任务
	if settings.Get().UserInfo.Username == "" || settings.Get().UserInfo.Password == "" {
		// 如果没有完成，那么就不开启
		b.logger.Infoln("Need do Setup, then do CornJob")
	} else {
		// 是否完成了 Setup，如果完成了，那么就开启第一次的扫描
		go func() {
			b.logger.Infoln("Setup is Done")
			// 尊重 RunScanAtStartUp 配置：开启时启动后立即扫描一次
			b.cronHelper.Start(settings.Get().CommonSettings.RunScanAtStartUp)
		}()
	}
}
