# etaMonitor 项目构建指南

本项目采用前后端分离架构，支持一键全自动构建。你可以直接在项目根目录下使用 Makefile 进行所有常用操作。

## 版本信息

构建时会自动注入版本号、构建时间和 Git 提交信息。默认版本号为 1.0.1，你可以通过 VERSION 环境变量指定版本号：

```sh
# 使用默认版本号构建
make

# 指定版本号构建
VERSION=1.2.0 make
```

启动时会显示版本信息：
```
=== etaMonitor 1.2.0 ===
Build Time: 2025-08-14T21:30:45+0800
Git Commit: a1b2c3d
========================
```

## 依赖环境

- Go 1.18 及以上
- Node.js 16+ 和 npm
- Linux/macOS/WSL（Windows 用户建议用 WSL）
- tar 工具（发布打包时需要）

## 一键构建

在项目根目录下执行：

```sh
make
```

等价于：
```sh
make build
```

该命令会自动：
1. 进入 backend 目录
2. 构建前端（自动安装依赖并构建）
3. 拷贝前端产物到后端嵌入目录
4. 构建后端可执行文件 `etamonitor`（输出到 output 目录）

## 运行

构建完成后，直接运行：

```sh
./output/etamonitor
```

或使用：

```sh
make run
```

（等价于进入 output 目录后执行 ./etamonitor）

## 发布打包

### 创建多平台发布包

使用以下命令创建适用于多个操作系统和架构的发布包：

```sh
make release
```

或指定版本号：

```sh
VERSION=1.2.0 make release
```

该命令会：

1. 构建前端资源
2. 为以下平台交叉编译二进制文件：
   - **Linux**: amd64, arm64, arm, 386
   - **Windows**: amd64, 386, arm64
   - **macOS**: amd64 (Intel), arm64 (Apple Silicon)
   - **FreeBSD**: amd64, arm64

3. 为每个平台创建包含以下文件的 tar.gz 压缩包：
   - 对应平台的可执行文件（`etamonitor` 或 `etamonitor.exe`）
   - `LICENSE` 文件
   - `README.md` 文件

4. 生成的压缩包命名格式：`etamonitor-{版本号}-{操作系统}-{架构}.tar.gz`

### 发布包示例

构建完成后，`release/` 目录下会包含类似以下文件：

```
release/
├── etamonitor-1.2.0-linux-amd64.tar.gz
├── etamonitor-1.2.0-linux-arm64.tar.gz
├── etamonitor-1.2.0-linux-arm.tar.gz
├── etamonitor-1.2.0-linux-386.tar.gz
├── etamonitor-1.2.0-windows-amd64.tar.gz
├── etamonitor-1.2.0-windows-386.tar.gz
├── etamonitor-1.2.0-windows-arm64.tar.gz
├── etamonitor-1.2.0-darwin-amd64.tar.gz
├── etamonitor-1.2.0-darwin-arm64.tar.gz
├── etamonitor-1.2.0-freebsd-amd64.tar.gz
└── etamonitor-1.2.0-freebsd-arm64.tar.gz
```

用户下载对应平台的压缩包后，解压即可使用：

```sh
# 示例：Linux AMD64 用户
tar -xzf etamonitor-1.2.0-linux-amd64.tar.gz
cd etamonitor-1.2.0-linux-amd64/
./etamonitor
```

## 其他常用命令

- 仅构建前端：

  ```sh
  make frontend
  ```

- 仅构建后端：

  ```sh
  make backend
  ```

- 清理所有构建产物：

  ```sh
  make clean
  ```

  注意：`make clean` 会同时清理 `output/`、`release/` 和前端构建产物。

## 目录结构说明

- `frontend/`：前端源码
- `backend/`：后端源码及 Makefile
- `backend/internal/static/frontend-dist/`：前端构建产物，go:embed 自动嵌入
- `output/etamonitor`：开发构建的后端二进制文件
- `release/`：发布打包的多平台压缩包

## 开发流程建议

1. **日常开发**：使用 `make` 进行快速构建和测试
2. **本地测试**：使用 `make run` 启动服务
3. **版本发布**：使用 `VERSION=x.y.z make release` 创建发布包
4. **清理环境**：使用 `make clean` 清理所有构建产物

## 常见问题

- **go:embed 找不到文件**：请确保已执行 `make frontend`，且 `backend/internal/static/frontend-dist/` 下有内容。
- **Node/npm 未安装**：请先安装 Node.js 和 npm。
- **端口被占用**：请检查 8080 端口是否被其他程序占用（你可以通过配置文件来指定端口号）。
- **版本信息显示 unknown**：请确保在 Git 仓库中构建，否则无法获取 Git 提交信息。
- **交叉编译失败**：某些平台可能因为 CGO 依赖而编译失败，这是正常现象，成功的平台会正常打包。
- **发布包过大**：如果二进制文件过大，可以在 GO_LDFLAGS 中添加 `-s -w` 参数来减小文件大小：
  ```makefile
  GO_LDFLAGS := -s -w -X "etamonitor/internal/config.Version=$(VERSION)" ...
  ```
- **tar 命令未找到**：Windows 用户请在 WSL 环境中执行，或安装 tar 工具。

## 版本管理建议

- 使用语义化版本号（Semantic Versioning）：`主版本.次版本.修订版`
- 主版本：不兼容的 API 修改
- 次版本：向后兼容的功能性新增
- 修订版：向后兼容的问题修正
