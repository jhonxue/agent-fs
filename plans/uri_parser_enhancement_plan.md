# URI 解析增强计划

## 背景

当前 URI 解析器 (`pkg/uri/parser.go`) 只支持简化的 URI 格式，如 `s3://bucket/key`。

用户提出需求：**完整的 URL 应该能够从中提取 host 和 port，S3 URL 需要支持 vhost 和 path style 两种模式**。

## 需求分析

### 当前支持的格式
```
# 当前支持
s3://bucket/key
r2://bucket/key
oss://bucket/key
```

### 需要支持的格式

#### 1. 带 Endpoint 的完整 URL
支持从完整的 URL 解析出 host、port、bucket、key：

```
# VHost 风格（虚拟主机风格）
https://bucket.s3.amazonaws.com/key
https://bucket.r2.cloudflarestorage.com/key
https://bucket.oss-cn-hangzhou.aliyuncs.com/key

# Path 风格
https://s3.amazonaws.com/bucket/key
https://cos.ap-guangzhou.myqcloud.com/bucket/key
```

#### 2. 带端口的 URL
```
https://minio.example.com:9000/bucket/key
http://localhost:9000/bucket/key
192.168.1.100:9000/bucket/key
```

## 设计方案

### 1. 扩展 URI 结构体

```go
// URI represents a parsed storage URI with scheme-based routing
type URI struct {
    Scheme    string // "file", "s3", "r2", "minio", etc.
    Host      string // Hostname (e.g., "s3.amazonaws.com")
    Port      int    // Port number (0 if not specified)
    Bucket    string // For cloud storage: bucket name
    Key       string // For cloud storage: object key
    Path      string // For local storage: file path
    Query     string // Optional query string
    IsVHost   bool   // true for vhost style, false for path style
}
```

### 2. 新增解析函数

#### ParseWithEndpoint(fullURL string) (*URI, error)
解析完整的 URL，自动检测 vhost 或 path style 模式。

#### parseS3URL(fullURL string) (*URI, error)
专门处理 S3 兼容的 URL：
- 检测是否是 VHost 风格：`bucket.endpoint/key`
- 检测是否是 Path 风格：`endpoint/bucket/key`
- 提取 host 和 port

### 3. 识别算法

```
1. 解析 scheme (https/http)
2. 提取 host:port
3. 检查 bucket 位置：
   - VHost 风格: host 包含 bucket 作为子域名
   - Path 风格: path 以 /bucket/ 开头
4. 根据风格提取 bucket 和 key
```

## 实现步骤

### 阶段 1: 修改 URI 结构体

- [ ] 在 `pkg/uri/parser.go` 中扩展 URI 结构体，添加 `Host`, `Port`, `IsVHost` 字段
- [ ] 注意：向后兼容，现有代码不应受影响

### 阶段 2: 实现解析逻辑

- [ ] 添加 `ParseWithEndpoint()` 函数
- [ ] 实现 `parseS3URL()` 核心逻辑
- [ ] 支持自动检测 VHost vs Path 风格
- [ ] 处理带端口和不带端口的情况

### 阶段 3: 测试

- [ ] 添加单元测试覆盖各种 URL 格式
- [ ] 测试向后兼容性（现有 URI 格式不受影响）
- [ ] 测试边界情况（IP:port, localhost, etc.）

### 阶段 4: 文档更新

- [ ] 更新 CLAUDE.md 中的 URI 格式说明
- [ ] 更新 README.md 中的使用示例

## 测试用例

| 输入 | Scheme | Host | Port | Bucket | Key | IsVHost |
|------|--------|------|------|--------|-----|---------|
| `https://bucket.s3.amazonaws.com/key` | s3 | s3.amazonaws.com | 443 | bucket | key | true |
| `https://s3.amazonaws.com/bucket/key` | s3 | s3.amazonaws.com | 443 | bucket | key | false |
| `https://bucket.r2.cloudflarestorage.com/a/b.txt` | r2 | r2.cloudflarestorage.com | 443 | bucket | a/b.txt | true |
| `http://localhost:9000/bucket/key` | minio | localhost | 9000 | bucket | key | false |
| `192.168.1.100:9000/bucket/key` | minio | 192.168.1.100 | 9000 | bucket | key | false |
| `s3://bucket/key` (backward compat) | s3 | - | - | bucket | key | - |

## 相关文件

- `pkg/uri/parser.go` - 主要修改
- `pkg/uri/parser_test.go` - 添加测试
- `CLAUDE.md` - 文档更新
- `README.md` - 文档更新
