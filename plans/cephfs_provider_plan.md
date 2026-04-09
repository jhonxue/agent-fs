# CephFS Provider Implementation Plan (v2)

---

**状态**: 已完成

**当前进度说明**: 
- ✅ 已添加 go-ceph 依赖 (v0.38.0)
- ✅ 已创建 CephFS Provider (`pkg/provider/cephfs_provider.go`)
- ✅ 已实现 StorageProvider 接口的所有核心方法 (Read, Write, Delete, List, Stat, Exists, Copy)
- ✅ 已配置环境变量支持 (CEPHFS_MON_HOSTS, CEPHFS_AUTH_ID, CEPHFS_KEYRING_PATH, CEPHFS_CONF_PATH, CEPHFS_SECRET)
- ✅ 已注册 Provider 到 registry
- ✅ 已更新 URI Parser 支持 `cephfs://` scheme
- ✅ 已实现 FileHandle 方法 (返回错误，提示使用 Read 方法)

---

## Goal
采用 go-ceph 库的 libcephfs API 直接访问 CephFS 存储，集成到现有的 provider 架构中。

**重要变更 (v2)**:
- URI Scheme: `cephfs://` (而非 `ceph://`)
- 访问方式: 直接使用 libcephfs API (而非挂载点)

## Architecture

```mermaid
graph TB
    subgraph pkg/provider
        StorageProvider[StorageProvider Interface]
        FileProvider[FileProvider]
        S3Provider[S3CompatibleProvider]
        CephFSProvider[CephFSProvider]
    end
    
    subgraph pkg/uri
        Parser[URI Parser]
    end
    
    subgraph pkg/provider/cephfs_provider.go
        LibCephFS[go-ceph cephfs API]
    end
    
    StorageProvider <|-- FileProvider
    StorageProvider <|-- S3Provider
    StorageProvider <|-- CephFSProvider
    Parser -->|cephfs://| CephFSProvider
    CephFSProvider -->|CGO libcephfs| LibCephFS
```

## Why Direct libcephfs API?

| 特性 | 挂载方式 | libcephfs API |
|------|----------|----------------|
| 需要预先挂载 | ✅ | ❌ |
| 直接访问集群 | ❌ | ✅ |
| 支持多路径 | 需多个挂载点 | ✅ |
| 性能 | FUSE开销 | 更好 |
| CephFS特性 | 基本支持 | 完整支持 |

## Implementation Steps

### 1. 添加 go-ceph 依赖
- 在 `go.mod` 中添加 `github.com/ceph/go-ceph v0.29.0+` 依赖
- 运行 `go mod tidy` 更新依赖
- 系统依赖: `libcephfs-dev` (Ubuntu) 或 `ceph-devel` (RHEL)

### 2. 创建 CephFS Provider
- 新建文件: `pkg/provider/cephfs_provider.go`
- 实现 [`StorageProvider`](pkg/provider/provider.go:53) 接口

#### 核心方法实现:

| 方法 | 说明 | go-ceph cephfs API |
|------|------|---------------------|
| `Scheme()` | 返回 "cephfs" | - |
| `Read(path)` | 读取文件 | `cephfs.Read` |
| `Write(path, data)` | 写入文件 | `cephfs.Write` |
| `Delete(path)` | 删除文件 | `cephfs.Unlink` |
| `List(path)` | 列出目录 | `cephfs.ReadDir` |
| `Stat(path)` | 获取文件信息 | `cephfs.Stat` |
| `Exists(path)` | 检查文件存在 | `cephfs.Stat` |
| `Copy(src, dst)` | 复制文件 | `cephfs.CopyFile` or manual read/write |

#### 配置方式 - 使用环境变量:
```go
// 通过环境变量配置 (都可选，有默认值)
CEPHFS_CONF_PATH     // ceph.conf 配置文件路径
CEPHFS_MON_HOSTS     // Ceph监视器地址，多个用逗号分隔
CEPHFS_AUTH_ID       // Ceph用户ID (例如 admin)
CEPHFS_KEYRING_PATH  // keyring文件路径 (可选)
CEPHFS_SECRET        // 密钥 (可选，支持环境变量传入)
```

#### URI 格式:
```
cephfs://[bucket/]path
cephfs://volume/subdir/file.txt
```

**注意**: 由于 libcephfs 直接访问 Ceph 集群，没有 bucket 概念。
- URI中的第一个路径部分作为 CephFS 中的根路径前缀
- 例如 `cephfs://mydata/file.txt` 访问 CephFS 中的 `/mydata/file.txt`

### 3. 确保 libcephfs 初始化
- 使用 `cephfs.Init()` 初始化库
- 使用 `cephfs.CreateFromConfigFile()` 或 `cephfs.CreateFromMonHost()` 创建 CephFS 实例
- 在程序退出时调用 `cephfs.Shutdown()`

### 4. 注册 Provider
- 在启动时注册到默认registry: `provider.Register("cephfs", NewCephFSProvider)`

### 5. 更新 URI Parser
- 在 [`pkg/uri/parser.go`](pkg/uri/parser.go:52) 的 switch 语句中添加 `"cephfs"` case

### 6. 添加 FileProviderInterface 支持
- 实现 `FileHandle()` 方法返回底层文件描述符
- 使用 `cephfs.Open()` 获取文件描述符
- 参考 [`FileProvider.FileHandle()`](pkg/provider/file_provider.go:41)

## File Structure

```
pkg/provider/
├── provider.go           # 已存在
├── registry.go           # 已存在
├── file_provider.go      # 已存在
├── s3_provider.go        # 已存在
├── oss_provider.go       # 已存在
├── cos_provider.go      # 已存在
└── cephfs_provider.go   # 新增 (直通 libcephfs)
```

## Usage Examples

```bash
# 设置环境变量
export CEPHFS_MON_HOSTS="192.168.1.1:6789,192.168.1.2:6789"
export CEPHFS_AUTH_ID="admin"
export CEPHFS_KEYRING_PATH="/etc/ceph/ceph.client.admin.keyring"

# 读取 CephFS 文件
afs fs read cephfs:///path/to/file.txt

# 列出 CephFS 目录
afs fs ls cephfs:///my-directory/

# 复制文件 (CephFS内部)
afs fs cp cephfs:///source/file.txt cephfs:///dest/file.txt

# 跨Provider复制
afs fs cp s3://bucket/data.txt cephfs:///backup/
```

## Connection Options (优先级顺序)

1. 配置文件 `CEPHFS_CONF_PATH` (例如 `/etc/ceph/ceph.conf`)
2. 监视器地址 `CEPHFS_MON_HOSTS` + 认证 `CEPHFS_AUTH_ID` + `CEPHFS_KEYRING_PATH`
3. 默认值: 尝试读取 `/etc/ceph/ceph.conf` 和 keyring

## Edge Cases

1. **Ceph集群未连接**: 返回友好错误信息 (例如 "failed to connect to Ceph cluster")
2. **权限不足**: 透传 ceph 错误信息
3. **连接失败**: 提供重试机制，可配置重试次数
4. **大文件**: 使用流式读写 (io.ReaderCloser)
5. **路径处理**: CephFS 使用 "/" 作为路径分隔符，需要规范化 URI 中的路径

## Testing Strategy

1. 单元测试: 使用 mock 或 stub 模式 (需要开发 stub)
2. 集成测试: 需要真实的 Ceph 集群 (Cephadm/rook等)
3. 错误场景测试: 各种异常情况

**注意**: 由于 libcephfs 是 CGO 依赖，可能需要: 
- 构建 tag 来控制是否启用
- 提供纯 Go 的 stub 实现用于测试 (但目前 go-ceph 没有官方 stub)

## Time Estimate
- 依赖添加 + 环境确认: 30min
- Provider实现 (核心方法): 2-3 hours
- URI 解析完善: 30min
- 集成测试: 1-2 hours

## Risk Mitigation

| 风险 | 缓解措施 |
|------|----------|
| CGO 依赖 libcephfs | 提供构建 tag (如 `cephfs`) 控制是否编译 |
| 测试困难 | 提供文档说明如何连接测试集群 |
| 系统依赖 | 在 README 中说明 libcephfs 安装方式 |
| 连接问题 | 提供详细的连接错误信息 |
