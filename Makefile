MODULE   := github.com/BenjaminBanwart/gw-bench
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT   := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE     := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS  := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

.PHONY: build test lint docker helm-lint clean

build:
	go build $(LDFLAGS) -o bin/gw-bench ./cmd/gw-bench/
	go build $(LDFLAGS) -o bin/test-backend ./cmd/test-backend/

test:
	go test -race ./...

lint:
	golangci-lint run ./...

docker:
	docker build -t gw-bench:$(VERSION) -f Dockerfile.runner .
	docker build -t gw-bench-backend:$(VERSION) -f Dockerfile.backend .

helm-lint:
	helm lint deploy/helm/gw-bench/
	helm template gw-bench deploy/helm/gw-bench/ -f deploy/helm/gw-bench/values.example.yaml

clean:
	rm -rf bin/
