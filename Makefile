.PHONY: all build clean run watch docker-build

all: build

build:
	@go build -o bin/eventstream *.go 

run: build
	@./bin/eventstream

clean:
	@rm -rf bin

watch:
	@air -c .air.toml

docker-build:
	@docker buildx build --platform linux/amd64,linux/arm64 -t ghcr.io/papawattu/cleanlog-eventstream:latest .

docker-push: docker-build
	@docker push ghcr.io/papawattu/cleanlog-eventstream:latest

