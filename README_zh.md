# etaMonitor: 一个Minecraft服务器监控系统

[![Go Version](https://img.shields.io/badge/Go-1.23.0-blue.svg)](https://golang.org/)
[![Vue.js](https://img.shields.io/badge/Vue.js-3.3+-4FC08D.svg)](https://vuejs.org/)
[![License](https://img.shields.io/badge/License-MPL%20v2-blue.svg)](LICENSE)

_查看我们的 [Demo站点](https://em.etaris.moe)_

- [中文文档](README_zh.md)
- [English](README.md)

## 概述

etaMonitor 是一款功能强大的自部署 Minecraft 服务器监控系统，提供实时监控、数据分析和美观的 Web 界面。

### 核心特性

- 🌐 **实时监控**: 基于 WebSocket 的实时服务器状态监控
- 💾 **数据持久化**: SQLite 数据库存储历史监控数据
- 📊 **数据分析**: 玩家在线统计与趋势分析，支持图表可视化
- 🔔 **活动通知**: 实时玩家加入/退出通知（15分钟内活动记录）
- 🖥️ **多服务器**: 支持同时监控多个 Minecraft 服务器
- 🎮 **版本兼容**: 自动检测 Java 版/基岩版服务器
- 🌐 **SRV 记录**: 支持 SRV DNS 记录解析
- 🔐 **安全认证**: JWT 身份验证和 HTTPS 安全提醒
- 💼 **数据库管理**: 支持数据库备份、恢复和智能整理优化
- 🎨 **现代界面**: 基于 MDUI 的 Material Design 3 风格界面
- 📱 **响应式设计**: 完美适配桌面和移动设备，图表控件自适应屏幕尺寸

## 快速开始

### 1. 下载安装

从 [GitHub Releases](https://github.com/HwlloChen/etaMonitor/releases/latest) 下载对应平台的最新版本：

### 2. 首次启动

```bash
./etamonitor
```

首次启动会自动：

- 创建默认配置文件 `config.json`, 详见[配置文件](#配置文件)
- 要求设置管理员账户和密码
- 初始化 SQLite 数据库

### 3. 访问系统

默认访问地址：`http://127.0.0.1:11451`

使用刚才设置的管理员账户登录，即可开始添加和监控 Minecraft 服务器。

## 配置文件

### 指定配置文件路径

```bash
./etamonitor -c /path/to/your/config.json
```

### 配置详解

完整配置文件示例：

```json
{
  "server": {
    "host": "127.0.0.1",
    "port": "11451", 
    "environment": "release"
  },
  "database": {
    "path": "./data/etamonitor.db"
  },
  "jwt": {
    "secret": "your-secret-key-change-in-production",
    "expires_in": "24h"
  },
  "monitor": {
    "interval": "10s",
    "ping_timeout": "10s", 
    "max_concurrent": 10,
    "activity_retention_time": "15m"
  },
  "logging": {
    "level": "info",
    "format": "json"
  },
  "cors": {
    "allow_origins": ["*"],
    "allow_credentials": true
  }
}
```

#### 配置说明

**服务器配置**:

- `server.host`: 监听地址
- `server.port`: 监听端口（默认 11451）
- `server.environment`: 运行环境 (release/debug)

**数据库配置**:

- `database.path`: SQLite 数据库文件路径

**JWT 配置**:

- `jwt.secret`: JWT 密钥（生产环境务必修改，默认随机生成）
- `jwt.expires_in`: JWT 令牌有效期 (例如: 24h, 7d)

**监控配置**:

- `monitor.interval`: 监控检查间隔（建议 5-30 秒）
- `monitor.ping_timeout`: 服务器 Ping 超时时间
- `monitor.max_concurrent`: 最大并发监控数量
- `monitor.activity_retention_time`: 玩家活动记录保留时间 (例如: 15m, 30m)

**日志配置**:

- `logging.level`: 日志级别 (debug, info, warn, error, fatal)
- `logging.format`: 日志格式 (text, json)

**CORS 配置**:

- `cors.allow_origins`: 允许跨域请求的来源
- `cors.allow_credentials`: 允许发送 Cookie

### 环境变量支持

支持通过环境变量覆盖配置：

```bash
# 基本配置
export HOST=0.0.0.0
export PORT=8080
export GIN_MODE=release

# 数据库配置
export DB_PATH=/var/lib/etamonitor/data.db

# JWT配置
export JWT_SECRET=your-production-secret-key
export JWT_EXPIRES_IN=7d

# 监控配置
export MONITOR_INTERVAL=15s
export PING_TIMEOUT=5s
export MAX_CONCURRENT=20
export ACTIVITY_RETENTION_TIME=30m

# 日志配置
export LOG_LEVEL=warn
export LOG_FORMAT=text
```

## 部署指南

### 持久化运行（Systemd）

创建系统服务文件：

```bash
sudo nano /etc/systemd/system/etamonitor.service
```

```ini
[Unit]
Description=etaMonitor - Minecraft Server Monitor
After=network.target

[Service]
Type=simple
WorkingDirectory=/path/to/etamonitor
ExecStart=/path/to/etamonitor/etamonitor # 更改为可执行程序路径
Restart=always
RestartSec=5
# Environment=GIN_MODE=release
# Environment=HOST=127.0.0.1
# Environment=PORT=11451

[Install]
WantedBy=multi-user.target
```

启用和启动服务：

```bash
sudo systemctl daemon-reload
sudo systemctl enable etamonitor
sudo systemctl start etamonitor
sudo systemctl status etamonitor
```

### 反向代理（Nginx）

此处使用 Nginx 进行反向代理以启用 HTTPS：

```nginx
server {
    listen 80;
    server_name your-domain.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name your-domain.com;

    # SSL 配置
    ssl_certificate /path/to/your/certificate.crt;
    ssl_certificate_key /path/to/your/private.key;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-RSA-AES128-GCM-SHA256:ECDHE-RSA-AES256-GCM-SHA384;

    # 安全头
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header Referrer-Policy "no-referrer-when-downgrade" always;
    add_header Content-Security-Policy "default-src 'self' http: https: data: blob: 'unsafe-inline'" always;

    location / {
        proxy_pass http://127.0.0.1:11451;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # WebSocket 支持
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_read_timeout 86400;
    }
}
```

## 编译项目

详见项目根目录的 [`BUILD_PROJECT_zh.md`](BUILD_PROJECT_zh.md) 文件，包含完整的编译和发布指南。

### 快速编译

```bash
# 克隆项目
git clone https://github.com/HwlloChen/etaMonitor.git
cd etaMonitor

# 一键构建
make

# 运行
make run
```

## 使用说明

### 添加服务器

1. 登录管理面板
2. 点击"添加服务器"
3. 输入服务器信息：
   - 服务器名称
   - 地址
   - 端口
   - 版本类型（Java/基岩版）

### 监控功能

- **实时状态**: 服务器在线状态、玩家数量、延迟
- **历史数据**: 在线玩家数量趋势图表，支持准确的时间轴显示
- **玩家活动**: 最近 15 分钟内玩家加入/退出记录
- **服务器详情**: 版本信息、MOTD、Favicon 等
- **数据管理**: 管理员面板支持数据库备份、恢复和优化整理功能

## 故障排查

### 日志查看

```bash
# 查看系统日志（如使用 systemd）
sudo journalctl -u etamonitor -f

# 查看应用输出
./etamonitor 2>&1 | tee etamonitor.log
```

## 贡献

欢迎贡献代码、报告问题或提出建议！

你可以通过Star⭐来支持项目

## 许可证

本项目采用 Mozilla Public License Version 2.0 许可证，详见 [LICENSE](LICENSE) 文件。

## 致谢

- [CraftHead](https://crafthead.net/): 提供 Minecraft 头像 API
- [MDUI](https://mdui.org/): Material Design 组件库
- [Gin](https://gin-gonic.com/): Go Web 框架
- [Vue.js](https://vuejs.org/): 前端框架
