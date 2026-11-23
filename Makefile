.PHONY: build test vet image image-slim icons run

build:
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o pmbin .

test:
	go test ./...

vet:
	go vet ./...

image:
	docker build -t lastcast:latest .

image-slim:
	docker build --build-arg MEDIA_TOOLS=0 -t lastcast:slim .

icons:
	python tools/make_icon.py

run: build
	PORT=8563 DATA_DIR=./data WEB_DIR=./web ./pmbin
