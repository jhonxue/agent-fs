<div align="center">

# agent-fs

### 为 AI Agent 打造的跨平台文件操作 CLI 工具

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go)](https://golang.org/)
[![CLI](https://img.shields.io/badge/CLI-Cobra-29BEB0?logo=terminal)](https://github.com/spf13/cobra)

[English](#english) | [中文](#中文)

</div>

---

<a name="中文"></a>

## 中文

### 目录

- [简介](#简介)
- [快速开始](#快速开始)
- [安装](#安装)
- [使用指南](#使用指南)
  - [本地文件操作](#本地文件操作)
  - [云存储操作](#云存储操作)
  - [配置管理](#配置管理)
- [云存储配置](#云存储配置)
- [命令参考](#命令参考)
- [输出格式](#输出格式)
- [安全特性](#安全特性)
- [常见问题](#常见问题)

---

### 简介

`agent-fs` (命令: `afs`) 是一个专为 **AI Agent** 设计的轻量级命令行工具，提供本地文件操作和云存储同步功能。抹平物理机与云存储的鸿沟，输出标准化的 JSON 格式，让 AI Agent 在受限上下文中安全、高效地完成文件操作。

**核心功能：**

| 功能 | 说明 |
|------|------|
| `fs` | 统一文件系统操作（read/ls/cp/info），支持本地和云存储 |
| `config` | 配置管理（set/get） |

**为什么选择 `afs`？**

- **AI 友好** - 标准 JSON 输出，易于解析
- **Token 感知** - 支持切片读取大文件，避免超出上下文
- **安全沙箱** - 支持路径限制，防止误操作
- **多云支持** - 兼容 S3、R2、MinIO 等主流云存储

---

### 快速开始

#### 30 秒上手

```bash
# 1. 下载并安装
curl -fsSL https://raw.githubusercontent.com/geekjourneyx/agent-fs/main/scripts/install.sh | bash

# 2. 验证安装
afs version

# 3. 查看帮助
afs --help
afs fs --help
```

#### 第一个命令

```bash
# 获取文件信息（支持本地、CephFS 和云存储 URI）
afs fs info s3://bucket/file.json
afs fs info /path/to/file.json

# 读取文件（支持 --head, --tail, --bytes 切片读取）
afs fs read /var/log/app.log --tail 50
# 读取 CephFS 文件示例
afs fs read cephfs:///path/to/file.txt

# 列出目录/前缀下的文件
afs fs ls s3://bucket/prefix/

# 跨存储类型复制文件
afs fs cp s3://bucket/data.json file:///tmp/data.json
```

---

### 安装

#### 方式一：一键安装（推荐）

**Linux / macOS**

```bash
curl -fsSL https://raw.githubusercontent.com/geekjourneyx/agent-fs/main/scripts/install.sh | bash
```

安装完成后，`afs` 命令将可用。验证安装：

```bash
afs version
# 输出: afs version x.x.x
```

#### 方式二：手动下载

从 [Releases](https://github.com/geekjourneyx/agent-fs/releases) 页面下载对应平台的二进制文件：

```bash
# Linux amd64
wget https://github.com/geekjourneyx/agent-fs/releases/latest/download/afs-linux-amd64 -O afs
chmod +x afs
sudo mv afs /usr/local/bin/

# macOS ARM64
wget https://github.com/geekjourneyx/agent-fs/releases/latest/download/afs-darwin-arm64 -O afs
chmod +x afs
sudo mv afs /usr/local/bin/

# Windows
# 下载 afs-windows-amd64.exe 并重命名为 afs.exe，添加到 PATH
```

#### 方式三：从源码构建

```bash
git clone https://github.com/geekjourneyx/agent-fs.git
cd agent-fs
# 标准构建
go build -o afs .

# 可选：启用 CephFS 支持（需安装 libcephfs 且启用 cgo）
# Debian/Ubuntu: sudo apt-get install -y ceph libcephfs2 libcephfs-dev
# CentOS/RHEL:   sudo yum install -y ceph ceph-devel
# macOS (brew):  brew install ceph
# 然后使用 cephfs,cgo 构建标签：
go build -tags "cephfs cgo" -o afs .
```

#### 方式四：作为 Skill 集成

**OpenClaw（本地 AI 助理）**

```bash
mkdir -p ~/.openclaw/workspace/skills/afs
curl -o ~/.openclaw/workspace/skills/afs/SKILL.md \
  https://raw.githubusercontent.com/geekjourneyx/agent-fs/main/skills/afs/SKILL.md
```

**Claude Code（AI 辅助开发）**

```bash
npx skills add https://github.com/geekjourneyx/agent-fs --skill afs
```

---

### 使用指南

#### 本地文件操作

##### 获取文件/目录信息

```bash
# 获取文件/目录信息
afs fs info /path/to/file.json

# 也支持云存储 URI
afs fs info s3://bucket/file.json

# 输出示例：
# {
#   "success": true,
#   "action": "local_info",
#   "data": {
#     "name": "file.json",
#     "path": "/path/to/file.json",
#     "type": "file",
#     "size_bytes": 1024,
#     "mode": "0644",
#     "modified_time": "2026-03-01T12:00:00Z",
#     "is_dir": false
#   }
# }
```

##### 读取文件内容

`afs` 支持 **Token 感知的切片读取**，避免读取大文件导致上下文溢出：

```bash
# 读取末尾 N 行（查看日志常用）
afs fs read /var/log/app.log --tail 50

# 读取开头 N 行
afs fs read /var/log/app.log --head 20

# 读取前 N 字节
afs fs read /data.bin --bytes 1024

# 读取完整文件（默认上限 10MB）
afs fs read /path/to/small.txt
```

##### 打包与解压

```bash
# 创建 zip 归档
afs fs zip /data/logs --out backup.zip

# 解压到指定目录
afs fs unzip backup.zip --dest /restore

# 输出示例：
# {
#   "success": true,
#   "action": "local_unzip",
#   "data": {
#     "zip_path": "backup.zip",
#     "destination": "/restore",
#     "extracted_files": 10,
#     "size_bytes": 20480
#   }
# }
```

---

#### 云存储操作

> **注意**：使用云存储功能前，需要先完成 [云存储配置](#云存储配置)。

##### 上传文件

```bash
# 上传单个文件
afs fs cp local.txt remote/path/

# 上传目录（自动压缩）
afs fs cp /logs remote/logs/ --zip

# 指定 provider
afs fs cp file.txt remote/ --provider r2
```

##### 下载文件

```bash
# 下载文件
afs fs cp remote/file.txt ./

# 下载并自动解压
afs fs cp remote/archive.zip ./ --unzip

# 覆盖本地文件
afs fs cp remote/file.txt ./ --overwrite
```

##### 列出对象

```bash
# 列出指定前缀的对象
afs fs ls remote/path/

# 限制返回数量
afs fs ls remote/path/ --limit 50

# 输出示例：
# {
#   "success": true,
#   "action": "cloud_list",
#   "data": {
#     "provider": "s3",
#     "objects": [...],
#     "count": 10,
#     "total_bytes": 102400
#   }
# }
```

##### 生成访问 URL

`afs fs` 支持两种 URL 类型：

**Presigned URL（默认，推荐用于私密文件）**

```bash
# 生成带签名的 URL（默认 15 分钟有效期）
afs fs url remote/file.txt

# 自定义过期时间（秒）
afs fs url remote/file.txt --expires 3600
```

- **特点**：带签名认证，有过期时间
- **安全性**：✅ 安全，链接过期后无法访问
- **适用场景**：临时分享私密文件、日志、配置等

**Public URL（需要配置 Public Access）**

```bash
# 生成公共访问 URL
afs fs url remote/public.jpg --public
```

- **特点**：无需认证，永久可访问（直到文件删除）
- **安全性**：⚠️ 任何人都可以访问
- **适用场景**：公开的网站资源、公共下载文件

> **⚠️ 安全提示**：使用 `--public` 时会显示安全警告。只有确实需要公开访问的文件才使用此选项。

> **注**：CephFS 等本地/分布式文件系统不支持 URL 生成，仅对象存储（S3 兼容）支持。

---

#### 配置管理

`afs` 支持三种配置方式，优先级从高到低：

1. **CLI 参数**（`--provider r2`）
2. **环境变量**（`export AFS_S3_BUCKET=xxx`）
3. **配置文件**（`.agent-fs.yaml` 或 `~/.agent-fs.yaml`）

##### 设置配置

```bash
# 设置到当前目录配置文件
afs config set s3.endpoint https://xxx.r2.cloudflarestorage.com
afs config set s3.bucket my-bucket
afs config set s3.access_key_id YOUR_ACCESS_KEY
afs config set s3.secret_access_key YOUR_SECRET_KEY

# 设置到全局配置文件（~/.agent-fs.yaml）
afs config set s3.endpoint https://xxx.r2.cloudflarestorage.com --global
```

##### 查看配置

```bash
# 查看单个配置
afs config get s3.endpoint

# 输出示例：
# {
#   "success": true,
#   "action": "config_get",
#   "data": {
#     "config_file": "/root/.agent-fs.yaml",
#     "key": "s3.endpoint",
#     "value": "https://xxx.r2.cloudflarestorage.com"
#   }
# }
```

---

### 云存储配置

#### 支持的云存储提供商

| 提供商 | 标识 | Endpoint 示例 | 说明 |
|--------|------|---------------|------|
| AWS S3 | `s3` | `https://s3.amazonaws.com` | 亚马逊 S3 |
| Cloudflare R2 | `r2` | 自动生成（配置 account_id） | Cloudflare R2 |
| MinIO | `minio` | `http://localhost:9000` | 自建对象存储 |
| 阿里云 OSS | `oss` | `https://oss-cn-hangzhou.aliyuncs.com` | 阿里云对象存储 |
| 腾讯云 COS | `cos` | `https://cos.ap-guangzhou.myqcloud.com` | 腾讯云对象存储 |
| Backblaze B2 | `b2` | `https://s3.us-west-004.backblazeb2.com` | B2 S3 兼容模式 |
| Wasabi | `wasabi` | `https://s3.wasabisys.com` | Wasabi 热云存储 |
| CephFS | `cephfs` | `N/A` | 分布式文件系统路径 |

查看完整列表：

```bash
afs fs providers
# 输出应包含 cephfs 提供商（当已编译并注册 CephFS provider 时）
```

#### Cloudflare R2 配置示例

**步骤 1：创建 R2 存储桶**

1. 登录 [Cloudflare Dashboard](https://dash.cloudflare.com/)
2. 进入 **R2** → 点击 **Create bucket**
3. 输入存储桶名称（全球唯一），选择区域

**步骤 2：获取 API Token**

1. 在 R2 页面点击 **Manage R2 API Tokens**
2. 点击 **Create API Token** → 选择 **Admin API Token**
3. 复制 **Access Key ID** 和 **Secret Access Key**（只显示一次）

**步骤 3：获取 Account ID**

在 Dashboard 右侧可以看到 **Account ID**，或从 URL 获取：
```
https://dash.cloudflare.com/<Account ID>/r2
```

**步骤 4：配置 afs**

```bash
# 使用 config 命令
afs config set r2.account_id "你的AccountID"
afs config set r2.bucket "你的存储桶名称"
afs config set r2.access_key_id "你的AccessKeyID"
afs config set r2.secret_access_key "你的SecretKey"

# 或使用环境变量
export AFS_S3_ENDPOINT=https://YOUR_ACCOUNT_ID.r2.cloudflarestorage.com
export AFS_S3_BUCKET=your-bucket
export AFS_S3_ACCESS_KEY_ID=your-access-key
export AFS_S3_SECRET_ACCESS_KEY=your-secret-key
```

**步骤 5：（可选）配置 Public Access**

如果需要生成公共访问 URL：

1. 在 R2 存储桶设置中启用 **Public access**
2. Cloudflare 会分配公共域名，格式如：`https://pub-xxx.r2.dev`
3. 配置 CDN 域名：
   ```bash
   afs config set r2.cdn_host https://pub-xxx.r2.dev
   ```

#### 通用 S3 兼容存储配置

```bash
# 配置文件方式
afs config set s3.endpoint https://your-endpoint.com
afs config set s3.bucket your-bucket
afs config set s3.access_key_id your-access-key
afs config set s3.secret_access_key your-secret-key
afs config set s3.region us-east-1

# 环境变量方式
export AFS_S3_ENDPOINT=https://your-endpoint.com
export AFS_S3_BUCKET=your-bucket
export AFS_S3_ACCESS_KEY_ID=your-access-key
export AFS_S3_SECRET_ACCESS_KEY=your-secret-key
```

#### CephFS 构建与运行（可选）

CephFS 为可选特性，启用条件与依赖如下：

- 构建标签：使用 `-tags "cephfs cgo"`
- 系统依赖：`libcephfs`（及开发头文件）
  - Debian/Ubuntu: `sudo apt-get install -y ceph libcephfs2 libcephfs-dev`
  - CentOS/RHEL:   `sudo yum install -y ceph ceph-devel`
  - macOS:         `brew install ceph`
- 运行依赖：
  - 配置文件：`/etc/ceph/ceph.conf`（或设置环境变量 `CEPHFS_CONF_PATH`）
  - 凭证：`CEPHFS_KEYRING_PATH` 或 `CEPHFS_SECRET`
  - 替代配置：如未提供 conf，可通过 `CEPHFS_MON_HOSTS` 指定 MON 列表（逗号分隔），并设置 `CEPHFS_AUTH_ID`（默认 `admin`）
- 注意事项：
  - CephFS 不支持 URL 生成（`afs fs url`），该能力仅适用于对象存储（S3 兼容）
  - tail 大文件时建议在当前版本优先使用本地文件系统路径

---

### 命令参考

#### 本地命令

| 命令 | 说明 | 示例 |
|------|------|------|
| `afs fs info <path>` | 获取文件/目录元数据 | `afs fs info file.txt` |
| `afs fs read <path> --tail N` | 读取末尾 N 行 | `afs fs read log.txt --tail 50` |
| `afs fs read <path> --head N` | 读取开头 N 行 | `afs fs read log.txt --head 20` |
| `afs fs read <path> --bytes N` | 读取前 N 字节 | `afs fs read data.bin --bytes 1024` |
| `afs fs zip <source> --out <file>` | 创建 zip 归档 | `afs fs zip /data --out backup.zip` |
| `afs fs unzip <zip> --dest <dir>` | 解压归档 | `afs fs unzip backup.zip --dest /restore` |

#### 云存储命令

| 命令 | 说明 | 示例 |
|------|------|------|
| `afs fs cp <local> <remote>` | 上传文件 | `afs fs cp file.txt remote/` |
| `afs fs cp <remote> <local>` | 下载文件 | `afs fs cp remote/file.txt ./` |
| `afs fs ls [prefix]` | 列出对象 | `afs fs ls remote/path/ --limit 50` |
| `afs fs url <remote_key>` | 生成 Presigned URL | `afs fs url remote/file.txt --expires 3600` |
| `afs fs url <remote_key> --public` | 生成公共 URL | `afs fs url remote/image.jpg --public` |
| `afs fs providers` | 列出支持的提供商 | `afs fs providers` |

#### 配置命令

| 命令 | 说明 | 示例 |
|------|------|------|
| `afs config set <key> <value>` | 设置配置 | `afs config set s3.bucket my-bucket` |
| `afs config get <key>` | 获取配置 | `afs config get s3.bucket` |
| `afs config set <key> <value> --global` | 设置全局配置 | `afs config set s3.bucket my-bucket --global` |

#### 通用选项

| 选项 | 说明 | 适用于 |
|------|------|--------|
| `--provider <name>` | 指定云存储提供商 | 所有 fs 命令 |
| `--zip` | 上传前自动压缩 | `fs cp` |
| `--unzip` | 下载后自动解压 | `fs cp` |
| `--overwrite` | 覆盖本地文件 | `fs cp` |
| `--expires <seconds>` | URL 过期时间 | `fs url` |
| `--public` | 生成公共 URL | `fs url` |
| `--details` | 包含详细信息 | `fs info` |

---

### 输出格式

所有 `afs` 命令返回标准化的 JSON 格式，便于 AI Agent 解析。

#### 成功响应

```json
{
  "success": true,
  "action": "local_read",
  "data": {
    "path": "/path/to/file.log",
    "content": "...",
    "line_count": 50,
    "byte_count": 2048,
    "truncated": false,
    "slice_type": "tail"
  },
  "error": null
}
```

#### 失败响应

```json
{
  "success": false,
  "action": "cloud_upload",
  "data": null,
  "error": {
    "code": "ERR_UPLOAD",
    "message": "upload failed: connection timeout"
  }
}
```

#### 错误码说明

| 错误码 | 说明 |
|--------|------|
| `ERR_INVALID_ARGUMENT` | 参数错误 |
| `ERR_PATH_TRAVERSAL` | 路径穿越检测 |
| `ERR_NOT_FOUND` | 文件或目录不存在 |
| `ERR_CONFLICT` | 目标已存在 |
| `ERR_PROVIDER` | 云存储提供商错误 |
| `ERR_UPLOAD` | 上传失败 |
| `ERR_DOWNLOAD` | 下载失败 |
| `ERR_ARCHIVE` | 归档操作失败 |
| `ERR_CONFIG` | 配置错误 |
| `ERR_INTERNAL` | 内部错误 |

---

### 安全特性

#### 沙箱模式

通过设置 `AFS_WORKSPACE` 环境变量，可以限制 `afs` 的操作范围，防止误删除系统文件：

```bash
# 设置工作区
export AFS_WORKSPACE=/safe/workspace

# 以下操作会被拒绝
afs fs info /etc/passwd
# ERROR: access denied: path is outside AFS_WORKSPACE
```

#### 路径穿越防护

自动检测并阻止 `../` 路径穿越攻击，确保所有操作在允许的范围内。

#### 敏感信息保护

- 密钥信息不会在输出中暴露
- 建议使用环境变量或配置文件存储密钥
- 避免在命令行中直接传递密钥

---

### 常见问题

#### Q: 如何查看 afs 版本？

```bash
afs version
```

#### Q: 如何获取帮助信息？

```bash
afs --help           # 查看主帮助
afs fs --help        # 查看 fs 命令帮助（支持本地和云存储）
```

#### Q: 配置文件在哪里？

配置文件优先级（高到低）：
1. 当前目录：`.agent-fs.yaml`
2. 用户目录：`~/.agent-fs.yaml`

#### Q: 支持哪些云存储？

`afs` 支持所有 S3 兼容的对象存储，包括 AWS S3、Cloudflare R2、MinIO、阿里云 OSS、腾讯云 COS 等。运行 `afs fs providers` 查看完整列表。

#### Q: 如何处理大文件？

使用 `--tail`、`--head` 或 `--bytes` 参数切片读取：

```bash
# 读取日志末尾 100 行
afs fs read large.log --tail 100

# 读取前 1KB
afs fs read large.bin --bytes 1024
```

#### Q: Public URL 和 Presigned URL 有什么区别？

| 特性 | Presigned URL | Public URL |
|------|---------------|------------|
| 认证 | 需要签名 | 不需要 |
| 有效期 | 可设置过期时间 | 永久有效 |
| 安全性 | 更安全 | 任何人可访问 |
| 配置 | 无需额外配置 | 需要开启 Public Access |

#### Q: 如何调试问题？

```bash
# 1. 检查配置
afs config get s3.endpoint

# 2. 测试连接
afs fs ls / --limit 1

# 3. 查看详细错误信息（stderr）
afs fs cp file.txt remote/ 2>&1
```

---

<a name="english"></a>

## English

### Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Installation](#installation)
- [Usage Guide](#usage-guide)
  - [Local File Operations](#local-file-operations)
  - [Cloud Storage Operations](#cloud-storage-operations)
  - [Configuration Management](#configuration-management)
- [Cloud Storage Setup](#cloud-storage-setup)
- [Command Reference](#command-reference)
- [Output Format](#output-format)
- [Security Features](#security-features)
- [FAQ](#faq)

---

### Overview

`agent-fs` (CLI: `afs`) is a lightweight command-line tool designed for **AI Agents**, providing local file operations and cloud storage synchronization. It bridges local and cloud storage with standardized JSON output, enabling AI Agents to safely and efficiently handle file operations within constrained contexts.

**Core Features:**

| Feature | Description |
|---------|-------------|
| `local` | Local file operations (zip/unzip/info/read) |
| `cloud` | Cloud storage sync (upload/download/list/url) |
| `config` | Configuration management (set/get) |

**Why `afs`?**

- **AI-Friendly** - Standard JSON output, easy to parse
- **Token-Aware** - Supports chunked reading for large files
- **Secure Sandbox** - Path restrictions to prevent accidental operations
- **Multi-Cloud** - Compatible with S3, R2, MinIO, and more

---

### Quick Start

#### 30-Second Setup

```bash
# 1. Install
curl -fsSL https://raw.githubusercontent.com/geekjourneyx/agent-fs/main/scripts/install.sh | bash

# 2. Verify
afs version

# 3. Get help
afs --help
afs fs --help
```

#### Your First Command

```bash
# Get file info
afs fs info /path/to/file.json

# Read last 50 lines
afs fs read /var/log/app.log --tail 50

# Upload to cloud (requires configuration)
afs fs cp local.txt remote/path/
```

---

### Installation

#### Method 1: One-Line Install (Recommended)

**Linux / macOS**

```bash
curl -fsSL https://raw.githubusercontent.com/geekjourneyx/agent-fs/main/scripts/install.sh | bash
```

Verify installation:

```bash
afs version
# Output: afs version x.x.x
```

#### Method 2: Manual Download

Download from [Releases](https://github.com/geekjourneyx/agent-fs/releases):

```bash
# Linux amd64
wget https://github.com/geekjourneyx/agent-fs/releases/latest/download/afs-linux-amd64 -O afs
chmod +x afs
sudo mv afs /usr/local/bin/

# macOS ARM64
wget https://github.com/geekjourneyx/agent-fs/releases/latest/download/afs-darwin-arm64 -O afs
chmod +x afs
sudo mv afs /usr/local/bin/
```

#### Method 3: Build from Source

```bash
git clone https://github.com/geekjourneyx/agent-fs.git
cd agent-fs
go build -o afs .
```

---

### Usage Guide

#### Local File Operations

##### Get File/Directory Info

```bash
# Get file info
afs fs info /path/to/file.json

# Get directory info (includes file count and total size)
afs fs info /path/to/dir --details
```

##### Read File Content

`afs` supports **token-aware chunked reading** to avoid context overflow:

```bash
# Read last N lines (useful for logs)
afs fs read /var/log/app.log --tail 50

# Read first N lines
afs fs read /var/log/app.log --head 20

# Read first N bytes
afs fs read /data.bin --bytes 1024

# Read complete file (default limit: 10MB)
afs fs read /path/to/small.txt
```

##### Archive Operations

```bash
# Create zip archive
afs fs zip /data/logs --out backup.zip

# Extract to directory
afs fs unzip backup.zip --dest /restore
```

---

#### Cloud Storage Operations

> **Note**: Configure cloud storage before using these commands. See [Cloud Storage Setup](#cloud-storage-setup).

##### Upload Files

```bash
# Upload single file
afs fs cp local.txt remote/path/

# Upload directory (auto-compress)
afs fs cp /logs remote/logs/ --zip

# Specify provider
afs fs cp file.txt remote/ --provider r2
```

##### Download Files

```bash
# Download file
afs fs cp remote/file.txt ./

# Download and auto-extract
afs fs cp remote/archive.zip ./ --unzip

# Overwrite local file
afs fs cp remote/file.txt ./ --overwrite
```

##### List Objects

```bash
# List objects with prefix
afs fs ls remote/path/

# Limit results
afs fs ls remote/path/ --limit 50
```

##### Generate Access URL

`afs fs` supports two URL types:

**Presigned URL (Default, for Private Files)**

```bash
# Generate signed URL (default 15 min expiration)
afs fs url remote/file.txt

# Custom expiration (seconds)
afs fs url remote/file.txt --expires 3600
```

- **Features**: Authenticated with signature, expires after set time
- **Security**: ✅ Secure, link becomes invalid after expiration
- **Use Case**: Temporary sharing of private files, logs, configs

**Public URL (Requires Public Access Configuration)**

```bash
# Generate public access URL
afs fs url remote/public.jpg --public
```

- **Features**: No authentication, permanently accessible (until deleted)
- **Security**: ⚠️ Accessible by anyone with the link
- **Use Case**: Public website assets, public download files

> **⚠️ Security Note**: A security warning is displayed when using `--public`. Only use this option for files that need to be publicly accessible.

---

#### Configuration Management

`afs` supports three configuration methods (priority: high to low):

1. **CLI flags** (`--provider r2`)
2. **Environment variables** (`export AFS_S3_BUCKET=xxx`)
3. **Config files** (`.agent-fs.yaml` or `~/.agent-fs.yaml`)

##### Set Configuration

```bash
# Set to current directory config
afs config set s3.endpoint https://xxx.r2.cloudflarestorage.com
afs config set s3.bucket my-bucket
afs config set s3.access_key_id YOUR_ACCESS_KEY
afs config set s3.secret_access_key YOUR_SECRET_KEY

# Set to global config (~/.agent-fs.yaml)
afs config set s3.endpoint https://xxx.r2.cloudflarestorage.com --global
```

##### View Configuration

```bash
# View single config
afs config get s3.endpoint
```

---

### Cloud Storage Setup

#### Supported Providers

| Provider | ID | Endpoint Example | Description |
|----------|----|----|----|
| AWS S3 | `s3` | `https://s3.amazonaws.com` | Amazon S3 |
| Cloudflare R2 | `r2` | Auto-generated (set account_id) | Cloudflare R2 |
| MinIO | `minio` | `http://localhost:9000` | Self-hosted object storage |
| Alibaba OSS | `oss` | `https://oss-cn-hangzhou.aliyuncs.com` | Alibaba Cloud OSS |
| Tencent COS | `cos` | `https://cos.ap-guangzhou.myqcloud.com` | Tencent Cloud COS |
| Backblaze B2 | `b2` | `https://s3.us-west-004.backblazeb2.com` | B2 S3 compatible |
| Wasabi | `wasabi` | `https://s3.wasabisys.com` | Wasabi hot cloud storage |
| CephFS | `cephfs` | N/A | Distributed filesystem (requires libcephfs) |

View full list:

```bash
afs fs providers
```

#### Cloudflare R2 Setup Example

**Step 1: Create R2 Bucket**

1. Login to [Cloudflare Dashboard](https://dash.cloudflare.com/)
2. Go to **R2** → Click **Create bucket**
3. Enter bucket name (globally unique), select region

**Step 2: Get API Token**

1. In R2 page, click **Manage R2 API Tokens**
2. Click **Create API Token** → Select **Admin API Token**
3. Copy **Access Key ID** and **Secret Access Key** (shown only once)

**Step 3: Get Account ID**

Found on the right side of Dashboard, or from URL:
```
https://dash.cloudflare.com/<Account ID>/r2
```

**Step 4: Configure afs**

```bash
# Using config command
afs config set r2.account_id "YOUR_ACCOUNT_ID"
afs config set r2.bucket "YOUR_BUCKET_NAME"
afs config set r2.access_key_id "YOUR_ACCESS_KEY_ID"
afs config set r2.secret_access_key "YOUR_SECRET_KEY"

# Or using environment variables
export AFS_S3_ENDPOINT=https://YOUR_ACCOUNT_ID.r2.cloudflarestorage.com
export AFS_S3_BUCKET=your-bucket
export AFS_S3_ACCESS_KEY_ID=your-access-key
export AFS_S3_SECRET_ACCESS_KEY=your-secret-key
```

**Step 5: (Optional) Configure Public Access**

For public access URLs:

1. Enable **Public access** in R2 bucket settings
2. Cloudflare assigns a public domain like: `https://pub-xxx.r2.dev`
3. Configure CDN host:
   ```bash
   afs config set r2.cdn_host https://pub-xxx.r2.dev
   ```

---

### Command Reference

#### Local Commands

| Command | Description | Example |
|---------|-------------|---------|
| `afs fs info <path>` | Get file/directory metadata | `afs fs info file.txt` |
| `afs fs read <path> --tail N` | Read last N lines | `afs fs read log.txt --tail 50` |
| `afs fs read <path> --head N` | Read first N lines | `afs fs read log.txt --head 20` |
| `afs fs read <path> --bytes N` | Read first N bytes | `afs fs read data.bin --bytes 1024` |
| `afs fs zip <source> --out <file>` | Create zip archive | `afs fs zip /data --out backup.zip` |
| `afs fs unzip <zip> --dest <dir>` | Extract archive | `afs fs unzip backup.zip --dest /restore` |

#### Cloud Commands

| Command | Description | Example |
|---------|-------------|---------|
| `afs fs cp <local> <remote>` | Upload file | `afs fs cp file.txt remote/` |
| `afs fs cp <remote> <local>` | Download file | `afs fs cp remote/file.txt ./` |
| `afs fs ls [prefix]` | List objects | `afs fs ls remote/path/ --limit 50` |
| `afs fs url <remote_key>` | Generate Presigned URL | `afs fs url remote/file.txt --expires 3600` |
| `afs fs url <remote_key> --public` | Generate public URL | `afs fs url remote/image.jpg --public` |
| `afs fs providers` | List supported providers | `afs fs providers` |

#### Configuration Commands

| Command | Description | Example |
|---------|-------------|---------|
| `afs config set <key> <value>` | Set configuration | `afs config set s3.bucket my-bucket` |
| `afs config get <key>` | Get configuration | `afs config get s3.bucket` |
| `afs config set <key> <value> --global` | Set global config | `afs config set s3.bucket my-bucket --global` |

#### Global Options

| Option | Description | Applies to |
|--------|-------------|------------|
| `--provider <name>` | Specify cloud provider | All fs commands |
| `--zip` | Auto-compress before upload | `fs cp` |
| `--unzip` | Auto-extract after download | `fs cp` |
| `--overwrite` | Overwrite local file | `fs cp` |
| `--expires <seconds>` | URL expiration time | `fs url` |
| `--public` | Generate public URL | `fs url` |
| `--details` | Include detailed info | `fs info` |

---

### Output Format

All `afs` commands return standardized JSON format for easy AI Agent parsing.

#### Success Response

```json
{
  "success": true,
  "action": "local_read",
  "data": {
    "path": "/path/to/file.log",
    "content": "...",
    "line_count": 50,
    "byte_count": 2048,
    "truncated": false,
    "slice_type": "tail"
  },
  "error": null
}
```

#### Error Response

```json
{
  "success": false,
  "action": "cloud_upload",
  "data": null,
  "error": {
    "code": "ERR_UPLOAD",
    "message": "upload failed: connection timeout"
  }
}
```

---

### Security Features

#### Sandbox Mode

Set `AFS_WORKSPACE` environment variable to restrict operations:

```bash
export AFS_WORKSPACE=/safe/workspace

# Operations outside this directory will be blocked
afs fs info /etc/passwd
# ERROR: access denied: path is outside AFS_WORKSPACE
```

#### Path Traversal Protection

Automatically detects and blocks `../` path traversal attacks.

#### Sensitive Data Protection

- Credentials are not exposed in output
- Use environment variables or config files for credentials
- Avoid passing secrets via command line

---

### FAQ

#### Q: How to check afs version?

```bash
afs version
```

#### Q: How to get help?

```bash
afs --help           # Main help
afs fs --help        # fs command help (supports local and cloud)
```

#### Q: Where is the config file?

Config file priority (high to low):
1. Current directory: `.agent-fs.yaml`
2. User directory: `~/.agent-fs.yaml`

#### Q: Which cloud providers are supported?

All S3-compatible object storage, including AWS S3, Cloudflare R2, MinIO, Alibaba OSS, Tencent COS, etc. Run `afs fs providers` for full list.

#### Q: How to handle large files?

Use `--tail`, `--head`, or `--bytes` for chunked reading:

```bash
# Read last 100 lines
afs fs read large.log --tail 100

# Read first 1KB
afs fs read large.bin --bytes 1024
```

#### Q: What's the difference between Public URL and Presigned URL?

| Feature | Presigned URL | Public URL |
|---------|---------------|------------|
| Auth | Requires signature | No auth |
| Expiration | Can set expiration | Permanent |
| Security | More secure | Anyone can access |
| Config | No extra config | Requires Public Access enabled |

---

## License

[MIT License](LICENSE)

---

## Author
