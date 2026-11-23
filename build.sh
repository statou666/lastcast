#!/usr/bin/env sh
# Build both Lastcast flavours:
#   ./build.sh 0.1.0
# tags: lastcast:0.1.0        / lastcast:full / lastcast:latest   (ffmpeg + yt-dlp)
#       lastcast:0.1.0-slim   / lastcast:slim                     (no media tools)
set -e
VERSION="${1:-dev}"

docker build \
  --build-arg MEDIA_TOOLS=1 --build-arg VERSION="$VERSION" \
  -t "lastcast:$VERSION" -t lastcast:full -t lastcast:latest .

docker build \
  --build-arg MEDIA_TOOLS=0 --build-arg VERSION="$VERSION" \
  -t "lastcast:$VERSION-slim" -t lastcast:slim .

echo "built lastcast:$VERSION (full) and lastcast:$VERSION-slim"
