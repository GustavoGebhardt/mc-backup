.PHONY: build clean test

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GOMODCACHE := $(shell go env GOPATH)/pkg/mod

build:
	docker run --rm \
		-v $(PWD):/src \
		-v $(GOMODCACHE):/go/pkg/mod \
		-w /src \
		golang:1.24-alpine \
		go build -ldflags "-X main.version=$(VERSION)" -o mc-backup ./cmd/mc-backup

test:
	docker run --rm \
		-v $(PWD):/src \
		-v $(GOMODCACHE):/go/pkg/mod \
		-w /src \
		golang:1.24-alpine \
		go test ./...

clean:
	rm -f mc-backup
