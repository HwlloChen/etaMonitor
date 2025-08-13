# 版本信息
VERSION ?= 1.0.0
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date +%FT%T%z)

# Go 编译参数
GO_LDFLAGS := -X "etamonitor/internal/config.Version=$(VERSION)" \
              -X "etamonitor/internal/config.BuildTime=$(BUILD_TIME)" \
              -X "etamonitor/internal/config.GitCommit=$(GIT_COMMIT)"

.PHONY: all clean build build-frontend build-backend

all: clean build

clean:
	rm -rf internal/static/frontend-dist
	rm -rf ../output

build: build-frontend build-backend

build-frontend:
	@echo "Building frontend..."
	@chmod +x scripts/build_frontend.sh
	@./scripts/build_frontend.sh

build-backend:
	@echo "Ensuring Go dependencies..."
	@go mod tidy
	@echo "Building backend..."
	@mkdir -p ../output
	@go build -ldflags '$(GO_LDFLAGS)' -o ../output/etamonitor ./cmd/main.go

run:
	@echo "Starting etaMonitor..."
	cd ../output && ./etamonitor
