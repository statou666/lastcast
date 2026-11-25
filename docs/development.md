[← 返回 README](../README.md)

# 开发与路线图

## 开发

```bash
# 编译（不需要 CGO）
CGO_ENABLED=0 go build -o lastcast .

# 测试
go test ./...
go vet ./...

# 生成图标（需要 Pillow；图标是深靛底 + 橙色漩涡 + 白色麦克风）
python tools/make_icon.py
```

代码结构（Go，无框架）：

| 文件 | 作用 |
|---|---|
| `main.go` | 配置、路由、HTTP 处理、打包 |
| `auth.go` | 用户、会话、权限中间件 |
| `store.go` | SQLite schema 与迁移 |
| `rss.go` | RSS/Atom 解析与条目分类 |
| `downloader.go` / `downloads.go` / `segmenter.go` | 下载、转码、分段、专辑标签 |
| `stream.go` | Range 流式代理、B 站/PeerTube 解析 |
| `bilibili.go` / `ytdlp.go` | B 站 API 与 yt-dlp 集成 |
| `tts.go` | TTS 合成与流式朗读 |
| `gpodder.go` | gPodder API v2 与代理 RSS |
| `queue.go` / `favorites.go` / `batch.go` | 播放列表、收藏、批量操作 |
| `history.go` | 最近播放记录（仅真正播放的条目，上限 300） |
| `settings.go` | 设置、OPML、统计 |
| `web/` | 无构建步骤的 SPA（原生 JS + CSS） |

## 路线图

- [x] 视频源 / 文字源统一播放
- [x] 下载成带封面和元数据的有声书
- [x] gPodder API v2（即时播放）
- [x] 多用户
- [x] 分类、播放列表、收藏、时间线、多选管理
- [x] 完整版 / 精简版双镜像
- [x] 下载页实时进度、最近播放历史
- [ ] UI 继续打磨（更精细的动效与排版）
- [ ] 章节 / 时间戳（chapters）
- [ ] 更完善的离线与缓存策略
- [ ] 移动端原生外壳（可选）

欢迎提 Issue 和 PR，尤其是新源类型的适配和播放器兼容性问题。
