# 对象存储 Provider 架构分析与统一 Scheme 方案

## 一、当前架构概述

当前项目在 `pkg/provider` 目录下实现了一个统一的存储 Provider 架构，采用 **接口抽象 + 注册机制** 的设计模式：

```
mermaid
graph TB
    subgraph "pkg/provider"
        P[StorageProvider 接口] --> R[registry.go 注册中心]
        R --> S3[s3_provider.go]
        R --> R2[r2_provider.go]
        R --> Minio[minio_provider.go]
        R --> COS[cos_provider.go]
        R --> OSS[oss_provider.go]
        R --> File[file_provider.go]
        R --> Ceph[cephfs_provider.go]
    end
    
    subgraph "pkg/uri"
        Parser[parser.go URI解析器] --> P
    end
    
    subgraph "pkg/cloud"
        Disp[dispatcher.go 分发器] --> S3Cloud[s3_provider.go]
    end
```

## 二、Provider 接口定义分析

### 2.1 核心接口 StorageProvider

位置：[`pkg/provider/provider.go`](pkg/provider/provider.go:53)

```go
type StorageProvider interface {
    Scheme() string
    Read(ctx context.Context, path string) (io.ReadCloser, error)
    Write(ctx context.Context, path string, data io.Reader) error
    Delete(ctx context.Context, path string) error
    List(ctx context.Context, path string) ([]FileInfo, error)
    Stat(ctx context.Context, path string) (*FileInfo, error)
    Exists(ctx context.Context, path string) (bool, error)
    Copy(ctx context.Context, srcPath, dstPath string) error
}
```

### 2.2 文件结构

| 文件 | 描述 | Scheme |
|------|------|--------|
| [`pkg/provider/s3_provider.go`](pkg/provider/s3_provider.go) | S3/R2/MinIO 统一实现 | s3, r2, minio |
| [`pkg/provider/cos_provider.go`](pkg/provider/cos_provider.go) | 腾讯云 COS | cos |
| [`pkg/provider/oss_provider.go`](pkg/provider/oss_provider.go) | 阿里云 OSS | oss |
| [`pkg/provider/file_provider.go`](pkg/provider/file_provider.go) | 本地文件系统 | file |
| [`pkg/provider/cephfs_provider.go`](pkg/provider/cephfs_provider.go) | CephFS 分布式存储 | cephfs |
| [`pkg/provider/registry.go`](pkg/provider/registry.go) | Provider 注册中心 | - |

## 三、各 Provider 共同点分析

### 3.1 统一的接口实现

所有云存储 Provider (S3/R2/MinIO/COS/OSS) 都实现了以下统一模式：

1. **底层客户端统一** - 都使用 `s3client.Client`，基于 AWS SDK v2
2. **配置来源一致** - 都从环境变量读取配置
3. **对象键 (Key) 语义** - 文件路径在云存储中作为对象 Key
4. **错误处理一致** - 使用 `smithy.APIError` 判断 NotFound
5. **元数据结构统一** - 返回统一的 `FileInfo` 结构

### 3.2 配置文件参数对比

| Provider | ENDPOINT | REGION | BUCKET | ACCESS_KEY_ID | SECRET_ACCESS_KEY | PathStyle |
|----------|----------|--------|--------|---------------|-------------------|------------|
| S3 | 可选 | 必填 | 必填 | 必填 | 必填 | false (虚拟主机) |
| R2 | 必填 | 必填 | 必填 | 必填 | 必填 | true (路径) |
| MinIO | 必填 | 必填 | 必填 | 必填 | 必填 | true (路径) |
| COS | 必填 | 必填 | 必填 | 必填 | 必填 | false (虚拟主机) |
| OSS | 必填 | 必填 | 必填 | 必填 | 必填 | true (路径) |

### 3.3 代码实现高度相似

`S3CompatibleProvider`、`COSProvider`、`OSSProvider` 的实现代码有 90% 以上的相似度，都包含：

- 相同的 CRUD 方法实现
- 相同的 `ListObjects` 调用
- 相同的 `HeadObject` 用于 Stat/Exists
- 相同的 Copy 逻辑

## 四、各 Provider 差异点分析

### 4.1 Endpoint 风格差异

| Provider | Endpoint 示例 | 虚拟主机风格 | 路径风格 |
|----------|---------------|--------------|----------|
| AWS S3 | s3.amazonaws.com | ✅ bucket.s3.amazonaws.com | ✅ s3.amazonaws.com/bucket |
| Cloudflare R2 | xxx.r2.cloudflarestorage.com | ❌ | ✅ |
| MinIO | localhost:9000 | 可能 | ✅ |
| 阿里云 OSS | oss-cn-hangzhou.aliyuncs.com | ✅ bucket.oss-cn-hangzhou.aliyuncs.com | ✅ |
| 腾讯云 COS | cos.ap-guangzhou.myqcloud.com | ✅ bucket.cos.ap-guangzhou.myqcloud.com | ✅ |

### 4.2 PathStyle 配置差异

- **S3**: 默认虚拟主机风格 (`PathStyle = false`)
- **COS**: 虚拟主机风格 (`PathStyle = false`)
- **OSS**: 必须使用路径风格 (`PathStyle = true`)
- **R2/MinIO**: 路径风格 (`PathStyle = true`)

### 4.3 认证方式

所有 Provider 都使用 AWS Signature v4 签名，与 S3 兼容。

### 4.4 Provider 注册差异

```go
// S3/R2/MinIO 统一入口
func NewS3CompatibleProvider(scheme string)

// COS 独立入口
func NewCOSProvider()

// OSS 独立入口
func NewOSSProvider()
```

## 五、URI 解析逻辑分析

### 5.1 当前支持的 URI 格式

位置：[`pkg/uri/parser.go`](pkg/uri/parser.go:38)

```go
// 支持的格式：
// - file:///absolute/path
// - s3://bucket/key
// - r2://bucket/key
// - minio://bucket/key
// - oss://bucket/key
// - cos://bucket/key
// - https://bucket.s3.amazonaws.com/key (VHost)
// - https://s3.amazonaws.com/bucket/key (Path)
```

### 5.2 Scheme 检测逻辑

位置：[`pkg/uri/parser.go`](pkg/uri/parser.go:159)

```go
func detectSchemeFromHost(host string) string {
    // AWS S3
    if strings.HasSuffix(hostLower, ".amazonaws.com") { return "s3" }
    // Cloudflare R2
    if strings.Contains(hostLower, "r2.cloudflarestorage.com") { return "r2" }
    // 阿里云 OSS
    if strings.Contains(hostLower, ".aliyuncs.com") { return "oss" }
    // 腾讯云 COS
    if strings.Contains(hostLower, ".myqcloud.com") { return "cos" }
    // 默认 MinIO
    return "minio"
}
```

## 六、Provider 注册与实例化机制

### 6.1 注册中心设计

位置：[`pkg/provider/registry.go`](pkg/provider/registry.go)

```go
type Registry struct {
    mu         sync.RWMutex
    providers  map[string]ProviderFactory  // scheme -> 工厂函数
    instances  map[string]StorageProvider   // scheme -> 单例
}

// 注册时机：init() 函数中自动注册
func init() {
    Register("s3", NewS3Provider)
    Register("r2", NewR2Provider)
    Register("minio", NewMinioProvider)
    Register("cos", NewCOSProvider)
    Register("oss", NewOSSProvider)
    Register("file", NewFileProvider)
}
```

### 6.2 获取 Provider 流程

```go
// 单例模式获取
provider, err := registry.Get(ctx, "s3")
// 内部逻辑：
// 1. 先检查缓存实例
// 2. 无则调用工厂函数创建
// 3. 缓存并返回
```

## 七、统一为 http/https Scheme 的技术难点

### 7.1 核心挑战

1. **多租户/多配置问题**
   - 当前 scheme (s3/r2/oss/cos) 可通过不同环境变量区分配置
   - 统一为 http/https 后，需要从 URL 本身或查询参数中识别配置

2. **Endpoint 推断困难**
   - s3/r2/cos 都有各自的域名模式
   - 从 URL 推断 provider 类型需要复杂的 hostname 匹配

3. **虚拟主机 vs 路径风格混用**
   - 不同 provider 的默认风格不同
   - 需要在 URL 解析时明确判断

4. **认证信息传递**
   - 当前通过环境变量传递 AccessKey/SecretKey
   - 统一 scheme 后需要支持 URL 或配置文件中传递

5. **向后兼容性**
   - 现有 s3://、r2:// 等 scheme 需要保持可用

### 7.2 设计方案对比

| 方案 | 优点 | 缺点 |
|------|------|------|
| A: 全部改为 https:// | 统一入口 | 需要复杂 URL 解析 |
| B: 新增 https:// 支持 | 渐进式迁移 | 两套逻辑并行 |
| C: 虚拟 Provider 路由 | 解耦清晰 | 实现复杂度高 |

### 7.3 推荐方案：虚拟 Provider 路由

```go
// 新增虚拟 Provider，根据 URL 自动路由
type HTTPProvider struct {
    // 内嵌实际的 provider
    delegate StorageProvider
}

// URL 格式：https://{endpoint}/{bucket}/{key}?provider=r2
// 或自动检测：https://account.r2.cloudflarestorage.com/bucket/key

func (p *HTTPProvider) Scheme() string {
    return "https"  // 同时支持 http/https
}

func (p *HTTPProvider) delegateByURL(rawURL string) (StorageProvider, error) {
    // 自动检测 provider 类型
    // 1. 解析 hostname 模式
    // 2. 查询参数指定 provider
    // 3. 返回对应 delegate
}
```

## 八、实施建议

### 8.1 分阶段实施

1. **第一阶段：抽象公共逻辑**
   - 提取 S3CompatibleProvider 的公共底层
   - 消除 COS/OSS 的代码重复

2. **第二阶段：新增 HTTP Provider**
   - 保持现有 s3:// 等 scheme 不变
   - 新增 https:// 支持

3. **第三阶段：配置灵活性增强**
   - 支持从配置文件读取 provider 配置
   - 添加 URL 查询参数覆盖环境变量

### 8.2 关键文件清单

需要修改的文件：

- [`pkg/provider/s3_provider.go`](pkg/provider/s3_provider.go) - 重构为通用底层
- [`pkg/provider/cos_provider.go`](pkg/provider/cos_provider.go) - 委托给 S3 兼容实现
- [`pkg/provider/oss_provider.go`](pkg/provider/oss_provider.go) - 委托给 S3 兼容实现
- [`pkg/provider/registry.go`](pkg/provider/registry.go) - 新增 HTTP Provider 注册
- [`pkg/uri/parser.go`](pkg/uri/parser.go) - 增强 HTTP URL 解析

### 8.3 预期收益

- 减少约 60% 的代码重复
- 统一的错误处理和重试逻辑
- 更灵活的 provider 配置方式
- 支持任意 S3 兼容存储服务
