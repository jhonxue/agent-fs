package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/geekjourneyx/agent-fs/pkg/apperr"
	"github.com/geekjourneyx/agent-fs/pkg/cloud"
	"github.com/geekjourneyx/agent-fs/pkg/config"
	"github.com/geekjourneyx/agent-fs/pkg/output"
	"github.com/geekjourneyx/agent-fs/pkg/provider"
	"github.com/geekjourneyx/agent-fs/pkg/s3client"
	"github.com/geekjourneyx/agent-fs/pkg/sandbox"
	"github.com/geekjourneyx/agent-fs/pkg/uri"
)

var (
	fsReadHead    int64
	fsReadTail    int64
	fsReadBytes   int64
	fsURLExpires  int64
	fsURLPublic   bool
)

var fsCmd = &cobra.Command{
	Use:   `fs`,
	Short: `Unified filesystem operations (local and cloud)`,
	Long: `Unified filesystem operations supporting both local and cloud storage.

Supported schemes:
  file  - Local filesystem
  s3    - Amazon S3
  r2    - Cloudflare R2
  minio - MinIO
  cos   - Tencent Cloud COS
  oss   - Alibaba Cloud OSS

Examples:
  afs fs read /path/to/file.txt
  afs fs read s3://bucket/path/to/file.txt
  afs fs ls ./data/
  afs fs ls s3://bucket/prefix/
  afs fs cp s3://bucket/data.json file:///tmp/`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var fsReadCmd = &cobra.Command{
	Use:   `read <path>`,
	Short: `Read file content from local or cloud storage`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runFsRead(args[0])
	},
}

var fsLsCmd = &cobra.Command{
	Use:   `ls <path>`,
	Short: `List files in a directory or prefix`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runFsLs(args[0])
	},
}

var fsCpCmd = &cobra.Command{
	Use:   `cp <source> <destination>`,
	Short: `Copy files between local and cloud storage`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runFsCp(args[0], args[1])
	},
}

var fsInfoCmd = &cobra.Command{
	Use:   `info <path>`,
	Short: `Get file information`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runFsInfo(args[0])
	},
}

var fsProvidersCmd = &cobra.Command{
	Use:   `providers`,
	Short: `List supported storage providers`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runFsProviders()
	},
}

var fsUrlCmd = &cobra.Command{
	Use:   `url <path>`,
	Short: `Generate access URL for a cloud object`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runFsURL(args[0])
	},
}

// parsePath parses a path string using ParseWithConfig for full URL support.
// It uses config to validate and match provider by bucket+endpoint+ak+sk.
// Falls back to ParseWithEndpoint if no config is available.
func parsePath(raw string) (*uri.URI, error) {
	// Try to load config for validation
	cfg := config.Default()
	// Try default config path if not yet loaded
	if !cfg.HasProviders() {
		_ = config.LoadFromDefault()
	}

	// Use ParseWithConfig to validate against config (if available)
	parsed, err := uri.ParseWithConfig(raw, cfg)
	if err != nil {
		// Fall back to ParseWithEndpoint for simple paths
		parsed, err = uri.ParseWithEndpoint(raw)
		if err != nil {
			// Final fallback to original Parse
			return uri.Parse(raw)
		}
	}
	return parsed, nil
}

func init() {
	// Read flags
	fsReadCmd.Flags().Int64VarP(&fsReadHead, "head", "n", 0, "Read first N lines")
	fsReadCmd.Flags().Int64VarP(&fsReadTail, "tail", "t", 0, "Read last N lines")
	fsReadCmd.Flags().Int64VarP(&fsReadBytes, "bytes", "b", 0, "Read first N bytes")

	// Validate flags: head and tail are mutually exclusive
	fsReadCmd.MarkFlagsMutuallyExclusive("head", "tail")

	// URL flags
	fsUrlCmd.Flags().Int64Var(&fsURLExpires, "expires", 900, "Expiration time in seconds (default: 900, 15 minutes)")
	fsUrlCmd.Flags().BoolVar(&fsURLPublic, "public", false, "Generate public URL instead of presigned URL")

	// Add subcommands
	fsCmd.AddCommand(fsReadCmd)
	fsCmd.AddCommand(fsLsCmd)
	fsCmd.AddCommand(fsCpCmd)
	fsCmd.AddCommand(fsInfoCmd)
	fsCmd.AddCommand(fsProvidersCmd)
	fsCmd.AddCommand(fsUrlCmd)

	// Register fs command
	rootCmd.AddCommand(fsCmd)
}

func runFsRead(path string) error {
	parsed, err := parsePath(path)
	if err != nil {
		return apperr.New(`fs_read`, apperr.CodeInvalidArg, fmt.Sprintf(`invalid path: %v`, err))
	}

	ctx := context.Background()
	p, err := provider.Get(ctx, parsed.Scheme)
	if err != nil {
		available := provider.SupportedSchemes()
		return apperr.New(`fs_read`, apperr.CodeNotFound,
			fmt.Sprintf(`provider not found for scheme: %s. Available: %v`, parsed.Scheme, available))
	}

	// Get the path based on scheme
	var filePath string
	switch parsed.Scheme {
	case "file":
		// Sandbox validation for file scheme to prevent path traversal attacks
		resolvedPath, err := sandbox.ResolveReadPath(parsed.Path)
		if err != nil {
			return err // sandbox error already wrapped with proper error code
		}
		filePath = resolvedPath
	case "cephfs":
		filePath = parsed.Path
	default:
		filePath = parsed.Key
	}

	// Check flags in order: bytes, tail, head
	if fsReadBytes > 0 {
		return readWithBytes(ctx, p, filePath, fsReadBytes)
	}
	if fsReadTail > 0 {
		return readWithTail(ctx, p, filePath, fsReadTail)
	}
	if fsReadHead > 0 {
		return readWithHead(ctx, p, filePath, fsReadHead)
	}

	rc, err := p.Read(ctx, filePath)
	if err != nil {
		return apperr.Wrap(`fs_read`, apperr.CodeNotFound, `failed to read file`, err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return apperr.Wrap(`fs_read`, apperr.CodeInternal, `failed to read data`, err)
	}

	content := string(data)
	lineCount := countLines(content)

	result := map[string]interface{}{
		"path":        path,
		"content":     content,
		"line_count":  lineCount,
		"byte_count":  len(data),
		"truncated":   false,
		"slice_type":  "full",
	}

	return output.PrintSuccess("fs_read", result)
}

func readWithHead(ctx context.Context, p provider.StorageProvider, path string, lines int64) error {
	rc, err := p.Read(ctx, path)
	if err != nil {
		return apperr.Wrap(`fs_read`, apperr.CodeNotFound, `failed to read file`, err)
	}
	defer rc.Close()

	// Use scanner for memory-efficient line-by-line reading
	scanner := bufio.NewScanner(rc)
	// Increase buffer to support long lines (default is 64KB)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 10*1024*1024) // up to 10MB per line

	var headLines []string
	for scanner.Scan() {
		headLines = append(headLines, scanner.Text())
		if int64(len(headLines)) >= lines {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return apperr.Wrap(`fs_read`, apperr.CodeInternal, `failed to scan file`, err)
	}

	// Join lines and calculate byte count
	content := strings.Join(headLines, "\n")
	byteCount := len(content)

	result := map[string]interface{}{
		"path":        path,
		"content":     content,
		"line_count":  len(headLines),
		"byte_count":  byteCount,
		"truncated":   false,
		"slice_type":  "head",
	}

	return output.PrintSuccess("fs_read_head", result)
}

func readWithBytes(ctx context.Context, p provider.StorageProvider, path string, bytes int64) error {
	rc, err := p.Read(ctx, path)
	if err != nil {
		return apperr.Wrap(`fs_read`, apperr.CodeNotFound, `failed to read file`, err)
	}
	defer rc.Close()

	// Read up to bytes+1 to detect truncation accurately
	limitedReader := &io.LimitedReader{R: rc, N: bytes + 1}
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return apperr.Wrap(`fs_read`, apperr.CodeInternal, `failed to read data`, err)
	}

	truncated := int64(len(data)) > bytes
	if truncated {
		data = data[:bytes]
	}

	content := string(data)
	byteCount := len(data)

	result := map[string]interface{}{
		"path":        path,
		"content":     content,
		"line_count":  countLines(content),
		"byte_count":  byteCount,
		"truncated":   truncated,
		"slice_type":  "bytes",
	}

	return output.PrintSuccess("fs_read_bytes", result)
}

func readWithTail(ctx context.Context, p provider.StorageProvider, path string, lines int64) error {
	// Try to use seek for efficient tail reading (works for file-based providers)
	if fp, ok := p.(provider.FileProviderInterface); ok {
		f, err := fp.FileHandle(ctx, path)
		if err == nil {
			defer f.Close()
			return readTailWithSeek(f, path, lines)
		}
	}

	// Fallback for cloud storage: read full file and extract tail
	// WARNING: This is memory-intensive for large files (>100MB recommended limit)
	// For cloud storage, consider using provider-specific range requests if available
	rc, err := p.Read(ctx, path)
	if err != nil {
		return apperr.Wrap(`fs_read`, apperr.CodeNotFound, `failed to read file`, err)
	}
	defer rc.Close()

	// Use limited reader to prevent memory exhaustion (limit: 100MB)
	const maxTailFileSize = 100 * 1024 * 1024
	limitedReader := &io.LimitedReader{R: rc, N: maxTailFileSize + 1}
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return apperr.Wrap(`fs_read`, apperr.CodeInternal, `failed to read data`, err)
	}

	// Check if file exceeded size limit
	if int64(len(data)) > maxTailFileSize {
		return apperr.New(`fs_read`, apperr.CodeInvalidArg,
			`tail operation on cloud storage requires reading entire file; file too large (>100MB). Consider downloading file first for local tail operation`)
	}

	content := string(data)
	allLines := strings.Split(content, "\n")

	startIdx := 0
	if len(allLines) > int(lines) {
		startIdx = len(allLines) - int(lines)
	}
	tailLines := allLines[startIdx:]

	result := map[string]interface{}{
		"path":        path,
		"content":     strings.Join(tailLines, "\n"),
		"line_count":  len(tailLines),
		"byte_count":  len(strings.Join(tailLines, "\n")),
		"truncated":   false,
		"slice_type":  "tail",
	}

	return output.PrintSuccess("fs_read_tail", result)
}

// readTailWithSeek uses io.Seek to efficiently read file tail
func readTailWithSeek(f *os.File, path string, lines int64) error {
	// Get file size
	fileSize, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		return apperr.Wrap(`fs_read`, apperr.CodeInternal, `failed to seek file`, err)
	}

	// Estimate buffer size (average line length * lines, with buffer)
	avgLineLen := int64(256)
	bufferSize := avgLineLen * lines * 2
	if bufferSize < 4096 {
		bufferSize = 4096 // minimum buffer
	}
	// Ensure buffer size doesn't exceed file size
	if bufferSize > fileSize {
		bufferSize = fileSize
	}

	// Calculate start position, ensure it's non-negative
	startPos := fileSize - bufferSize
	if startPos < 0 {
		startPos = 0
	}

	_, err = f.Seek(startPos, io.SeekStart)
	if err != nil {
		return apperr.Wrap(`fs_read`, apperr.CodeInternal, `failed to seek file`, err)
	}

	data := make([]byte, bufferSize)
	n, err := f.Read(data)
	if err != nil && err != io.EOF {
		return apperr.Wrap(`fs_read`, apperr.CodeInternal, `failed to read data`, err)
	}
	data = data[:n]

	content := string(data)
	allLines := strings.Split(content, "\n")

	startIdx := 0
	if len(allLines) > int(lines) {
		startIdx = len(allLines) - int(lines)
	}
	tailLines := allLines[startIdx:]

	result := map[string]interface{}{
		"path":        path,
		"content":     strings.Join(tailLines, "\n"),
		"line_count":  len(tailLines),
		"byte_count":  len(strings.Join(tailLines, "\n")),
		"truncated":   startPos > 0,
		"slice_type":  "tail",
	}

	return output.PrintSuccess("fs_read_tail", result)
}

// countLines returns the number of lines in a string
// Empty content returns 0, otherwise counts newline characters + 1
func countLines(content string) int {
	if content == "" {
		return 0
	}
	return strings.Count(content, "\n") + 1
}

func runFsLs(path string) error {
	parsed, err := parsePath(path)
	if err != nil {
		return apperr.New(`fs_ls`, apperr.CodeInvalidArg, fmt.Sprintf(`invalid path: %v`, err))
	}

	ctx := context.Background()
	p, err := provider.Get(ctx, parsed.Scheme)
	if err != nil {
		return apperr.New(`fs_ls`, apperr.CodeNotFound, fmt.Sprintf(`provider not found for scheme: %s`, parsed.Scheme))
	}

	var filePath string
	switch parsed.Scheme {
	case "file":
		// Sandbox validation for file scheme to prevent path traversal attacks
		resolvedPath, err := sandbox.ResolveReadPath(parsed.Path)
		if err != nil {
			return err // sandbox error already wrapped with proper error code
		}
		filePath = resolvedPath
	case "cephfs":
		filePath = parsed.Path
	default:
		filePath = parsed.Key
	}

	files, err := p.List(ctx, filePath)
	if err != nil {
		return apperr.Wrap(`fs_ls`, apperr.CodeNotFound, `failed to list files`, err)
	}

	result := map[string]interface{}{
		"path":  path,
		"files": files,
		"count": len(files),
	}

	return output.PrintSuccess("fs_ls", result)
}

func runFsCp(src, dst string) error {
	srcParsed, err := parsePath(src)
	if err != nil {
		return apperr.New(`fs_cp`, apperr.CodeInvalidArg, fmt.Sprintf(`invalid source path: %v`, err))
	}

	dstParsed, err := parsePath(dst)
	if err != nil {
		return apperr.New(`fs_cp`, apperr.CodeInvalidArg, fmt.Sprintf(`invalid destination path: %v`, err))
	}

	ctx := context.Background()

	// Get source provider
	srcProvider, err := provider.Get(ctx, srcParsed.Scheme)
	if err != nil {
		return apperr.New(`fs_cp`, apperr.CodeNotFound, fmt.Sprintf(`provider not found for scheme: %s`, srcParsed.Scheme))
	}

	// Get destination provider
	dstProvider, err := provider.Get(ctx, dstParsed.Scheme)
	if err != nil {
		return apperr.New(`fs_cp`, apperr.CodeNotFound, fmt.Sprintf(`provider not found for scheme: %s`, dstParsed.Scheme))
	}

	// Get the paths with sandbox validation for file scheme
	var srcPath, dstPath string
	switch srcParsed.Scheme {
	case "file":
		// Sandbox validation for source path (read operation)
		resolvedPath, err := sandbox.ResolveReadPath(srcParsed.Path)
		if err != nil {
			return err // sandbox error already wrapped with proper error code
		}
		srcPath = resolvedPath
	case "cephfs":
		srcPath = srcParsed.Path
	default:
		srcPath = srcParsed.Key
	}
	switch dstParsed.Scheme {
	case "file":
		// Sandbox validation for destination path (write operation)
		resolvedPath, err := sandbox.ResolveWritePath(dstParsed.Path)
		if err != nil {
			return err // sandbox error already wrapped with proper error code
		}
		dstPath = resolvedPath
	case "cephfs":
		dstPath = dstParsed.Path
	default:
		dstPath = dstParsed.Key
	}

	// Check if both providers have identical configuration
	// If so, we can use the provider's native Copy method for better performance
	srcConfig := srcProvider.ConfigInfo()
	dstConfig := dstProvider.ConfigInfo()

	useNativeCopy := srcConfig.Equals(dstConfig)

	if useNativeCopy {
		// Use native copy for better performance (copy happens on server side)
		err := dstProvider.Copy(ctx, dstPath, srcPath)
		if err != nil {
			return apperr.Wrap(`fs_cp`, apperr.CodeInternal, `failed to copy file using native copy`, err)
		}

		result := map[string]interface{}{
			"source":      src,
			"destination": dst,
			"success":     true,
			"method":      "native_copy",
		}

		return output.PrintSuccess("fs_cp", result)
	}

	// Fallback: use read-write method to ensure proper permission control
	// This is needed when providers have different credentials or configurations
	rc, err := srcProvider.Read(ctx, srcPath)
	if err != nil {
		return apperr.Wrap(`fs_cp`, apperr.CodeNotFound, `failed to read source file`, err)
	}
	defer rc.Close()

	// Stream data directly from source to destination provider
	err = dstProvider.Write(ctx, dstPath, rc)
	if err != nil {
		return apperr.Wrap(`fs_cp`, apperr.CodeInternal, `failed to write destination file`, err)
	}

	result := map[string]interface{}{
		"source":      src,
		"destination": dst,
		"success":     true,
		"method":      "read_write",
	}

	return output.PrintSuccess("fs_cp", result)
}


func runFsInfo(path string) error {
	parsed, err := parsePath(path)
	if err != nil {
		return apperr.New(`fs_info`, apperr.CodeInvalidArg, fmt.Sprintf(`invalid path: %v`, err))
	}

	ctx := context.Background()
	p, err := provider.Get(ctx, parsed.Scheme)
	if err != nil {
		return apperr.New(`fs_info`, apperr.CodeNotFound, fmt.Sprintf(`provider not found for scheme: %s`, parsed.Scheme))
	}

	var filePath string
	switch parsed.Scheme {
	case "file":
		// Sandbox validation for file scheme to prevent path traversal attacks
		resolvedPath, err := sandbox.ResolveReadPath(parsed.Path)
		if err != nil {
			return err // sandbox error already wrapped with proper error code
		}
		filePath = resolvedPath
	case "cephfs":
		filePath = parsed.Path
	default:
		filePath = parsed.Key
	}

	info, err := p.Stat(ctx, filePath)
	if err != nil {
		return apperr.Wrap(`fs_info`, apperr.CodeNotFound, `failed to get file info`, err)
	}

	result := map[string]interface{}{
		"path":          path,
		"name":         info.Name,
		"size":         info.Size,
		"is_dir":       info.IsDir,
		"last_modified": info.LastModified,
	}

	return output.PrintSuccess("fs_info", result)
}

func runFsProviders() error {
	schemes := provider.SupportedSchemes()
	
	// Get provider info from cloud package
	cloudProviders := cloud.GetProviders()
	
	providerList := make([]map[string]any, 0, len(schemes))
	for _, scheme := range schemes {
		info := map[string]any{
			"scheme": scheme,
		}
		// Try to find matching cloud provider
		for _, cp := range cloudProviders {
			if strings.EqualFold(cp.Name, scheme) {
				info["name"] = cp.Name
				info["description"] = cp.Description
				break
			}
		}
		providerList = append(providerList, info)
	}
	
	return output.PrintSuccess("fs_providers", map[string]any{
		"providers": providerList,
		"note":      "All S3-compatible storage is supported. Configure via environment variables.",
	})
}

func runFsURL(pathArg string) error {
	parsed, err := parsePath(pathArg)
	if err != nil {
		return apperr.New(`fs_url`, apperr.CodeInvalidArg, fmt.Sprintf(`invalid path: %v`, err))
	}
	
	// Only cloud storage supports URL generation
	switch parsed.Scheme {
	case "file", "cephfs":
		return apperr.New(`fs_url`, apperr.CodeInvalidArg, `URL generation is not supported for local filesystem`)
	}
	
	// Load provider config
	providerName := resolveFsProvider(parsed.Scheme)
	cfg, err := loadFsProviderConfig(providerName)
	if err != nil {
		return err
	}
	
	cloudProvider, err := cloud.NewS3CompatibleProvider(providerName, cfg)
	if err != nil {
		return apperr.Wrap(`fs_url`, apperr.CodeProvider, `failed to create provider`, err)
	}
	
	dispatcher := cloud.NewDispatcher(map[string]cloud.Provider{
		providerName: cloudProvider,
	})
	
	result, err := dispatcher.URL(context.Background(), providerName, cloud.URLRequest{
		RemoteKey:  parsed.Key,
		Expiration: fsURLExpires,
		PublicOnly: fsURLPublic,
	})
	if err != nil {
		return apperr.Wrap(`fs_url`, apperr.CodeInternal, `failed to generate URL`, err)
	}

	// Warn user about public URL security implications
	if fsURLPublic {
		warnFsPublicURL(providerName)
	}

	return output.PrintSuccess("fs_url", map[string]any{
		"provider":     result.Provider,
		"remote_key":   result.RemoteKey,
		"url":          result.URL,
		"expires_in":   result.ExpiresIn,
		"expires_at":   result.ExpiresAt,
		"is_presigned": result.IsPresigned,
	})
}

func resolveFsProvider(scheme string) string {
	return strings.ToLower(scheme)
}

func loadFsProviderConfig(providerName string) (s3client.Config, error) {
	prefixes := []string{
		fmt.Sprintf("providers.%s", strings.ToLower(providerName)),
		strings.ToLower(providerName),
		"providers.s3",
		"s3",
	}

	cfg := s3client.Config{
		Endpoint:        pickStringFs(prefixes, "endpoint"),
		Region:          pickStringFs(prefixes, "region"),
		Bucket:          pickStringFs(prefixes, "bucket"),
		AccessKeyID:     pickStringFs(prefixes, "access_key_id", "accesskeyid", "access_key", "accesskey"),
		SecretAccessKey: pickStringFs(prefixes, "secret_access_key", "access_key_secret", "secretkey", "secret_key"),
		PathPrefix:      pickStringFs(prefixes, "path", "path_prefix"),
		CDNHost:         pickStringFs(prefixes, "cdn_host", "domain"),
		PathStyle:       pickBoolFs(prefixes, false, "path_style", "pathstyle"),
		UseSSL:          pickBoolFs(prefixes, true, "use_ssl"),
	}

	if strings.EqualFold(providerName, "r2") && cfg.Endpoint == "" {
		accountID := pickStringFs(prefixes, "account_id", "accountid")
		if accountID != "" {
			cfg.Endpoint = fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)
		}
	}
	if cfg.Bucket == "" {
		return s3client.Config{}, apperr.New("fs_url", apperr.CodeConfig, "missing provider bucket configuration")
	}
	if cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" {
		return s3client.Config{}, apperr.New("fs_url", apperr.CodeConfig, "missing provider access key configuration")
	}
	return cfg, nil
}

func pickStringFs(prefixes []string, keys ...string) string {
	for _, prefix := range prefixes {
		for _, key := range keys {
			fullKey := prefix + "." + key
			if value := strings.TrimSpace(viper.GetString(fullKey)); value != "" {
				return value
			}
		}
	}
	return ""
}

func pickBoolFs(prefixes []string, defaultValue bool, keys ...string) bool {
	for _, prefix := range prefixes {
		for _, key := range keys {
			fullKey := prefix + "." + key
			if viper.IsSet(fullKey) {
				return viper.GetBool(fullKey)
			}
		}
	}
	return defaultValue
}

func warnFsPublicURL(provider string) {
	msg := `
⚠️  SECURITY WARNING: You are using --public flag

The generated URL will be publicly accessible without authentication.
This is only suitable for:
  • Public website assets (images, CSS, JS)
  • Public download files
  • Files intended for public sharing

DO NOT use --public for:
  • Private configuration files
  • Sensitive data or logs
  • Any files requiring access control

For Cloudflare R2: Public Access must be enabled in your bucket settings.
`
	fmt.Fprint(os.Stderr, msg)
}
