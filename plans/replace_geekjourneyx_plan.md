# 替换 geekjourneyx 为 jhonxue 的计划

## 任务概述

将工程中所有 `geekjourneyx` 替换为 `jhonxue`，涉及 Go module 名称、import 路径、文档链接等。

## 搜索结果分析

共找到 39 处需要替换（含本计划文档），分布在以下文件中：

### 1. Go Module 配置

| 文件 | 行号 | 内容类型 |
|------|------|----------|
| `go.mod` | 1 | module 名称定义 |

### 2. Go 源文件 (import 路径)

| 文件 | 行号 | 内容类型 |
|------|------|----------|
| `main.go` | 7-9 | import 语句 |
| `cmd/config.go` | 12-14 | import 语句 |
| `cmd/fs.go` | 14-23 | import 语句 |
| `pkg/local/read.go` | 9 | import 语句 |
| `pkg/local/info.go` | 9 | import 语句 |
| `pkg/provider/provider.go` | 9 | import 语句 |
| `pkg/provider/file_provider.go` | 9 | import 语句 |
| `pkg/provider/s3_base.go` | 15 | import 语句 |
| `pkg/provider/s3_base_test.go` | 8 | import 语句 |
| `pkg/provider/s3_provider.go` | 16 | import 语句 |
| `pkg/provider/oss_provider.go` | 6 | import 语句 |
| `pkg/provider/cos_provider.go` | 6 | import 语句 |
| `pkg/provider/factory.go` | 8-9 | import 语句 |
| `pkg/cloud/s3_provider.go` | 7-9 | import 语句 |
| `pkg/cloud/dispatcher.go` | 7-8 | import 语句 |
| `pkg/permission/manager.go` | 7-8 | import 语句 |
| `pkg/sandbox/path.go` | 8 | import 语句 |
| `pkg/uri/parser.go` | 10-12 | import 语句 |
| `pkg/archive/archive.go` | 10 | import 语句 |
| `pkg/provider/file_provider.go` | 9 | import 语句 |
### 3. 文档和脚本文件

| 文件 | 行号 | 内容类型 |
|------|------|----------|
| `README.md` | 64, 102, 114, 118, 123, 134, 154, 160, 705, 737, 749, 753, 758, 766 | GitHub URL 和安装命令 |
| `CLAUDE.md` | 9, 83, 520 | 仓库 URL 和项目参考 |
| `scripts/install.sh` | 10 | REPO 变量定义 |
| `skills/afs/SKILL.md` | 14 | 安装脚本 URL |

## 替换策略

### 全局替换规则

将所有出现的 `geekjourneyx` 替换为 `jhonxue`：

```
github.com/geekjourneyx/agent-fs → github.com/jhonxue/agent-fs
geekjourneyx/agent-fs → jhonxue/agent-fs
```

### 执行方式

使用 `sed` 命令进行批量替换：

```bash
# 在所有 .go 文件中替换
find . -name "*.go" -exec sed -i 's/geekjourneyx/jhonxue/g' {} +

# 在 go.mod 中替换
sed -i 's/geekjourneyx/jhonxue/g' go.mod

# 在 README.md 中替换
sed -i 's/geekjourneyx/jhonxue/g' README.md

# 在 CLAUDE.md 中替换
sed -i 's/geekjourneyx/jhonxue/g' CLAUDE.md

# 在 scripts/install.sh 中替换
sed -i 's/geekjourneyx/jhonxue/g' scripts/install.sh

# 在 skills/afs/SKILL.md 中替换
sed -i 's/geekjourneyx/jhonxue/g' skills/afs/SKILL.md
```

## 注意事项

1. **替换后需要验证**：
   - 运行 `go mod tidy` 确保依赖正确
   - 运行 `go build ./...` 确保编译通过
   - 运行 `go test ./...` 确保测试通过

2. **Git 远程仓库**：
   - 如果已经关联了远程仓库，需要更新 remote URL
   - `git remote set-url origin https://github.com/jhonxue/agent-fs.git`

3. **不需要替换的文件**：
   - `.git/` 目录内的文件（Git 自动管理）
   - 其他可能不在搜索结果中的临时文件

## 执行计划

| 步骤 | 操作 | 验证 |
|------|------|------|
| 1 | 替换 go.mod | `go mod tidy` |
| 2 | 替换所有 .go 文件 | `go build ./...` |
| 3 | 替换 README.md | 人工检查链接格式 |
| 4 | 替换 CLAUDE.md | 人工检查链接格式 |
| 5 | 替换 scripts/install.sh | 人工检查脚本逻辑 |
| 6 | 替换 skills/afs/SKILL.md | 人工检查链接格式 |
| 7 | 运行测试 | `go test ./...` |
| 8 | 更新 git remote（如需要） | `git remote -v` |

---

*文档创建时间：2026-04-15*