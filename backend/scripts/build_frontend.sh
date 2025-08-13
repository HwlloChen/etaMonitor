#!/bin/bash
set -e

# 检查 node 和 npm
command -v node >/dev/null 2>&1 || { echo >&2 "node 未安装，请先安装 node.js"; exit 1; }
command -v npm >/dev/null 2>&1 || { echo >&2 "npm 未安装，请先安装 npm"; exit 1; }

# 进入前端目录
cd ../frontend

# 安装依赖
echo "Installing frontend dependencies..."
npm install

# 构建前端
echo "Building frontend..."
npm run build


# 创建嵌入目录
mkdir -p ../backend/internal/static/frontend-dist

# 清空目标目录，防止旧文件残留
rm -rf ../backend/internal/static/frontend-dist/*

# 复制构建结果
echo "Copying build files..."
cp -a dist/* ../backend/internal/static/frontend-dist/

echo "Frontend build complete!"
