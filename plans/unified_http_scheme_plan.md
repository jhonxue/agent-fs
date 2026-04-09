# 对象存储统一 HTTP/HTTPS Scheme 设计方案

## 背景与目标

当前项目使用多种 scheme（`s3://`, `r2://`, `oss://`, `cos://`, `minio://`）来区分不同的对象存储提供商。本方案统一这些 scheme 并支持完整 URL 格式：

1. **使用完整 URL** - `https://bucket.s3.amazonaws.com/key`
2. **向后兼容 Legacy Scheme** - 同样要求完整 URL，能解析出 bucket 和 endpoint
3. **配置驱动** - 通过配置文件定义 provider 参数（AK/SK 等）
4. **别名优先** - 别名与 scheme 同名时以配置文件中的别名为准

## 设计原则

- **向后兼容** - 现有的 `s3://`, `oss://` 等 scheme 继续可用，但必须提供完整 URL
- **配置驱动** - 凭证从配置文件获取，通过 bucket + endpoint 匹配
- **验证优先** - URL 包含的信息必须与配置匹配，否则报错
- **智能推断** - URL 信息不全时使用配置文件中的默认值

---

## 配置文件设计

### 配置格式（YAML）

```yaml
providers:
  # S3 (AWS) - 别名与 scheme 相同
  s3:
    type: s3
    endpoint: s3.amazonaws.com
    bucket: my-s3-bucket        # 可选：预定义的 bucket
    region: us-east-1
    access_key: ${AWS_ACCESS_KEY_ID}
    secret_key: ${AWS_SECRET_ACCESS_KEY}

  # Cloudflare R2
  r2:
    type: s3
    endpoint: ${ACCOUNT_ID}.r2.cloudflarestorage.com
    access_key: ${R2_ACCESS_KEY_ID}
    secret_key: ${R2_SECRET_ACCESS_KEY}
    path_style: true

  # 阿里云 OSS (使用别名)
  myoss:
    type: oss
    endpoint: oss-cn-hangzhou.aliyuncs.com
    bucket: my-oss-bucket
    access_key: ${OSS_ACCESS_KEY_ID}
    secret_key: ${OSS_SECRET_ACCESS_KEY}

  # MinIO (使用别名)
  local-minio:
    type: s3
    endpoint: localhost:9000
    bucket: my-minio-bucket
    access_key: minioadmin
    secret_key: minioadmin
    path_style: true
```

### 配置结构体

```go
// Config 存储服务配置
type Config struct {
    Providers map[string]*ProviderConfig `yaml:"providers"`
}

// ProviderConfig Provider 配置
type ProviderConfig struct {
    Type       string `yaml:"type"`       // s3, oss, cos
    Endpoint   string `yaml:"endpoint"`     // 端点地址
    Bucket     string `yaml:"bucket"`       // 预定义的 bucket
    Region     string `yaml:"region"`      // 区域 (可选)
    AccessKey  string `yaml:"access_key"`  // 访问密钥
    SecretKey  string `yaml:"secret_key"`  // 秘密密钥
    PathStyle  bool   `yaml:"path_style"`  // 是否使用 Path Style
}
```

### 配置匹配规则

**别名与 scheme 同名时的优先级：**
- 当 scheme = "s3" 且配置中存在名为 "s3" 的别名时，**以配置中的别名为准**
- Legacy scheme（如 `s3://`）同样要求完整 URL

**URL 信息验证逻辑：**
1. URL 包含 bucket 和 endpoint → 与配置对比，必须匹配
2. URL 只包含 endpoint → 使用配置中的 bucket
3. URL 只包含 bucket → 使用配置中的 endpoint
4. URL 两者都不包含 → 完全使用配置中的信息

---

## URI 解析策略

### URL 格式

| 使用方式 | 格式 | 示例 |
|----------|------|------|
| **HTTP/HTTPS 完整 URL** | `https://[bucket.]endpoint/key` | `https://bucket.s3.amazonaws.com/key` |
| **Legacy Scheme 完整 URL** | `s3://bucket/key` (需完整 URL) | `s3://mybucket.s3.amazonaws.com/data.txt` |
| **别名** | `myalias://bucket/key` | `myoss://my-bucket/data.txt` |

### 解析流程

```
输入 URI
    │
    ▼
┌─────────────────┐
│ 获取 Scheme     │
└────────┬────────┘
         │
    ┌────▼─────────────────────────────┐
    │ 1. scheme = "http" 或 "https"?    │
    │ 2. scheme 是否在配置文件中有别名? │
    └───────────────┬───────────────────┘
         ┌──────────┴──────────┐
         ▼                     ▼
       YES                    NO
         │                     │
         ▼                     ▼
┌──────────────┐    ┌────────────────────┐
│ 解析完整URL   │    │ 视为 Legacy Scheme │
│ (已有实现)    │    │ - 必须包含完整URL  │
│              │    │ - 去配置文件匹配   │
└──────────────┘    └────────────────────┘
```

### 配置文件匹配验证

**当 scheme 与配置中别名相同时：**

| URL 信息 | 配置文件信息 | 处理方式 |
|----------|--------------|----------|
| URL 有 bucket + endpoint | 配置有 bucket + endpoint | **验证是否匹配**，不匹配报错 |
| URL 有 endpoint | 配置有 bucket | 使用配置的 bucket |
| URL 有 bucket | 配置有 endpoint | 使用配置的 endpoint |
| URL 无 bucket + endpoint | 配置有 bucket + endpoint | 使用配置中的全部信息 |

**Legacy Scheme 处理（如 s3://）：**
1. 解析 URL 获取 bucket 和 endpoint
2. 在配置文件中查找与 endpoint 匹配的 provider
3. 如果 URL 的 bucket ≠ 配置中的 bucket，报错
4. 使用匹配的配置文件创建 provider

---

## Provider 注册机制

### 注册表设计

```go
// ProviderRegistry provider 注册表
type ProviderRegistry struct {
    mu        sync.RWMutex
    providers map[string]ProviderFactory
    configs   map[string]*ProviderConfig
}

// ProviderFactory provider 工厂函数
type ProviderFactory func(*ProviderConfig) (StorageProvider, error)
```

### 注册内置 Provider

```go
func init() {
    RegisterProvider("s3", NewS3Provider)
    RegisterProvider("oss", NewOSSProvider)
    RegisterProvider("cos", NewCOSProvider)
    RegisterProvider("r2", NewS3Provider)  // R2 也是 S3 兼容
    RegisterProvider("minio", NewS3Provider)
}
```

---

## 实现步骤

### 阶段 1: 配置系统

- [ ] 创建 `pkg/config/config.go` - 加载和管理配置
- [ ] 支持从环境变量读取敏感信息 (`${ENV_VAR}`)
- [ ] 支持配置文件路径指定（默认 `~/.afs/config.yaml`）

### 阶段 2: 扩展 URI 解析器

- [ ] 修改 `ParseWithEndpoint()` 支持配置文件中的别名
- [ ] 添加 `LoadAlias()` 方法从配置获取 alias
- [ ] 在 URI struct 中添加 `Alias` 字段

### 阶段 3: Provider 工厂

- [ ] 创建 `pkg/provider/factory.go` - 根据配置创建 provider
- [ ] 实现基于配置的 S3 客户端创建
- [ ] 复用现有的 s3client.Client

### 阶段 4: 命令行集成

- [ ] 添加 `--config` 或 `-c` 参数指定配置文件
- [ ] 在 `cmd/root.go` 中初始化配置

### 阶段 5: 向后兼容

- [ ] 保持现有的 `s3://`, `oss://` 等 scheme 可用
- [ ] 环境变量认证继续工作
- [ ] 添加迁移指南文档

---

## 使用示例

### 方式 1: 完整 URL

```bash
# HTTP/HTTPS 完整 URL (自动检测 provider 类型)
afs fs ls https://mybucket.s3.amazonaws.com/

# Path 风格 (R2)
afs fs ls https://account.r2.cloudflarestorage.com/mybucket/

# 带端口 (MinIO)
afs fs ls http://localhost:9000/mybucket/
```

### 方式 2: 配置文件别名

```yaml
# ~/.afs/config.yaml
providers:
  mys3:
    type: s3
    endpoint: s3.amazonaws.com
    bucket: mybucket
    region: us-east-1
    access_key: ${AWS_ACCESS_KEY_ID}
    secret_key: ${AWS_SECRET_ACCESS_KEY}
```

```bash
# 使用别名访问（完全使用配置中的信息）
mys3:///

# 使用别名 + bucket（URL 有 bucket，验证与配置是否一致）
mys3://mybucket/  # 验证 URL bucket = 配置 bucket
```

### 方式 3: Legacy Scheme (向后兼容)

Legacy scheme 同样要求完整 URL，必须能解析出 bucket 和 endpoint：

```bash
# 完整 URL 格式 (需要配置文件中的凭证)
afs fs ls s3://mybucket.s3.amazonaws.com/data.txt

# 如果配置文件中 s3 别名的 endpoint 不匹配，报错
# Error: URL endpoint (other-endpoint.s3.amazonaws.com) 
#        does not match configured endpoint (s3.amazonaws.com)
```

### 验证示例

| 配置 | URL | 结果 |
|------|-----|------|
| endpoint: s3.amazonaws.com, bucket: mybucket | `s3://mybucket.s3.amazonaws.com/key` | ✅ 匹配成功 |
| endpoint: s3.amazonaws.com, bucket: mybucket | `s3://other-bucket.s3.amazonaws.com/key` | ❌ bucket 不匹配 |
| endpoint: s3.amazonaws.com, bucket: mybucket | `s3://mybucket.other-endpoint.com/key` | ❌ endpoint 不匹配 |
| endpoint: s3.amazonaws.com, bucket: mybucket | `s3://mybucket.s3.amazonaws.com/` | ✅ bucket 和 endpoint 都匹配 |

---

## 关键技术决策

1. **配置加载时机** - 启动时加载，可热重载
2. **凭证优先级** - URL 内嵌 > 环境变量 > 配置文件
3. **错误处理** - 配置文件错误时回退到 legacy 模式
4. **缓存策略** - Provider 实例缓存，避免重复创建

---

## 风险与缓解

| 风险 | 缓解措施 |
|------|----------|
| 配置泄露 | 使用环境变量引用，不存储明文凭证 |
| 兼容性问题 | 保留 legacy scheme，完全向后兼容 |
| 性能影响 | Provider 实例缓存 |
| 复杂度增加 | 清晰的分层设计 |