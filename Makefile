.PHONY: all build clean run watch docker-build docker-push

all: build

build:
	@go build -o bin/eventstream ./main.go

run: build
	@./bin/eventstream

clean:
	@rm -rf bin

watch:
	@air -c .air.toml

docker-build:
	@docker buildx build --platform linux/amd64 -t ghcr.io/papawattu/cleanlog-eventstream:latest .

docker-push: docker-build
	@docker push ghcr.io/papawattu/cleanlog-eventstream:latest

