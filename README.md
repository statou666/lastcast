<div align="center">

<img src="web/icons/icon-192.png" width="110" alt="Lastcast">

# Lastcast

**自托管播客 / 视频 / 文字 RSS 管理器 —— 单二进制、SQLite、Docker 部署**

把音频播客、视频播客（B 站 / YouTube / PeerTube…）和文字 RSS 统一成「可以听的节目」：
订阅、在线播放、下载成带封面和元数据的有声书、多设备同步进度，还能被 AntennaPod 等播客客户端通过 gPodder API 连接。

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![Docker](https://img.shields.io/badge/Docker-ready-2496ED?logo=docker&logoColor=white)](#部署)
[![License](https://img.shields.io/badge/license-MIT-green)](#license)
[![PWA](https://img.shields.io/badge/PWA-mobile%20ready-5A0FC8)](#手机--pwa)

</div>

---

## 目录

- [为什么做这个](#为什么做这个)
- [功能特性](#功能特性)
- [支持的源类型](#支持的源类型)
- [部署](#部署)
  - [方式一：Docker（完整版，推荐）](#方式一docker完整版推荐)
  - [方式二：Docker 精简版（不带 ffmpeg / yt-dlp）](#方式二docker-精简版不带-ffmpeg--yt-dlp)
  - [方式三：docker compose](#方式三docker-compose)
  - [方式四：直接运行二进制](#方式四直接运行二进制)
  - [首次使用](#首次使用)
  - [反向代理 / HTTPS](#反向代理--https)
  - [升级与备份](#升级与备份)
- [配置说明](#配置说明)
- [文本转语音（TTS）](#文本转语音tts)
- [gPodder 客户端（AntennaPod 等）](#gpodder-客户端antennapod-等)
- [数据目录结构](#数据目录结构)
- [常见问题](#常见问题)
- [开发](#开发)
- [路线图](#路线图)
- [License](#license)
- [支持作者](#支持作者)

---

## 为什么做这个

这是我通过 AI 写的播客管理器，因为我想在 Docker 上部署自己的播客，但是没找到合适的，所以自己用 AI 写了个。

但是实现了一些别的功能。

**特色功能：**

1. **能够订阅和收听视频 / 文字 RSS 源**：因为视频播客的兴起和 RSS 的没落，所以与时俱进，让视频和文字都整合到播客里——视频通过 ffmpeg 和 yt-dlp 来实现，文字通过 TTS 服务来实现（TTS 需自己配置）。
2. **下载功能**：能将音频以有声书的方式下载下来，自带封面和元数据，可以用这个应用管理，也可以用专门的有声书应用来管理。
3. **gPodder API**：所有源都能通过支持 gPodder API 的应用连接和播放；这个功能由 AI 实现，不一定好用，欢迎测试。
4. **多用户管理**：可以多用户共同使用、互不影响，但是普通用户没有修改全局设置和管理其他用户的能力。

推荐配合 [RSSHub](https://github.com/DIYgod/RSSHub) 使用：任何 RSSHub 能生成的源（B 站、YouTube、知乎、微博……）都可以直接订阅，能够扩展这个应用的使用范围。

---

## 功能特性

### 订阅管理

- **RSS / Atom 通用解析**，不预设源的类型：音频、视频、文字都按条目内容自动识别。
- **分类**：订阅可按自定义分类分组显示，分类完全由你管理。
- **源详情编辑**：可修改源地址（改完立即重新抓取）、分类、类型（自动 / 音频 / 视频 / 文字）、「此源更新时自动下载新集」和「默认抓取全文」。
- **文字源全文抓取**：打开「默认抓取全文」后，新条目会在后台抓取原网页正文；条目详情里也随时可以点「获取全文」。正文和文章内图片都会显示在详情页（通用正文/图片提取，不针对某个站点写规则）。
- **OPML 导入 / 导出**：可以直接**选择本机的 `.opml` 文件**（也会自动读入内容，或手动粘贴），导入时逐源抓取并自动识别它是音频、视频还是文字；分类不从 OPML 推断，保持你自己管理。
- 定时抓取（默认 45 分钟），支持 ETag / Last-Modified，条目按 GUID 去重。
- **更新时间线**：更新页按「更新于」倒序、按天分组，每条都显示多久之前更新。
- **多选批量操作**：更新页、源详情可批量下载 / 加入播放列表 / 收藏 / 标记已听完；订阅页可批量删除；播放列表、收藏、下载页也都有对应的批量管理。

### 播放

- **默认流式播放，不落盘**：`/stream/{id}` 把源站媒体以 Range 方式透传给浏览器，点开就播、可拖动，不占磁盘。
- **视频源当音频听**：B 站 / PeerTube / 直链视频都只取音轨在线播放；B 站通过 bvid 调官方 API 解析 DASH 音频，冷启动约 1 秒。
- **YouTube 等「只有网页」的源**：内置 yt-dlp 兜底解析（完整版镜像），同样即时播放。
- **文字源 TTS 朗读**：严格按**段落**分段（一个段落一段；段落过长再在段内按句/字数切小段），生成一段就能听一段，不用等整集。播放只写缓存，不会自动变成下载；想要永久文件请在条目/订阅里点「下载」。
- 播放条常驻：±30 秒、0.8–2 倍速、音量、睡眠定时（15/30/45/60 分钟或本集播完即停）、节目笔记。
- **跨设备进度**：每台设备独立记录进度并同步，换设备继续听。
- **播放列表（队列）**：有序、下一首播放、上下调整、随机、清空、自动连播；跨设备同步。
- **收藏**：RSS 阅读器式星标，默认只收藏不下载，可在设置里开启「收藏单集自动下载」。

### 下载 / 有声书

- 下载目录按订阅分文件夹：`下载目录/<订阅标题>/<条目标题>.<扩展名>`。
- 可选下载格式：自动（跟随源）/ MP3 / M4A；同编码时直接 copy，不重复转码。
- 每个成品都写成「专辑曲目」：内嵌封面，并写入 `title / album / artist(源名) / album_artist(源名) / track=N/总数 / date(发布日期) / genre=Podcast / description(条目正文) / synopsis / comment`；订阅文件夹里还有 `cover.jpg`。
- 标签写入失败时会自动重试（先去掉封面、再把 m4a 退回 mp3），不会因为封面问题丢掉全部元数据。
- 视频源下载时只保留音轨；长视频会分块下载再本地提取，避免 CDN 长连接假死导致文件截断。
- 下载独立于订阅：删除订阅**不会**删除已下载文件，可在「下载」页单独管理、批量删除。
- 打包：一个订阅可导出为 zip（`book.json` + 按顺序命名的音频），方便导入有声书应用。
- 缓存与下载分离：缓存可随时清理 / 按上限自动淘汰，下载永久保留。

### 多用户

- PBKDF2-SHA256 加盐密码 + HttpOnly 会话 Cookie，30 天有效。
- 每个用户拥有自己的订阅、下载、进度、播放列表、收藏和统计；媒体接口会校验归属。
- 管理员管理用户和实例级设置；普通用户可以改自己的密码，但不能改全局设置。
- 首次启动进入 `/setup` 创建管理员，已有旧数据会自动归属到第一个账号。

### 其他

- **gPodder API v2**：见下方专节。
- **HTTP 代理**：设置页可填代理地址，作用于订阅抓取、媒体流、下载、B 站 API、yt-dlp、ffmpeg/ffprobe。
- **PWA**：手机浏览器「添加到主屏幕」即可全屏使用，深色 / 浅色主题，移动端底部标签栏 + 自适应播放条。
- **TTS 音色自动检测**：设置页打开时会自动检测，点「检测音色」重新检测；检测结果会**列成可点击的音色列表**（显示 id 和音色名），也可以直接手填任意 voice id。调用的是兼容服务商的 `/audio/voices`（Kokoro、openedai-speech 等），不绑定具体实现。
- 切换文字源时旧音频会立即暂停，不再等新条目的分段生成；后台渲染和播放取消防竞争处理。
- **中英双语**界面。
- 单二进制、无 CGO 依赖，SQLite 存数据，低配设备（树莓派等）也能跑。

---

## 支持的源类型

| 源类型 | 识别方式 | 播放方式 | 下载 |
|---|---|---|---|
| 音频播客 | 条目带音频 enclosure | 直接流式代理，或播放已下载文件 | 原始音频 + 标签封面 |
| 视频播客（B 站、PeerTube、直链 mp4…） | HTML 播放页 / 视频 enclosure / bvid / 视频站点特征 | 解析出音轨后 Range 代理，即时播放 | 提取音轨为单个 m4a/mp3 |
| 文字 RSS | 没有 enclosure，只有正文 | TTS 分段合成，可边生成边播 | 合成完整 mp3 |
| 需要网页解析的源（YouTube 等） | 无直链、只有播放页 | 内置 yt-dlp 解析后代理 | yt-dlp 抓最佳音轨再转码 |

> 程序**不会预设** RSS 来源是什么类型，也不会在后台「预解析」：只有你真正播放或下载时才按顺序尝试解析。

---

## 部署

运行环境只需要 Docker（或一个能跑静态二进制的 Linux）。镜像分两个版本，按需选择：

| | 完整版 `lastcast:latest` | 精简版 `lastcast:slim` |
|---|---|---|
| 体积 | 约 511 MB | 约 43.5 MB |
| ffmpeg / ffprobe / yt-dlp | 内置 | 不带，需外部提供 |
| 在线播放 / TTS | ✅ | ✅ |
| 下载（转码、标签、封面） | ✅ | ⚠️ 无 ffmpeg 时退化为原始音频直接保存 |
| YouTube 等需要 yt-dlp 的源 | ✅ | ⚠️ 需外部 yt-dlp |

### 预构建镜像（推荐，CI 构建）

仓库用 GitHub Actions 自动构建 **linux/amd64 + linux/arm64** 两种架构，推到两个地方：

| 来源 | 地址 | 说明 |
|---|---|---|
| GitHub Container Registry（默认） | `ghcr.io/statou666/lastcast:latest` / `:latest-slim` | 无需额外配置，tag 触发即构建 |
| Docker Hub（可选） | `docker.io/<你的用户名>/lastcast` | 在仓库 Secrets 里配置后自动同步 |

直接使用：

```bash
docker pull ghcr.io/statou666/lastcast:latest       # 完整版
docker pull ghcr.io/statou666/lastcast:latest-slim  # 精简版
```

镜像 tag 跟着 git tag 走：`v0.1.1` → `0.1.1`、`0.1.1-slim`，同时更新 `latest` / `latest-slim`。

### 方式一：Docker（完整版，推荐）

```bash
# 1. 构建镜像（多阶段构建，会自动编译 Go 二进制；第一次会下载依赖，约 1–3 分钟）
git clone https://github.com/statou666/lastcast.git
cd lastcast
docker build -t lastcast:latest .

# 2. 创建数据卷（只需一次）
docker volume create lastcast-data

# 3. 启动
docker run -d \
  --name lastcast \
  --restart unless-stopped \
  -p 8563:8563 \
  -v lastcast-data:/data \
  --memory 256m \
  lastcast:latest
```

打开 `http://<主机IP>:8563`，首次会进入 `/setup` 创建管理员账号。

如果想把下载目录放在宿主机上（方便直接拿文件）：

```bash
docker run -d --name lastcast --restart unless-stopped \
  -p 8563:8563 \
  -v lastcast-data:/data \
  -v /srv/lastcast/downloads:/data/downloads \
  -e TZ=Asia/Shanghai \
  lastcast:latest
```

> 容器以 UID `65532` 运行。挂载宿主机目录时请确保该目录可写：
> `sudo chown -R 65532:65532 /srv/lastcast/downloads`。
> 直接使用命名卷（`lastcast-data`）则不需要处理权限。

### 方式二：Docker 精简版（不带 ffmpeg / yt-dlp）

适合已经有 ffmpeg / yt-dlp、或想保持镜像极小的场景：

```bash
docker build --build-arg MEDIA_TOOLS=0 -t lastcast:slim .

docker run -d --name lastcast --restart unless-stopped \
  -p 8563:8563 -v lastcast-data:/data lastcast:slim
```

没有媒体工具时：在线播放、TTS 正常；下载会退化为「原始音频直接保存」；YouTube 等必须 yt-dlp 的源会明确报错。

要让精简版恢复完整能力，可以把二进制挂进容器（**必须是 musl 静态版**；glibc 静态版 ffmpeg 在 Alpine 里无法解析域名）：

```bash
docker run -d --name lastcast --restart unless-stopped \
  -p 8563:8563 -v lastcast-data:/data \
  -v /opt/tools/ffmpeg:/usr/local/bin/ffmpeg:ro \
  -v /opt/tools/ffprobe:/usr/local/bin/ffprobe:ro \
  -v /opt/tools/yt-dlp:/usr/local/bin/yt-dlp:ro \
  lastcast:slim
```

或者把工具放进 `/data/bin`，用环境变量指定：

```bash
-e FFMPEG_PATH=/data/bin/ffmpeg
-e FFPROBE_PATH=/data/bin/ffprobe
-e YTDLP_PATH=/data/bin/yt-dlp
```

### 方式三：docker compose

仓库自带两个 compose 文件：

```bash
# 完整版（数据卷 lastcast-data 会自动创建）
docker compose up -d --build

# 精简版
docker compose -f docker-compose.slim.yml up -d --build
```

compose 里已经包含 TTS、抓取间隔、内存限制等示例配置，按需修改即可。

### 方式四：直接运行二进制

```bash
CGO_ENABLED=0 go build -o lastcast .
PORT=8563 DATA_DIR=./data WEB_DIR=./web ./lastcast
```

需要同目录下有 `web/` 静态文件；ffmpeg / yt-dlp 可选，路径用 `FFMPEG_PATH` / `YTDLP_PATH` 指定。

### 首次使用

1. 浏览器打开 `http://localhost:8563`，进入 `/setup` 创建管理员（用户名 2–32 字符，密码至少 6 位）。
2. 设置 → 用户 里可以添加其他账号。
3. 订阅页添加第一个源，或导入 OPML。
4. 设置 → 语音合成 配置 TTS 后，文字源才能「朗读」。
5. 手机：浏览器菜单「添加到主屏幕」，即可作为 PWA 使用。

### 反向代理 / HTTPS

Lastcast 本身只监听 HTTP。放在 Nginx / Caddy 后面时：

```nginx
location / {
    proxy_pass http://127.0.0.1:8563;
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_buffering off;          # 媒体流式播放
    proxy_read_timeout 3600s;
}
```

带 `X-Forwarded-Proto: https` 时，登录 Cookie 会自动加 `Secure`。gPodder 客户端和 PWA 都能在 HTTPS 下正常工作。

### 升级与备份

```bash
# 备份数据库（强烈建议每次升级前做）
docker cp lastcast:/data/podcastman.db ./podcastman-$(date +%F).db
docker cp lastcast:/data/podcastman.db-wal ./podcastman-$(date +%F).db-wal

# 拉新代码 / 新镜像后重建容器，数据卷不变
docker rm -f lastcast
docker run -d ...（同上面的启动命令）
```

数据库是 SQLite，启动时会自动迁移表结构；升级不需要手动改库。

---

## 配置说明

所有设置都可以在 **设置** 页面修改（会持久化到数据库，重启不丢）。也可以用环境变量提供默认值：

| 环境变量 | 默认 | 说明 |
|---|---|---|
| `PORT` | `8563` | 监听端口 |
| `DATA_DIR` | `./data` | 数据目录（数据库、缓存、下载） |
| `WEB_DIR` | `./web` | 静态文件目录 |
| `FETCH_INTERVAL_MIN` | `45` | 定时抓取间隔（分钟） |
| `DOWNLOAD_CONCURRENCY` | `2` | 同时下载数（1–8） |
| `AUTO_DOWNLOAD_NEW` | `0` | 全局「新集自动下载」 |
| `STORAGE_LIMIT_MB` | `0` | 缓存+下载总上限，0 = 不限 |
| `CACHE_DIR` / `DOWNLOAD_DIR` | `cache` / `downloads` | 相对 `DATA_DIR` 的子目录 |
| `CACHE_LIMIT_MB` | `0` | 缓存上限，0 = 不限 |
| `DOWNLOAD_FORMAT` | `auto` | `auto` / `mp3` / `m4a` |
| `PLAY_MODE` | `stream` | `stream` 流式（不落盘）/ `segment` 缓存分段 |
| `FFMPEG_PATH` | `ffmpeg` | ffmpeg 可执行文件路径 |
| `FFPROBE_PATH` | 同目录 `ffprobe` | ffprobe 路径 |
| `YTDLP_PATH` | `yt-dlp` | yt-dlp 路径 |
| `PROXY_URL` | 空 | HTTP 代理（如 `http://127.0.0.1:7890`） |
| `TTS_PROVIDER` | `off` | `off` / `openai`（OpenAI 兼容接口） |
| `TTS_BASE_URL` | `https://api.openai.com/v1` | TTS 服务地址 |
| `TTS_KEY` | 空 | TTS API Key |
| `TTS_MODEL` | `tts-1` | 模型名 |
| `TTS_VOICE` | `alloy` | 音色 |
| `TTS_SPEED` | `1.0` | 语速 |
| `TTS_MAX_CHARS` | `3000` | 每次 TTS 请求最大字符数 |
| `TTS_AUTO` | `1` | 新文字条目自动朗读 |
| `UI_THEME` | `dark` | `dark` / `light` |

> 设置页保存过的值优先于环境变量；`PORT` / `DATA_DIR` / `WEB_DIR` 属于进程级配置，只能用环境变量。

---

## 文本转语音（TTS）

TTS 用来把文字 RSS 朗读成音频。任何兼容 OpenAI `/audio/speech` 的服务都可以用（OpenAI、LocalAI、one-api、vLLM-TTS 等）。

在 **设置 → 语音合成** 中填写：

| 项 | 示例 |
|---|---|
| 服务商 | `OpenAI-compatible` |
| Base URL | `http://127.0.0.1:5050/v1`（容器内可用 `http://host.docker.internal:5050/v1`） |
| API Key | 你的 key（没有就留空） |
| 模型 | `tts-1` |
| 音色 | 打开设置会自动检测并列出音色，点击选用；也可手填，如 `af_heart`、`zh-CN-XiaoxiaoNeural`、`alloy` |

保存后点 **测试**，能返回字节数就说明配置成功。文字条目的播放/下载都会走这个服务。

---

## gPodder 客户端（AntennaPod 等）

Lastcast 实现了 gPodder API v2：订阅、设备、播放进度双向同步；并且**视频源和文字源都能即时播放**，不需要先下载。

在 AntennaPod 中：

1. **设置 → 同步 → gPodder**；
2. 服务器填 `http://<主机>:8563`（设置页「播客客户端」卡片会显示当前地址）；
3. 用户名 / 密码填 Lastcast 的账号；
4. 在 Lastcast 的 **设置 → 播客客户端** 里复制某个源的地址（形如 `…/gpodder/feed/<token>/<id>.xml`），在 AntennaPod 里添加该源。

原理：订阅列表返回的是动态生成的代理 RSS，每条 enclosure 指回 Lastcast：

- 视频/音频 → `/stream/{id}`：Range 代理，点开即播、可拖动；
- 文字 → `/speak/{id}`：边合成边流式输出 mp3，首段生成完即可播放。

代理 URL 带每用户 token（客户端不带 Cookie）；gPodder 本身使用 HTTP Basic 认证。**建议只在局域网 / HTTPS 下使用。**

---

## 数据目录结构

```
/data
├── podcastman.db          # SQLite（用户、订阅、条目、进度、收藏、下载记录）
├── podcastman.db-wal
├── cache/                 # 可丢弃缓存
│   ├── covers/            # 封面缓存
│   ├── tmp/               # 下载/转码临时文件
│   └── <slug>/<id>/       # 分段缓存
└── downloads/             # 永久下载
    └── <用户名>/<订阅标题>/<条目标题>.<ext>
        ├── cover.jpg
        └── ...
```

- 删除订阅不会删除 `downloads/` 里的文件。
- 缓存可以随时清空（设置 → 缓存），不会影响下载。
- 备份只需备份 `podcastman.db`（如果容器运行中，建议同时备份 `-wal`）。

---

## 常见问题

**Q：为什么更新页的条目没有封面？**
视频源/文字源会取正文里的第一张图作为封面，音频源用订阅封面；旧数据在启动时会自动回填。没有图片的源会显示占位块。

**Q：B 站源能直接播吗？需要登录吗？**
公开视频可以直接播：程序调 B 站公开 API 取音轨 URL，再通过本机代理流式播放。需要登录/会员的内容暂不支持。

**Q：YouTube 源报「no playable audio stream」？**
完整版镜像内置 yt-dlp；精简版需要自己挂载 yt-dlp。另外部分站点需要代理，可在设置里填 HTTP 代理。

**Q：为什么完整版镜像这么大？**
里面打包了 ffmpeg（约 136 MB）和 yt-dlp（带 Python 依赖）。不需要的话用 `lastcast:slim`（约 43.5 MB），功能会按上表降级。

**Q：普通用户能做什么？**
每个用户只能看到/管理自己的订阅、下载、进度和收藏；全局设置和用户管理仅管理员可用。所有用户都可以改自己的密码。

**Q：数据库能换 PostgreSQL / MySQL 吗？**
目前只支持 SQLite（纯 Go 实现、无 CGO），单文件、易备份，适合个人和小团队。

**Q：支持多实例 / 集群吗？**
不支持。数据在本地 SQLite 和本地磁盘，设计目标是单机自托管。

**Q：忘记管理员密码怎么办？**
可以在容器里用 sqlite 修改 `users` 表（删除该用户后重新 `/setup`，或写入新的 PBKDF2 哈希）。建议平时保留数据库备份。

---

## 开发

```bash
# 编译（不需要 CGO）
CGO_ENABLED=0 go build -o pmbin .

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
| `settings.go` | 设置、OPML、统计 |
| `web/` | 无构建步骤的 SPA（原生 JS + CSS） |

---

## 路线图

- [x] 视频源 / 文字源统一播放
- [x] 下载成带封面和元数据的有声书
- [x] gPodder API v2（即时播放）
- [x] 多用户
- [x] 分类、播放列表、收藏、时间线、多选管理
- [x] 完整版 / 精简版双镜像
- [ ] UI 继续打磨（更精细的动效与排版）
- [ ] 章节 / 时间戳（chapters）
- [ ] 更完善的离线与缓存策略
- [ ] 移动端原生外壳（可选）

欢迎提 Issue 和 PR，尤其是新源类型的适配和播放器兼容性问题。

---

## License

MIT

---

## 支持作者

给我点 token 吧，给点吧。

<div align="center">
  <img src="docs/alipay.jpg" width="280" alt="支付宝收款码">
  <p>支付宝扫码</p>
</div>
