package uri

import (
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/jhonxue/agent-fs/pkg/config"
	"github.com/jhonxue/agent-fs/pkg/provider"
)

// SupportedSchemes returns all URI schemes that have registered providers
func SupportedSchemes() []string {
	return provider.SupportedSchemes()
}

// URI represents a parsed storage URI with scheme-based routing
type URI struct {
	Scheme  string // "file", "s3", "r2", "minio", etc.
	Host    string // Hostname (e.g., "s3.amazonaws.com")
	Port    int    // Port number (0 if not specified)
	Bucket  string // For cloud storage: bucket name
	Key     string // For cloud storage: object key
	Path    string // For local storage: file path
	Query   string // Optional query string
	IsVHost bool   // true for vhost style (bucket.endpoint), false for path style (endpoint/bucket)
	Alias   string // Provider alias from config file (e.g., "mys3", "myoss")
}

// Parse parses a URI string into a URI struct.
// Supported formats:
//   - file:///absolute/path or file://relative/path
//   - s3://bucket/key
//   - r2://bucket/key
//   - minio://bucket/key
//   - /absolute/path (defaults to file://)
//   - relative/path (defaults to file://)
func Parse(raw string) (*URI, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("empty URI")
	}

	// Handle paths without scheme
	if !strings.Contains(raw, "://") {
		return parseLocalPath(raw)
	}

	// Parse URI with scheme
	parts := strings.SplitN(raw, "://", 2)
	scheme := parts[0]
	rest := parts[1]

	switch scheme {
	case "file":
		return parseFileURI(rest)
	case "s3", "r2", "minio", "oss", "cos":
		return parseCloudURI(scheme, rest)
	case "cephfs":
		return parseCephFSURI(rest)
	case "hdfs":
		return parseHDFSURI(rest)
	case "gcs", "azure", "ceph":
		return nil, fmt.Errorf("scheme %q not yet implemented: registry pending provider registration", scheme)
	default:
		return nil, fmt.Errorf("unsupported scheme: %s", scheme)
	}
}

// ParseWithEndpoint parses a full URL with endpoint and extracts host, port, bucket, key.
// Supported formats:
//   - https://bucket.s3.amazonaws.com/key (VHost style)
//   - https://s3.amazonaws.com/bucket/key (Path style)
//   - http://localhost:9000/bucket/key
//   - s3://bucket/key (backward compatible)
func ParseWithEndpoint(raw string) (*URI, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("empty URL")
	}

	// If no scheme, treat as relative path
	if !strings.Contains(raw, "://") {
		return parseLocalPath(raw)
	}

	// Check if it's a full URL (http/https)
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return parseFullURL(raw)
	}

	// Fall back to original parse for scheme://bucket/key format
	return Parse(raw)
}

// ParseWithConfig parses a URI and matches it with provider configuration.
// It validates the URL information against the configured provider.
// If config is nil, it behaves like ParseWithEndpoint.
func ParseWithConfig(raw string, cfg *config.Config) (*URI, error) {
	// First, parse the URI
	uri, err := ParseWithEndpoint(raw)
	if err != nil {
		return nil, err
	}

	// If no config, return as-is
	if cfg == nil || !cfg.HasProviders() {
		return uri, nil
	}

	scheme := uri.Scheme
	urlBucket := uri.Bucket
	urlEndpoint := uri.Host
	if uri.Port > 0 {
		urlEndpoint = fmt.Sprintf("%s:%d", urlEndpoint, uri.Port)
	}

	// Check if scheme matches a configured alias (aliases take priority)
	if providerCfg, ok := cfg.GetProvider(scheme); ok {
		// Alias takes priority over automatic detection
		uri.Alias = scheme
		// Validate URL info against config
		if err := config.MatchURLWithConfig(providerCfg, urlBucket, urlEndpoint); err != nil {
			return nil, fmt.Errorf("configuration mismatch: %w", err)
		}
		// Use config's bucket/endpoint if URL doesn't have them
		if uri.Bucket == "" && providerCfg.Bucket != "" {
			uri.Bucket = providerCfg.Bucket
		}
		if uri.Host == "" && providerCfg.Endpoint != "" {
			uri.Host = providerCfg.Endpoint
		}
		return uri, nil
	}

	// If scheme is not an alias, try to match by endpoint
	if providerCfg, alias := cfg.GetProviderByBucketAndEndpoint(urlBucket, urlEndpoint); providerCfg != nil {
		uri.Alias = alias
		uri.Scheme = providerCfg.Type
		// Use config's values to fill in missing info
		if uri.Bucket == "" && providerCfg.Bucket != "" {
			uri.Bucket = providerCfg.Bucket
		}
		return uri, nil
	}

	// No matching config, return parsed URI as-is (maybe it's a legacy URL)
	return uri, nil
}

// parseFullURL parses a full HTTP/HTTPS URL
func parseFullURL(raw string) (*URI, error) {
	// Use net/url for parsing
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	// Determine scheme based on hostname patterns
	scheme := detectSchemeFromHost(u.Host)
	if scheme == "" {
		return nil, fmt.Errorf("unsupported endpoint: %s", u.Host)
	}

	host := u.Hostname()
	port := 0
	if p := u.Port(); p != "" {
		port, err = strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("invalid port: %w", err)
		}
	}

	// Detect vhost vs path style
	path := u.Path
	var bucket, key string
	var isVHost bool

	if isVHostStyle(host, scheme) {
		// VHost style: bucket is part of hostname
		bucket = extractBucketFromHost(host, scheme)
		key = strings.TrimPrefix(path, "/")
		isVHost = true
	} else {
		// Path style: /bucket/key
		parts := strings.SplitN(strings.TrimPrefix(path, "/"), "/", 2)
		bucket = parts[0]
		key = ""
		if len(parts) > 1 {
			key = parts[1]
		}
		isVHost = false
	}

	if bucket == "" {
		return nil, fmt.Errorf("missing bucket in URL: %s", raw)
	}

	// Handle query string
	if key != "" {
		if idx := strings.Index(key, "?"); idx != -1 {
			key = key[:idx]
		}
	}

	return &URI{
		Scheme:  scheme,
		Host:    host,
		Port:    port,
		Bucket:  bucket,
		Key:     key,
		IsVHost: isVHost,
	}, nil
}

// detectSchemeFromHost detects the scheme (s3, r2, oss, cos, minio) from hostname
func detectSchemeFromHost(host string) string {
	hostLower := strings.ToLower(host)

	// AWS S3 - check exact match and specific regional patterns
	// Priority: exact match > .s3.amazonaws.com > .amazonaws.com.cn
	if hostLower == "s3.amazonaws.com" ||
		strings.HasSuffix(hostLower, ".s3.amazonaws.com") ||
		strings.HasSuffix(hostLower, ".amazonaws.com.cn") {
		return "s3"
	}

	// Cloudflare R2 - exact match and specific patterns
	if strings.HasSuffix(hostLower, ".r2.cloudflarestorage.com") {
		return "r2"
	}

	// Aliyun OSS - specific patterns only (bucket.oss-cn-{region}.aliyuncs.com)
	// Must have .oss-cn- in host to avoid matching unrelated domains
	if strings.Contains(hostLower, ".oss-cn-") && strings.HasSuffix(hostLower, ".aliyuncs.com") {
		return "oss"
	}

	// Tencent COS - specific patterns only (bucket.cos.{region}.myqcloud.com)
	// Must have cos. prefix or .cos. in host to avoid matching unrelated domains
	if (strings.HasPrefix(hostLower, "cos.") || strings.Contains(hostLower, ".cos.")) && strings.HasSuffix(hostLower, ".myqcloud.com") {
		return "cos"
	}

	// MinIO (custom endpoints) - default for unknown
	// This handles cases like: localhost:9000, minio.example.com, custom S3-compatible services
	return "minio"
}

// isVHostStyle checks if the URL uses vhost style (bucket.endpoint)
func isVHostStyle(host, scheme string) bool {
	hostLower := strings.ToLower(host)

	// If host is an IP address, it's not vhost style
	if isIPAddress(host) {
		return false
	}

	// VHost style patterns for each provider
	switch scheme {
	case "s3":
		// bucket.s3.amazonaws.com
		if strings.Contains(hostLower, ".s3.amazonaws.com") {
			return true
		}
	case "r2":
		// bucket.r2.cloudflarestorage.com
		if strings.Contains(hostLower, ".r2.cloudflarestorage.com") {
			return true
		}
	case "oss":
		// bucket.oss-cn-region.aliyuncs.com
		if strings.Contains(hostLower, ".oss-cn-") && strings.Contains(hostLower, ".aliyuncs.com") {
			return true
		}
	case "cos":
		// bucket.cos.ap-region.myqcloud.com
		if strings.Contains(hostLower, ".cos.") && strings.Contains(hostLower, ".myqcloud.com") {
			return true
		}
	case "minio":
		// For custom endpoints, check if first part looks like a bucket
		// bucket.minio.example.com
		parts := strings.SplitN(host, ".", 2)
		if len(parts) > 1 && !strings.Contains(parts[0], ":") {
			return true
		}
	}

	return false
}

// isIPAddress checks if the host is an IP address (IPv4 or IPv6)
func isIPAddress(host string) bool {
	// Check for IPv4
	parts := strings.Split(host, ".")
	if len(parts) == 4 {
		for _, p := range parts {
			if _, err := strconv.Atoi(p); err != nil {
				return false
			}
		}
		return true
	}
	// Check for IPv6 (simplified check)
	return strings.Contains(host, ":")
}

// extractBucketFromHost extracts bucket name from vhost style hostname
func extractBucketFromHost(host, scheme string) string {
	switch scheme {
	case "s3":
		// bucket.s3.amazonaws.com -> bucket
		parts := strings.SplitN(host, ".", 3)
		if len(parts) >= 3 {
			return parts[0]
		}
	case "r2":
		// bucket.r2.cloudflarestorage.com -> bucket
		parts := strings.SplitN(host, ".", 3)
		if len(parts) >= 3 {
			return parts[0]
		}
	case "oss":
		// bucket.oss-cn-hangzhou.aliyuncs.com -> bucket
		parts := strings.SplitN(host, ".", 4)
		if len(parts) >= 4 {
			return parts[0]
		}
	case "cos":
		// bucket.cos.ap-guangzhou.myqcloud.com -> bucket
		parts := strings.SplitN(host, ".", 4)
		if len(parts) >= 4 {
			return parts[0]
		}
	case "minio":
		// bucket.minio.example.com -> bucket
		parts := strings.SplitN(host, ".", 2)
		if len(parts) >= 2 {
			return parts[0]
		}
	}

	return ""
}

// parseLocalPath handles paths without scheme, defaulting to file://
func parseLocalPath(raw string) (*URI, error) {
	return &URI{
		Scheme: "file",
		Path:   raw,
	}, nil
}

// parseFileURI handles file:// URIs
func parseFileURI(rest string) (*URI, error) {
	// file:///tmp/test.txt = absolute path /tmp/test.txt
	// file://host/path      = host is ignored by design -> /path
	// file://relative/path  = NOT recommended; prefer path without scheme. If provided, it's treated as host+path and the host is ignored.

	// Handle empty rest (file://)
	if rest == "" {
		return &URI{
			Scheme: "file",
			Path:   "",
		}, nil
	}

	// If starts with slash, it's an absolute path (normalize leading slashes)
	if strings.HasPrefix(rest, "/") {
		// Count leading slashes
		i := 0
		for i < len(rest) && rest[i] == '/' {
			i++
		}
		if i >= 2 {
			// file:///path -> absolute /path
			return &URI{Scheme: "file", Path: "/" + rest[i:]}, nil
		}
		// e.g., "/path"
		return &URI{Scheme: "file", Path: rest}, nil
	}

	// No leading slash: treat as authority+path if a slash exists (drop the authority/host)
	if idx := strings.IndexByte(rest, '/'); idx >= 0 {
		// Ignore host, enforce absolute path
		return &URI{Scheme: "file", Path: "/" + rest[idx+1:]}, nil
	}

	// No slash at all -> treat as relative path string
	return &URI{Scheme: "file", Path: rest}, nil
}

// parseCloudURI handles cloud storage URIs (s3://, r2://, etc.)
func parseCloudURI(scheme, rest string) (*URI, error) {
	// Split bucket and key
	parts := strings.SplitN(rest, "/", 2)
	bucket := parts[0]
	if bucket == "" {
		return nil, fmt.Errorf("missing bucket in %s URI", scheme)
	}
	key := ""
	if len(parts) > 1 {
		key = parts[1]
	}

	// Handle query string
	if key != "" {
		if idx := strings.Index(key, "?"); idx != -1 {
			key = key[:idx]
		}
	}

	return &URI{
		Scheme: scheme,
		Bucket: bucket,
		Key:    key,
	}, nil
}

// parseCephFSURI handles CephFS URIs (cephfs://path)
// CephFS doesn't have buckets, so the entire path is the key
func parseCephFSURI(rest string) (*URI, error) {
	// Ensure path starts with /
	key := rest
	if key != "" && !strings.HasPrefix(key, "/") {
		key = "/" + key
	}

	// Handle query string
	if key != "" {
		if idx := strings.Index(key, "?"); idx != -1 {
			key = key[:idx]
		}
	}

	return &URI{
		Scheme: "cephfs",
		Path:   key,
		Key:    key,
	}, nil
}

// parseHDFSURI handles HDFS URIs (hdfs://namenode:port/path)
// HDFS uses a Namenode address and path, similar to CephFS
func parseHDFSURI(rest string) (*URI, error) {
	// Parse format: namenode:port/path or just /path
	var host string
	var port int
	var key string

	// Check if there's a host:port prefix
	if idx := strings.Index(rest, "/"); idx != -1 {
		// Has path component
		hostPort := rest[:idx]
		key = rest[idx:]
		if idx2 := strings.Index(hostPort, ":"); idx2 != -1 {
			host = hostPort[:idx2]
			var err error
			port, err = strconv.Atoi(hostPort[idx2+1:])
			if err != nil {
				return nil, fmt.Errorf("invalid port in HDFS URI: %w", err)
			}
		} else {
			host = hostPort
		}
	} else {
		// No path, treat entire rest as host:port or path
		if idx := strings.Index(rest, ":"); idx != -1 {
			host = rest[:idx]
			var err error
			port, err = strconv.Atoi(rest[idx+1:])
			if err != nil {
				return nil, fmt.Errorf("invalid port in HDFS URI: %w", err)
			}
		} else {
			// Just a path, no host
			key = rest
		}
	}

	// Ensure path starts with /
	if key != "" && !strings.HasPrefix(key, "/") {
		key = "/" + key
	}

	// Handle query string
	if key != "" {
		if idx := strings.Index(key, "?"); idx != -1 {
			key = key[:idx]
		}
	}

	return &URI{
		Scheme: "hdfs",
		Host:   host,
		Port:   port,
		Path:   key,
		Key:    key,
	}, nil
}

// IsLocal returns true if the URI points to local filesystem
func (u *URI) IsLocal() bool {
	return u.Scheme == "file"
}

// IsCloud returns true if the URI points to cloud storage
func (u *URI) IsCloud() bool {
	return !u.IsLocal()
}

// String returns the URI as a string
func (u *URI) String() string {
	switch u.Scheme {
	case "file":
		// Ensure proper format: file:///absolute or file://relative
		// Absolute paths should have exactly 3 slashes total
		if strings.HasPrefix(u.Path, "/") {
			return "file:///" + strings.TrimPrefix(u.Path, "/")
		}
		return "file://" + u.Path
	case "cephfs":
		return "cephfs://" + u.Path
	case "s3", "r2", "minio", "gcs", "azure", "ceph", "oss", "cos":
		if u.Key != "" {
			return fmt.Sprintf("%s://%s/%s", u.Scheme, u.Bucket, u.Key)
		}
		return fmt.Sprintf("%s://%s", u.Scheme, u.Bucket)
	default:
		return fmt.Sprintf("%s://%s/%s", u.Scheme, u.Bucket, u.Key)
	}
}

// BaseName returns the base name of the URI (file name or object key)
func (u *URI) BaseName() string {
	switch u.Scheme {
	case "file", "cephfs":
		return path.Base(u.Path)
	default:
		return path.Base(u.Key)
	}
}

// DirName returns the directory portion of the URI
func (u *URI) DirName() string {
	switch u.Scheme {
	case "file", "cephfs":
		return path.Dir(u.Path)
	default:
		return path.Dir(u.Key)
	}
}

// Join joins the URI with additional path components
func (u *URI) Join(extra ...string) string {
	parts := append([]string{u.String()}, extra...)
	return strings.Join(parts, "/")
}
