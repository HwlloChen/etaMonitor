# 版本信息
VERSION ?= 1.0.2
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date +%FT%T%z)

# Go 编译参数
GO_LDFLAGS := -X "etamonitor/internal/config.Version=$(VERSION)" \
              -X "etamonitor/internal/config.BuildTime=$(BUILD_TIME)" \
              -X "etamonitor/internal/config.GitCommit=$(GIT_COMMIT)"

# 交叉编译目标平台
PLATFORMS := darwin/amd64 darwin/arm64 \
             freebsd/amd64 freebsd/arm64 \
             linux/386 linux/amd64 linux/arm linux/arm64 linux/loong64 linux/ppc64le linux/riscv64 linux/s390x \
             windows/386 windows/amd64 windows/arm64
# 发布目录
RELEASE_DIR := ../release

.PHONY: all clean build build-frontend build-backend run release

all: clean build

clean:
	rm -rf internal/static/frontend-dist
	rm -rf ../output
	rm -rf $(RELEASE_DIR)

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

release: build-frontend
	@echo "Building release packages for all platforms..."
	@rm -rf $(RELEASE_DIR)
	@mkdir -p $(RELEASE_DIR)
	@go mod tidy
	@for platform in $(PLATFORMS); do \
		OS=$$(echo $$platform | cut -d'/' -f1); \
		ARCH=$$(echo $$platform | cut -d'/' -f2); \
		binary_name="etamonitor"; \
		if [ "$$OS" = "windows" ]; then \
			binary_name="etamonitor.exe"; \
		fi; \
		output_dir="$(RELEASE_DIR)/etamonitor-$(VERSION)-$$OS-$$ARCH"; \
		echo "Building $$OS/$$ARCH..."; \
		mkdir -p $$output_dir; \
		GOOS=$$OS GOARCH=$$ARCH CGO_ENABLED=0 go build -ldflags '$(GO_LDFLAGS)' -o $$output_dir/$$binary_name ./cmd/main.go || continue; \
		cp ../LICENSE $$output_dir/ 2>/dev/null || touch $$output_dir/LICENSE; \
		cp ../README.md $$output_dir/ 2>/dev/null || touch $$output_dir/README.md; \
		(cd $(RELEASE_DIR) && tar -czf etamonitor-$(VERSION)-$$OS-$$ARCH.tar.gz etamonitor-$(VERSION)-$$OS-$$ARCH/ ); \
		(cd $(RELEASE_DIR) && sha256sum etamonitor-$(VERSION)-$$OS-$$ARCH.tar.gz > etamonitor-$(VERSION)-$$OS-$$ARCH.tar.gz.sha256 2>/dev/null || shasum -a 256 etamonitor-$(VERSION)-$$OS-$$ARCH.tar.gz > etamonitor-$(VERSION)-$$OS-$$ARCH.tar.gz.sha256); \
		rm -rf $$output_dir; \
	done
	@echo "Release packages and SHA256 checksums created in $(RELEASE_DIR)/"
	@ls -la $(RELEASE_DIR)/