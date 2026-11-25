[← 返回 README](../README.md)

# 配置与使用

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

播放时**边播边预取下一段**：当前段播放的同时拉取下一段，段与段之间几乎无间隔；每段开头垫 120ms 静音，避免 TTS 吞掉前几个字。

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

## 数据目录结构

```
/data
├── lastcast.db            # SQLite（用户、订阅、条目、进度、收藏、下载、历史）
├── lastcast.db-wal
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
- 备份只需备份 `lastcast.db`（如果容器运行中，建议同时备份 `-wal`）。
- 旧版 `podcastman.db` 在首次启动时会自动改名为 `lastcast.db`，数据不丢。
