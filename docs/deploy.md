[← 返回 README](../README.md)

# 部署

运行环境只需要 Docker（或一个能跑静态二进制的 Linux）。镜像分两个版本，按需选择：

| | 完整版 `lastcast:latest` | 精简版 `lastcast:slim` |
|---|---|---|
| 体积 | 约 511 MB | 约 43.5 MB |
| ffmpeg / ffprobe / yt-dlp | 内置 | 不带，需外部提供 |
| 在线播放 / TTS | ✅ | ✅ |
| 下载（转码、标签、封面） | ✅ | ⚠️ 无 ffmpeg 时退化为原始音频直接保存 |
| YouTube 等需要 yt-dlp 的源 | ✅ | ⚠️ 需外部 yt-dlp |

## 预构建镜像（推荐，CI 构建）

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

## 方式一：Docker（完整版，推荐）

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

## 方式二：Docker 精简版（不带 ffmpeg / yt-dlp）

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

## 方式三：docker compose

仓库自带两个 compose 文件：

```bash
# 完整版（数据卷 lastcast-data 会自动创建）
docker compose up -d --build

# 精简版
docker compose -f docker-compose.slim.yml up -d --build
```

compose 里已经包含 TTS、抓取间隔、内存限制等示例配置，按需修改即可。

## 方式四：直接运行二进制

```bash
CGO_ENABLED=0 go build -o lastcast .
PORT=8563 DATA_DIR=./data WEB_DIR=./web ./lastcast
```

需要同目录下有 `web/` 静态文件；ffmpeg / yt-dlp 可选，路径用 `FFMPEG_PATH` / `YTDLP_PATH` 指定。

## 首次使用

1. 浏览器打开 `http://localhost:8563`，进入 `/setup` 创建管理员（用户名 2–32 字符，密码至少 6 位）。
2. 设置 → 用户 里可以添加其他账号。
3. 订阅页添加第一个源，或导入 OPML。
4. 设置 → 语音合成 配置 TTS 后，文字源才能「朗读」。
5. 手机：浏览器菜单「添加到主屏幕」，即可作为 PWA 使用。

## 反向代理 / HTTPS

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

## 升级与备份

```bash
# 备份数据库（强烈建议每次升级前做）
docker cp lastcast:/data/lastcast.db ./lastcast-$(date +%F).db
docker cp lastcast:/data/lastcast.db-wal ./lastcast-$(date +%F).db-wal

# 拉新代码 / 新镜像后重建容器，数据卷不变
docker rm -f lastcast
docker run -d ...（同上面的启动命令）
```

数据库是 SQLite，启动时会自动迁移表结构；升级不需要手动改库。旧版 `podcastman.db` 在首次启动时会自动改名为 `lastcast.db`，数据不丢。
