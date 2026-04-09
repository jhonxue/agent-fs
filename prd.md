# 开发文档：`agent-fs` (命令: `afs`) - 面向 AI Agent 的跨平台文件操作 CLI

## 1. 项目概述 (Project Overview)

**项目名称**: `agent-fs`
**CLI 唤醒词**: `afs`
**核心定位**: 一个专为 AI Agent（智能体）设计的、极度轻量、跨平台、无状态的底层文件操作工具箱。
**解决的痛点**: 抹平物理机与云存储的鸿沟，提供结构化的输入输出，让 AI Agent 在受限的上下文中安全、高效地完成“本地文件打包 -> 云端上传 -> 下载 -> 解压读取”的完整生命周期闭环。暂时不包含复杂的文档格式解析。

## 2. 技术选型 (Tech Stack)

* **开发语言**: Go 1.24+ (追求极致的冷启动速度和极小的二进制体积)。
* **核心 CLI 框架**: `spf13/cobra` (用于构建子命令) + `spf13/viper` (用于多级配置读取：CLI 参数 > 环境变量 > YAML 配置文件)。
* **云存储协议抽象**: 采用标准 S3 协议 SDK (`github.com/aws/aws-sdk-go-v2/service/s3`)。通过配置自定义 Endpoint，实现对 AWS S3、Cloudflare R2、阿里云 OSS 等主流云存储的一揽子兼容。
* **CI/CD**: GitHub Actions (实现多操作系统、多架构的自动化跨平台交叉编译和 Release 发布)。

## 3. 核心设计原则 (Agent-First Design Principles)

在开发过程中，必须严格遵守以下针对大模型友好的原则：

1. **机器可读输出 (Machine-Readable Output)**:
* 默认情况下（或通过 `--json` flag），所有命令的 `stdout` **必须**输出标准化的 JSON 格式，绝不能混入进度条、彩色控制字符或人类问候语。
* 所有的错误信息必须输出到 `stderr`，并在 JSON 结构中包含明确的 `error_code` 和 `message`。


2. **安全沙箱 (Path Jail / Sandbox)**:
* 支持通过环境变量 `AFS_WORKSPACE` 锁定工作目录。所有的读写操作必须经过 `filepath.Clean` 校验，严禁通过 `../../` 路径穿越跳出工作区，防止 Agent 误删系统级核心文件。


3. **Token 感知 (Token-Aware)**:
* 对于文件读取，严禁一次性将 GB 级日志全部输出。必须提供 `--tail`, `--head`, 或 `--bytes` 等切片读取能力。



## 4. 目录结构设计 (Directory Layout)

采用标准的 Go 项目布局：

```text
agent-fs/
├── cmd/                # Cobra 命令注册入口
│   ├── root.go         # 根命令
│   ├── fs.go           # 统一文件操作 (info/read/cp/ls/url等)
│   └── config.go       # 凭证与配置管理
├── pkg/                # 核心逻辑封装
│   ├── sandbox/        # 路径安全校验隔离层
│   ├── archive/        # Zip 打包/解压引擎
│   ├── s3client/       # 云存储 SDK 封装层
│   └── output/         # JSON 结构化输出化格式器
├── main.go             # 程序主入口
├── go.mod
└── .github/workflows/  # CI/CD 构建脚本

```

## 5. 核心命令与功能规范 (Commands & Features)

### 5.1 本地文件操作 (`afs fs`)

| 命令 | 参数示例 | 功能描述 | Agent 核心价值 |
| --- | --- | --- | --- |
| `info` | `afs fs info ./data` | 获取文件/目录大小、权限、修改时间。 | 操作前预判，防止处理超大文件超时。 |
| `zip` | `afs fs zip ./logs --out archive.zip` | 将文件或目录打包为 Zip 格式。 | 减少多文件上传的 API 调用次数。 |
| `unzip` | `afs fs unzip archive.zip --dest ./tmp` | 解压文件到指定安全目录。 | 提取云端下载的数据包。 |
| `read` | `afs fs read error.log --tail 100` | 读取文件末尾/开头的指定行数或字节。 | 节省 Context 窗口，精准获取报错日志。 |

### 5.2 云端存储交互 (`afs fs`)

*所有云端命令默认依赖配置好的 S3 凭证 (Endpoint, AccessKey, SecretKey, Bucket)。*

| 命令 | 参数示例 | 功能描述 | Agent 核心价值 |
| --- | --- | --- | --- |
| `cp` | `afs fs cp ./archive.zip remote/path/` | 上传本地文件到云存储，支持大文件并发分片上传。 | 将生成的产物固化到云端。 |
| `cp` | `afs fs cp remote/path/file.csv ./` | 从云存储下载文件到本地沙箱。 | 获取外部数据源进行分析。 |
| `ls` | `afs fs ls remote/path/ --limit 10` | 列出云端目录下的文件列表。 | 探查云存储中的现有资源。 |

### 5.3 配置管理 (`afs config`)

| 命令 | 参数示例 | 功能描述 |
| --- | --- | --- |
| `set` | `afs config set s3.endpoint https://xxx.r2.cloudflarestorage.com` | 将配置写入 `~/.agent-fs.yaml` 或当前工作目录。 |

## 6. 标准化 JSON 输出规范 (I/O Specification)

为了确保 Agent 能够通过 Python `subprocess` 或 Node.js `exec` 稳定解析结果，所有的成功返回必须符合以下数据结构：

```json
{
  "success": true,
  "action": "fs_cp",
  "data": {
    "local_path": "/path/to/local/archive.zip",
    "remote_url": "https://cdn.example.com/remote/path/archive.zip",
    "size_bytes": 1048576,
    "time_taken_ms": 1250
  },
  "error": null
}

```

失败返回示例（输出到 stderr 并在 stdout 返回 JSON）：

```json
{
  "success": false,
  "action": "local_read",
  "data": null,
  "error": {
    "code": "ERR_PATH_TRAVERSAL",
    "message": "Access denied: attempted to read outside the sandbox workspace."
  }
}

```