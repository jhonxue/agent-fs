# Provider 架构分析与重构规划

## 一、当前架构概述

### 1.1 整体架构设计

当前 `pkg/provider` 目录采用**工厂模式 + 注册机制**的架构设计，实现了统一存储抽象层，支持多种云存储和本地文件系统。

```mermaid
flowchart TB
    subgraph Interface层
        SP[StorageProvider Interface]
    end
    
    subgraph Provider实现层
        S3CP[S3CompatibleProvider - s3/r2/minio]
        OSSP[OSSProvider - 阿里云OSS]
        COSP[COSProvider - 腾讯云COS]
        FP[FileProvider - 本地文件系统]
        CFP[CephFSProvider - CephFS]
    end
    
    subgraph 注册与管理层
        REG[Registry - 注册表]
        FAC[Factory - 工厂]
        CFG[ProviderConfig - 配置管理]
    end
    
    subgraph 客户端层
        S3C[s3client.Client - S3 SDK封装]
    end
    
    SP --> S3CP
    SP --> OSSP
    SP --> COSP
    SP --> FP
    SP --> CFP
    
    S3CP --> S3C
    OSSP --> S3C
    COSP --> S3C
    
    REG --> S3CP
    REG --> OSSP
    REG --> COSP
    REG --> FP
    REG --> CFP
    
    FAC --> CFG
    FAC --> S3CP
</mermaid>

### 1.2 核心接口定义

[`StorageProvider`](pkg/provider/provider.go:76) 接口定义了统一的存储操作：

| 方法 | 说明 | 返回值 |
|------|------|--------|
| `Scheme()` | 返回协议标识 | `string` |
| `Read(ctx, path)` | 读取文件 | `io.ReadCloser, error` |
| `Write(ctx, path, data)` | 写入文件 | `error` |
| `Delete(ctx, path)` | 删除文件 | `error` |
| `List(ctx, path)` | 列出文件 | `[]FileInfo, error` |
| `Stat(ctx, path)` | 获取文件信息 | `*FileInfo, error` |
| `Exists(ctx, path)` | 检查文件存在 | `bool, error` |
| `Copy(ctx, src, dst)` | 复制文件 | `error` |
| `ConfigInfo()` | 返回配置信息 | `ProviderConfigInfo` |

### 1.3 Provider 类型分类

| 类型 | 文件 | 实现结构 | 底层依赖 | 特点 |
|------|------|----------|----------|------|
| S3 兼容 | [`s3_provider.go`](pkg/provider/s3_provider.go) | `S3CompatibleProvider` | `s3client.Client` | 支持 s3/r2/minio |
| 阿里云 OSS | [`oss_provider.go`](pkg/provider/oss_provider.go) | `OSSProvider` | `s3client.Client` | PathStyle=true |
| 腾讯云 COS | [`cos_provider.go`](pkg/provider/cos_provider.go) | `COSProvider` | `s3client.Client` | PathStyle=false |
| 本地文件 | [`file_provider.go`](pkg/provider/file_provider.go) | `FileProvider` | `os` 包 | 无云依赖 |
| CephFS | [`cephfs_provider.go`](pkg/provider/cephfs_provider.go) | `CephFSProvider` | `go-ceph` | 需要 CGO |

---

## 二、问题分析

### 2.1 代码重复问题 - 高严重性

#### 2.1.1 S3-based Provider 完全重复的方法

通过代码对比分析，[`OSSProvider`](pkg/provider/oss_provider.go) 和 [`COSProvider`](pkg/provider/cos_provider.go) 与 [`S3CompatibleProvider`](pkg/provider/s3_provider.go) 存在**100% 代码重复**：

| 方法 | OSSProvider | COSProvider | S3CompatibleProvider | 重复程度 |
|------|-------------|-------------|----------------------|----------|
| `Read()` | 第69-81行 | 第69-81行 | 第106-118行 | **100%** |
| `Write()` | 第84-94行 | 第84-94行 | 第121-131行 | **100%** |
| `Delete()` | 第97-105行 | 第97-105行 | 第134-142行 | **100%** |
| `List()` | 第108-126行 | 第108-126行 | 第145-163行 | **100%** |
| `Stat()` | 第129-162行 | 第129-162行 | 第166-199行 | **100%** |
| `Exists()` | 第165-178行 | 第165-178行 | 第202-215行 | **100%** |
| `Copy()` | 第181-191行 | 第181-191行 | 第218-228行 | **100%** |

**重复代码示例**（[`Read()`](pkg/provider/oss_provider.go:69) 方法）:

```go
// OSSProvider.Read() - oss_provider.go:69-81
func (p *OSSProvider) Read(ctx context.Context, key string) (io.ReadCloser, error) {
    input := &s3.GetObjectInput{
        Bucket: aws.String(p.bucket),
        Key:    aws.String(key),
    }
    output, err := p.client.GetObject(ctx, input)
    if err != nil {
        return nil, err
    }
    return output.Body, nil
}

// COSProvider.Read() - cos_provider.go:69-81 (完全相同)
// S3CompatibleProvider.Read() - s3_provider.go:106-118 (完全相同)
```

#### 2.1.2 结构体字段完全重复

三个 S3-based Provider 的结构体字段完全相同：

```go
// 所有三个 Provider 都有相同的字段定义
type OSSProvider struct {
    scheme    string
    client    *s3client.Client
    bucket    string
    endpoint  string
    accessKey string
    secretKey string
    pathStyle bool
    useSSL    bool
}
```

#### 2.1.3 重复代码统计

| Provider | 总行数 | 与 S3CompatibleProvider 重复行数 | 重复比例 |
|----------|--------|----------------------------------|----------|
| OSSProvider | 209 | ~150 | **72%** |
| COSProvider | 209 | ~150 | **72%** |
| 合计重复 | - | ~300 | - |

### 2.2 架构问题

#### 2.2.1 双重工厂机制导致混乱

当前存在两个工厂机制：

1. [`Registry`](pkg/provider/registry.go:12) - 基于环境变量的工厂注册
2. [`configuredFactories`](pkg/provider/factory.go:14) - 基于配置文件的工厂注册

**问题**:
- 两种机制并存，职责不清晰
- [`factory.go`](pkg/provider/factory.go) 中的 [`CreateS3ProviderFromConfig()`](pkg/provider/factory.go:36) 创建的 Provider 未完整存储配置信息（缺少 endpoint、accessKey 等字段）

```go
// factory.go:52 - 不完整的 Provider 创建
return &S3CompatibleProvider{
    scheme: scheme,
    client: client,
    bucket: providerCfg.Bucket,
    // 缺少: endpoint, accessKey, secretKey, pathStyle, useSSL
}
```

#### 2.2.2 ConfigInfo 与 ProviderConfig 结构重复

[`ProviderConfigInfo`](pkg/provider/provider.go:52) 和 [`ProviderConfig`](pkg/config/config.go:12) 字段几乎相同：

```go
// provider.go:52 - ProviderConfigInfo
type ProviderConfigInfo struct {
    Scheme      string
    Bucket      string
    Endpoint    string
    AccessKey   string
    SecretKey   string
    PathStyle   bool
    UseSSL      bool
}

// config.go:12 - ProviderConfig
type ProviderConfig struct {
    Type       string  // 对应 Scheme
    Endpoint   string
    Bucket     string
    Region     string  // ProviderConfigInfo 缺少
    AccessKey  string
    SecretKey  string
    PathStyle  bool
    UseSSL     bool
}
```

#### 2.2.3 缺少统一的错误类型体系

当前只有两个简单的错误类型：
- [`providerConfigError`](pkg/provider/provider.go:19)
- [`notImplementedError`](pkg/provider/provider.go:28)

缺少：
- 统一的错误码定义
- 错误包装机制
- 错误恢复建议

### 2.3 安全问题

#### 2.3.1 敏感信息暴露风险

[`ProviderConfigInfo`](pkg/provider/provider.go:52) 包含原始敏感字段：

| 风险点 | 说明 | 影响范围 |
|--------|------|----------|
| `AccessKey` | 明文存储访问密钥 | 日志输出、调试信息、错误消息 |
| `SecretKey` | 明文存储密钥 | 同上，更严重 |
| `Equals()` 方法 | 直接比较敏感字段 | 可能被用于日志 |

**潜在暴露场景**：

```go
// 潜在风险：日志输出或错误消息
config := provider.ConfigInfo()
log.Printf("Provider config: %v", config) // 泄露 AccessKey/SecretKey
```

#### 2.3.2 配置验证信息泄露

[`config.go:313`](pkg/config/config.go:313) 中错误消息包含敏感信息：

```go
key := fmt.Sprintf("%s:%s:%s:%s", cfg.Bucket, cfg.Endpoint, cfg.AccessKey, cfg.SecretKey)
// 错误消息可能泄露完整配置
return fmt.Errorf("duplicate provider configurations found: %v", duplicates)
```

#### 2.3.3 缺少敏感信息脱敏机制

当前代码缺少：
- `SafeString()` 方法用于安全日志输出
- 敏感字段哈希化
- 错误消息脱敏

### 2.4 可维护性问题

#### 2.4.1 测试覆盖不完整

[`provider_test.go`](pkg/provider/provider_test.go) 只包含：
- `Equals()` 方法测试（已有）
- `FileInfo` 结构测试
- `ReadCloser` 测试
- Mock Provider 测试

缺少：
- 每个 Provider 的集成测试
- 配置验证测试
- 错误处理测试
- 边界条件测试

#### 2.4.2 CephFS Provider 构建标签问题

[`cephfs_provider.go`](pkg/provider/cephfs_provider.go) 使用 `//go:build cephfs && cgo`，但缺少：
- 非 CGO 环境下的 stub 实现
- 条件编译文档说明
- 构建指南

#### 2.4.3 FileProvider 缺少沙盒配置灵活性

[`FileProvider`](pkg/provider/file_provider.go) 的沙盒验证使用硬编码的 [`sandbox`](pkg/sandbox) 包：
- 无法动态配置允许的路径
- 缺少沙盒规则文档

---

## 三、重构方案

### 3.1 高优先级重构 - 立即实施

#### 3.1.1 创建 S3BaseProvider 消除代码重复

**方案**: 创建 [`S3BaseProvider`](pkg/provider/s3_base.go) 作为基础结构：

```go
// pkg/provider/s3_base.go
package provider

type S3BaseProvider struct {
    scheme    string
    client    *s3client.Client
    bucket    string
    endpoint  string
    accessKey string
    secretKey string
    pathStyle bool
    useSSL    bool
}

// 统一实现所有 S3 操作方法
func (p *S3BaseProvider) Read(ctx context.Context, key string) (io.ReadCloser, error) { ... }
func (p *S3BaseProvider) Write(ctx context.Context, key string, data io.Reader) error { ... }
func (p *S3BaseProvider) Delete(ctx context.Context, key string) error { ... }
func (p *S3BaseProvider) List(ctx context.Context, prefix string) ([]FileInfo, error) { ... }
func (p *S3BaseProvider) Stat(ctx context.Context, key string) (*FileInfo, error) { ... }
func (p *S3BaseProvider) Exists(ctx context.Context, key string) (bool, error) { ... }
func (p *S3BaseProvider) Copy(ctx context.Context, srcKey, dstKey string) error { ... }
func (p *S3BaseProvider) ConfigInfo() ProviderConfigInfo { ... }
```

**OSSProvider/COSProvider 简化为**：

```go
// pkg/provider/oss_provider.go
type OSSProvider struct {
    S3BaseProvider
}

func NewOSSProvider() (StorageProvider, error) {
    return newS3BasedProvider("oss", s3client.Config{
        PathStyle: true,  // OSS 特有配置
        UseSSL:    true,
    })
}

// pkg/provider/cos_provider.go
type COSProvider struct {
    S3BaseProvider
}

func NewCOSProvider() (StorageProvider, error) {
    return newS3BasedProvider("cos", s3client.Config{
        PathStyle: false, // COS 特有配置
        UseSSL:    true,
    })
}
```

**收益**:
- 减少约 300 行重复代码
- 新增 S3-based Provider 只需 20 行代码
- 维护成本降低 70%

#### 3.1.2 修复 CreateS3ProviderFromConfig 不完整问题

**方案**: 在 [`factory.go`](pkg/provider/factory.go) 中补全配置字段：

```go
func CreateS3ProviderFromConfig(providerCfg *config.ProviderConfig, scheme string) (StorageProvider, error) {
    clientCfg := s3client.Config{
        Endpoint:        providerCfg.Endpoint,
        Region:          providerCfg.Region,
        Bucket:          providerCfg.Bucket,
        AccessKeyID:     providerCfg.AccessKey,
        SecretAccessKey: providerCfg.SecretKey,
        PathStyle:       providerCfg.PathStyle,
        UseSSL:          providerCfg.UseSSL,
    }

    client, err := s3client.New(context.Background(), clientCfg)
    if err != nil {
        return nil, fmt.Errorf("failed to create S3 client: %w", err)
    }

    return &S3BaseProvider{
        scheme:    scheme,
        client:    client,
        bucket:    providerCfg.Bucket,
        endpoint:  providerCfg.Endpoint,
        accessKey: providerCfg.AccessKey,
        secretKey: providerCfg.SecretKey,
        pathStyle: providerCfg.PathStyle,
        useSSL:    providerCfg.UseSSL,
    }, nil
}
```

### 3.2 中优先级重构 - 后续迭代

#### 3.2.1 添加敏感信息脱敏机制

**方案 A**: 添加 `SafeString()` 方法：

```go
// pkg/provider/provider.go
func (c ProviderConfigInfo) SafeString() string {
    return fmt.Sprintf("scheme=%s, bucket=%s, endpoint=%s, pathStyle=%v, useSSL=%v",
        c.Scheme, c.Bucket, c.Endpoint, c.PathStyle, c.UseSSL)
}

// 使用方式
log.Printf("Provider config: %s", config.SafeString())
```

**方案 B**: 使用哈希值替代原始密钥比较：

```go
type ProviderIdentity struct {
    Scheme        string
    Bucket        string
    Endpoint      string
    AccessKeyHash string  // SHA256 哈希
    SecretKeyHash string  // SHA256 哈希
    PathStyle     bool
    UseSSL        bool
}

func (c ProviderConfigInfo) Identity() ProviderIdentity {
    return ProviderIdentity{
        Scheme:        c.Scheme,
        Bucket:        c.Bucket,
        Endpoint:      c.Endpoint,
        AccessKeyHash: sha256Hash(c.AccessKey),
        SecretKeyHash: sha256Hash(c.SecretKey),
        PathStyle:     c.PathStyle,
        UseSSL:        c.UseSSL,
    }
}
```

#### 3.2.2 统一错误类型体系

**方案**: 创建 [`errors.go`](pkg/provider/errors.go)：

```go
package provider

type ProviderError struct {
    Code    ErrorCode
    Message string
    Cause   error
}

type ErrorCode string

const (
    ErrConfigMissing    ErrorCode = "CONFIG_MISSING"
    ErrAuthFailed       ErrorCode = "AUTH_FAILED"
    ErrBucketNotFound   ErrorCode = "BUCKET_NOT_FOUND"
    ErrObjectNotFound   ErrorCode = "OBJECT_NOT_FOUND"
    ErrPermissionDenied ErrorCode = "PERMISSION_DENIED"
)

func (e *ProviderError) Error() string {
    return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
}

func (e *ProviderError) Unwrap() error {
    return e.Cause
}
```

#### 3.2.3 补充测试覆盖

需要添加的测试：

| 测试类型 | 测试内容 | 优先级 |
|----------|----------|--------|
| 单元测试 | `S3BaseProvider` 所有方法 | 高 |
| 单元测试 | Provider 创建失败场景 | 中 |
| 集成测试 | 各 Provider 基本操作 | 中 |
| 边界测试 | 大文件、空文件、特殊字符路径 | 低 |

### 3.3 低优先级重构 - 可选优化

#### 3.3.1 合并 ProviderConfigInfo 和 ProviderConfig

考虑使用统一的配置结构：

```go
// pkg/provider/provider.go
type ProviderConfig struct {
    Scheme      string
    Bucket      string
    Endpoint    string
    Region      string
    AccessKey   string  // sensitive
    SecretKey   string  // sensitive
    PathStyle   bool
    UseSSL      bool
}

// 用于比较，不暴露敏感信息
func (c ProviderConfig) CompareKey() string {
    return fmt.Sprintf("%s:%s:%s:%x:%x:%v:%v",
        c.Scheme, c.Bucket, c.Endpoint,
        sha256(c.AccessKey), sha256(c.SecretKey),
        c.PathStyle, c.UseSSL)
}
```

#### 3.3.2 添加跨 Bucket 复制支持

```go
// 检查是否支持跨 bucket 服务端复制
func (p *S3BaseProvider) CanCrossBucketCopy(dst ProviderConfigInfo) bool {
    return p.scheme == dst.Scheme && 
           p.endpoint == dst.Endpoint &&
           p.accessKey == dst.AccessKey &&
           p.secretKey == dst.SecretKey
}
```

---

## 四、实施计划

### 4.1 第一阶段 - 立即实施

```mermaid
flowchart LR
    subgraph 步骤1
        A1[创建 S3BaseProvider]
        A2[重构 OSSProvider]
        A3[重构 COSProvider]
    end
    
    subgraph 步骤2
        B1[修复 CreateS3ProviderFromConfig]
        B2[统一工厂机制]
    end
    
    subgraph 步骤3
        C1[添加 SafeString 方法]
        C2[更新日志输出点]
    end
    
    subgraph 步骤4
        D1[添加单元测试]
        D2[验证功能正确]
    end
    
    步骤1 --> 步骤2 --> 步骤3 --> 步骤4
</mermaid>
```

| 序号 | 任务 | 涉及文件 | 依赖 |
|------|------|----------|------|
| 1 | 创建 `s3_base.go` | 新建 | 无 |
| 2 | 重构 `oss_provider.go` | 修改 | 1 |
| 3 | 重构 `cos_provider.go` | 修改 | 1 |
| 4 | 重构 `s3_provider.go` 使用 S3BaseProvider | 修改 | 1 |
| 5 | 修复 `factory.go` | 修改 | 1-4 |
| 6 | 添加 `SafeString()` | 修改 provider.go | 无 |
| 7 | 搜索并修复日志泄露点 | 多文件 | 6 |
| 8 | 添加测试用例 | 修改 provider_test.go | 1-7 |

### 4.2 第二阶段 - 后续迭代

| 序号 | 任务 | 说明 |
|------|------|------|
| 1 | 创建统一错误类型 | `errors.go` |
| 2 | 合并配置结构 | 可选 |
| 3 | 添加集成测试框架 | 需要 mock 或测试环境 |
| 4 | 跨 bucket 复制优化 | 性能提升 |

---

## 五、风险评估

### 5.1 潜在风险

| 风险 | 影响程度 | 发生概率 | 缓解措施 |
|------|----------|----------|----------|
| 重构破坏现有功能 | 高 | 中 | 充分的单元测试和集成测试 |
| Provider 初始化失败 | 中 | 低 | 保持错误处理逻辑不变 |
| 性能下降 | 低 | 低 | 嵌入结构不会增加开销 |
| 配置兼容性问题 | 中 | 低 | 保持配置字段名称不变 |

### 5.2 回滚策略

- 所有重构通过 Git 分支进行
- 每个阶段完成后进行功能验证
- 保留旧代码结构作为备份
- 渐进式重构，每步可独立回滚

### 5.3 验证清单

```markdown
- [ ] 所有 Provider 能正常初始化
- [ ] Read/Write/Delete 操作正常
- [ ] List/Stat/Exists 操作正常
- [ ] Copy 操作正常
- [ ] ConfigInfo.Equals() 正常工作
- [ ] 日志无敏感信息泄露
- [ ] 测试覆盖率达标
```

---

## 六、总结

### 6.1 问题严重性评估

| 类别 | 严重性 | 影响范围 | 建议处理时间 |
|------|--------|----------|--------------|
| 代码重复 | **高** | 维护成本高 | 立即 |
| 工厂机制混乱 | **中** | 扩展困难 | 第一阶段 |
| 安全问题 | **中** | 合规风险 | 第一阶段 |
| 测试覆盖不足 | **中** | 质量风险 | 第一阶段 |
| 配置结构重复 | **低** | 设计瑕疵 | 后续 |

### 6.2 预期收益

| 收益项 | 预期效果 |
|--------|----------|
| 代码量减少 | 减少 ~300 行重复代码 |
| 新 Provider 添加成本 | 从 200 行降到 20 行 |
| 维护成本 | 降低 70% |
| 安全合规 | 消除敏感信息泄露风险 |
| 测试可靠性 | 覆盖率从 ~30% 提升到 ~80% |

### 6.3 后续建议

1. **立即执行第一阶段重构**，消除代码重复和安全隐患
2. **建立 Provider 开发指南**，规范新 Provider 添加流程
3. **完善测试基础设施**，支持 mock 和集成测试
4. **考虑添加 Provider 状态监控**，支持健康检查和诊断

---

## 附录：代码对比详情

### A. OSSProvider vs COSProvider 方法对比

| 方法 | OSS 行号 | COS 行号 | 差异 |
|------|----------|----------|------|
| Read | 69-81 | 69-81 | 无 |
| Write | 84-94 | 84-94 | 无 |
| Delete | 97-105 | 97-105 | 无 |
| List | 108-126 | 108-126 | 无 |
| Stat | 129-162 | 129-162 | 无 |
| Exists | 165-178 | 165-178 | 无 |
| Copy | 181-191 | 181-191 | 无 |
| ConfigInfo | 194-204 | 194-204 | 无 |

### B. S3-compatible Provider 配置差异

| Provider | PathStyle | UseSSL | 验证要求 |
|----------|-----------|--------|----------|
| S3 | false（默认） | true | 仅需 Bucket |
| R2 | true | true | Endpoint + Bucket |
| MinIO | true | true | Endpoint + Bucket |
| OSS | true | true | Endpoint + Bucket |
| COS | false | true | Endpoint + Bucket |