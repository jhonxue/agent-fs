package uri

import (
	"fmt"
	"path"
	"strings"

	"github.com/geekjourneyx/agent-fs/pkg/provider"
)

// SupportedSchemes returns all URI schemes that have registered providers
func SupportedSchemes() []string {
	return provider.SupportedSchemes()
}

// URI represents a parsed storage URI with scheme-based routing
type URI struct {
	Scheme  string // "file", "s3", "r2", "minio", etc.
	Bucket  string // For cloud storage: bucket name
	Key     string // For cloud storage: object key
	Path    string // For local storage: file path
	Query   string // Optional query string
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
	case "gcs", "azure", "ceph":
		return nil, fmt.Errorf("scheme %q not yet implemented: registry pending provider registration", scheme)
	default:
		return nil, fmt.Errorf("unsupported scheme: %s", scheme)
	}
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
