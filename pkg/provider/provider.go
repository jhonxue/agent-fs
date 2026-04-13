package provider

import (
	"context"
	"fmt"
	"io"
	"time"
)

// FileInfo represents metadata about a file or object
type FileInfo struct {
	Name         string    `json:"name"`           // File name
	Size         int64     `json:"size"`           // Size in bytes
	IsDir        bool      `json:"is_dir"`         // Is this a directory
	LastModified time.Time `json:"last_modified"`  // Last modification time
	ETag         string    `json:"etag,omitempty"` // For cloud storage: ETag
	Path         string    `json:"path"`           // Full path
}

// providerConfigError is an error for missing provider configuration
type providerConfigError struct {
	msg string
}

func (e *providerConfigError) Error() string {
	return e.msg
}

// notImplementedError is an error for methods not yet implemented
type notImplementedError struct {
	msg string
}

func (e *notImplementedError) Error() string {
	return e.msg
}

// mockReader implements io.Reader for testing
type mockReader struct {
	data []byte
	pos  int
}

func (r *mockReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

// ProviderConfigInfo contains configuration info for comparison
type ProviderConfigInfo struct {
	Scheme      string
	Bucket      string
	Endpoint    string
	AccessKey   string
	SecretKey   string
	PathStyle   bool
	UseSSL      bool
}

// Equals compares two ProviderConfigInfo instances for equality
// Returns true if all fields are identical
func (c ProviderConfigInfo) Equals(other ProviderConfigInfo) bool {
	return c.Scheme == other.Scheme &&
		c.Bucket == other.Bucket &&
		c.Endpoint == other.Endpoint &&
		c.AccessKey == other.AccessKey &&
		c.SecretKey == other.SecretKey &&
		c.PathStyle == other.PathStyle &&
		c.UseSSL == other.UseSSL
}

// SafeString returns a string representation without sensitive fields
// Use this for logging and debugging to avoid credential exposure
func (c ProviderConfigInfo) SafeString() string {
	return fmt.Sprintf("scheme=%s, bucket=%s, endpoint=%s, pathStyle=%v, useSSL=%v",
		c.Scheme, c.Bucket, c.Endpoint, c.PathStyle, c.UseSSL)
}

// StorageProvider is the unified interface for all storage backends
type StorageProvider interface {
	// Scheme returns the URI scheme this provider handles (e.g., "file", "s3", "r2")
	Scheme() string

	// Read reads a file and returns its contents
	// path is the key (for cloud) or path (for local)
	Read(ctx context.Context, path string) (io.ReadCloser, error)

	// Write writes data to a file
	Write(ctx context.Context, path string, data io.Reader) error

	// Delete deletes a file
	Delete(ctx context.Context, path string) error

	// List lists files in a directory/prefix
	List(ctx context.Context, path string) ([]FileInfo, error)

	// Stat returns metadata about a file
	Stat(ctx context.Context, path string) (*FileInfo, error)

	// Exists checks if a file exists
	Exists(ctx context.Context, path string) (bool, error)

	// Copy copies a file from source to destination within the same provider
	Copy(ctx context.Context, srcPath, dstPath string) error

	// ConfigInfo returns the provider configuration for comparison
	// Used to determine if two providers are equivalent (same scheme, bucket, endpoint, credentials)
	ConfigInfo() ProviderConfigInfo
}

// ReadCloser combines io.Reader and io.Closer for streaming file data.
// This struct embeds both interfaces and provides a nil-safe Close method.
type ReadCloser struct {
	io.Reader
	io.Closer
}

// Close closes the reader, handling nil Closer gracefully.
// This method shadows the embedded io.Closer to provide nil-safe behavior:
// if Closer field is nil, Close returns nil instead of panicking.
func (rc *ReadCloser) Close() error {
	if rc.Closer == nil {
		return nil
	}
	return rc.Closer.Close()
}
