.PHONY: build clean test

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
build:
	docker run --rm \
		-v $(PWD):/src \
		-w /src \
		golang:1.26-alpine \
		go build -ldflags "-X main.version=$(VERSION)" -o mc-backup ./cmd/mc-backup

test:
	docker run --rm \
		-v $(PWD):/src \
		-w /src \
		golang:1.26-alpine \
		go test ./...

clean:
	rm -f mc-backup
