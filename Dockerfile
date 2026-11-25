# Two flavours from one file:
#   full : ffmpeg + yt-dlp baked in (default)  -> docker build -t lastcast:latest .
#   slim : no media tools                        -> docker build --build-arg MEDIA_TOOLS=0 -t lastcast:slim .
#
# The binary is built in a Go builder stage, so `docker build` needs no local pmbin.
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
COPY web ./web
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/lastcast .

FROM alpine:latest
ARG MEDIA_TOOLS=1
ARG VERSION=dev
RUN apk add --no-cache ca-certificates && \
    if [ "$MEDIA_TOOLS" = "1" ]; then apk add --no-cache ffmpeg yt-dlp; fi
LABEL org.opencontainers.image.title="Lastcast" \
      org.opencontainers.image.description="Self-hosted podcast/video library with streaming playback, TTS and gPodder sync" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.licenses="MIT"
WORKDIR /app
COPY --from=build /out/lastcast /app/lastcast
COPY web /app/web
ENV PORT=8563 DATA_DIR=/data WEB_DIR=/app/web HOME=/data TMPDIR=/data/cache/tmp
EXPOSE 8563
USER 65532:65532
ENTRYPOINT ["/app/lastcast"]
