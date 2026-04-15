---
name: afs
description: Agent-first cross-platform file operations CLI. Use when AI Agent needs local file operations (zip/unzip/info/read) or cloud storage operations (upload/download/list) with S3-compatible providers.
---

# afs - Agent-First File Operations CLI

CLI tool for AI Agents to perform local file operations and cloud storage synchronization with S3-compatible providers.

## Quick start

**Install**:
```bash
curl -fsSL https://raw.githubusercontent.com/jhonxue/agent-fs/main/scripts/install.sh | bash
```

**Basic usage**:
```bash
# Get file info (local or cloud via URI)
afs fs info file:///path/to/file

# Read file with slicing (Token-aware)
afs fs read file:///path/to/log --tail 50

# Create and extract archives
afs fs zip file:///data --out backup.zip
afs fs unzip file://backup.zip --dest /restore

# Cloud operations (use s3://, r2://, oss:// URI schemes)
afs fs cp ./file.txt s3://bucket/remote/path/
afs fs ls s3://bucket/remote/ --limit 20
afs fs url s3://bucket/remote/file.txt --expires 3600  # Generate access URL
afs fs providers  # List supported providers
```

## Commands

All operations use the unified `afs fs` command with URI schemes to distinguish local/cloud storage:
- `file://` - Local filesystem (default)
- `s3://`, `r2://`, `oss://`, `cos://`, `cephfs://`, `hdfs://` - Cloud storage providers

| Command | Purpose |
|---------|---------|
| `afs fs info` | Get file/directory metadata (local or cloud) |
| `afs fs read` | Read file with slicing options |
| `afs fs zip` | Create zip archive from file/directory |
| `afs fs unzip` | Extract zip archive to destination |
| `afs fs cp` | Copy/upload/download files |
| `afs fs ls` | List objects in storage |
| `afs fs url` | Generate access URL (Presigned URL or Public URL) |
| `afs fs providers` | List supported storage providers |
| `afs config` | Manage configuration |

## File operations (unified)

All operations use URI schemes to distinguish local/cloud storage:
- `file:///path` - Local filesystem
- `s3://bucket/key` - Amazon S3
- `r2://bucket/key` - Cloudflare R2
- `oss://bucket/key` - Aliyun OSS
- `cos://bucket/key` - Tencent COS
- `cephfs:///path` - CephFS
- `hdfs://namenode:8020/path` - HDFS

### Get file/directory info

```bash
# Local file info
afs fs info file:///path/to/file

# Local directory with details
afs fs info file:///path/to/dir --details

# Cloud object info
afs fs info s3://bucket/path/to/file
```

**Output**:
```json
{
  "success": true,
  "action": "local_info",
  "data": {
    "name": "file.txt",
    "path": "/path/to/file.txt",
    "type": "file",
    "size_bytes": 1024,
    "mode": "0644",
    "modified_time": "2024-01-01T00:00:00Z",
    "is_dir": false
  },
  "error": null
}
```

### Read file with slicing (Token-aware)

Designed for large files - avoid reading entire GB logs into context:

```bash
# Read last N lines from local file (perfect for error logs)
afs fs read file:///var/log/app.log --tail 100

# Read first N lines from local file
afs fs read file:///path/to/config.yaml --head 20

# Read first N bytes from local file
afs fs read file:///path/to/data.bin --bytes 4096

# Read entire file (limited to 10MB)
afs fs read file:///path/to/small.txt
```

**Output**:
```json
{
  "success": true,
  "action": "local_read",
  "data": {
    "path": "/path/to/file.log",
    "content": "...",
    "line_count": 50,
    "byte_count": 2048,
    "truncated": true,
    "slice_type": "tail"
  },
  "error": null
}
```

### Create archive

```bash
# Zip a local file or directory
afs fs zip file:///data --out backup.zip

# Zip with nested paths
afs fs zip file:///project/src --out archive.zip
```

**Output**:
```json
{
  "success": true,
  "action": "local_zip",
  "data": {
    "source_path": "/data",
    "zip_path": "backup.zip",
    "size_bytes": 1048576
  },
  "error": null
}
```

### Extract archive

```bash
# Extract to directory
afs fs unzip file://backup.zip --dest /restore
```

**Output**:
```json
{
  "success": true,
  "action": "local_unzip",
  "data": {
    "zip_path": "backup.zip",
    "destination": "/restore",
    "extracted_files": 42,
    "size_bytes": 1048576
  },
  "error": null
}
```

## Cloud storage operations

### Upload (cp)

**⚠️ Security requirement**: Remote path MUST follow `date/hash/` format for cloud uploads. This prevents conflicts and organizes files by time.

```bash
# Upload file (REQUIRED format: YYYYMMDD/hash/filename)
DATE=$(date +%Y%m%d)
HASH=$(md5sum /path/file | cut -c1-8)
afs fs cp ./local_file.txt s3://bucket/${DATE}/${HASH}/file.txt

# Upload directory with automatic compression
afs fs cp /data/logs/ r2://bucket/20260301/a3b4c5d6/logs.zip --zip

# Example: backup with timestamp
afs fs cp ./backup.zip s3://bucket/backups/$(date +%Y%m%d)/$(date +%H%M%S).zip
```

**Path format breakdown**:
- `YYYYMMDD` - Date for organization (e.g., `20260301`)
- `8-char-hash` - Short unique identifier (e.g., `a3b4c5d6`)
- `filename` - Actual filename

**Output**:
```json
{
  "success": true,
  "action": "cloud_upload",
  "data": {
    "provider": "s3",
    "local_path": "/local/file.txt",
    "remote_key": "remote/path/file.txt",
    "remote_url": "https://...",
    "size_bytes": 1024,
    "time_taken_ms": 250,
    "compressed": false
  },
  "error": null
}
```

### Download (cp)

```bash
# Download file
afs fs cp s3://bucket/remote/file.txt /local/path/

# Download and auto-extract
afs fs cp s3://bucket/remote/archive.zip /local/dir/ --unzip
```

**Output**:
```json
{
  "success": true,
  "action": "cloud_download",
  "data": {
    "provider": "s3",
    "remote_key": "remote/file.txt",
    "local_path": "/local/file.txt",
    "size_bytes": 1024,
    "time_taken_ms": 150,
    "decompressed": false
  },
  "error": null
}
```

### List objects (ls)

```bash
# List objects with prefix
afs fs ls s3://bucket/backups/ --limit 50

# List all objects in bucket
afs fs ls r2://bucket/
```

**Output**:
```json
{
  "success": true,
  "action": "cloud_list",
  "data": {
    "provider": "s3",
    "objects": [
      {
        "key": "backups/file1.txt",
        "size_bytes": 1024,
        "last_modified": "2024-01-01T00:00:00Z",
        "etag": "\"d41d8cd98f00b204e9800998ecf8427e\""
      }
    ],
    "count": 1,
    "total_bytes": 1024,
    "prefix": "backups/",
    "is_truncated": false
  },
  "error": null
}
```

### Generate access URL

Generate presigned URL for temporary private access or public URL for publicly accessible objects:

```bash
# Generate presigned URL (default 15 minutes valid)
afs fs url s3://bucket/remote/file.txt

# Custom expiration time (in seconds)
afs fs url s3://bucket/remote/file.txt --expires 3600  # 1 hour
afs fs url s3://bucket/remote/file.txt --expires 60    # 1 minute

# Generate public URL (for objects with public read access)
afs fs url s3://bucket/remote/public.jpg --public

# Specify provider
afs fs url r2://bucket/remote/file.txt --expires 7200
```

**Output (Presigned URL)**:
```json
{
  "success": true,
  "action": "cloud_url",
  "data": {
    "provider": "s3",
    "remote_key": "remote/file.txt",
    "url": "https://my-bucket.s3.amazonaws.com/remote/file.txt?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=...&X-Amz-Date=...&X-Amz-Expires=900&X-Amz-SignedHeaders=host&X-Amz-Signature=...",
    "expires_in": 900,
    "expires_at": "2024-03-01T18:15:30Z",
    "is_presigned": true
  }
}
```

**Output (Public URL)**:
```json
{
  "success": true,
  "action": "cloud_url",
  "data": {
    "provider": "s3",
    "remote_key": "remote/public.jpg",
    "url": "https://my-bucket.s3.amazonaws.com/remote/public.jpg",
    "is_presigned": false
  }
}
```

**URL flags**:
- `--expires N` - Expiration time in seconds (default: 900 = 15 minutes)
- `--public` - Generate public URL instead of presigned URL

### List supported providers

```bash
# List all supported cloud storage providers
afs fs providers
```

**Output**:
```json
{
  "success": true,
  "action": "cloud_providers",
  "data": {
    "providers": [
      {
        "name": "s3",
        "description": "AWS S3",
        "endpoint_example": "https://s3.amazonaws.com"
      },
      {
        "name": "r2",
        "description": "Cloudflare R2",
        "endpoint_example": "https://{account_id}.r2.cloudflarestorage.com",
        "config_note": "Set account_id for auto-generated endpoint, or specify endpoint manually"
      },
      {
        "name": "minio",
        "description": "MinIO Self-Hosted Object Storage",
        "endpoint_example": "http://localhost:9000",
        "config_note": "Set path_style=true for non-virtual-hosted-style access"
      },
      {
        "name": "oss",
        "description": "Alibaba Cloud Object Storage Service (OSS)",
        "endpoint_example": "https://oss-cn-hangzhou.aliyuncs.com",
        "config_note": "Replace region in endpoint: oss-{region}.aliyuncs.com"
      },
      {
        "name": "cos",
        "description": "Tencent Cloud Object Storage (COS)",
        "endpoint_example": "https://cos.ap-guangzhou.myqcloud.com",
        "config_note": "Replace region in endpoint: cos.{region}.myqcloud.com"
      },
      {
        "name": "b2",
        "description": "Backblaze B2 (S3 Compatible)",
        "endpoint_example": "https://s3.us-west-004.backblazeb2.com"
      },
      {
        "name": "wasabi",
        "description": "Wasabi Hot Cloud Storage",
        "endpoint_example": "https://s3.wasabisys.com"
      }
    ],
    "note": "Any S3-compatible storage is supported. Configure custom endpoint via config."
  }
}
```

## Configuration

Config file locations (priority order):
1. `.agent-fs.yaml` (current directory)
2. `~/.agent-fs.yaml` (home directory)

**Environment variables**:
- `AFS_WORKSPACE` - Sandbox workspace root (security, prevents path traversal)
- `AFS_S3_ENDPOINT` - S3 endpoint URL
- `AFS_S3_BUCKET` - S3 bucket name
- `AFS_S3_ACCESS_KEY_ID` - S3 access key
- `AFS_S3_SECRET_ACCESS_KEY` - S3 secret key
- `HDFS_NAMENODE_ADDRS` - HDFS NameNode addresses (comma-separated)
- `HDFS_USERNAME` - HDFS username (default: hadoop)

**Config commands**:
```bash
# Set provider configuration
afs config set s3.endpoint https://xxx.r2.cloudflarestorage.com
afs config set s3.bucket my-bucket
afs config set s3.access_key_id AKIAIOSFODNN7EXAMPLE
afs config set s3.secret_access_key wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY

# View configuration
afs config get s3.endpoint

# Use global config
afs config set s3.endpoint https://xxx.r2.cloudflarestorage.com --global
```

**Provider-specific config**:
```yaml
# ~/.agent-fs.yaml
s3:
  endpoint: https://s3.amazonaws.com
  region: us-east-1
  bucket: my-bucket
  access_key_id: AKIAIOSFODNN7EXAMPLE
  secret_access_key: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY

r2:
  account_id: your_account_id
  bucket: my-r2-bucket
  access_key_id: your_access_key
  secret_access_key: your_secret_key
  endpoint: https://your_account_id.r2.cloudflarestorage.com
```

## Supported providers

**Officially supported (with preset configuration)**:

| Provider | ID | Endpoint Example | Description |
|----------|----|-----------------|-------------|
| AWS S3 | `s3` | `https://s3.amazonaws.com` | Amazon S3 |
| Cloudflare R2 | `r2` | Auto-generated (set account_id) | Cloudflare R2 |
| MinIO | `minio` | `http://localhost:9000` | Self-hosted object storage |
| Aliyun OSS | `oss` | `https://oss-cn-hangzhou.aliyuncs.com` | Aliyun Object Storage |
| Tencent COS | `cos` | `https://cos.ap-guangzhou.myqcloud.com` | Tencent Cloud Object Storage |
| Backblaze B2 | `b2` | `https://s3.us-west-004.backblazeb2.com` | B2 S3 compatible |
| Wasabi | `wasabi` | `https://s3.wasabisys.com` | Wasabi Hot Cloud Storage |
| HDFS | `hdfs` | `hdfs://namenode:8020` | Hadoop Distributed File System |

**Other S3-compatible storage**:

Any S3-protocol compatible object storage is supported:
- DigitalOcean Spaces
- Google Cloud Storage (Interoperability)
- Oracle Cloud Infrastructure
- Custom MinIO deployments

**Query supported providers**:
```bash
afs fs providers
```

## Output format

All commands return standardized JSON for AI Agent parsing:

**Success**:
```json
{
  "success": true,
  "action": "command_name",
  "data": {...},
  "error": null
}
```

**Failure**:
```json
{
  "success": false,
  "action": "command_name",
  "data": null,
  "error": {
    "code": "ERR_NOT_FOUND",
    "message": "File does not exist"
  }
}
```

## Security

### Cloud upload path requirement

**MANDATORY**: All cloud uploads MUST use `YYYYMMDD/hash/` path format:

```bash
# Correct format
afs fs cp ./file.txt s3://bucket/20260301/a3b4c5d6/file.txt

# Incorrect (will be rejected)
afs fs cp ./file.txt s3://bucket/file.txt
afs fs cp ./file.txt s3://bucket/uploads/file.txt
```

**Reasons**:
- Prevents path conflicts between concurrent operations
- Organizes files by date for easy management
- Hash provides uniqueness for same-day uploads
- Avoids accidental file overwrites

### Sandbox mode

Restrict operations to a specific directory:

```bash
export AFS_WORKSPACE=/safe/workspace

# Now all operations are restricted to /safe/workspace
afs fs info file:///etc/passwd  # ERROR: path is outside AFS_WORKSPACE
```

### Path traversal protection

All paths are validated against `AFS_WORKSPACE`. The `../` sequences are blocked automatically.

## Common workflows

### Agent file backup workflow

```bash
# 1. Get info before processing
afs fs info file:///data --details

# 2. Create compressed archive
afs fs zip file:///data --out backup.zip

# 3. Upload to cloud (REQUIRED: date/hash/ format)
DATE=$(date +%Y%m%d)
HASH=$(md5sum backup.zip | cut -c1-8)
afs fs cp ./backup.zip s3://bucket/backups/${DATE}/${HASH}/backup.zip

# 4. Verify upload
afs fs ls s3://bucket/backups/${DATE}/ --limit 1
```

### Error log analysis workflow

```bash
# 1. Check log size
afs fs info file:///var/log/app.log

# 2. Read last 100 lines for errors
afs fs read file:///var/log/app.log --tail 100

# 3. If needed, get more context
afs fs read file:///var/log/app.log --tail 500
```

### Cloud sync workflow

```bash
# 1. List remote files
afs fs ls s3://bucket/projects/ --limit 100

# 2. Download needed file
afs fs cp s3://bucket/projects/config.yaml ./

# 3. Extract if compressed
afs fs unzip file://config.zip --dest ./
```

## Error codes

| Code | Description |
|------|-------------|
| `ERR_INVALID_ARGUMENT` | Invalid command argument |
| `ERR_PATH_TRAVERSAL` | Path outside workspace |
| `ERR_NOT_FOUND` | File/directory not found |
| `ERR_CONFLICT` | Destination already exists |
| `ERR_PROVIDER` | Unsupported/invalid provider |
| `ERR_UPLOAD` | Upload failed |
| `ERR_DOWNLOAD` | Download failed |
| `ERR_ARCHIVE` | Archive operation failed |
| `ERR_CONFIG` | Configuration error |
| `ERR_INTERNAL` | Internal error |

## Implementation notes

- **Go 1.24+** required
- **Cross-platform**: Linux, macOS, Windows
- **S3 protocol**: Compatible with any S3-compatible storage
- **Machine-readable output**: All commands return JSON
- **Token-aware**: Built-in file slicing for large files

## Project structure

```
agent-fs/
├── cmd/                    # CLI commands
│   ├── root.go             # Root command
│   ├── fs.go               # Unified file operations (info/read/zip/unzip/cp/ls/url)
│   └── config.go           # Config management
├── pkg/                    # Core logic
│   ├── local/              # Local file operations
│   │   ├── info.go         # File info
│   │   └── read.go         # File reading
│   ├── provider/           # Storage providers (S3, R2, OSS, COS, CephFS)
│   ├── cloud/              # Cloud abstraction layer
│   ├── unified/            # Unified operations (adapter, dispatcher)
│   ├── uri/                # URI parser
│   ├── sandbox/            # Path security
│   ├── output/             # JSON output
│   └── apperr/             # Error handling
└── main.go                 # Entry point
```
