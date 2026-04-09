# agent-fs - 项目记忆

## 项目概述

**agent-fs** 是一个专为 **AI Agent** 设计的跨平台文件操作 CLI 工具，提供本地文件操作和云存储同步功能。抹平物理机与云存储的鸿沟，输出标准化的 JSON 格式，让 AI Agent 在受限上下文中安全、高效地完成文件操作。

- **项目名**: agent-fs
- **CLI 名**: afs
- **仓库**: https://github.com/geekjourneyx/agent-fs
- **语言**: Go 1.24+
- **框架**: Cobra + Viper
- **核心依赖**: AWS SDK v2 (S3 协议)

## 设计原则

1. **单一职责**: 专注于文件传输和云存储同步，不做内容处理
2. **AI 友好**: 所有命令返回标准 JSON 格式，易于解析
3. **Token 感知**: 支持切片读取，避免大文件超出上下文
4. **安全沙箱**: 支持路径限制，防止误操作

## 项目结构

```
agent-fs/
├── cmd/                    # CLI 命令
│   ├── root.go             # 根命令、版本、配置初始化
│   ├── fs.go                # 统一文件系统操作 (read/ls/info/cp/url)
│   └── config.go           # 配置管理 (set/get)
├── pkg/                    # 核心逻辑
│   ├── apperr/             # 错误处理（错误码、解析）
│   ├── archive/            # Zip 打包/解压
│   ├── cloud/              # 云存储抽象（Provider接口、Dispatcher）
│   ├── local/              # 本地文件操作
│   ├── s3client/           # S3 客户端封装
│   ├── sandbox/            # 路径安全校验
│   └── output/             # JSON 输出格式化
├── main.go                 # 程序入口
├── scripts/                # 安装脚本
│   └── install.sh
├── .github/workflows/
│   ├── ci.yml              # 持续集成
│   └── release.yml         # 发布流程
├── skills/afs/             # Claude Code Skill
│   └── SKILL.md
├── .golangci.yml           # 静态检查配置
├── CLAUDE.md               # 本文件
├── README.md               # 项目说明（双语）
└── prd.md                  # 产品需求文档
```

---

## 提交前工作流程

每次代码提交前，必须按顺序执行以下步骤：

### 1. 编译检查

```bash
cd /mnt/data/hub/agent-fs
go build -o afs .
```

确保编译无错误。

### 2. 运行测试

```bash
go test ./... -v
```

确保所有测试通过。

### 3. 代码格式化

```bash
# 确保安装 goimports
if [ ! -f ~/go/bin/goimports ]; then
    go install golang.org/x/tools/cmd/goimports@latest
fi

go fmt ./...
~/go/bin/goimports -local github.com/geekjourneyx/agent-fs -w .
```

确保代码格式统一。

### 4. 静态检查

```bash
# 确保安装 golangci-lint
if [ ! -f ~/go/bin/golangci-lint ]; then
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
fi

go vet ./...
~/go/bin/golangci-lint run
```

修复所有 vet 和 lint 问题。

### 5. 功能测试

手动测试核心功能：

```bash
# 测试本地操作
./afs fs info main.go
./afs fs read main.go --head 5

# 测试云操作（需要配置）
./afs fs ls s3://bucket/test/
./afs fs providers

# 测试配置
./afs config get s3.bucket
```

### 6. 文档检查

**README.md 检查清单**:

- [ ] 中英文双语完整
- [ ] 示例代码可运行
- [ ] 安装说明清晰
- [ ] 配置项表格完整
- [ ] 项目结构文档与实际目录结构一致
- [ ] 命令参考表格完整
- [ ] 输出格式示例有 JSON 示例

**SKILL.md 检查清单**:

- [ ] 命令列表与实际一致
- [ ] 示例代码可运行
- [ ] 输出格式示例正确
- [ ] 配置说明准确

### 7. 更新 CHANGELOG.md

提交信息格式：

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Type**:
- `feat`: 新功能
- `fix`: 修复 bug
- `docs`: 文档变更
- `style`: 代码格式（不影响功能）
- `refactor`: 重构
- `test`: 测试相关
- `chore`: 构建过程或辅助工具变更
- `revert`: 回滚

**Scope**:
- `local`: 本地操作
- `cloud`: 云存储操作
- `config`: 配置管理
- `output`: 输出格式
- `sandbox`: 安全沙箱
- `archive`: 归档操作
- `s3client`: S3 客户端

**重要**: Commit 中不能有 claude 账号和邮箱信息！

正确示例：
```
feat(local): add sandbox workspace validation

Add path traversal protection to prevent operations
outside AFS_WORKSPACE. This enhances security for AI Agent
file operations.

Closes #123
```

### 8. 更新 CHANGELOG.md

在 `CHANGELOG.md` 顶部添加新版本：

```markdown
## [1.x.x] - 2026-MM-DD

### Added
- 新增功能说明

### Changed
- 变更说明

### Fixed
- 修复说明

### Removed
- 移除说明
```

---

## 发版流程

### 1. 发版前检查

**自动检查脚本（推荐）**:

```bash
# 使用发版前检查脚本（包含所有检查项）
./scripts/release-check.sh 1.0.0
```

**手动检查**:

```bash
# 1. 确保所有检查通过
go test ./...
go vet ./...
~/go/bin/golangci-lint run
go build -o afs .

# 2. 版本号一致性检查
# 确保 main.go 和 cmd/root.go 中的 Version 都是 "dev"
# （发版时通过 ldflags 自动注入版本号）

# 3. 手动测试核心功能
./afs version           # 应显示 "dev"
./afs fs info README.md
./afs fs providers
```

### 2. 版本号说明

**版本号管理方式**:

- **开发时**: `main.go` 和 `cmd/root.go` 中硬编码 `Version = "dev"`
- **发版时**: 通过 GitHub Actions ldflags 动态注入版本号
- **安装脚本**: 使用 `VERSION=${VERSION:-"latest"}` 默认下载最新版本

**需要保持一致的地方**:

| 位置 | 版本号 | 说明 |
|------|--------|------|
| `main.go` | `Version = "dev"` | 默认开发版本 |
| `cmd/root.go` | `Version = "dev"` | 必须与 main.go 一致 |
| GitHub Actions ldflags | `-X main.Version=${VERSION}` | 构建时注入 |
| Git Tag | `v1.0.0` | 与发版版本一致 |

### 3. 更新 CHANGELOG.md

在 `CHANGELOG.md` 顶部添加新版本：

```markdown
## [1.x.x] - 2026-MM-DD

### Added
- 新增功能说明

### Changed
- 变更说明

### Fixed
- 修复说明

### Removed
- 移除说明
```

### 4. 提交变更

```bash
git add .
git commit -m "chore(release): prepare for vx.x.x"
```

### 5. 创建标签

```bash
git tag -a vx.x.x -m "Release vx.x.x"
```

### 6. 用户确认

向用户展示：
- 当前版本号（应为 "dev"）
- Git 标签（vx.x.x）
- 变更内容（CHANGELOG.md）
- 二进制文件下载地址

**等待用户确认后再推送！**

### 7. 推送远程

```bash
git push origin main
git push origin vx.x.x
```

---

## 技术要点

### 配置管理

- 配置文件位置：当前目录 `.agent-fs.yaml` > 用户目录 `~/.agent-fs.yaml`
- 配置优先级：CLI 参数 > 环境变量 > 配置文件
- 环境变量前缀：`AFS_`

### 环境变量

| 变量 | 说明 | 示例 |
|------|------|------|
| `AFS_WORKSPACE` | 沙箱工作目录 | `/safe/workspace` |
| `AFS_S3_ENDPOINT` | S3 端点 | `https://xxx.r2.cloudflarestorage.com` |
| `AFS_S3_BUCKET` | S3 存储桶 | `my-bucket` |
| `AFS_S3_ACCESS_KEY_ID` | 访问密钥 ID | `xxx` |
| `AFS_S3_SECRET_ACCESS_KEY` | 访问密钥 | `xxx` |
| `AFS_S3_CDN_HOST` | CDN 域名 | `https://pub-xxx.r2.dev` |

### 输出格式

所有命令返回标准化 JSON：

```json
{
  "success": true,
  "action": "command_name",
  "data": {...},
  "error": null
}
```

### 错误码定义

| 错误码 | 说明 |
|--------|------|
| `ERR_INVALID_ARGUMENT` | 参数错误 |
| `ERR_PATH_TRAVERSAL` | 路径穿越检测 |
| `ERR_NOT_FOUND` | 文件或目录不存在 |
| `ERR_CONFLICT` | 目标已存在 |
| `ERR_PROVIDER` | 不支持的提供商 |
| `ERR_UPLOAD` | 上传失败 |
| `ERR_DOWNLOAD` | 下载失败 |
| `ERR_ARCHIVE` | 归档操作失败 |
| `ERR_CONFIG` | 配置错误 |
| `ERR_INTERNAL` | 内部错误 |

### 支持的云存储

| ID | 名称 | Endpoint 示例 |
|----|------|---------------|
| `s3` | AWS S3 | `https://s3.amazonaws.com` |
| `r2` | Cloudflare R2 | 自动生成（配置 account_id） |
| `minio` | MinIO | `http://localhost:9000` |
| `oss` | 阿里云 OSS | `https://oss-cn-hangzhou.aliyuncs.com` |
| `cos` | 腾讯云 COS | `https://cos.ap-guangzhou.myqcloud.com` |
| `b2` | Backblaze B2 | `https://s3.us-west-004.backblazeb2.com` |
| `wasabi` | Wasabi | `https://s3.wasabisys.com` |
| `cephfs` | CephFS | N/A（分布式文件系统路径） |

---

## 命令参考

### fs 命令

`fs` 是统一的文件系统操作命令，支持本地和云存储。

**支持的 URI Scheme:**
- `file` - 本地文件系统（默认，可省略）
- `s3` - Amazon S3
- `r2` - Cloudflare R2
- `minio` - MinIO
- `cos` - 腾讯云 COS
- `oss` - 阿里云 OSS
- `cephfs` - CephFS 分布式文件系统

```bash
# 读取文件内容
afs fs read <path>                    # 读取本地文件
afs fs read s3://bucket/key           # 读取云存储文件
afs fs read <path> --head N           # 读取开头 N 行
afs fs read <path> --tail N           # 读取末尾 N 行
afs fs read <path> --bytes N          # 读取前 N 字节

# 列出文件
afs fs ls <path>                      # 列出本地目录
afs fs ls s3://bucket/prefix/         # 列出云存储对象
afs fs ls <path> --limit N            # 限制返回数量

# 获取文件信息
afs fs info <path>                    # 获取本地文件元数据
afs fs info s3://bucket/key           # 获取云存储对象元数据

# 复制文件
afs fs cp <source> <dest>             # 复制文件
afs fs cp s3://bucket/key ./local/    # 从云存储下载
afs fs cp ./local/file s3://bucket/   # 上传到云存储

# 生成 URL
afs fs url <path>                     # 生成 Presigned URL
afs fs url <path> --expires 3600      # 自定义过期时间
afs fs url <path> --public            # 生成公共 URL

# 查看支持的提供商
afs fs providers                      # 列出所有支持的云存储提供商
```

### config 命令

```bash
afs config set <key> <value>              # 设置配置
afs config get <key>                       # 获取配置
afs config set <key> <value> --global      # 设置全局配置
```

---

## 开发规范

### 文件命名

- 使用小写字母
- 多单词用下划线分隔：`local_info.go`
- 测试文件：`xxx_test.go`

### 包命名

- 全小写，无下划线
- 简短描述性名称：`apperr`, `sandbox`

### 错误处理

```go
// 使用 apperr 包装错误
return apperr.Wrap(`action_name`, apperr.CodeNotFound, `file not found`, err)

// 创建新错误
return apperr.New(`action_name`, apperr.CodeInvalidArg, `invalid argument`)
```

### JSON 输出

```go
// 成功输出
return output.PrintSuccess(`action_name`, data)

// 失败输出
return apperr.Wrap(`action_name`, code, message, err)
// 在 main.go 中统一处理为 JSON
```

### 路径处理

所有用户输入路径必须经过 `sandbox.ResolveReadPath` 或 `sandbox.ResolveWritePath`：

```go
resolvedPath, err := sandbox.ResolveReadPath(userInput)
// 自动处理路径穿越、沙箱限制等
```

---

## CI/CD 配置

### 持续集成 (.github/workflows/ci.yml)

- Go 1.24
- `go vet` 检查
- `go test -v -race -coverprofile` 测试
- `golangci-lint` 静态检查

### 发布流程 (.github/workflows/release.yml)

- 多平台交叉编译：linux/darwin/windows × amd64/arm64
- 生成 SHA256SUMS
- 自动创建 GitHub Release

---

## 单一职责边界

### ✅ afs 应该做的

- 文件上传/下载（云存储同步）
- 文件信息查询
- 文件内容读取（切片）
- Zip 打包/解压
- 配置管理
- 安全沙箱

### ❌ afs 不应该做的

- 图片/视频处理（媒体编辑）
- PDF/Excel 处理（文档处理）
- 格式转换（内容转换）
- 水印/滤镜（内容编辑）

**原则**: 复杂的文件处理交给专业工具（ffmpeg, ImageMagick 等），afs 只负责传输。

---

## 相关链接

- AWS SDK v2: https://github.com/aws/aws-sdk-go-v2
- Cobra: https://github.com/spf13/cobra
- Viper: https://github.com/spf13/viper
- 项目参考: https://github.com/geekjourneyx/jina-cli

