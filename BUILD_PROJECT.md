# etaMonitor 项目构建指南

本项目采用前后端分离架构，支持一键全自动构建。你可以直接在项目根目录下使用 Makefile 进行所有常用操作。

## 版本信息

构建时会自动注入版本号、构建时间和 Git 提交信息。默认版本号为 0.1.0，你可以通过 VERSION 环境变量指定版本号：

```sh
# 使用默认版本号构建
make

# 指定版本号构建
VERSION=1.0.0 make
```

启动时会显示版本信息：
```
=== etaMonitor 1.0.0 ===
Build Time: 2025-08-13T16:30:45+0800
Git Commit: a1b2c3d
========================
```

## 依赖环境

- Go 1.18 及以上
- Node.js 16+ 和 npm
- Linux/macOS/WSL（Windows 用户建议用 WSL）

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

## 目录结构说明

- `frontend/`：前端源码
- `backend/`：后端源码及 Makefile
- `backend/internal/static/frontend-dist/`：前端构建产物，go:embed 自动嵌入
- `output/etamonitor`：最终生成的后端二进制文件

## 常见问题

- **go:embed 找不到文件**：请确保已执行 `make frontend`，且 `backend/internal/static/frontend-dist/` 下有内容。
- **Node/npm 未安装**：请先安装 Node.js 和 npm。
- **端口被占用**：请检查 8080 端口是否被其他程序占用(你可以通过配置文件来指定端口号)。
- **版本信息显示 unknown**：请确保在 Git 仓库中构建，否则无法获取 Git 提交信息。
- **想要发布新版本**：使用 `VERSION=x.y.z make` 指定版本号构建。
