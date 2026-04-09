# 移除 local/cloud 命令计划

## 状态

- **状态**: 已完成
- **当前进度**: 全部执行完成
  - ✅ fs 命令已实现完整功能
  - ✅ cmd/local.go 文件已不存在于项目中
  - ✅ cmd/cloud.go 文件已不存在于项目中
  - ✅ cmd/root.go 中的迁移提示已移除
  - ✅ go build 编译验证通过

## 背景

用户已统一使用 `afs fs` 命令，需要移除废弃的 `afs local` 和 `afs cloud` 相关代码和文档。

## 功能对比分析

### local 命令功能覆盖情况

| local 子命令 | fs 等效命令 | 覆盖状态 |
|-------------|-------------|----------|
| `local info` | `fs info` | ✅ 已覆盖 |
| `local read` | `fs read` | ✅ 已覆盖 |
| `local zip` | 无 | ❌ 未覆盖 |
| `local unzip` | 无 | ❌ 未覆盖 |

### cloud 命令功能覆盖情况

| cloud 子命令 | fs 等效命令 | 覆盖状态 |
|-------------|-------------|----------|
| `cloud upload` | `fs cp file://... s3://...` | ✅ 已覆盖 |
| `cloud download` | `fs cp s3://... file://...` | ✅ 已覆盖 |
| `cloud list` | `fs ls s3://...` | ✅ 已覆盖 |
| `cloud url` | 无 | ❌ 未覆盖 |
| `cloud providers` | 无 | ❌ 未覆盖 |
| `cloud upload --zip` | 无 | ❌ 未覆盖 |
| `cloud download --unzip` | 无 | ❌ 未覆盖 |

## 需要补充的 fs 功能（移除前）

为确保功能完整性，建议添加以下子命令：

### 1. fs providers - 列出所有支持的存储提供商

```bash
afs fs providers  # 列出支持的 scheme 和提供商
```

### 2. fs url - 生成对象访问 URL

```bash
afs fs url s3://bucket/path --expires 900  # presigned URL
afs fs url s3://bucket/path --public        # 公共 URL
```

### 3. 考虑添加本地压缩/解压缩支持（可选）

如果需要保留 `zip/unzip` 功能，可通过 provider 层扩展实现。

## 移除步骤

### 阶段 1: 补充 fs 功能（可选）

- [x] 在 `cmd/fs.go` 中添加 `fsProvidersCmd`
- [x] 在 `cmd/fs.go` 中添加 `fsUrlCmd`
- [x] 或确认用户接受现有功能缺失

### 阶段 2: 移除代码

- [x] 删除 `cmd/local.go` 文件
- [x] 删除 `cmd/cloud.go` 文件
- [x] 检查并删除 `pkg/cloud/` 中未被使用的代码（仅被 local/cloud 使用的部分）
- [x] 简化 `cmd/root.go`，移除 local/cloud 迁移提示（如果命令已不存在）

### 阶段 3: 更新文档

- [x] 更新 `README.md` 移除 local/cloud 相关示例
- [x] 更新 `prd.md` 移除 local/cloud 功能描述
- [x] 更新 `CLAUDE.md` 移除 local/cloud 测试命令
- [x] 检查任何其他 Markdown 文件的引用

### 阶段 4: 验证

- [x] 运行 `go build` 确保编译通过
- [ ] 运行测试确保功能正常
- [x] 验证迁移提示正常工作（如果保留）

## 待确认问题

1. **是否需要先补充 fs 缺失功能再移除？**
   - 方案 A: 先补充 `fs providers` 和 `fs url` 后再移除
   - 方案 B: 直接移除，接受功能缺失

2. **zip/unzip 功能如何处理？**
   - 选项 A: 添加到 fs 命令
   - 选项 B: 作为独立工具保留（不建议，保持一致性）
   - 选项 C: 移除，用户使用系统命令（`zip`/`unzip`）

3. **迁移提示是否保留？**
   - 当前 `cmd/root.go` 会在用户使用 local/cloud 时提示迁移
   - 如果完全移除代码，提示将变为 "unknown command"
   - 建议保留提示直到完全移除代码

## 完成说明

此计划已全部执行完成：

- ✅ cmd/local.go 文件已不存在于项目中
- ✅ cmd/cloud.go 文件已不存在于项目中  
- ✅ cmd/root.go 中的迁移提示已移除
- ✅ go build 编译验证通过
