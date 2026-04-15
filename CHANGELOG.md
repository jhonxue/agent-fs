# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- HDFS storage provider support via RPC protocol (`hdfs://` scheme)
- New environment variables: `HDFS_NAMENODE_ADDRS`, `HDFS_USERNAME`

## [1.0.0] - 2026-03-01

### Added
- Initial release of agent-fs
- Local file operations: info, read, zip, unzip
- Cloud storage operations: upload, download, list, url
- Configuration management: set, get
- Security sandbox with workspace restriction (AFS_WORKSPACE)
- Token-aware file reading with slicing options (head/tail/bytes)
- Support for 7+ S3-compatible storage providers (S3, R2, MinIO, AliOSS, TXCOS, B2, Wasabi)
- Standardized JSON output for AI Agent parsing
- Presigned URL and Public URL generation
- Decompression bomb protection for zip operations
- Claude Code Skill integration
