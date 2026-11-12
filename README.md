# ChineseSubFinder（个人维护版）

自动给电影、连续剧下载中文字幕。

这是 [ChineseSubFinder](https://github.com/ChineseSubFinder/ChineseSubFinder) v0.55.2 的个人维护版本。上游已停止维护（[停更公告](SeeYou/README.md)），它依赖的云端接口和多个字幕站也已下线。本版本修好了能修的部分，删掉了失效的部分，并加了一些新功能。

## 升级前必读：不可逆的改动

从上游版本或本版本的旧镜像升级前，先备份 `/config` 目录（Unraid 上就是 appdata 里的这个容器目录）。

| 改动 | 影响 | 回退到旧镜像时 |
|---|---|---|
| Web 登录密码改为 bcrypt 哈希保存 | 首次启动时，配置文件里的明文密码会被改写成哈希 | 旧版本无法登录。处理办法：手动把 `ChineseSubFinderSettings.json` 里 `user_info.password` 改成明文密码，或者直接恢复备份 |
| 配置文件新增字段（`notification_settings`、`emby_settings.server_type`、字幕优先级等） | 旧配置可以直接加载，缺的字段用默认值 | 旧版本会忽略这些字段；如果旧版本再保存一次配置，这些字段就会丢失 |
| 缓存数据库新增 `job_histories` 等表 | 自动建表 | 旧版本不读这些表，不受影响 |
| 删除 SubtitleBest、a4k、字幕共享上传、Chrome/验证码相关功能，以及版本升级提示 | 相关设置项不再显示，配置里的旧值会被忽略 | — |
| 登录 token 改为 30 天内不用就过期 | 升级后需要重新登录一次 | — |
| 登录失败限速 | 同一地址连续失败 5 次后锁定 30 秒，之后逐次加倍，最长 5 分钟。如果前面有反向代理，所有客户端共用一个计数 | — |

`/api/v1` 接口（API Key 鉴权）没有变化，MoviePilot 插件可以继续使用。

## 可用字幕源

- 迅雷、射手（按文件哈希匹配）
- ASSRT（需要在“字幕源设置”里填 Token）

zimuku 每一步都要过验证码，暂不支持。a4k、SubtitleBest 已失效，已移除。

## 主要改动

字幕
- “进阶配置”里新增字幕挑选优先级：优先选与视频版本匹配的字幕，再按语言偏好（双语/简体/繁体）和站点顺序挑。
- 库页面的“字幕搜索”新增“程序搜索”：在后台搜索所有可用字幕源，列出全部候选字幕，可以预览后手动挑一个使用。
- “字幕已满意，不再自动更换”锁定后，自动下载、MoviePilot 和 Jellyfin Webhook 都不会再替换这个视频的字幕。
- 没有 TMDB API Key 时，改用 nfo 里的中文名和原名搜索。仍然建议填写 TMDB Key。
- 修复压缩包里的连续剧字幕：文件名不是 `.SxxEyy.` 格式时会被丢掉。

媒体服务器
- “媒体库设置”可以选择 Emby 或 Jellyfin。
- 下载完字幕后，按目录通知媒体服务器刷新；MoviePilot 等通过 API 添加的任务也会刷新。如果容器内外路径不同，在“媒体库设置”里填路径映射。
- Jellyfin Webhook：Jellyfin 新入库一集或一部电影时，自动加入字幕下载队列。地址在“通知”页可以复制（`/api/v1/jellyfin-webhook?api_key=…`）。

任务与通知
- 总览页显示今日下载数、任务队列、下次扫描时间，以及字幕源、媒体服务器、TMDB 的连接状态。
- 新增“下载历史”页面：显示每个任务的结果、用的字幕和字幕站，以及统计数据。
- “下载队列”可以筛选失败任务，单个或全部重试。失败的任务会逐步降低优先级。
- “通知”设置：任务成功或失败时，通过 Webhook、Telegram 或 Bark 发送通知。

稳定性
- 修复启动时卡在字幕源检查、Web 界面迟迟打不开的问题。
- 修复多处崩溃（panic）；任务崩溃时会标记为失败，并记录堆栈。
- 配置文件、任务队列和下载缓存改为原子写入，断电时不会写坏。
- 镜像自带健康检查。收到停止信号时会等正在运行的任务最多 8 秒；被中断的任务在下次启动时重新排队。
- 网页和接口启用 gzip；大媒体库首次扫描时入队更快。

## 和 MoviePilot 配合

MoviePilot 官方插件库里的 **ChineseSubFinder** 插件（“整理入库时通知 ChineseSubFinder 下载字幕”）仍然可以使用，它调用的是 `/api/v1/add-job`。

1. 在本程序的“实验室”里开启 API Key，复制 Key。
2. 在 MoviePilot 里安装 ChineseSubFinder 插件，填写：
   - 服务器地址，例如 `http://192.168.1.10:19035`
   - API Key
   - 如果两个容器里的媒体路径不同，填写本地路径（MoviePilot 里的路径）和远端路径（本程序里的路径）
3. 连续剧任务需要剧集目录里已经有 nfo，所以要在 MoviePilot 里开启刮削。

## 部署

- 容器内的目录：`/config`（配置、缓存、日志），`/media`（媒体，设置里的路径必须在 `/media` 下），Web 端口 `19035`。环境变量 `PUID`、`PGID`、`UMASK`、`TZ`、`PERMS` 的含义和上游一样，见 [docker/readme.md](docker/readme.md)。
- 媒体库要经过刮削。电影目录里最好有带 IMDB ID 的 nfo。目录结构要求见上游文档：[电影](https://github.com/ChineseSubFinder/ChineseSubFinder/blob/docs/DesignFile/%E7%94%B5%E5%BD%B1%E5%92%8C%E8%BF%9E%E7%BB%AD%E5%89%A7%E7%9B%AE%E5%BD%95%E7%BB%93%E6%9E%84%E7%A4%BA%E4%BE%8B.md)、[连续剧](https://github.com/ChineseSubFinder/ChineseSubFinder/blob/docs/DesignFile/%E8%BF%9E%E7%BB%AD%E5%89%A7%E7%9B%AE%E5%BD%95%E7%BB%93%E6%9E%84%E8%A6%81%E6%B1%82.md)。

## 编译

需要 Go 1.26、Node 22，以及 CGO（gcc）。前端会被嵌入可执行文件，所以要先编译前端：

```bash
npm --prefix frontend ci && npm --prefix frontend run build
go build -ldflags="-s -w" -o chinesesubfinder ./cmd/chinesesubfinder
```

测试：`go test ./...`。默认跳过需要联网的测试；运行测试时会使用临时配置目录，不会写入真实配置。

## 上游文档

下面的上游文档大部分仍然适用：

- [使用教程](https://github.com/ChineseSubFinder/ChineseSubFinder/tree/docs/DesignFile/使用教程)
- [对外 HTTP API](https://github.com/ChineseSubFinder/ChineseSubFinder/tree/docs/DesignFile/ApiKey%E8%AE%BE%E8%AE%A1)
- [字幕命名格式](https://github.com/ChineseSubFinder/ChineseSubFinder/blob/docs/DesignFile/%E5%85%B3%E4%BA%8E%E5%AD%97%E5%B9%95%E5%90%8D%E7%A7%B0%E5%91%BD%E5%90%8D%E6%A0%BC%E5%BC%8F%E8%AF%B4%E6%98%8E.md)
- [如何判断视频是否需要下载、更新字幕](https://github.com/ChineseSubFinder/ChineseSubFinder/blob/docs/DesignFile/%E5%A6%82%E4%BD%95%E5%88%A4%E6%96%AD%E8%A7%86%E9%A2%91%E6%98%AF%E5%90%A6%E9%9C%80%E8%A6%81%E4%B8%8B%E8%BD%BD%E3%80%81%E6%9B%B4%E6%96%B0%E5%AD%97%E5%B9%95%E7%9A%84.md)
- [问题列表](https://github.com/ChineseSubFinder/ChineseSubFinder/blob/docs/DesignFile/%E9%97%AE%E9%A2%98%E5%88%97%E8%A1%A8.md)

## 感谢

原作者 [allanpk716](https://github.com/allanpk716) 和上游贡献者：[iMyon](https://github.com/iMyon)（Web 前端）、[devome](https://github.com/devome)（Linux 和 Docker）、[宅宅还是度度](https://weibo.com/u/2884534224)（Logo）。

还要感谢这些项目：[GoSubTitleSearcher](https://github.com/Andyfoo/GoSubTitleSearcher)、[subfinder](https://github.com/ausaki/subfinder)、[sat](https://github.com/golandscape/sat)、[ffsubsync](https://github.com/smacke/ffsubsync)、[gohls](https://github.com/shimberger/gohls)。
